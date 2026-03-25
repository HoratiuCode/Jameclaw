package main

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/jameclaw/cmd/jameclaw/internal"
	"github.com/sipeed/jameclaw/pkg/config"
)

func TestNewJameclawCommand(t *testing.T) {
	cmd := NewJameclawCommand()

	require.NotNil(t, cmd)

	short := fmt.Sprintf("%s jameclaw - Personal AI Assistant v%s\n\n", internal.Logo, config.GetVersion())

	assert.Equal(t, "jameclaw", cmd.Use)
	assert.Equal(t, short, cmd.Short)

	assert.True(t, cmd.HasSubCommands())
	assert.True(t, cmd.HasAvailableSubCommands())

	assert.False(t, cmd.HasFlags())

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	allowedCommands := []string{
		"agent",
		"auth",
		"cron",
		"gateway",
		"migrate",
		"model",
		"onboard",
		"skills",
		"status",
		"version",
	}

	subcommands := cmd.Commands()
	assert.Len(t, subcommands, len(allowedCommands))

	for _, subcmd := range subcommands {
		found := slices.Contains(allowedCommands, subcmd.Name())
		assert.True(t, found, "unexpected subcommand %q", subcmd.Name())

		assert.False(t, subcmd.Hidden)
	}
}

func TestRunInteractiveDefaultCommandSelection(t *testing.T) {
	originalIsInteractive := defaultCommandIsInteractive
	originalOnboardComplete := startupOnboardComplete
	originalSelector := defaultCommandSelector
	originalAgent := runDefaultAgentCommand
	originalWeb := runDefaultWebCommand
	originalTUI := runDefaultTUICommand
	originalOutput := defaultCommandOutput
	t.Cleanup(func() {
		defaultCommandIsInteractive = originalIsInteractive
		startupOnboardComplete = originalOnboardComplete
		defaultCommandSelector = originalSelector
		runDefaultAgentCommand = originalAgent
		runDefaultWebCommand = originalWeb
		runDefaultTUICommand = originalTUI
		defaultCommandOutput = originalOutput
	})

	defaultCommandIsInteractive = func() bool { return true }
	startupOnboardComplete = func() bool { return true }
	defaultCommandOutput = io.Discard

	cases := []struct {
		name   string
		choice string
		want   string
	}{
		{name: "agent", choice: "agent", want: "agent"},
		{name: "web", choice: "web", want: "web"},
		{name: "tui", choice: "tui", want: "tui"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			defaultCommandSelector = func() (string, error) { return tc.choice, nil }
			runDefaultAgentCommand = func() error {
				got = "agent"
				return nil
			}
			runDefaultWebCommand = func() error {
				got = "web"
				return nil
			}
			runDefaultTUICommand = func() error {
				got = "tui"
				return nil
			}

			err := runInteractiveDefaultCommand()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRunInteractiveDefaultCommandPromptsAgainAfterInvalidChoice(t *testing.T) {
	originalIsInteractive := defaultCommandIsInteractive
	originalOnboardComplete := startupOnboardComplete
	originalSelector := defaultCommandSelector
	originalAgent := runDefaultAgentCommand
	originalOutput := defaultCommandOutput
	t.Cleanup(func() {
		defaultCommandIsInteractive = originalIsInteractive
		startupOnboardComplete = originalOnboardComplete
		defaultCommandSelector = originalSelector
		runDefaultAgentCommand = originalAgent
		defaultCommandOutput = originalOutput
	})

	defaultCommandIsInteractive = func() bool { return true }
	startupOnboardComplete = func() bool { return true }

	var output bytes.Buffer
	defaultCommandOutput = &output

	choices := []string{"bad-option", "agent"}
	defaultCommandSelector = func() (string, error) {
		choice := choices[0]
		choices = choices[1:]
		return choice, nil
	}

	called := false
	runDefaultAgentCommand = func() error {
		called = true
		return nil
	}

	err := runInteractiveDefaultCommand()
	require.NoError(t, err)
	assert.True(t, called)
	assert.Contains(t, output.String(), `Unknown startup option "bad-option"`)
}

func TestRunInteractiveDefaultCommandRunsOnboardBeforeSelection(t *testing.T) {
	originalIsInteractive := defaultCommandIsInteractive
	originalOnboard := runStartupOnboard
	originalOnboardComplete := startupOnboardComplete
	originalSelector := defaultCommandSelector
	originalWeb := runDefaultWebCommand
	originalOutput := defaultCommandOutput
	t.Cleanup(func() {
		defaultCommandIsInteractive = originalIsInteractive
		runStartupOnboard = originalOnboard
		startupOnboardComplete = originalOnboardComplete
		defaultCommandSelector = originalSelector
		runDefaultWebCommand = originalWeb
		defaultCommandOutput = originalOutput
	})

	defaultCommandIsInteractive = func() bool { return true }
	defaultCommandOutput = io.Discard

	onboardComplete := false
	runStartupOnboard = func() bool {
		onboardComplete = true
		return false
	}
	startupOnboardComplete = func() bool { return onboardComplete }
	defaultCommandSelector = func() (string, error) { return "web", nil }

	called := false
	runDefaultWebCommand = func() error {
		called = true
		return nil
	}

	err := runInteractiveDefaultCommand()
	require.NoError(t, err)
	assert.True(t, onboardComplete)
	assert.True(t, called)
}

func TestNormalizeStartupChoice(t *testing.T) {
	assert.Equal(t, "agent", normalizeStartupChoice("1"))
	assert.Equal(t, "agent", normalizeStartupChoice("modify"))
	assert.Equal(t, "web", normalizeStartupChoice("2"))
	assert.Equal(t, "web", normalizeStartupChoice("browser"))
	assert.Equal(t, "tui", normalizeStartupChoice("3"))
	assert.Equal(t, "", normalizeStartupChoice("unknown"))
}
