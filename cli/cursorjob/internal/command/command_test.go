package command

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/benfdking/sdsf/cli/cursorjob/internal/cursor"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// harness is an App wired to an in-process fake API, with its output captured.
type harness struct {
	app    *App
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func newHarness(t *testing.T, handler http.HandlerFunc) *harness {
	t.Helper()
	httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		recorder := httptest.NewRecorder()
		handler(recorder, req)
		resp := recorder.Result()
		resp.Request = req
		return resp, nil
	})}

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	return &harness{
		app: &App{
			Stdout: stdout,
			Stderr: stderr,
			Getenv: func(string) string { return "" },
			Client: cursor.NewClient("https://api.cursor.test", "test", httpClient),
		},
		stdout: stdout,
		stderr: stderr,
	}
}

func (h *harness) run(args ...string) int {
	return h.app.Execute(context.Background(), args)
}

func TestExecuteUnknownCommandIsUsageError(t *testing.T) {
	h := newHarness(t, func(http.ResponseWriter, *http.Request) {})

	if code := h.run("frobnicate"); code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(h.stderr.String(), "unknown command") {
		t.Errorf("stderr = %q", h.stderr)
	}
}

func TestExecuteWithNoArgsPrintsUsage(t *testing.T) {
	h := newHarness(t, func(http.ResponseWriter, *http.Request) {})

	if code := h.run(); code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(h.stderr.String(), "Usage:") {
		t.Errorf("stderr = %q", h.stderr)
	}
}

func TestRunSubmitsJobAndPrintsIDs(t *testing.T) {
	var got cursor.CreateAgentRequest

	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{
			"agent":{"id":"bc-42","url":"https://cursor.com/agents/bc-42"},
			"run":{"id":"run-42","agentId":"bc-42","status":"CREATING"}
		}`))
	})

	code := h.run("run",
		"--repo", "git@github.com:benfdking/sdsf.git",
		"--ref", "main",
		"--model", "claude-opus-4-8",
		"--name", "nightly",
		"--auto-pr",
		"--env", "CI=true",
		"upgrade the linear provider")

	if code != ExitOK {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, h.stderr)
	}
	if got.Prompt.Text != "upgrade the linear provider" {
		t.Errorf("prompt = %q", got.Prompt.Text)
	}
	// The scp-style remote must reach the API as an https URL.
	if len(got.Repos) != 1 || got.Repos[0].URL != "https://github.com/benfdking/sdsf" {
		t.Errorf("repos = %+v", got.Repos)
	}
	if got.Repos[0].StartingRef != "main" {
		t.Errorf("startingRef = %q", got.Repos[0].StartingRef)
	}
	if got.Model == nil || got.Model.ID != "claude-opus-4-8" {
		t.Errorf("model = %+v", got.Model)
	}
	if !got.AutoCreatePR || got.Name != "nightly" {
		t.Errorf("autoCreatePR = %v, name = %q", got.AutoCreatePR, got.Name)
	}
	if got.EnvVars["CI"] != "true" {
		t.Errorf("envVars = %+v", got.EnvVars)
	}

	out := h.stdout.String()
	for _, want := range []string{"bc-42", "run-42"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout %q missing %q", out, want)
		}
	}
	// The reattach hint belongs on stderr so `cursorjob run ... | ...` stays clean.
	if !strings.Contains(h.stderr.String(), "cursorjob attach bc-42") {
		t.Errorf("stderr = %q, want an attach hint", h.stderr)
	}
}

func TestRunWithoutPromptIsUsageError(t *testing.T) {
	h := newHarness(t, func(http.ResponseWriter, *http.Request) {
		t.Error("no request should be made without a prompt")
	})

	if code := h.run("run", "--no-git"); code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
}

func TestRunReadsPromptFromStdin(t *testing.T) {
	var got cursor.CreateAgentRequest
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"agent":{"id":"bc-1"},"run":{"id":"run-1"}}`))
	})
	h.app.Stdin = strings.NewReader("  piped prompt\n")

	if code := h.run("run", "--no-git"); code != ExitOK {
		t.Fatalf("exit code = %d (stderr: %s)", code, h.stderr)
	}
	if got.Prompt.Text != "piped prompt" {
		t.Errorf("prompt = %q, want the trimmed stdin content", got.Prompt.Text)
	}
}

