package command

import (
	"context"
	"fmt"
	"time"

	"github.com/benfdking/sdsf/cli/cursorjob/internal/cursor"
)

func cmdList(ctx context.Context, a *App, args []string) error {
	fs := a.newFlagSet("list")
	common := addCommonFlags(fs)
	limit := fs.Int("limit", 20, "Maximum agents to return (max 100)")
	all := fs.Bool("all", false, "Include archived agents")

	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: cursorjob list [flags]")
		fmt.Fprintln(a.Stderr, "\nLists agents, most recently updated first.")
		fmt.Fprintln(a.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	client, err := a.client(common)
	if err != nil {
		return err
	}

	includeArchived := *all
	page, err := client.ListAgents(ctx, cursor.ListAgentsOptions{
		Limit:           *limit,
		IncludeArchived: &includeArchived,
	})
	if err != nil {
		return err
	}

	if common.json {
		return writeJSON(a.Stdout, page)
	}
	printAgentTable(a.Stdout, page.Items, time.Now())
	return nil
}

func cmdRuns(ctx context.Context, a *App, args []string) error {
	fs := a.newFlagSet("runs")
	common := addCommonFlags(fs)
	limit := fs.Int("limit", 20, "Maximum runs to return (max 100)")

	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: cursorjob runs [flags] <agent-id>")
		fmt.Fprintln(a.Stderr, "\nLists the runs belonging to an agent.")
		fmt.Fprintln(a.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return usagef("exactly one agent id is required")
	}

	client, err := a.client(common)
	if err != nil {
		return err
	}

	page, err := client.ListRuns(ctx, fs.Arg(0), *limit, "")
	if err != nil {
		return err
	}

	if common.json {
		return writeJSON(a.Stdout, page)
	}
	printRunTable(a.Stdout, page.Items, time.Now())
	return nil
}

func cmdShow(ctx context.Context, a *App, args []string) error {
	fs := a.newFlagSet("show")
	common := addCommonFlags(fs)

	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: cursorjob show [flags] <agent-id> [run-id]")
		fmt.Fprintln(a.Stderr, "\nShows a run's status and result. Defaults to the agent's latest run.")
		fmt.Fprintln(a.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 || fs.NArg() > 2 {
		return usagef("expected an agent id and an optional run id")
	}

	client, err := a.client(common)
	if err != nil {
		return err
	}

	agentID := fs.Arg(0)
	runID, err := client.ResolveRunID(ctx, agentID, fs.Arg(1))
	if err != nil {
		return err
	}

	run, err := client.GetRun(ctx, agentID, runID)
	if err != nil {
		return err
	}

	if common.json {
		return writeJSON(a.Stdout, run)
	}
	printRunDetail(a.Stdout, run)
	return nil
}

func cmdCancel(ctx context.Context, a *App, args []string) error {
	fs := a.newFlagSet("cancel")
	common := addCommonFlags(fs)

	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: cursorjob cancel [flags] <agent-id> [run-id]")
		fmt.Fprintln(a.Stderr, "\nCancels an in-flight run. Defaults to the agent's latest run.")
		fmt.Fprintln(a.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 || fs.NArg() > 2 {
		return usagef("expected an agent id and an optional run id")
	}

	client, err := a.client(common)
	if err != nil {
		return err
	}

	agentID := fs.Arg(0)
	runID, err := client.ResolveRunID(ctx, agentID, fs.Arg(1))
	if err != nil {
		return err
	}

	if err := client.CancelRun(ctx, agentID, runID); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "cancelled %s\n", runID)
	return nil
}
