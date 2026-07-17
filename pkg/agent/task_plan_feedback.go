package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/jameclaw/pkg/bus"
	"github.com/sipeed/jameclaw/pkg/constants"
	"github.com/sipeed/jameclaw/pkg/providers"
	"github.com/sipeed/jameclaw/pkg/utils"
)

const taskPlanSystemPrompt = `Create a concise, user-visible execution plan for the request below.
Return only 3-4 Markdown bullets. Describe concrete outcomes and checks, not private reasoning.
Do not perform the task, make claims of completion, ask questions, or mention tools unless useful to the user.`

const (
	taskPlanTimeout        = 20 * time.Second
	taskPlanPublishTimeout = 3 * time.Second
)

// publishTaskPlan sends a transparent plan before the first task execution.
// It is best-effort: planning must never prevent the actual request from running.
func (al *AgentLoop) publishTaskPlan(ctx context.Context, ts *turnState, model string) {
	if !al.shouldPublishTaskPlan(ts) {
		return
	}
	maxSteps := al.cfg.Agents.Defaults.GetTaskPlanMaxSteps()
	planCtx, cancel := context.WithTimeout(ctx, taskPlanTimeout)
	defer cancel()
	response, err := ts.agent.Provider.Chat(planCtx, []providers.Message{
		{Role: "system", Content: fmt.Sprintf("%s Limit the plan to at most %d bullets.", taskPlanSystemPrompt, maxSteps)},
		{Role: "user", Content: "Request:\n" + ts.userMessage},
	}, nil, model, map[string]any{"max_tokens": 350, "temperature": 0.2})
	if err != nil || response == nil || strings.TrimSpace(response.Content) == "" {
		return
	}
	plan := normalizeTaskPlan(response.Content, maxSteps)
	if plan == "" {
		return
	}
	pubCtx, pubCancel := context.WithTimeout(ctx, taskPlanPublishTimeout)
	defer pubCancel()
	if err := al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: ts.channel,
		ChatID:  ts.chatID,
		Content: "🧭 **Plan**\n\n" + plan,
	}); err != nil {
		return
	}
	al.emitReasoningStep(ts, "share_plan", "Shared the execution plan with the user before starting work.", nil)
}

func (al *AgentLoop) shouldPublishTaskPlan(ts *turnState) bool {
	if al == nil || ts == nil || al.bus == nil || !al.cfg.Agents.Defaults.TaskPlanFeedback.Enabled {
		return false
	}
	if ts.channel == "" || ts.chatID == "" || constants.IsInternalChannel(ts.channel) || ts.opts.NoHistory {
		return false
	}
	request := strings.TrimSpace(ts.userMessage)
	if strings.HasPrefix(request, "/") || len([]rune(request)) < al.cfg.Agents.Defaults.GetTaskPlanMinLength() {
		return false
	}
	return hasTaskPlanningSignal(request)
}

func hasTaskPlanningSignal(request string) bool {
	request = strings.ToLower(request)
	for _, signal := range []string{
		"build", "create", "implement", "fix", "debug", "research", "analyse", "analyze", "compare", "review", "write", "design", "generate", "develop", "investigate", "plan", "steps", "task", "project",
	} {
		if strings.Contains(request, signal) {
			return true
		}
	}
	return len(strings.Fields(request)) >= 13
}

func normalizeTaskPlan(content string, maxSteps int) string {
	maxChars := 1600
	lines := strings.Split(content, "\n")
	bullets := make([]string, 0, maxSteps)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "-•* \t0123456789.")
		if line == "" {
			continue
		}
		bullets = append(bullets, "- "+line)
		if len(bullets) == maxSteps {
			break
		}
	}
	if len(bullets) == 0 {
		return ""
	}
	return utils.Truncate(strings.Join(bullets, "\n"), maxChars)
}
