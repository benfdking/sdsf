package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/benfdking/sdsf/cli/cursorjob/internal/cursor"
)

const cmuxCommandTimeout = 2 * time.Second

// cmuxJob mirrors a watched Cursor run into the cmux workspace that launched
// cursorjob. The integration is deliberately best effort: missing binaries,
// stale workspace ids, and socket failures never affect the Cursor run.
type cmuxJob struct {
	app         *App
	binary      string
	workspaceID string
	surfaceID   string

	mu           sync.Mutex
	agentName    string
	runID        string
	seeded       bool
	gitSignature string
	branchLabel  string
	prLabel      string
	prURL        string
	workspace    string
	tab          string
	status       string
}

func (a *App) newCMUXJob() *cmuxJob {
	workspaceID := a.getenv("CMUX_WORKSPACE_ID")
	if workspaceID == "" {
		return nil
	}

	binary := a.getenv("CMUX_BUNDLED_CLI_PATH")
	if binary == "" {
		binary = a.getenv("CMUX_CODEX_HOOK_CMUX_BIN")
	}
	if binary == "" && a.RunCommand != nil {
		binary = "cmux"
	}
	if binary == "" {
		if resolved, err := exec.LookPath("cmux"); err == nil {
			binary = resolved
		}
	}
	if binary == "" {
		const bundled = "/Applications/cmux.app/Contents/Resources/bin/cmux"
		if info, err := os.Stat(bundled); err == nil && !info.IsDir() {
			binary = bundled
		}
	}
	if binary == "" {
		return nil
	}

	return &cmuxJob{
		app:         a,
		binary:      binary,
		workspaceID: workspaceID,
		surfaceID:   a.getenv("CMUX_SURFACE_ID"),
	}
}

// startDetachedCMUXWatch preserves cmux updates for the CLI's normal detached
// submission mode. It starts a quiet `cursorjob wait` child targeted at the
// same workspace/surface; run --follow uses the current process instead.
func (a *App) startDetachedCMUXWatch(common *commonFlags, agentID, runID string) {
	if a.newCMUXJob() == nil {
		return
	}
	executable, err := os.Executable()
	if err != nil {
		return
	}

	args := []string{"wait", agentID, runID}
	env := os.Environ()
	if common != nil {
		if common.apiKey != "" {
			env = replaceEnv(env, "CURSOR_API_KEY", common.apiKey)
		}
		if common.baseURL != "" {
			env = replaceEnv(env, "CURSOR_API_BASE_URL", common.baseURL)
		}
	}

	if a.StartCommand != nil {
		_ = a.StartCommand(executable, args, env)
		return
	}

	cmd := exec.Command(executable, args...)
	cmd.Env = env
	cmd.Dir = a.WorkDir
	if err := cmd.Start(); err != nil {
		return
	}
	_ = cmd.Process.Release()
}

