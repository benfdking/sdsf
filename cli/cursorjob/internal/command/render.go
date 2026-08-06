package command

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/benfdking/sdsf/cli/cursorjob/internal/cursor"
)

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// writeJSONLine emits compact single-line JSON, so streaming output stays
// newline-delimited and pipeable into jq line by line.
func writeJSONLine(w io.Writer, value any) error {
	return json.NewEncoder(w).Encode(value)
}

func newTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// shortAge renders a timestamp as a compact relative age ("3m", "2h"). Unknown
// or unparseable timestamps render as "-" so a format change on the API side
// degrades the column instead of failing the command.
func shortAge(timestamp string, now time.Time) string {
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return "-"
	}
	d := max(now.Sub(parsed), 0)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func formatDuration(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

func truncateDisplay(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

// printAgentTable renders the agent listing.
func printAgentTable(w io.Writer, agents []cursor.Agent, now time.Time) {
	if len(agents) == 0 {
		fmt.Fprintln(w, "No agents found.")
		return
	}
	table := newTable(w)
	fmt.Fprintln(table, "AGENT ID\tSTATUS\tAGE\tLATEST RUN\tNAME")
	for _, agent := range agents {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
			agent.ID,
			orDash(agent.Status),
			shortAge(agent.UpdatedAt, now),
			orDash(agent.LatestRunID),
			truncateDisplay(agent.Name, 48),
		)
	}
	_ = table.Flush()
}

// printRunTable renders the run listing for a single agent.
func printRunTable(w io.Writer, runs []cursor.Run, now time.Time) {
	if len(runs) == 0 {
		fmt.Fprintln(w, "No runs found.")
		return
	}
	table := newTable(w)
	fmt.Fprintln(table, "RUN ID\tSTATUS\tAGE\tDURATION\tBRANCH")
	for _, run := range runs {
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
			run.ID,
			orDash(run.Status),
			shortAge(run.CreatedAt, now),
			formatDuration(run.DurationMs),
			orDash(primaryBranch(run.Git)),
		)
	}
	_ = table.Flush()
}

// printRunDetail renders a single run, including whatever the agent produced.
func printRunDetail(w io.Writer, run *cursor.Run) {
	table := newTable(w)
	fmt.Fprintf(table, "Run:\t%s\n", run.ID)
	fmt.Fprintf(table, "Agent:\t%s\n", run.AgentID)
	fmt.Fprintf(table, "Status:\t%s\n", run.Status)
	if run.CreatedAt != "" {
		fmt.Fprintf(table, "Created:\t%s\n", run.CreatedAt)
	}
	if run.DurationMs > 0 {
		fmt.Fprintf(table, "Duration:\t%s\n", formatDuration(run.DurationMs))
	}
	_ = table.Flush()

	printGit(w, run.Git)

	if strings.TrimSpace(run.Result) != "" {
		fmt.Fprintf(w, "\n%s\n", strings.TrimSpace(run.Result))
	}
}

func printGit(w io.Writer, git *cursor.Git) {
	if git == nil || len(git.Branches) == 0 {
		return
	}
	fmt.Fprintln(w)
	for _, branch := range git.Branches {
		switch {
		case branch.PRURL != "":
			fmt.Fprintf(w, "PR:     %s\n", branch.PRURL)
		case branch.Branch != "":
			fmt.Fprintf(w, "Branch: %s (%s)\n", branch.Branch, branch.RepoURL)
		}
	}
}

func primaryBranch(git *cursor.Git) string {
	if git == nil {
		return ""
	}
	for _, branch := range git.Branches {
		if branch.Branch != "" {
			return branch.Branch
		}
	}
	return ""
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
