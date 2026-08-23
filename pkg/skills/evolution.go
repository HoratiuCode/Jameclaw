package skills

import (
	"errors"
	"fmt"
	"strings"
)

type EvolutionAction string

const (
	EvolutionCreate EvolutionAction = "create"
	EvolutionEdit   EvolutionAction = "edit"
	EvolutionPatch  EvolutionAction = "patch"
)

type EvolutionRequest struct {
	Action      EvolutionAction `json:"action"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Content     string          `json:"content,omitempty"`
	Find        string          `json:"find,omitempty"`
	Replace     string          `json:"replace,omitempty"`
}

type EvolutionResult struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func ApplyEvolution(workspace string, req EvolutionRequest) (EvolutionResult, error) {
	if strings.TrimSpace(req.Name) == "" {
		return EvolutionResult{}, errors.New("skill name is required")
	}
	manager := NewSkillManager(workspace)
	health, err := manager.Apply(ManagedSkillRequest{
		Action:  string(req.Action),
		Name:    req.Name,
		Content: req.Content,
		Find:    req.Find,
		Replace: req.Replace,
	})
	if err != nil {
		return EvolutionResult{}, err
	}
	if health.Path == "" {
		return EvolutionResult{}, fmt.Errorf("skill %q was not created", req.Name)
	}
	return EvolutionResult{Name: health.Name, Path: health.Path}, nil
}

func normalizeSkillName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
