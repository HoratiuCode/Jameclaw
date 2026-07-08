package memory

import "context"

// Provider is the stable boundary for long-term memory backends.
type Provider interface {
	Name() string
	BuildSystemPrompt(ctx context.Context) (string, error)
	PreTurn(ctx context.Context, userMessage string) (string, error)
	PostTurn(ctx context.Context, userMessage, assistantMessage string) error
}

type Manager struct {
	providers []Provider
}

func NewManager(providers ...Provider) *Manager {
	return &Manager{providers: append([]Provider(nil), providers...)}
}

func (m *Manager) Add(provider Provider) {
	if provider == nil {
		return
	}
	m.providers = append(m.providers, provider)
}

func (m *Manager) Providers() []Provider {
	if m == nil {
		return nil
	}
	return append([]Provider(nil), m.providers...)
}

func (m *Manager) BuildSystemPrompt(ctx context.Context) []string {
	if m == nil {
		return nil
	}
	blocks := make([]string, 0, len(m.providers))
	for _, provider := range m.providers {
		if block, err := provider.BuildSystemPrompt(ctx); err == nil && block != "" {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func (m *Manager) PreTurn(ctx context.Context, userMessage string) []string {
	if m == nil {
		return nil
	}
	blocks := make([]string, 0, len(m.providers))
	for _, provider := range m.providers {
		if block, err := provider.PreTurn(ctx, userMessage); err == nil && block != "" {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func (m *Manager) PostTurn(ctx context.Context, userMessage, assistantMessage string) {
	if m == nil {
		return
	}
	for _, provider := range m.providers {
		_ = provider.PostTurn(ctx, userMessage, assistantMessage)
	}
}