func TestDetachedRunInsideCMUXStartsBackgroundWait(t *testing.T) {
	h := newHarness(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"agent":{"id":"bc-1","name":"task"},"run":{"id":"run-1","agentId":"bc-1"}}`))
	})
	h.app.Getenv = func(key string) string {
		return map[string]string{
			"CMUX_WORKSPACE_ID":     "workspace-1",
			"CMUX_BUNDLED_CLI_PATH": "/test/bin/cmux",
		}[key]
	}
	var childArgs []string
	h.app.StartCommand = func(_ string, args, _ []string) error {
		childArgs = append([]string(nil), args...)
		return nil
	}

	if code := h.run("run", "--json", "--no-git", "do the task"); code != ExitOK {
		t.Fatalf("exit code = %d (stderr: %s)", code, h.stderr)
	}
	if got := strings.Join(childArgs, " "); got != "wait bc-1 run-1" {
		t.Errorf("background command = %q, want wait for submitted run", got)
	}
}

func TestRunRejectsInvalidMode(t *testing.T) {
	h := newHarness(t, func(http.ResponseWriter, *http.Request) {
		t.Error("no request should be made for an invalid mode")
	})

	if code := h.run("run", "--no-git", "--mode", "turbo", "do a thing"); code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
}

// The whole point of attach: block, show the transcript, exit on the outcome.
func TestAttachStreamsTranscriptAndExitsZeroOnFinished(t *testing.T) {
	var runCalls atomic.Int32

	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/stream"):
			_, _ = w.Write([]byte("id: 1\nevent: assistant\ndata: {\"text\":\"hello \"}\n\n" +
				"id: 2\nevent: assistant\ndata: {\"text\":\"world\"}\n\n" +
				"id: 3\nevent: result\ndata: {\"status\":\"FINISHED\"}\n\n" +
				"id: 4\nevent: done\ndata: {}\n\n"))
		case r.URL.Path == "/v1/agents/bc-1":
			_, _ = w.Write([]byte(`{"id":"bc-1","latestRunId":"run-1"}`))
		default:
			// In flight on the first check, so the stream is actually opened.
			if runCalls.Add(1) == 1 {
				_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING"}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"run-1","status":"FINISHED","durationMs":3000,
				"git":{"branches":[{"repoUrl":"github.com/benfdking/sdsf","branch":"cursor/fix","prUrl":"https://github.com/benfdking/sdsf/pull/2"}]}}`))
		}
	})

	if code := h.run("attach", "bc-1"); code != ExitOK {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, h.stderr)
	}
	if out := h.stdout.String(); !strings.Contains(out, "hello world") {
		t.Errorf("stdout = %q, want the assistant transcript", out)
	}
	stderr := h.stderr.String()
	if !strings.Contains(stderr, "FINISHED") {
		t.Errorf("stderr = %q, want the outcome", stderr)
	}
	if !strings.Contains(stderr, "https://github.com/benfdking/sdsf/pull/2") {
		t.Errorf("stderr = %q, want the PR url", stderr)
	}
}

// Exit codes are the contract scripts depend on, so pin each terminal status.
func TestAttachExitCodeMatchesTerminalStatus(t *testing.T) {
	tests := map[string]int{
		cursor.RunStatusFinished:  ExitOK,
		cursor.RunStatusError:     ExitRunError,
		cursor.RunStatusCancelled: ExitCancelled,
		cursor.RunStatusExpired:   ExitExpired,
	}

	for status, wantCode := range tests {
		t.Run(status, func(t *testing.T) {
			h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/stream") {
					t.Error("a terminal run should not be streamed")
				}
				_, _ = w.Write([]byte(`{"id":"run-1","status":"` + status + `"}`))
			})

			if code := h.run("attach", "bc-1", "run-1"); code != wantCode {
				t.Errorf("exit code = %d, want %d", code, wantCode)
			}
		})
	}
}

