package cursor

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// fastWatch keeps the reconnect and poll timings short enough for tests.
func fastWatch(onEvent func(Event)) WatchOptions {
	return WatchOptions{
		OnEvent:        onEvent,
		PollInterval:   time.Millisecond,
		ReconnectDelay: time.Millisecond,
	}
}

// A run that has already finished should be returned straight from GetRun,
// without opening a stream at all.
func TestWatchRunReturnsImmediatelyForTerminalRun(t *testing.T) {
	var streamCalls atomic.Int32

	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/agents/bc-1/runs/run-1/stream" {
			streamCalls.Add(1)
		}
		_, _ = w.Write([]byte(`{"id":"run-1","agentId":"bc-1","status":"FINISHED","durationMs":1500,"result":"done"}`))
	})

	run, err := client.WatchRun(context.Background(), "bc-1", "run-1", fastWatch(nil))
	if err != nil {
		t.Fatalf("WatchRun: %v", err)
	}
	if run.Status != RunStatusFinished || run.Result != "done" {
		t.Errorf("run = %+v", run)
	}
	if got := streamCalls.Load(); got != 0 {
		t.Errorf("opened %d streams for an already-finished run, want 0", got)
	}
}

func TestWatchRunStreamsToCompletion(t *testing.T) {
	var runCalls atomic.Int32

	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/agents/bc-1/runs/run-1":
			// First call reports the run in flight; later calls report it done.
			if runCalls.Add(1) == 1 {
				_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING"}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"run-1","status":"FINISHED","durationMs":2000,"result":"all green"}`))
		case "/v1/agents/bc-1/runs/run-1/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("id: 1\nevent: status\ndata: {\"status\":\"RUNNING\"}\n\n" +
				"id: 2\nevent: assistant\ndata: {\"text\":\"working\"}\n\n" +
				"id: 3\nevent: result\ndata: {\"status\":\"FINISHED\",\"text\":\"all green\"}\n\n" +
				"id: 4\nevent: done\ndata: {}\n\n"))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})

	var seen []string
	run, err := client.WatchRun(context.Background(), "bc-1", "run-1", fastWatch(func(e Event) {
		seen = append(seen, e.Type)
	}))
	if err != nil {
		t.Fatalf("WatchRun: %v", err)
	}
	if run.Status != RunStatusFinished {
		t.Errorf("status = %q, want FINISHED", run.Status)
	}
	if run.Result != "all green" {
		t.Errorf("result = %q", run.Result)
	}
	if len(seen) != 4 {
		t.Errorf("saw events %v, want 4", seen)
	}
}

func TestWatchRunRefreshesGitMetadataWhileStreaming(t *testing.T) {
	var runCalls atomic.Int32

	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/agents/bc-1/runs/run-1":
			switch runCalls.Add(1) {
			case 1:
				_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING"}`))
			case 2:
				_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING","git":{"branches":[{"branch":"cursor/live-update"}]}}`))
			default:
				_, _ = w.Write([]byte(`{"id":"run-1","status":"FINISHED","git":{"branches":[{"branch":"cursor/live-update"}]}}`))
			}
		case "/v1/agents/bc-1/runs/run-1/stream":
			w.Header().Set("Content-Type", "text/event-stream")
			time.Sleep(30 * time.Millisecond)
			_, _ = w.Write([]byte("event: result\ndata: {\"status\":\"FINISHED\"}\n\nevent: done\ndata: {}\n\n"))
		}
	})

	var sawLiveBranch atomic.Bool
	opts := fastWatch(nil)
	opts.SnapshotInterval = time.Millisecond
	opts.OnSnapshot = func(run *Run) {
		if run.Status == RunStatusRunning && run.Git != nil && len(run.Git.Branches) > 0 && run.Git.Branches[0].Branch == "cursor/live-update" {
			sawLiveBranch.Store(true)
		}
	}

	run, err := client.WatchRun(context.Background(), "bc-1", "run-1", opts)
	if err != nil {
		t.Fatalf("WatchRun: %v", err)
	}
	if run.Status != RunStatusFinished {
		t.Errorf("status = %q, want FINISHED", run.Status)
	}
	if !sawLiveBranch.Load() {
		t.Error("did not receive branch metadata while the SSE stream was still running")
	}
}

// When the stream's retention window has passed the run is still fine, so the
// wait must switch to polling rather than fail.
func TestWatchRunFallsBackToPollingWhenStreamExpired(t *testing.T) {
	var runCalls atomic.Int32

	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/agents/bc-1/runs/run-1":
			if runCalls.Add(1) < 3 {
				_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING"}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"run-1","status":"ERROR"}`))
		case "/v1/agents/bc-1/runs/run-1/stream":
			w.WriteHeader(http.StatusGone)
			_, _ = w.Write([]byte(`{"code":"stream_expired","message":"gone"}`))
		}
	})

	var notices []string
	opts := fastWatch(nil)
	opts.OnNotice = func(msg string) { notices = append(notices, msg) }

	run, err := client.WatchRun(context.Background(), "bc-1", "run-1", opts)
	if err != nil {
		t.Fatalf("WatchRun: %v", err)
	}
	if run.Status != RunStatusError {
		t.Errorf("status = %q, want ERROR", run.Status)
	}
	if len(notices) == 0 {
		t.Error("expected a notice explaining the fallback to polling")
	}
}

