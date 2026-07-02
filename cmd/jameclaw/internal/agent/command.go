package agent

import (
	"github.com/spf13/cobra"
)

func NewAgentCommand() *cobra.Command {
	var (
		message    string
		sessionKey string
		model      string
		debug      bool
		reasoning  string
	)

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Interact with the agent directly",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return agentCmd(message, sessionKey, model, debug, reasoning)
		},
	}

	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Send a single message (non-interactive mode)")
	cmd.Flags().StringVarP(&sessionKey, "session", "s", "", "Session key")
	cmd.Flags().StringVarP(&model, "model", "", "", "Model to use")
	cmd.Flags().StringVar(&reasoning, "reasoning", "off", "Reasoning display mode: off, summary, or debug")

	return cmd
}
