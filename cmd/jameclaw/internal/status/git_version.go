package status

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const gitCheckTimeout = 3 * time.Second

type gitCommandRunner func(context.Context, ...string) (string, error)

type repoVersionStatus struct {
	LocalCommit  string
	RemoteCommit string
	RemoteRef    string
	IsOutdated   bool
}

func detectRepoVersionStatus() (*repoVersionStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCheckTimeout)
	defer cancel()
	return detectRepoVersionStatusWithRunner(ctx, runGitCommand)
}

func detectRepoVersionStatusWithRunner(ctx context.Context, runner gitCommandRunner) (*repoVersionStatus, error) {
	if _, err := runner(ctx, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil, err
	}

	localCommit, err := runner(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return nil, err
	}

	remoteRef, remoteCommit, err := resolveRemoteHead(ctx, runner)
	if err != nil {
		return &repoVersionStatus{LocalCommit: localCommit}, nil
	}

	status := &repoVersionStatus{
		LocalCommit:  localCommit,
		RemoteCommit: remoteCommit,
		RemoteRef:    remoteRef,
	}
	status.IsOutdated = localCommit != "" && remoteCommit != "" && localCommit != remoteCommit
	return status, nil
}

func resolveRemoteHead(ctx context.Context, runner gitCommandRunner) (string, string, error) {
	output, err := runner(ctx, "ls-remote", "--symref", "origin", "HEAD")
	if err != nil {
		return "", "", err
	}

	var remoteRef string
	var remoteCommit string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "ref: ") && strings.HasSuffix(line, "\tHEAD") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				remoteRef = fields[1]
			}
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) == 2 && parts[1] == "HEAD" {
			remoteCommit = strings.TrimSpace(parts[0])
		}
	}

	if remoteCommit == "" {
		return "", "", fmt.Errorf("remote HEAD commit not found")
	}

	if remoteRef == "" {
		remoteRef = "origin/HEAD"
	}

	return remoteRef, shortCommit(remoteCommit), nil
}

func runGitCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func shortCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}
