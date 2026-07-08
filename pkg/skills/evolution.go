package skills

import (
	"errors"
	"os"
	"path/filepath"
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
	name := normalizeSkillName(req.Name)
	if name == "" {
		return EvolutionResult{}, errors.New("skill name is required")
	}
	skillDir := filepath.Join(workspace, "skills", name)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	switch req.Action {
	case EvolutionCreate:
		if strings.TrimSpace(req.Content) == "" {
			req.Content = "# " + name + "\n\n" + strings.TrimSpace(req.Description) + "\n"
		}
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return EvolutionResult{}, err
		}
		if _, err := os.Stat(skillFile); err == nil {
			return EvolutionResult{}, errors.New("skill already exists")
		}
		if err := os.WriteFile(skillFile, []byte(req.Content), 0o644); err != nil {
			return EvolutionResult{}, err
		}
	case EvolutionEdit:
		if strings.TrimSpace(req.Content) == "" {
			return EvolutionResult{}, errors.New("content is required")
		}
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return EvolutionResult{}, err
		}
		if err := os.WriteFile(skillFile, []byte(req.Content), 0o644); err != nil {
			return EvolutionResult{}, err
		}
	case EvolutionPatch:
		if req.Find == "" {
			return EvolutionResult{}, errors.New("find is required")
		}
		data, err := os.ReadFile(skillFile)
		if err != nil {
			return EvolutionResult{}, err
		}
		next := strings.Replace(string(data), req.Find, req.Replace, 1)
		if next == string(data) {
			return EvolutionResult{}, errors.New("find text not found")
		}
		if err := os.WriteFile(skillFile, []byte(next), 0o644); err != nil {
			return EvolutionResult{}, err
		}
	default:
		return EvolutionResult{}, errors.New("unknown evolution action")
	}

	return EvolutionResult{Name: name, Path: skillFile}, nil
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
