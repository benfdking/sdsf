// Command cursorjob submits jobs to Cursor's cloud agents and attaches to them
// until they finish.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/benfdking/sdsf/cli/cursorjob/internal/command"
)

func main() {
	// Cancelling the context detaches cleanly and prints how to reattach; the
	// run itself keeps going server-side. A second signal aborts immediately.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(command.NewApp().Execute(ctx, os.Args[1:]))
}
