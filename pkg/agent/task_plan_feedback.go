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

const taskPlanSystemPrompt = `Decide whether the request is clear enough to begin work.
Proceed independently whenever a reasonable, low-risk, reversible assumption lets the work move forward. State that assumption briefly as the first plan bullet when useful.
Return exactly "CLARIFY: " followed by one concise question only when the missing detail affects an irreversible, external, security-sensitive, or materially different outcome. Do not provide a plan in that case.
Otherwise, return only a concise, user-visible execution plan of 3-4 Markdown bullets. Describe concrete outcomes and checks, not private reasoning. End every bullet with "Tools: " followed by the exact tool names expected for that step, or "none" when no tool is needed. Do not perform the task or make claims of completion.`

const (
	taskPlanTimeout        = 20 * time.Second
	taskPlanPublishTimeout = 3 * time.Second
)

// publishTaskPlan sends a transparent plan before the first task execution.
// It is best-effort: planning must never prevent the actual request from running.
func (al *AgentLoop) publishTaskPlan(ctx context.Context, ts *turnState, model string) string {
	if !al.shouldPublishTaskPlan(ts) {
		return ""
	}
	maxSteps := al.cfg.Agents.Defaults.GetTaskPlanMaxSteps()
	planCtx, cancel := context.WithTimeout(ctx, taskPlanTimeout)
	defer cancel()
	toolNames := utils.Truncate(strings.Join(ts.agent.Tools.List(), ", "), 1400)
	response, err := ts.agent.Provider.Chat(planCtx, []providers.Message{
		{Role: "system", Content: fmt.Sprintf("%s Limit the plan to at most %d bullets. Choose tool names only from this available set: %s.", taskPlanSystemPrompt, maxSteps, toolNames)},
		{Role: "user", Content: "Request:\n" + ts.userMessage},
	}, nil, model, map[string]any{"max_tokens": 350, "temperature": 0.2})
	if err != nil || response == nil || strings.TrimSpace(response.Content) == "" {
		return ""
	}
	if question, ok := strings.CutPrefix(strings.TrimSpace(response.Content), "CLARIFY:"); ok {
		return strings.TrimSpace(question)
	}
	plan := normalizeTaskPlan(response.Content, maxSteps)
	if plan == "" {
		return ""
	}
	pubCtx, pubCancel := context.WithTimeout(ctx, taskPlanPublishTimeout)
	defer pubCancel()
	if err := al.bus.PublishOutbound(pubCtx, bus.OutboundMessage{
		Channel: ts.channel,
		ChatID:  ts.chatID,
		Content: "🧭 **Plan**\n\n" + plan,
	}); err != nil {
		return ""
	}
	ts.planPublished = true
	al.emitReasoningStep(ts, "share_plan", "Shared the execution plan with the user before starting work.", nil)
	return ""
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
	// Planning with a second model request makes ordinary work feel stalled.
	// Fast lane is the default: the main agent begins responding immediately.
	// A user can still explicitly ask for a plan or steps before work starts.
	if al.cfg.Agents.Defaults.TaskPlanFeedback.OnlyOnExplicitRequest {
		return requestsVisiblePlan(request)
	}
	return hasTaskPlanningSignal(request)
}

func requestsVisiblePlan(request string) bool {
	request = strings.ToLower(request)
	return strings.Contains(request, "plan") || strings.Contains(request, "steps") || strings.Contains(request, "how will you")
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
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "[ ]"), "[x]"), "[X]"))
		bullets = append(bullets, "- [ ] "+line)
		if len(bullets) == maxSteps {
			break
		}
	}
	if len(bullets) == 0 {
		return ""
	}
	return utils.Truncate(strings.Join(bullets, "\n"), maxChars)
}