func TestWaitSuppressesTranscript(t *testing.T) {
	var runCalls atomic.Int32
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stream") {
			_, _ = w.Write([]byte("event: assistant\ndata: {\"text\":\"chatter\"}\n\nevent: done\ndata: {}\n\n"))
			return
		}
		if runCalls.Add(1) == 1 {
			_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"run-1","status":"FINISHED"}`))
	})

	if code := h.run("wait", "bc-1", "run-1"); code != ExitOK {
		t.Fatalf("exit code = %d (stderr: %s)", code, h.stderr)
	}
	if strings.Contains(h.stdout.String(), "chatter") {
		t.Errorf("stdout = %q, want no transcript from wait", h.stdout)
	}
}

// --json must be uniformly newline-delimited: every line, including the final
// record, parses on its own so it can be piped into jq.
func TestAttachJSONEmitsNewlineDelimitedRecords(t *testing.T) {
	var runCalls atomic.Int32
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stream") {
			_, _ = w.Write([]byte("id: 1\nevent: assistant\ndata: {\"text\":\"hi\"}\n\nevent: done\ndata: {}\n\n"))
			return
		}
		if runCalls.Add(1) == 1 {
			_, _ = w.Write([]byte(`{"id":"run-1","status":"RUNNING"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"run-1","status":"FINISHED"}`))
	})

	if code := h.run("attach", "--json", "bc-1", "run-1"); code != ExitOK {
		t.Fatalf("exit code = %d (stderr: %s)", code, h.stderr)
	}

	lines := strings.Split(strings.TrimSpace(h.stdout.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("stdout = %q, want an event line per event plus the final run", h.stdout)
	}

	type record struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
		Run  *cursor.Run     `json:"run"`
	}
	decoded := make([]record, 0, len(lines))
	for i, line := range lines {
		var rec record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("line %d is not JSON: %v (%q)", i, err, line)
		}
		decoded = append(decoded, rec)
	}

	if decoded[0].Type != cursor.EventAssistant {
		t.Errorf("first record type = %q, want %q", decoded[0].Type, cursor.EventAssistant)
	}
	last := decoded[len(decoded)-1]
	if last.Type != "run" || last.Run == nil || last.Run.Status != cursor.RunStatusFinished {
		t.Errorf("last record = %+v, want the terminal run", last)
	}
}

func TestAttachWithoutAgentIDIsUsageError(t *testing.T) {
	h := newHarness(t, func(http.ResponseWriter, *http.Request) {
		t.Error("no request should be made without an agent id")
	})

	if code := h.run("attach"); code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
}

func TestListRendersTable(t *testing.T) {
	h := newHarness(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[
			{"id":"bc-1","status":"ACTIVE","name":"nightly upgrade","latestRunId":"run-1"},
			{"id":"bc-2","status":"ARCHIVED","name":"old","latestRunId":"run-2"}
		]}`))
	})

	if code := h.run("list"); code != ExitOK {
		t.Fatalf("exit code = %d (stderr: %s)", code, h.stderr)
	}
	out := h.stdout.String()
	for _, want := range []string{"AGENT ID", "bc-1", "nightly upgrade", "bc-2"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout %q missing %q", out, want)
		}
	}
}

