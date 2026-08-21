package cursor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Defaults for WatchOptions.
const (
	DefaultPollInterval      = 5 * time.Second
	DefaultSnapshotInterval  = 15 * time.Second
	DefaultMaxStreamAttempts = 5
	DefaultReconnectDelay    = time.Second
	maxBackoff               = 15 * time.Second
)

// WatchOptions tunes WatchRun.
type WatchOptions struct {
	// OnEvent receives every stream event, in order. Nil means discard, which
	// is how a quiet wait avoids paying for rendering.
	OnEvent func(Event)

	// OnNotice receives human-readable transport notes — reconnects, falling
	// back to polling — that are about the CLI rather than the run itself.
	OnNotice func(string)

	// OnSnapshot receives authoritative run snapshots from GetRun. When set,
	// WatchRun also refreshes the snapshot periodically while an SSE stream is
	// open. This exposes branch and PR metadata, which can appear before the
	// terminal result event. Callbacks may arrive from a background goroutine.
	OnSnapshot func(*Run)

	// SnapshotInterval is the gap between metadata refreshes while streaming.
	// It has no effect when OnSnapshot is nil.
	SnapshotInterval time.Duration

	// PollInterval is the gap between GetRun calls when polling. The API allows
	// 20 requests/minute per team, so keep this at or above a few seconds.
	PollInterval time.Duration

	// MaxStreamAttempts bounds consecutive failed stream attempts before giving
	// up on SSE and polling instead. The counter resets whenever a stream makes
	// progress, so a long run can reconnect indefinitely.
	MaxStreamAttempts int

	// ReconnectDelay is the base delay before reconnecting a dropped stream.
	// It doubles per consecutive failed attempt, capped at maxBackoff.
	ReconnectDelay time.Duration
}

func (o WatchOptions) pollInterval() time.Duration {
	if o.PollInterval <= 0 {
		return DefaultPollInterval
	}
	return o.PollInterval
}

func (o WatchOptions) snapshotInterval() time.Duration {
	if o.SnapshotInterval <= 0 {
		return DefaultSnapshotInterval
	}
	return o.SnapshotInterval
}

func (o WatchOptions) maxStreamAttempts() int {
	if o.MaxStreamAttempts <= 0 {
		return DefaultMaxStreamAttempts
	}
	return o.MaxStreamAttempts
}

// backoff is the delay before the next reconnect, doubling with each
// consecutive failure.
func (o WatchOptions) backoff(attempt int) time.Duration {
	base := o.ReconnectDelay
	if base <= 0 {
		base = DefaultReconnectDelay
	}
	backoff := base << attempt
	if backoff > maxBackoff || backoff <= 0 {
		return maxBackoff
	}
	return backoff
}

func (o WatchOptions) emit(event Event) {
	if o.OnEvent != nil {
		o.OnEvent(event)
	}
}

func (o WatchOptions) snapshot(run *Run) {
	if o.OnSnapshot != nil {
		o.OnSnapshot(run)
	}
}

func (o WatchOptions) notice(format string, args ...any) {
	if o.OnNotice != nil {
		o.OnNotice(fmt.Sprintf(format, args...))
	}
}

