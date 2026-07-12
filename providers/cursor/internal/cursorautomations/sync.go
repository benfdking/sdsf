package cursorautomations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// API is the slice of Client that Sync needs; tests substitute a fake.
type API interface {
	List(ctx context.Context) (json.RawMessage, error)
	Create(ctx context.Context, a Automation) (string, error)
	Update(ctx context.Context, a Automation, automationID string) error
}

// Result summarises a sync run for the caller to report and exit on.
type Result struct {
	Total     int
	Created   int
	Updated   int
	Unchanged int
	Failures  int

	AuthFailed        bool // some failure was a 401/403: the session token needs rotating
	SlackNotConnected bool // some failure was the Slack-not-connected 400
}

func (r *Result) fail(err error) {
	r.Failures++
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		r.AuthFailed = r.AuthFailed || apiErr.IsAuth()
		r.SlackNotConnected = r.SlackNotConnected || apiErr.IsSlackNotConnected()
	}
}

// Sync reconciles the code-defined automations against Cursor, writing only
// when something actually changed. Each automation resolves to its live
// counterpart (recorded id, else adopt by name, else create), then:
//
//   - unchanged (desired config matches the last applied payload AND the live
//     automation matches its post-write snapshot) → skipped, no API write, so a
//     no-op deploy can't retrigger or repost anything (AAI-688);
//   - changed → updated, logging a field-level diff of exactly what changed
//     (repo-side changes and live drift are reported separately);
//   - no baseline (first run with this feature, adopted, or state was lost) →
//     updated once to establish one;
//   - missing everywhere → created; create carries the full config, so there is
//     no follow-up update (the old create-then-update double write was itself a
//     source of duplicate automation messages).
//
// After any write it re-lists Cursor and records both baselines in state.
func Sync(ctx context.Context, client API, automations []Automation, state *State, statePath string, out io.Writer) Result {
	res := Result{Total: len(automations)}

	live, err := listLive(ctx, client)
	if err != nil {
		emit(out, "error listing automations: %v\n", err)
		res.Failures = res.Total
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			res.AuthFailed = apiErr.IsAuth()
			res.SlackNotConnected = apiErr.IsSlackNotConnected()
		}
		return res
	}

	liveByName := map[string]string{}
	liveIDs := map[string]bool{}
	liveByID := map[string]LiveAutomation{}
	for _, l := range live {
		if l.AutomationID != "" {
			liveByName[l.Name] = l.AutomationID
			liveIDs[l.AutomationID] = true
			liveByID[l.AutomationID] = l
		}
	}

	// written accumulates everything that reached Cursor this run, so baselines
	// can be captured from a single post-write list.
	type written struct {
		dir     string
		id      string
		desired json.RawMessage
	}
	var wrote []written

	// applyUpdate pushes the config and, on success, queues the automation for
	// baseline capture. Both the changed and the adopted paths end here.
	applyUpdate := func(a Automation, id string, desired json.RawMessage) {
		if err := client.Update(ctx, a, id); err != nil {
			emit(out, "  FAILED: %v\n", err)
			res.fail(err)
			return
		}
		emit(out, "  updated\n")
		res.Updated++
		wrote = append(wrote, written{dir: a.Dir, id: id, desired: desired})
	}

	for _, a := range automations {
		emit(out, "Syncing %s ... ", a.Dir)

		desired, err := a.DesiredBaseline()
		if err != nil {
			emit(out, "FAILED: %v\n", err)
			res.fail(err)
			continue
		}

		id, source := ResolveTarget(a.Dir, a.Config.Name, state, liveByName, liveIDs)
		switch source {
		case "state":
			dec, err := decide(state, a.Dir, desired, liveByID[id])
			if err != nil {
				emit(out, "FAILED: %v\n", err)
				res.fail(err)
				continue
			}
			if !dec.write {
				emit(out, "unchanged, skipping\n")
				res.Unchanged++
				continue
			}
			dec.report(out)
			applyUpdate(a, id, desired)

		case "adopt":
			state.Record(a.Dir, id)
			saveState(state, statePath, out)
			emit(out, "adopted by name; applying config to establish a baseline\n")
			applyUpdate(a, id, desired)

		case "create":
			created, err := client.Create(ctx, a)
			if err != nil {
				emit(out, "FAILED: %v\n", err)
				res.fail(err)
				continue
			}
			state.Record(a.Dir, created)
			saveState(state, statePath, out)
			emit(out, "created\n")
			res.Created++
			wrote = append(wrote, written{dir: a.Dir, id: created, desired: desired})
		}
	}

	if len(wrote) > 0 {
		// Baselines come from a fresh list, not the write responses, so they don't
		// depend on the (less well understood) create/update echo shapes. A failure
		// here leaves the baseline unestablished: the next run applies once more
		// and retries, rather than ever trusting a stale snapshot.
		//
		// TOCTOU caveat: a UI edit landing in the seconds between our write and
		// this list is absorbed into the baseline, so that one edit reads as
		// "unchanged" until the next repo change. Accepted: closing it would need
		// the echo-shape dependency this deliberately avoids.
		postLive, err := listLive(ctx, client)
		if err != nil {
			emit(out, "warning: could not list automations to capture baselines (next run will re-apply): %v\n", err)
		} else {
			postByID := map[string]LiveAutomation{}
			for _, l := range postLive {
				postByID[l.AutomationID] = l
			}
			for _, w := range wrote {
				l, ok := postByID[w.id]
				if !ok {
					emit(out, "warning: %s (id %s) missing from post-write list; baseline not recorded\n", w.dir, w.id)
					continue
				}
				liveView, err := ProjectLive(l.Raw)
				if err != nil {
					emit(out, "warning: could not project %s for its baseline (next run will re-apply): %v\n", w.dir, err)
					continue
				}
				state.RecordBaseline(w.dir, w.desired, liveView)
			}
		}
		saveState(state, statePath, out)
	}

	return res
}

