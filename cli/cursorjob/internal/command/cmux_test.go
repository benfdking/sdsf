package command

import (
	"context"
	"strings"
	"testing"

	"github.com/benfdking/sdsf/cli/cursorjob/internal/cursor"
)

func TestCMUXJobUpdatesNamesAndNotifiesOnChangesAndEnd(t *testing.T) {
	var calls [][]string
	app := &App{
		Getenv: func(key string) string {
			return map[string]string{
				"CMUX_WORKSPACE_ID":     "workspace-1",
				"CMUX_SURFACE_ID":       "surface-2",
				"CMUX_BUNDLED_CLI_PATH": "/test/bin/cmux",
			}[key]
		},
		RunCommand: func(_ context.Context, name string, args ...string) error {
			call := append([]string{name}, args...)
			calls = append(calls, call)
			return nil
		},
	}

	job := app.newCMUXJob()
	if job == nil {
		t.Fatal("expected cmux integration inside a cmux workspace")
	}
	job.start("Fix flaky tests", "run-1")
	job.observe(&cursor.Run{ID: "run-1", Status: cursor.RunStatusRunning})
	job.observe(&cursor.Run{ID: "run-1", Status: cursor.RunStatusRunning, Git: &cursor.Git{Branches: []cursor.GitBranch{{
		RepoURL: "github.com/acme/app",
		Branch:  "cursor/fix-flakes",
	}}}})
	job.observe(&cursor.Run{ID: "run-1", Status: cursor.RunStatusRunning, Git: &cursor.Git{Branches: []cursor.GitBranch{{
		RepoURL: "github.com/acme/app",
		Branch:  "cursor/fix-flakes",
		PRURL:   "https://github.com/acme/app/pull/42",
	}}}})
	job.finish(&cursor.Run{ID: "run-1", Status: cursor.RunStatusFinished, Git: &cursor.Git{Branches: []cursor.GitBranch{{
		RepoURL: "github.com/acme/app",
		Branch:  "cursor/fix-flakes",
		PRURL:   "https://github.com/acme/app/pull/42",
	}}}})

	joined := joinCommandCalls(calls)
	for _, want := range []string{
		"rename-workspace --workspace workspace-1 -- Fix flaky tests",
		"rename-tab --workspace workspace-1 --surface surface-2 -- cursor/fix-flakes",
		"rename-workspace --workspace workspace-1 -- Fix flaky tests · PR #42",
		"notify --title Cursor branch ready",
		"notify --title Cursor PR ready",
		"notify --title Cursor job finished",
		"--workspace workspace-1 --surface surface-2",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("commands missing %q:\n%s", want, joined)
		}
	}

	if got := countCommand(calls, "notify"); got != 3 {
		t.Errorf("notify calls = %d, want branch + PR + terminal notifications; calls:\n%s", got, joined)
	}
}

func TestCMUXJobCollapsesTerminalMetadataIntoOneNotification(t *testing.T) {
	var calls [][]string
	app := &App{
		Getenv: func(key string) string {
			return map[string]string{
				"CMUX_WORKSPACE_ID":     "workspace-1",
				"CMUX_SURFACE_ID":       "surface-2",
				"CMUX_BUNDLED_CLI_PATH": "/test/bin/cmux",
			}[key]
		},
		RunCommand: func(_ context.Context, name string, args ...string) error {
			calls = append(calls, append([]string{name}, args...))
			return nil
		},
	}

	job := app.newCMUXJob()
	job.start("Completed task", "run-1")
	terminal := &cursor.Run{ID: "run-1", Status: cursor.RunStatusFinished, Git: &cursor.Git{Branches: []cursor.GitBranch{{
		Branch: "cursor/done",
		PRURL:  "https://github.com/acme/app/pull/7",
	}}}}
	job.observe(terminal)
	job.finish(terminal)

	if got := countCommand(calls, "notify"); got != 1 {
		t.Errorf("notify calls = %d, want one terminal notification; calls:\n%s", got, joinCommandCalls(calls))
	}
}

func TestCMUXJobIsDisabledOutsideCMUX(t *testing.T) {
	app := &App{Getenv: func(string) string { return "" }}
	if job := app.newCMUXJob(); job != nil {
		t.Fatal("cmux integration should be disabled without CMUX_WORKSPACE_ID")
	}
}

func TestDetachedCMUXWatchStartsQuietWaitAndForwardsFlagCredentials(t *testing.T) {
	var gotArgs, gotEnv []string
	app := &App{
		WorkDir: "/test/repo",
		Getenv: func(key string) string {
			return map[string]string{
				"CMUX_WORKSPACE_ID":     "workspace-1",
				"CMUX_BUNDLED_CLI_PATH": "/test/bin/cmux",
			}[key]
		},
		StartCommand: func(_ string, args, env []string) error {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil
		},
	}

	app.startDetachedCMUXWatch(&commonFlags{
		apiKey:  "flag-key",
		baseURL: "https://cursor.example.test",
	}, "bc-1", "run-1")

	if got := strings.Join(gotArgs, " "); got != "wait bc-1 run-1" {
		t.Errorf("child args = %q, want quiet wait command", got)
	}
	if got := envValue(gotEnv, "CURSOR_API_KEY"); got != "flag-key" {
		t.Errorf("child CURSOR_API_KEY = %q", got)
	}
	if got := envValue(gotEnv, "CURSOR_API_BASE_URL"); got != "https://cursor.example.test" {
		t.Errorf("child CURSOR_API_BASE_URL = %q", got)
	}
}

func joinCommandCalls(calls [][]string) string {
	lines := make([]string, 0, len(calls))
	for _, call := range calls {
		lines = append(lines, strings.Join(call[1:], " "))
	}
	return strings.Join(lines, "\n")
}

func countCommand(calls [][]string, command string) int {
	count := 0
	for _, call := range calls {
		if len(call) > 1 && call[1] == command {
			count++
		}
	}
	return count
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}
