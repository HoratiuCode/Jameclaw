package status

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectRepoVersionStatusWithRunner_Outdated(t *testing.T) {
	t.Parallel()

	runner := func(_ context.Context, args ...string) (string, error) {
		switch key := argsKey(args...); key {
		case "rev-parse --is-inside-work-tree":
			return "true", nil
		case "rev-parse --short HEAD":
			return "abc1234", nil
		case "ls-remote --symref origin HEAD":
			return "ref: refs/heads/main\tHEAD\nfedcba987654321\tHEAD", nil
		default:
			return "", errors.New("unexpected git command: " + key)
		}
	}

	status, err := detectRepoVersionStatusWithRunner(context.Background(), runner)
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "abc1234", status.LocalCommit)
	assert.Equal(t, "fedcba9", status.RemoteCommit)
	assert.Equal(t, "refs/heads/main", status.RemoteRef)
	assert.True(t, status.IsOutdated)
}

func TestDetectRepoVersionStatusWithRunner_UpToDate(t *testing.T) {
	t.Parallel()

	runner := func(_ context.Context, args ...string) (string, error) {
		switch key := argsKey(args...); key {
		case "rev-parse --is-inside-work-tree":
			return "true", nil
		case "rev-parse --short HEAD":
			return "abc1234", nil
		case "ls-remote --symref origin HEAD":
			return "ref: refs/heads/main\tHEAD\nabc1234\tHEAD", nil
		default:
			return "", errors.New("unexpected git command: " + key)
		}
	}

	status, err := detectRepoVersionStatusWithRunner(context.Background(), runner)
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "abc1234", status.LocalCommit)
	assert.Equal(t, "abc1234", status.RemoteCommit)
	assert.False(t, status.IsOutdated)
}

func TestDetectRepoVersionStatusWithRunner_SkipsRemoteFailure(t *testing.T) {
	t.Parallel()

	runner := func(_ context.Context, args ...string) (string, error) {
		switch key := argsKey(args...); key {
		case "rev-parse --is-inside-work-tree":
			return "true", nil
		case "rev-parse --short HEAD":
			return "abc1234", nil
		case "ls-remote --symref origin HEAD":
			return "", errors.New("network error")
		default:
			return "", errors.New("unexpected git command: " + key)
		}
	}

	status, err := detectRepoVersionStatusWithRunner(context.Background(), runner)
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "abc1234", status.LocalCommit)
	assert.Empty(t, status.RemoteCommit)
	assert.False(t, status.IsOutdated)
}

func argsKey(args ...string) string {
	if len(args) == 0 {
		return ""
	}
	key := args[0]
	for i := 1; i < len(args); i++ {
		key += " " + args[i]
	}
	return key
}