func listLive(ctx context.Context, client API) ([]LiveAutomation, error) {
	raw, err := client.List(ctx)
	if err != nil {
		return nil, err
	}
	live, err := ParseList(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing automations list: %w", err)
	}
	return live, nil
}

// decision is the outcome of comparing an automation against its baselines.
type decision struct {
	write      bool
	noBaseline bool
	repoDiffs  []FieldDiff // desired config vs last applied payload
	driftDiffs []FieldDiff // live automation vs its post-write snapshot
}

func (d decision) report(out io.Writer) {
	if d.noBaseline {
		emit(out, "no baseline recorded; applying config to establish one\n")
	}
	if len(d.repoDiffs) > 0 {
		emit(out, "config changed:\n")
		for _, diff := range d.repoDiffs {
			emit(out, "    %s\n", diff)
		}
	}
	if len(d.driftDiffs) > 0 {
		emit(out, "live automation drifted from the last applied config (edited outside the repo?); re-asserting:\n")
		for _, diff := range d.driftDiffs {
			emit(out, "    %s\n", diff)
		}
	}
}

// decide works out whether an automation whose recorded id is still live needs
// a write, and if so why, field by field.
func decide(state *State, dir string, desired json.RawMessage, la LiveAutomation) (decision, error) {
	appliedPayload, appliedLive, ok := state.Baseline(dir)
	if !ok {
		return decision{write: true, noBaseline: true}, nil
	}

	repoDiffs, err := DiffJSON(appliedPayload, desired)
	if err != nil {
		return decision{}, fmt.Errorf("comparing desired config against the last applied payload: %w", err)
	}

	liveView, err := ProjectLive(la.Raw)
	if err != nil {
		return decision{}, fmt.Errorf("projecting the live automation for drift detection: %w", err)
	}
	driftDiffs, err := DiffJSON(appliedLive, liveView)
	if err != nil {
		return decision{}, fmt.Errorf("comparing the live automation against its baseline: %w", err)
	}

	return decision{
		write:      len(repoDiffs) > 0 || len(driftDiffs) > 0,
		repoDiffs:  repoDiffs,
		driftDiffs: driftDiffs,
	}, nil
}

func saveState(state *State, path string, out io.Writer) {
	if err := state.Save(path); err != nil {
		emit(out, "warning: could not save state to %s: %v\n", path, err)
	}
}

// emit writes progress output, deliberately discarding the write error: a
// failed progress line can't affect sync correctness.
func emit(out io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(out, format, args...)
}
