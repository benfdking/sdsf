# cursorjob

A command line client for [Cursor's Cloud Agents API](https://cursor.com/docs/cloud-agent/api/endpoints). Submit a job from your terminal, walk away, and attach to it later to block until it finishes.

It talks to `https://api.cursor.com/v1` over HTTPS — the jobs run on Cursor's infrastructure (or on your own `cursor-agent worker` fleet, via `--env-type`), not on your machine. Closing your laptop does not stop them.

## Why this exists

`cursor-agent -p` already runs a prompt to completion, but it runs it *in your terminal* as a foreground process — close the shell and the job dies, and there is no scriptable way to reattach. This CLI covers the other shape: submit, detach, reattach, and get a meaningful exit code.

## Install

```bash
go build -o ~/.local/bin/cursorjob ./cli/cursorjob
```

## Authentication

Create a key at **cursor.com/dashboard → API Keys**, then:

```bash
export CURSOR_API_KEY=crsr_...
```

`--api-key` overrides it per invocation. `CURSOR_API_BASE_URL` / `--base-url` point the client at a different host.

## Usage

Submit a job. With no `--repo`, the repository and starting ref are inferred from the working directory's `origin` remote and current branch:

```bash
$ cursorjob run "port the retry logic from the slack provider to linear"
Using repo https://github.com/benfdking/sdsf @ main
agent bc-6f2a...
run   run-91c3...
url   https://cursor.com/agents/bc-6f2a...
```

Attach to it whenever — the transcript streams to stdout, progress to stderr:

```bash
$ cursorjob attach bc-6f2a...
Attached to run-91c3... (ctrl-c to detach; the run keeps going)

I'll start by reading the existing retry helper...

FINISHED in 4m12s
PR:     https://github.com/benfdking/sdsf/pull/7
```

Submit and block in one step with `--follow`:

```bash
cursorjob run --follow --auto-pr "fix the flaky conversation test"
```

### cmux integration

When launched from a [cmux](https://cmux.com) workspace, every active watch
(`run --follow`, `followup --follow`, `attach`, or `wait`) mirrors the job into
the originating workspace:

- The workspace uses the Cursor agent/PR name (and the PR number once known).
- The originating tab uses the remote Cursor branch name.
- Sidebar status pills show the run, branch, and PR state.
- A cmux notification is posted when branch/PR metadata changes and when the
  run reaches a terminal state.

Cursor can publish branch metadata before the run ends, so cursorjob refreshes
it every 15 seconds while the transcript stream is open. A detached `run` or
`followup` starts a quiet background `wait`, preserving these updates after the
submitting command exits. Outside cmux—or if its local control socket is not
available—the integration is silently skipped and never changes the job's exit
status.

Block silently in a script, and branch on the outcome:

```bash
if cursorjob wait bc-6f2a...; then
  echo "agent finished"
else
  echo "agent did not finish cleanly (exit $?)"
fi
```

Other commands:

```bash
cursorjob list                      # agents, most recently updated first
cursorjob runs bc-6f2a...           # every run on an agent
cursorjob show bc-6f2a...           # latest run's status and final result
cursorjob followup bc-6f2a... "now also update the docs"
cursorjob cancel bc-6f2a...
```

Every command takes `--json`. For `attach` and `wait` the output is newline-delimited — one object per stream event, then a final `{"type":"run",...}` record — so it pipes straight into `jq`:

```bash
cursorjob attach --json bc-6f2a... | jq -r 'select(.type == "assistant") | .data.text'
```

## Exit codes

`attach`, `wait`, and `run --follow` exit on the run's terminal status, so shell logic works without parsing output:

| Code | Meaning |
| --- | --- |
| 0 | Run `FINISHED` |
| 1 | CLI, network, or auth failure — or you detached with ctrl-c |
| 2 | Usage error |
| 10 | Run `ERROR` |
| 11 | Run `CANCELLED` |
| 12 | Run `EXPIRED` |

## How waiting works

`WatchRun` prefers the API's SSE stream and falls back to polling, so a wait survives a flaky connection:

- A run that is already terminal returns straight from `GET /runs/{id}` — attaching to a finished job is instant, not an error.
- A dropped connection reconnects with `Last-Event-ID`, resuming rather than replaying. The retry counter resets whenever a stream makes progress, so a long run can reconnect indefinitely while a hard-failing one still gives up.
- Once the stream's retention window passes the API returns `410 stream_expired`; the wait switches to polling `GET /runs/{id}` instead of failing.
- The final status always comes from `GET /runs/{id}`, never inferred from a possibly-truncated stream.
- An unrecognised status is treated as in-flight, so a new server-side state can't make a wait exit early and report success.

Ctrl-c detaches and prints the reattach command; the run keeps going server-side.

## Build and test

```bash
go build ./...
go test ./...
```

Tests serve the API in-process through a custom `http.RoundTripper`, so the suite binds no sockets and needs no network.