// WatchRun blocks until a run reaches a terminal state and returns it.
//
// It prefers the SSE stream, reconnecting with Last-Event-ID when the
// connection drops, and degrades to polling GetRun when the stream is
// unavailable or its retention window has passed. Either way the returned Run
// comes from GetRun, so the final status is authoritative rather than inferred
// from a possibly-truncated stream.
func (c *Client) WatchRun(ctx context.Context, agentID, runID string, opts WatchOptions) (*Run, error) {
	// A run that is already finished needs no stream. This also makes attaching
	// to a completed run cheap and immediate instead of an error.
	run, err := c.GetRun(ctx, agentID, runID)
	if err != nil {
		return nil, err
	}
	opts.snapshot(run)
	if IsTerminal(run.Status) {
		return run, nil
	}

	// The event stream carries the transcript but only includes git metadata in
	// its terminal result. Refresh GetRun at a conservative interval so callers
	// can react when a branch is pushed or a PR is opened during a long run.
	stopSnapshots := c.watchSnapshots(ctx, agentID, runID, opts)
	defer stopSnapshots()

	var (
		lastEventID string
		attempt     int
	)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		progressed := false
		sawResult := false
		lastEventID, err = c.StreamRun(ctx, agentID, runID, lastEventID, func(event Event) error {
			progressed = true
			if event.Type == EventResult {
				sawResult = true
			}
			opts.emit(event)
			if event.Type == EventDone {
				return errStreamDone
			}
			return nil
		})

		switch {
		case err == nil, errors.Is(err, errStreamDone):
			// Stream ended cleanly. If it delivered a result the run is over;
			// otherwise confirm against the API before deciding to reconnect.
		case ctx.Err() != nil:
			return nil, ctx.Err()
		default:
			if apiErr, ok := errors.AsType[*APIError](err); ok {
				if apiErr.IsStreamExpired() {
					opts.notice("event stream expired for %s; polling instead", runID)
					stopSnapshots()
					return c.pollUntilTerminal(ctx, agentID, runID, opts)
				}
				if !apiErr.IsRetryable() {
					return nil, err
				}
			}
			opts.notice("stream interrupted (%v)", err)
		}

		run, getErr := c.GetRun(ctx, agentID, runID)
		if getErr != nil {
			return nil, getErr
		}
		opts.snapshot(run)
		if IsTerminal(run.Status) {
			return run, nil
		}
		if sawResult {
			// The stream said it was done but the API still reports the run as
			// in flight — trust the API and let polling settle it.
			opts.notice("stream closed before %s reached a terminal state; polling instead", runID)
			stopSnapshots()
			return c.pollUntilTerminal(ctx, agentID, runID, opts)
		}

		if progressed {
			attempt = 0
		} else {
			attempt++
		}
		if attempt >= opts.maxStreamAttempts() {
			opts.notice("giving up on the event stream after %d attempts; polling instead", attempt)
			stopSnapshots()
			return c.pollUntilTerminal(ctx, agentID, runID, opts)
		}
		if err := sleep(ctx, opts.backoff(attempt)); err != nil {
			return nil, err
		}
	}
}

// errStreamDone unwinds the scan loop on the `done` event without treating the
// stop as a failure.
var errStreamDone = errors.New("stream done")

// pollUntilTerminal is the no-streaming path: ask for the run on an interval
// until it settles.
func (c *Client) pollUntilTerminal(ctx context.Context, agentID, runID string, opts WatchOptions) (*Run, error) {
	ticker := time.NewTicker(opts.pollInterval())
	defer ticker.Stop()

	for {
		run, err := c.GetRun(ctx, agentID, runID)
		if err != nil {
			// Rate limiting and server blips shouldn't end a long wait; anything
			// else is a real failure.
			apiErr, ok := errors.AsType[*APIError](err)
			if !ok || !apiErr.IsRetryable() {
				return nil, err
			}
			opts.notice("poll failed (%v); retrying", err)
		} else if IsTerminal(run.Status) {
			opts.snapshot(run)
			return run, nil
		} else {
			opts.snapshot(run)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// watchSnapshots starts the low-rate metadata observer used while SSE owns the
// main wait loop. The returned function cancels it and waits for it to stop,
// ensuring no callback can outlive WatchRun.
func (c *Client) watchSnapshots(ctx context.Context, agentID, runID string, opts WatchOptions) func() {
	if opts.OnSnapshot == nil {
		return func() {}
	}

	observerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(opts.snapshotInterval())
		defer ticker.Stop()

		for {
			select {
			case <-observerCtx.Done():
				return
			case <-ticker.C:
				run, err := c.GetRun(observerCtx, agentID, runID)
				if err != nil {
					continue
				}
				opts.snapshot(run)
				if IsTerminal(run.Status) {
					return
				}
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			<-done
		})
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
