// Package gitrepo infers repository details from the working directory so the
// CLI can default --repo and --ref to "the repo I'm standing in".
package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// OriginURL returns the origin remote of the repository containing dir,
// normalised to an https GitHub URL.
func OriginURL(ctx context.Context, dir string) (string, error) {
	out, err := run(ctx, dir, "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("read git remote origin: %w", err)
	}
	return NormalizeRemote(out), nil
}

// CurrentBranch returns the checked-out branch, or an error when HEAD is
// detached.
func CurrentBranch(ctx context.Context, dir string) (string, error) {
	out, err := run(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("read current git branch: %w", err)
	}
	if out == "HEAD" {
		return "", fmt.Errorf("HEAD is detached; pass --ref explicitly")
	}
	return out, nil
}

// NormalizeRemote converts the remote forms git prints into the https URL the
// Cursor API expects: scp-style (git@host:owner/repo.git) and ssh:// both
// become https://host/owner/repo.
func NormalizeRemote(remote string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(remote, "git@"):
		trimmed := strings.TrimPrefix(remote, "git@")
		host, path, found := strings.Cut(trimmed, ":")
		if found {
			remote = "https://" + host + "/" + path
		}
	case strings.HasPrefix(remote, "ssh://git@"):
		remote = "https://" + strings.TrimPrefix(remote, "ssh://git@")
	case strings.HasPrefix(remote, "git://"):
		remote = "https://" + strings.TrimPrefix(remote, "git://")
	}

	remote = strings.TrimSuffix(remote, "/")
	return strings.TrimSuffix(remote, ".git")
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		// git puts the useful part on stderr, which Output() captures.
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