// A dropped connection mid-run must reconnect and resume from the last event
// id, not restart or give up.
func TestWatchRunReconnectsWithLastEventID(t *testing.T) {
	var (
		streamCalls   atomic.Int32
		lastEventIDs  []string
		runCallsCount atomic.Int32
	)

	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/agents/bc-1/runs/run-1":
			if runCallsCount.Add(1) <= 2 {
				_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING"}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"run-1","status":"FINISHED"}`))
		case "/v1/agents/bc-1/runs/run-1/stream":
			lastEventIDs = append(lastEventIDs, r.Header.Get("Last-Event-ID"))
			w.Header().Set("Content-Type", "text/event-stream")
			if streamCalls.Add(1) == 1 {
				// Truncated: events, then the connection ends with no result.
				_, _ = w.Write([]byte("id: 5\nevent: assistant\ndata: {\"text\":\"partial\"}\n\n"))
				return
			}
			_, _ = w.Write([]byte("id: 6\nevent: result\ndata: {\"status\":\"FINISHED\"}\n\nid: 7\nevent: done\ndata: {}\n\n"))
		}
	})

	run, err := client.WatchRun(context.Background(), "bc-1", "run-1", fastWatch(nil))
	if err != nil {
		t.Fatalf("WatchRun: %v", err)
	}
	if run.Status != RunStatusFinished {
		t.Errorf("status = %q, want FINISHED", run.Status)
	}
	if len(lastEventIDs) < 2 {
		t.Fatalf("stream opened %d times, want at least 2", len(lastEventIDs))
	}
	if lastEventIDs[0] != "" {
		t.Errorf("first Last-Event-ID = %q, want empty", lastEventIDs[0])
	}
	if lastEventIDs[1] != "5" {
		t.Errorf("resumed with Last-Event-ID %q, want 5", lastEventIDs[1])
	}
}

// A bad key is not something reconnecting will fix.
func TestWatchRunGivesUpOnNonRetryableStreamError(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/agents/bc-1/runs/run-1" {
			_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"unauthorized","message":"Invalid API key"}`))
	})

	_, err := client.WatchRun(context.Background(), "bc-1", "run-1", fastWatch(nil))
	if err == nil {
		t.Fatal("expected an error for a 401 on the stream")
	}
	apiErr, ok := errType(err)
	if !ok || !apiErr.IsAuth() {
		t.Errorf("error = %v, want an auth APIError", err)
	}
}

// Detaching (ctrl-c) must return promptly rather than block on the poll timer.
func TestWatchRunHonoursContextCancellation(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/agents/bc-1/runs/run-1" {
			_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING"}`))
			return
		}
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`{"code":"stream_expired"}`))
	})

	ctx, cancel := context.WithCancel(context.Background())
	opts := fastWatch(nil)
	opts.PollInterval = 50 * time.Millisecond
	opts.OnNotice = func(string) { cancel() }

	done := make(chan error, 1)
	go func() {
		_, err := client.WatchRun(ctx, "bc-1", "run-1", opts)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected a context error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WatchRun did not return after cancellation")
	}
}

func TestWatchOptionsBackoffDoublesAndCaps(t *testing.T) {
	opts := WatchOptions{ReconnectDelay: time.Second}
	for attempt, want := range map[int]time.Duration{0: time.Second, 1: 2 * time.Second, 3: 8 * time.Second, 10: maxBackoff} {
		if got := opts.backoff(attempt); got != want {
			t.Errorf("backoff(%d) = %v, want %v", attempt, got, want)
		}
	}
	if got := (WatchOptions{}).backoff(0); got != DefaultReconnectDelay {
		t.Errorf("default backoff = %v, want %v", got, DefaultReconnectDelay)
	}
}

// Guard the exact status→behaviour contract that scripts depend on.
func TestWatchRunReturnsEachTerminalStatus(t *testing.T) {
	for _, status := range []string{RunStatusFinished, RunStatusError, RunStatusCancelled, RunStatusExpired} {
		t.Run(status, func(t *testing.T) {
			client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"id":"run-1","status":%q}`, status)
			})
			run, err := client.WatchRun(context.Background(), "bc-1", "run-1", fastWatch(nil))
			if err != nil {
				t.Fatalf("WatchRun: %v", err)
			}
			if run.Status != status {
				t.Errorf("status = %q, want %q", run.Status, status)
			}
		})
	}
}
