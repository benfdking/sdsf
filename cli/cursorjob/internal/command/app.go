// Package command implements the cursorjob command line interface.
package command

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/benfdking/sdsf/cli/cursorjob/internal/cursor"
)

// Process exit codes. Run outcomes get distinct codes above the conventional
// range so a script can tell "the CLI broke" apart from "the agent failed".
const (
	ExitOK        = 0
	ExitError     = 1
	ExitUsage     = 2
	ExitRunError  = 10
	ExitCancelled = 11
	ExitExpired   = 12
)

// exitCodeForStatus maps a terminal run status onto a process exit code.
func exitCodeForStatus(status string) int {
	switch status {
	case cursor.RunStatusFinished:
		return ExitOK
	case cursor.RunStatusCancelled:
		return ExitCancelled
	case cursor.RunStatusExpired:
		return ExitExpired
	default:
		return ExitRunError
	}
}

// App is one invocation of the CLI. Every dependency the commands touch is a
// field so tests can drive the whole surface without a network or a real
// terminal.
type App struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Getenv  func(string) string
	WorkDir string

	// RunCommand, when set, replaces execution of best-effort local integration
	// commands. Tests use it to exercise cmux behavior without a running app.
	RunCommand func(context.Context, string, ...string) error
	// StartCommand similarly replaces detached process startup.
	StartCommand func(string, []string, []string) error

	// Client, when set, is used instead of building one from flags and the
	// environment.
	Client *cursor.Client
}

// NewApp builds an App bound to the real process.
func NewApp() *App {
	workDir, _ := os.Getwd()
	return &App{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Getenv:  os.Getenv,
		WorkDir: workDir,
	}
}

// usageError is a bad invocation, reported with the usage text and ExitUsage
// rather than a bare error.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func usagef(format string, args ...any) error {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

// exitCodeError carries a specific exit code up from a command.
type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *exitCodeError) Unwrap() error { return e.err }

type commandFunc func(context.Context, *App, []string) error

type subcommand struct {
	name    string
	summary string
	run     commandFunc
}

func subcommands() []subcommand {
	return []subcommand{
		{"run", "Submit a new job and print its ids", cmdRun},
		{"attach", "Stream a job's output and block until it finishes", cmdAttach},
		{"wait", "Block until a job finishes, printing only the outcome", cmdWait},
		{"followup", "Send a follow-up prompt to an existing agent", cmdFollowup},
		{"list", "List agents", cmdList},
		{"runs", "List an agent's runs", cmdRuns},
		{"show", "Show a single run in detail", cmdShow},
		{"cancel", "Cancel an in-flight run", cmdCancel},
		{"help", "Show this help", nil},
	}
}

// Execute parses args (without the program name) and returns a process exit
// code.
func (a *App) Execute(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.printUsage()
		return ExitUsage
	}

	name := args[0]
	switch name {
	case "-h", "--help", "help":
		a.printUsage()
		return ExitOK
	}

	for _, cmd := range subcommands() {
		if cmd.name != name || cmd.run == nil {
			continue
		}
		err := cmd.run(ctx, a, args[1:])
		return a.report(name, err)
	}

	fmt.Fprintf(a.Stderr, "cursorjob: unknown command %q\n\n", name)
	a.printUsage()
	return ExitUsage
}

// report turns a command's error into an exit code and a message.
func (a *App) report(name string, err error) int {
	if err == nil {
		return ExitOK
	}

	if usageErr, ok := errors.AsType[*usageError](err); ok {
		fmt.Fprintf(a.Stderr, "cursorjob %s: %s\n", name, usageErr.msg)
		fmt.Fprintf(a.Stderr, "Run 'cursorjob %s --help' for usage.\n", name)
		return ExitUsage
	}

	if codeErr, ok := errors.AsType[*exitCodeError](err); ok {
		if codeErr.err != nil {
			fmt.Fprintf(a.Stderr, "cursorjob %s: %v\n", name, codeErr.err)
		}
		return codeErr.code
	}

	if errors.Is(err, context.Canceled) {
		fmt.Fprintln(a.Stderr, "cursorjob: interrupted")
		return ExitError
	}

	if apiErr, ok := errors.AsType[*cursor.APIError](err); ok && apiErr.IsAuth() {
		fmt.Fprintf(a.Stderr, "cursorjob %s: %v\n", name, err)
		fmt.Fprintln(a.Stderr, "Check CURSOR_API_KEY — create a key at cursor.com/dashboard → API Keys.")
		return ExitError
	}

	fmt.Fprintf(a.Stderr, "cursorjob %s: %v\n", name, err)
	return ExitError
}

func (a *App) printUsage() {
	var b strings.Builder
	b.WriteString("cursorjob — submit and monitor Cursor cloud agents\n\n")
	b.WriteString("Usage:\n  cursorjob <command> [flags]\n\nCommands:\n")
	for _, cmd := range subcommands() {
		fmt.Fprintf(&b, "  %-9s %s\n", cmd.name, cmd.summary)
	}
	b.WriteString("\nAuthentication:\n")
	b.WriteString("  CURSOR_API_KEY      API key (or pass --api-key)\n")
	b.WriteString("  CURSOR_API_BASE_URL Override the API host (default https://api.cursor.com)\n")
	b.WriteString("\nExit codes:\n")
	b.WriteString("  0 finished   1 cli error   2 usage   10 run error   11 cancelled   12 expired\n")
	fmt.Fprint(a.Stderr, b.String())
}

// commonFlags are the connection flags every subcommand accepts.
type commonFlags struct {
	apiKey  string
	baseURL string
	json    bool
}

func addCommonFlags(fs *flag.FlagSet) *commonFlags {
	common := &commonFlags{}
	fs.StringVar(&common.apiKey, "api-key", "", "Cursor API key (default $CURSOR_API_KEY)")
	fs.StringVar(&common.baseURL, "base-url", "", "API base URL (default $CURSOR_API_BASE_URL)")
	fs.BoolVar(&common.json, "json", false, "Emit JSON instead of human-readable text")
	return common
}

// newFlagSet builds a flag set that reports errors through the usage plumbing
// rather than calling os.Exit.
func (a *App) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("cursorjob "+name, flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return &exitCodeError{code: ExitOK}
		}
		return &exitCodeError{code: ExitUsage}
	}
	return nil
}

// client returns the API client, building one from flags and the environment
// unless a test injected one.
func (a *App) client(common *commonFlags) (*cursor.Client, error) {
	if a.Client != nil {
		return a.Client, nil
	}

	apiKey := common.apiKey
	if apiKey == "" {
		apiKey = a.getenv("CURSOR_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("no API key: set CURSOR_API_KEY or pass --api-key (create one at cursor.com/dashboard → API Keys)")
	}

	baseURL := common.baseURL
	if baseURL == "" {
		baseURL = a.getenv("CURSOR_API_BASE_URL")
	}
	return cursor.NewClient(baseURL, apiKey, nil), nil
}

func (a *App) getenv(key string) string {
	if a.Getenv == nil {
		return ""
	}
	return a.Getenv(key)
}