func TestShowResolvesLatestRunWhenRunIDOmitted(t *testing.T) {
	var paths []string
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/v1/agents/bc-1" {
			_, _ = w.Write([]byte(`{"id":"bc-1","latestRunId":"run-7"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"run-7","status":"FINISHED","result":"summary text"}`))
	})

	if code := h.run("show", "bc-1"); code != ExitOK {
		t.Fatalf("exit code = %d (stderr: %s)", code, h.stderr)
	}
	if !strings.Contains(h.stdout.String(), "summary text") {
		t.Errorf("stdout = %q, want the run result", h.stdout)
	}
	if len(paths) != 2 || paths[1] != "/v1/agents/bc-1/runs/run-7" {
		t.Errorf("paths = %v, want the latest run to be resolved then fetched", paths)
	}
}

func TestCancelUsesResolvedRun(t *testing.T) {
	var cancelPath string
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/cancel") {
			cancelPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"id":"bc-1","latestRunId":"run-5"}`))
	})

	if code := h.run("cancel", "bc-1"); code != ExitOK {
		t.Fatalf("exit code = %d (stderr: %s)", code, h.stderr)
	}
	if cancelPath != "/v1/agents/bc-1/runs/run-5/cancel" {
		t.Errorf("cancel path = %q", cancelPath)
	}
}

func TestFollowupPostsRunToExistingAgent(t *testing.T) {
	var body cursor.CreateRunRequest
	var path string
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write([]byte(`{"run":{"id":"run-2","agentId":"bc-1","status":"CREATING"}}`))
	})

	if code := h.run("followup", "bc-1", "now also update the README"); code != ExitOK {
		t.Fatalf("exit code = %d (stderr: %s)", code, h.stderr)
	}
	if path != "/v1/agents/bc-1/runs" {
		t.Errorf("path = %q", path)
	}
	if body.Prompt.Text != "now also update the README" {
		t.Errorf("prompt = %q", body.Prompt.Text)
	}
}

// A missing key should explain itself rather than fail with a bare 401.
func TestMissingAPIKeyExplainsHowToSetOne(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	app := &App{
		Stdout: stdout,
		Stderr: stderr,
		Getenv: func(string) string { return "" },
	}

	if code := app.Execute(context.Background(), []string{"list"}); code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr.String(), "CURSOR_API_KEY") {
		t.Errorf("stderr = %q, want it to name the env var", stderr)
	}
}

func TestClientPrefersFlagOverEnvironment(t *testing.T) {
	app := &App{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Getenv: func(key string) string {
			if key == "CURSOR_API_KEY" {
				return "crsr_from_env"
			}
			return ""
		},
	}

	// Env alone is enough.
	if _, err := app.client(&commonFlags{}); err != nil {
		t.Fatalf("client with env key: %v", err)
	}
	// An explicit flag works even with no env at all.
	app.Getenv = func(string) string { return "" }
	if _, err := app.client(&commonFlags{apiKey: "crsr_from_flag"}); err != nil {
		t.Fatalf("client with flag key: %v", err)
	}
	// Neither is an error, not a panic or an anonymous 401 later.
	if _, err := app.client(&commonFlags{}); err == nil {
		t.Error("expected an error when no key is available")
	}
}

func TestExitCodeForStatus(t *testing.T) {
	tests := map[string]int{
		cursor.RunStatusFinished:  ExitOK,
		cursor.RunStatusError:     ExitRunError,
		cursor.RunStatusCancelled: ExitCancelled,
		cursor.RunStatusExpired:   ExitExpired,
		"SOMETHING_NEW":           ExitRunError,
	}
	for status, want := range tests {
		if got := exitCodeForStatus(status); got != want {
			t.Errorf("exitCodeForStatus(%q) = %d, want %d", status, got, want)
		}
	}
}

func TestKeyValueListRejectsMalformedPairs(t *testing.T) {
	list := keyValueList{}
	if err := list.Set("NOPE"); err == nil {
		t.Error("expected an error for a value without =")
	}
	if err := list.Set("=value"); err == nil {
		t.Error("expected an error for an empty key")
	}
	if err := list.Set("KEY=value=with=equals"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if list["KEY"] != "value=with=equals" {
		t.Errorf("value = %q, want everything after the first =", list["KEY"])
	}
}
