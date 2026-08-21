package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/benfdking/sdsf/cli/cursorjob/internal/cursor"
)

// watchConfig is how a run should be rendered while it is being waited on.
type watchConfig struct {
	// json emits one JSON object per stream event, for piping into jq.
	json bool
	// verbose adds reasoning and tool calls to the transcript.
	verbose bool
	// quiet suppresses the transcript, leaving only the final outcome.
	quiet        bool
	pollInterval time.Duration

	// agentName avoids another API request for run --follow, which already has
	// the newly-created agent response. Attach/followup resolve it on demand.
	agentName        string
	snapshotInterval time.Duration
}

func cmdAttach(ctx context.Context, a *App, args []string) error {
	return a.attachCommand(ctx, args, "attach", false)
}

func cmdWait(ctx context.Context, a *App, args []string) error {
	return a.attachCommand(ctx, args, "wait", true)
}

func (a *App) attachCommand(ctx context.Context, args []string, name string, quietDefault bool) error {
	fs := a.newFlagSet(name)
	common := addCommonFlags(fs)
	verbose := fs.Bool("verbose", false, "Also show reasoning and tool calls")
	quiet := fs.Bool("quiet", quietDefault, "Suppress the transcript and print only the outcome")
	timeout := fs.Duration("timeout", 0, "Give up after this long (0 waits indefinitely)")
	pollInterval := fs.Duration("poll-interval", cursor.DefaultPollInterval, "Gap between status checks when streaming is unavailable")

	fs.Usage = func() {
		fmt.Fprintf(a.Stderr, "Usage: cursorjob %s [flags] <agent-id> [run-id]\n", name)
		fmt.Fprintln(a.Stderr, "\nBlocks until the run reaches a terminal state. Defaults to the agent's latest run.")
		fmt.Fprintln(a.Stderr, "Exits 0 on FINISHED, 10 on ERROR, 11 on CANCELLED, 12 on EXPIRED.")
		fmt.Fprintln(a.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return usagef("an agent id is required")
	}
	if len(rest) > 2 {
		return usagef("expected at most an agent id and a run id, got %d arguments", len(rest))
	}
	agentID := rest[0]
	runID := ""
	if len(rest) == 2 {
		runID = rest[1]
	}

	client, err := a.client(common)
	if err != nil {
		return err
	}

	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	return a.watch(ctx, client, agentID, runID, watchConfig{
		json:         common.json,
		verbose:      *verbose,
		quiet:        *quiet,
		pollInterval: *pollInterval,
	})
}

// watch blocks on a run and reports its outcome, returning an exitCodeError so
// the process exit code reflects the run's terminal status.
func (a *App) watch(ctx context.Context, client *cursor.Client, agentID, runID string, cfg watchConfig) error {
	resolvedRunID, err := client.ResolveRunID(ctx, agentID, runID)
	if err != nil {
		return err
	}

	cmux := a.newCMUXJob()
	if cmux != nil {
		agentName := cfg.agentName
		if agentName == "" {
			// Naming is optional; a metadata lookup failure must not prevent the
			// actual run from being watched.
			if agent, getErr := client.GetAgent(ctx, agentID); getErr == nil {
				agentName = agent.Name
			}
		}
		cmux.start(agentName, resolvedRunID)
	}

	renderer := &eventRenderer{app: a, cfg: cfg}
	if !cfg.quiet && !cfg.json {
		fmt.Fprintf(a.Stderr, "Attached to %s (ctrl-c to detach; the run keeps going)\n\n", resolvedRunID)
	}

	watchOptions := cursor.WatchOptions{
		OnEvent:      renderer.handle,
		OnNotice:     renderer.notice,
		PollInterval: cfg.pollInterval,
	}
	if cmux != nil {
		watchOptions.OnSnapshot = cmux.observe
		watchOptions.SnapshotInterval = cfg.snapshotInterval
	}
	run, err := client.WatchRun(ctx, agentID, resolvedRunID, watchOptions)
	if err != nil {
		if ctx.Err() != nil {
			// Detaching is not a failure of the run, but it is not a success
			// either — report it distinctly from a finished run.
			fmt.Fprintf(a.Stderr, "\nDetached from %s; it is still running.\n", resolvedRunID)
			fmt.Fprintf(a.Stderr, "Reattach with: cursorjob attach %s %s\n", agentID, resolvedRunID)
			return &exitCodeError{code: ExitError}
		}
		return err
	}

	renderer.finish()
	cmux.finish(run)

	if cfg.json {
		// Same newline-delimited shape as the events, tagged so a consumer can
		// tell the final record apart from the stream.
		final := struct {
			Type string      `json:"type"`
			Run  *cursor.Run `json:"run"`
		}{Type: "run", Run: run}
		if err := writeJSONLine(a.Stdout, final); err != nil {
			return err
		}
	} else {
		a.printOutcome(run)
	}

	if code := exitCodeForStatus(run.Status); code != ExitOK {
		return &exitCodeError{code: code}
	}
	return nil
}

func (a *App) printOutcome(run *cursor.Run) {
	fmt.Fprintln(a.Stderr)
	summary := fmt.Sprintf("%s in %s", run.Status, formatDuration(run.DurationMs))
	fmt.Fprintf(a.Stderr, "%s\n", summary)
	printGit(a.Stderr, run.Git)
}

// eventRenderer turns stream events into terminal output.
//
// The transcript goes to stdout so it can be piped; progress chatter, tool
// calls, and reasoning go to stderr so the pipe stays clean.
type eventRenderer struct {
	app *App
	cfg watchConfig

	// midLine tracks whether assistant text ended without a newline, so notices
	// and the final summary don't get appended to a partial line.
	midLine bool
}

func (r *eventRenderer) handle(event cursor.Event) {
	if r.cfg.json {
		r.emitJSON(event)
		return
	}
	if r.cfg.quiet {
		return
	}

	switch event.Type {
	case cursor.EventAssistant:
		var payload cursor.TextPayload
		if err := event.Decode(&payload); err != nil || payload.Text == "" {
			return
		}
		fmt.Fprint(r.app.Stdout, payload.Text)
		r.midLine = !strings.HasSuffix(payload.Text, "\n")

	case cursor.EventThinking:
		if !r.cfg.verbose {
			return
		}
		var payload cursor.TextPayload
		if err := event.Decode(&payload); err != nil || payload.Text == "" {
			return
		}
		r.breakLine()
		fmt.Fprintf(r.app.Stderr, "· %s\n", truncateDisplay(payload.Text, 120))

	case cursor.EventToolCall:
		if !r.cfg.verbose {
			return
		}
		var payload cursor.ToolCallPayload
		if err := event.Decode(&payload); err != nil {
			return
		}
		// Only the start is interesting for progress; completions would double
		// every line without adding much.
		if payload.Status != "" && payload.Status != "running" {
			return
		}
		r.breakLine()
		fmt.Fprintf(r.app.Stderr, "→ %s %s\n", payload.Name, summarizeArgs(payload.Args))

	case cursor.EventStatus:
		var payload cursor.StatusPayload
		if err := event.Decode(&payload); err != nil || payload.Status == "" {
			return
		}
		r.breakLine()
		fmt.Fprintf(r.app.Stderr, "status: %s\n", payload.Status)

	case cursor.EventError:
		var payload cursor.ErrorPayload
		if err := event.Decode(&payload); err != nil {
			return
		}
		r.breakLine()
		fmt.Fprintf(r.app.Stderr, "error: %s %s\n", payload.Code, payload.Message)
	}
}

// emitJSON writes one line per event, keeping the payload as raw JSON so
// downstream tools see exactly what the API sent.
func (r *eventRenderer) emitJSON(event cursor.Event) {
	envelope := struct {
		ID   string          `json:"id,omitempty"`
		Type string          `json:"type"`
		Data json.RawMessage `json:"data,omitempty"`
	}{ID: event.ID, Type: event.Type}

	if json.Valid(event.Data) {
		envelope.Data = event.Data
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return
	}
	fmt.Fprintf(r.app.Stdout, "%s\n", encoded)
}

func (r *eventRenderer) notice(message string) {
	if r.cfg.quiet || r.cfg.json {
		return
	}
	r.breakLine()
	fmt.Fprintf(r.app.Stderr, "· %s\n", message)
}

func (r *eventRenderer) breakLine() {
	if r.midLine {
		fmt.Fprintln(r.app.Stdout)
		r.midLine = false
	}
}

func (r *eventRenderer) finish() { r.breakLine() }

// summarizeArgs renders tool arguments as a short one-liner for progress output.
func summarizeArgs(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var fields map[string]any
	if err := json.Unmarshal(args, &fields); err != nil {
		return truncateDisplay(string(args), 80)
	}
	// Prefer the field that identifies the target of the call.
	for _, key := range []string{"path", "file_path", "command", "query", "pattern", "url"} {
		if value, ok := fields[key]; ok {
			return truncateDisplay(fmt.Sprint(value), 80)
		}
	}
	return truncateDisplay(string(args), 80)
}
