package commands

import "github.com/sipeed/jameclaw/pkg/config"

// Runtime provides runtime dependencies to command handlers. It is constructed
// per-request by the agent loop so that per-request state (like session scope)
// can coexist with long-lived callbacks (like GetModelInfo).
type Runtime struct {
	Config             *config.Config
	GetModelInfo       func() (name, provider string)
	ListAgentIDs       func() []string
	ListDefinitions    func() []Definition
	ListSkillNames     func() []string
	GetActiveSkills    func() []string
	SetActiveSkills    func([]string) error
	GetEnabledChannels func() []string
	GetActiveTurn      func() any // Returning any to avoid circular dependency with agent package
	SwitchModel        func(value string) (oldModel string, err error)
	SwitchChannel      func(value string) error
	ClearHistory       func() error
	SessionStats       func() (messageCount, tokenEstimate, contextWindow int, summary string, err error)
	UndoLastTurn       func() (removedMessages int, err error)
	CompressSession    func() (dropped, remaining int, compressed bool, err error)
	ListAutomations    func() []AutomationSummary
	RunAutomation      func(identifier string) error
	SetAutomationState func(identifier string, enabled bool) error
	ReloadConfig       func() error
	StopAgent          func(hard bool) error
	PendingQueue       func() int
	ClearQueue         func() int
}

// AutomationSummary is the safe, channel-neutral view exposed to commands.
// It deliberately omits prompts and output so a status command cannot leak
// large or sensitive automation payloads into a messaging channel.
type AutomationSummary struct {
	ID         string
	Name       string
	Enabled    bool
	Running    bool
	Status     string
	Schedule   string
	LastStatus string
}
