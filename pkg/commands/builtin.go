package commands

// BuiltinDefinitions returns all built-in command definitions.
// Each command group is defined in its own cmd_*.go file.
// Definitions are stateless — runtime dependencies are provided
// via the Runtime parameter passed to handlers at execution time.
func BuiltinDefinitions() []Definition {
	return []Definition{
		startCommand(),
		statusCommand(),
		helpCommand(),
		showCommand(),
		listCommand(),
		useCommand(),
		learnCommand(),
		skillsCommand(),
		nameCommand(),
		emojiCommand(),
		personaCommand(),
		styleCommand(),
		switchCommand(),
		checkCommand(),
		clearCommand(),
		sessionStatsCommand(),
		undoCommand(),
		compactCommand(),
		stopCommand(),
		queueCommand(),
		modelCommand(),
		newSessionCommand(),
		sessionsCommand(),
		usageCommand(),
		insightsCommand(),
		approvalsCommand(),
		automationsCommand(),
		automationActionCommand("run", "Run an automation now", "run", nil),
		automationActionCommand("pause", "Pause an automation", "pause", boolPtr(false)),
		automationActionCommand("resume", "Resume an automation", "resume", boolPtr(true)),
		subagentsCommand(),
		reloadCommand(),
	}
}
