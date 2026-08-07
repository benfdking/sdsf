package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/benfdking/sdsf/cli/cursorjob/internal/cursor"
	"github.com/benfdking/sdsf/cli/cursorjob/internal/gitrepo"
)

// stringList collects a repeatable string flag.
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(value string) error {
	if value == "" {
		return fmt.Errorf("value must not be empty")
	}
	*l = append(*l, value)
	return nil
}

// keyValueList collects repeatable KEY=VALUE flags.
type keyValueList map[string]string

func (m keyValueList) String() string {
	pairs := make([]string, 0, len(m))
	for key, value := range m {
		pairs = append(pairs, key+"="+value)
	}
	return strings.Join(pairs, ",")
}

func (m keyValueList) Set(value string) error {
	key, val, found := strings.Cut(value, "=")
	if !found || key == "" {
		return fmt.Errorf("expected KEY=VALUE, got %q", value)
	}
	m[key] = val
	return nil
}

func cmdRun(ctx context.Context, a *App, args []string) error {
	fs := a.newFlagSet("run")
	common := addCommonFlags(fs)

	var repos stringList
	envVars := keyValueList{}
	fs.Var(&repos, "repo", "Repository URL to work in (repeatable; defaults to this repo's origin)")
	fs.Var(envVars, "env", "Environment variable for the run as KEY=VALUE (repeatable)")

	ref := fs.String("ref", "", "Starting git ref (defaults to the current branch when inferring the repo)")
	model := fs.String("model", "", "Model id, e.g. claude-opus-4-8")
	name := fs.String("name", "", "Display name for the agent")
	mode := fs.String("mode", "", "Execution mode: agent or plan")
	envType := fs.String("env-type", "", "Execution environment: cloud, pool, or machine")
	envName := fs.String("env-name", "", "Pool or machine name, when --env-type is pool or machine")
	autoPR := fs.Bool("auto-pr", false, "Open a pull request when the run finishes")
	skipReviewer := fs.Bool("skip-reviewer", false, "Skip the automatic reviewer request")
	currentBranch := fs.Bool("current-branch", false, "Push to the starting branch instead of a new one")
	noGit := fs.Bool("no-git", false, "Do not infer the repository from the working directory")

	follow := fs.Bool("follow", false, "Attach and block until the run finishes")
	fs.BoolVar(follow, "f", false, "Shorthand for --follow")
	verbose := fs.Bool("verbose", false, "With --follow, also show reasoning and tool calls")

	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: cursorjob run [flags] <prompt>")
		fmt.Fprintln(a.Stderr, "\nSubmits a job and prints its ids. Reads the prompt from stdin when none is given.")
		fmt.Fprintln(a.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	prompt, err := a.readPrompt(fs.Args())
	if err != nil {
		return err
	}

	if *mode != "" && *mode != "agent" && *mode != "plan" {
		return usagef("--mode must be agent or plan, got %q", *mode)
	}
	if *envName != "" && *envType == "" {
		return usagef("--env-name needs --env-type (pool or machine)")
	}

	client, err := a.client(common)
	if err != nil {
		return err
	}

	resolvedRepos, err := a.resolveRepos(ctx, repos, *ref, *noGit)
	if err != nil {
		return err
	}

	req := cursor.CreateAgentRequest{
		Prompt:              cursor.Prompt{Text: prompt},
		Name:                *name,
		Repos:               resolvedRepos,
		WorkOnCurrentBranch: *currentBranch,
		AutoCreatePR:        *autoPR,
		SkipReviewerRequest: *skipReviewer,
		Mode:                *mode,
	}
	if *model != "" {
		req.Model = &cursor.Model{ID: *model}
	}
	if *envType != "" {
		req.Env = &cursor.Env{Type: *envType, Name: *envName}
	}
	if len(envVars) > 0 {
		req.EnvVars = envVars
	}

	created, err := client.CreateAgent(ctx, req)
	if err != nil {
		return err
	}

	switch {
	case common.json && !*follow:
		return writeJSON(a.Stdout, created)
	case common.json:
		// Keep the newline-delimited shape the stream uses, so the ids are still
		// available as the first record.
		if err := writeJSONLine(a.Stdout, map[string]any{
			"type":  "created",
			"agent": created.Agent,
			"run":   created.Run,
		}); err != nil {
			return err
		}
	default:
		a.printSubmitted(&created.Agent, &created.Run)
	}

	if !*follow {
		return nil
	}
	return a.watch(ctx, client, created.Run.AgentID, created.Run.ID, watchConfig{
		json:    common.json,
		verbose: *verbose,
	})
}

func cmdFollowup(ctx context.Context, a *App, args []string) error {
	fs := a.newFlagSet("followup")
	common := addCommonFlags(fs)
	mode := fs.String("mode", "", "Execution mode: agent or plan")
	follow := fs.Bool("follow", false, "Attach and block until the run finishes")
	fs.BoolVar(follow, "f", false, "Shorthand for --follow")
	verbose := fs.Bool("verbose", false, "With --follow, also show reasoning and tool calls")

	fs.Usage = func() {
		fmt.Fprintln(a.Stderr, "Usage: cursorjob followup [flags] <agent-id> <prompt>")
		fmt.Fprintln(a.Stderr, "\nQueues another run on an existing agent, keeping its context.")
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
	agentID := rest[0]

	prompt, err := a.readPrompt(rest[1:])
	if err != nil {
		return err
	}

	client, err := a.client(common)
	if err != nil {
		return err
	}

	run, err := client.CreateRun(ctx, agentID, cursor.CreateRunRequest{
		Prompt: cursor.Prompt{Text: prompt},
		Mode:   *mode,
	})
	if err != nil {
		return err
	}

	switch {
	case common.json && !*follow:
		return writeJSON(a.Stdout, run)
	case common.json:
		if err := writeJSONLine(a.Stdout, map[string]any{"type": "created", "run": run}); err != nil {
			return err
		}
	default:
		fmt.Fprintf(a.Stdout, "run   %s\n", run.ID)
		fmt.Fprintf(a.Stdout, "agent %s\n", run.AgentID)
		fmt.Fprintf(a.Stderr, "\nAttach with: cursorjob attach %s %s\n", run.AgentID, run.ID)
	}

	if !*follow {
		return nil
	}
	return a.watch(ctx, client, run.AgentID, run.ID, watchConfig{
		json:    common.json,
		verbose: *verbose,
	})
}

func (a *App) printSubmitted(agent *cursor.Agent, run *cursor.Run) {
	fmt.Fprintf(a.Stdout, "agent %s\n", agent.ID)
	fmt.Fprintf(a.Stdout, "run   %s\n", run.ID)
	if agent.URL != "" {
		fmt.Fprintf(a.Stdout, "url   %s\n", agent.URL)
	}
	fmt.Fprintf(a.Stderr, "\nAttach with: cursorjob attach %s\n", agent.ID)
}

// readPrompt takes the prompt from the positional args, or from stdin when
// they are empty or a single "-". Reading stdin is only attempted when it is
// not an interactive terminal, so a bare `cursorjob run` prints usage instead
// of silently hanging.
func (a *App) readPrompt(args []string) (string, error) {
	joined := strings.TrimSpace(strings.Join(args, " "))
	if joined != "" && joined != "-" {
		return joined, nil
	}

	if joined != "-" && !a.stdinIsPiped() {
		return "", usagef("a prompt is required (pass it as arguments, or pipe it on stdin)")
	}
	if a.Stdin == nil {
		return "", usagef("a prompt is required")
	}

	data, err := io.ReadAll(a.Stdin)
	if err != nil {
		return "", fmt.Errorf("read prompt from stdin: %w", err)
	}
	prompt := strings.TrimSpace(string(data))
	if prompt == "" {
		return "", usagef("the prompt on stdin was empty")
	}
	return prompt, nil
}

func (a *App) stdinIsPiped() bool {
	file, ok := a.Stdin.(*os.File)
	if !ok {
		// A non-file stdin only happens in tests, where reading is the point.
		return a.Stdin != nil
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice == 0
}

// resolveRepos turns --repo/--ref into the API's repo list, falling back to the
// origin remote of the working directory.
func (a *App) resolveRepos(ctx context.Context, repos stringList, ref string, noGit bool) ([]cursor.Repo, error) {
	if len(repos) > 0 {
		resolved := make([]cursor.Repo, 0, len(repos))
		for _, repo := range repos {
			resolved = append(resolved, cursor.Repo{
				URL:         gitrepo.NormalizeRemote(repo),
				StartingRef: ref,
			})
		}
		return resolved, nil
	}

	if noGit {
		return nil, nil
	}

	origin, err := gitrepo.OriginURL(ctx, a.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("no --repo given and could not infer one: %w", err)
	}

	startingRef := ref
	if startingRef == "" {
		// Best effort: a detached HEAD or a fresh repo just means the server
		// picks the default branch.
		if branch, err := gitrepo.CurrentBranch(ctx, a.WorkDir); err == nil {
			startingRef = branch
		}
	}

	fmt.Fprintf(a.Stderr, "Using repo %s", origin)
	if startingRef != "" {
		fmt.Fprintf(a.Stderr, " @ %s", startingRef)
	}
	fmt.Fprintln(a.Stderr)

	return []cursor.Repo{{URL: origin, StartingRef: startingRef}}, nil
}