func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func (j *cmuxJob) start(agentName, runID string) {
	if j == nil {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()

	j.agentName = strings.TrimSpace(agentName)
	if j.agentName == "" {
		j.agentName = "Cursor job"
	}
	j.runID = runID
	j.renameWorkspaceLocked(j.agentName)
	j.setRunStatusLocked(cursor.RunStatusRunning)
	j.logLocked("progress", fmt.Sprintf("Watching Cursor run %s", runID))
}

// observe applies a fresh API snapshot. The first snapshot establishes a
// baseline, so attaching to an existing run does not produce notifications for
// old branches or PRs. Later in-flight git changes do produce one cmux ping.
func (j *cmuxJob) observe(run *cursor.Run) {
	if j == nil || run == nil {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()

	branchLabel, prLabel, prURL, signature := cmuxGitMetadata(run.Git)
	changed := j.seeded && signature != "" && signature != j.gitSignature
	j.seeded = true
	j.gitSignature = signature

	if branchLabel != "" && branchLabel != j.branchLabel {
		j.branchLabel = branchLabel
		j.renameTabLocked(branchLabel)
		j.setStatusLocked("cursorjob_branch", truncateDisplay(branchLabel, 64), "arrow.triangle.branch", "#64D2FF")
	}
	if prLabel != "" && (prLabel != j.prLabel || prURL != j.prURL) {
		j.prLabel = prLabel
		j.prURL = prURL
		j.setStatusLocked("cursorjob_pr", prLabel, "link", "#BF5AF2")
		j.renameWorkspaceLocked(truncateDisplay(j.agentName+" · "+prLabel, 100))
	}
	j.setRunStatusLocked(run.Status)

	if changed && !cursor.IsTerminal(run.Status) {
		title := "Cursor changes ready"
		if prURL != "" {
			title = "Cursor PR ready"
		} else if branchLabel != "" {
			title = "Cursor branch ready"
		}
		j.notifyLocked(title, cmuxGitSummary(branchLabel, prURL))
		j.logLocked("info", cmuxGitSummary(branchLabel, prURL))
	}
}

func (j *cmuxJob) finish(run *cursor.Run) {
	if j == nil || run == nil {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()

	// A terminal snapshot can contain the first branch/PR metadata. Apply it to
	// the names, but collapse all terminal changes into one final notification.
	branchLabel, prLabel, prURL, signature := cmuxGitMetadata(run.Git)
	j.seeded = true
	j.gitSignature = signature
	if branchLabel != "" && branchLabel != j.branchLabel {
		j.branchLabel = branchLabel
		j.renameTabLocked(branchLabel)
		j.setStatusLocked("cursorjob_branch", truncateDisplay(branchLabel, 64), "arrow.triangle.branch", "#64D2FF")
	}
	if prLabel != "" && (prLabel != j.prLabel || prURL != j.prURL) {
		j.prLabel, j.prURL = prLabel, prURL
		j.setStatusLocked("cursorjob_pr", prLabel, "link", "#BF5AF2")
		j.renameWorkspaceLocked(truncateDisplay(j.agentName+" · "+prLabel, 100))
	}
	j.setRunStatusLocked(run.Status)

	title := "Cursor job " + strings.ToLower(run.Status)
	body := cmuxGitSummary(j.branchLabel, j.prURL)
	if body == "" {
		body = fmt.Sprintf("Run %s ended with %s", j.runID, run.Status)
	}
	j.notifyLocked(title, body)
	level := "success"
	if run.Status != cursor.RunStatusFinished {
		level = "error"
	}
	j.logLocked(level, fmt.Sprintf("Cursor run %s: %s", j.runID, run.Status))
}

func (j *cmuxJob) renameWorkspaceLocked(title string) {
	if title == "" || title == j.workspace {
		return
	}
	j.workspace = title
	j.runLocked("rename-workspace", "--workspace", j.workspaceID, "--", title)
}

func (j *cmuxJob) renameTabLocked(title string) {
	if title == "" || title == j.tab || j.surfaceID == "" {
		return
	}
	j.tab = title
	j.runLocked("rename-tab", "--workspace", j.workspaceID, "--surface", j.surfaceID, "--", truncateDisplay(title, 100))
}

func (j *cmuxJob) setRunStatusLocked(status string) {
	value, icon, color := cmuxRunStatus(status)
	if value == "" || value == j.status {
		return
	}
	j.status = value
	j.setStatusLocked("cursorjob", value, icon, color)
}

func (j *cmuxJob) setStatusLocked(key, value, icon, color string) {
	args := []string{"set-status", key, value, "--workspace", j.workspaceID}
	if icon != "" {
		args = append(args, "--icon", icon)
	}
	if color != "" {
		args = append(args, "--color", color)
	}
	j.runLocked(args...)
}

func (j *cmuxJob) notifyLocked(title, body string) {
	args := []string{"notify", "--title", title, "--subtitle", j.agentName, "--body", body, "--workspace", j.workspaceID}
	if j.surfaceID != "" {
		args = append(args, "--surface", j.surfaceID)
	}
	j.runLocked(args...)
}

func (j *cmuxJob) logLocked(level, message string) {
	j.runLocked("log", "--level", level, "--source", "cursorjob", "--workspace", j.workspaceID, "--", message)
}

func (j *cmuxJob) runLocked(args ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), cmuxCommandTimeout)
	defer cancel()

	if j.app.RunCommand != nil {
		_ = j.app.RunCommand(ctx, j.binary, args...)
		return
	}
	cmd := exec.CommandContext(ctx, j.binary, args...)
	_ = cmd.Run()
}

func cmuxRunStatus(status string) (value, icon, color string) {
	switch status {
	case cursor.RunStatusCreating, cursor.RunStatusRunning, "":
		return "Running", "sparkle", "#0A84FF"
	case cursor.RunStatusFinished:
		return "Finished", "checkmark", "#30D158"
	case cursor.RunStatusCancelled:
		return "Cancelled", "xmark", "#FF9F0A"
	case cursor.RunStatusExpired:
		return "Expired", "clock", "#FF9F0A"
	case cursor.RunStatusError:
		return "Error", "warning", "#FF453A"
	default:
		return status, "sparkle", "#0A84FF"
	}
}

func cmuxGitMetadata(git *cursor.Git) (branchLabel, prLabel, prURL, signature string) {
	if git == nil {
		return "", "", "", ""
	}
	var branches, prs, prURLs, signatureParts []string
	for _, branch := range git.Branches {
		signatureParts = append(signatureParts, branch.RepoURL+"\x00"+branch.Branch+"\x00"+branch.PRURL)
		if branch.Branch != "" {
			branches = appendUnique(branches, branch.Branch)
		}
		if branch.PRURL != "" {
			prs = appendUnique(prs, cmuxPRLabel(branch.PRURL))
			prURLs = appendUnique(prURLs, branch.PRURL)
		}
	}
	sort.Strings(branches)
	sort.Strings(prs)
	sort.Strings(prURLs)
	sort.Strings(signatureParts)
	return strings.Join(branches, " · "), strings.Join(prs, " · "), strings.Join(prURLs, "\nPR: "), strings.Join(signatureParts, "\x01")
}

func cmuxPRLabel(prURL string) string {
	trimmed := strings.TrimSuffix(strings.TrimSpace(prURL), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) >= 2 && parts[len(parts)-2] == "pull" && parts[len(parts)-1] != "" {
		return "PR #" + parts[len(parts)-1]
	}
	return "PR"
}

func cmuxGitSummary(branch, prURL string) string {
	var lines []string
	if branch != "" {
		lines = append(lines, "Branch: "+branch)
	}
	if prURL != "" {
		lines = append(lines, "PR: "+prURL)
	}
	return strings.Join(lines, "\n")
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
