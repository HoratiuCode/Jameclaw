package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sipeed/jameclaw/pkg/skills"
)

// SkillManageTool is the agent-facing local skill authoring surface. All
// writes are delegated to skills.SkillManager so chat, API, and Desktop use
// the same validation and safety rules.
type SkillManageTool struct {
	manager *skills.SkillManager
}

func NewSkillManageTool(workspace string) *SkillManageTool {
	return &SkillManageTool{manager: skills.NewSkillManager(workspace)}
}

func (t *SkillManageTool) Name() string { return "skill_manage" }

func (t *SkillManageTool) Description() string {
	return "Create and improve reusable local skills. Actions: create, edit, patch, write_file, remove_file, health, archive, restore, curate. Skills require valid YAML frontmatter and are security-scanned before being saved. Use references/, templates/, scripts/, and assets/ for progressive disclosure."
}

func (t *SkillManageTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{"create", "edit", "patch", "write_file", "remove_file", "health", "archive", "restore", "curate", "bundle_list", "bundle_save"},
			},
			"name":               map[string]any{"type": "string", "description": "Lowercase hyphenated skill name."},
			"category":           map[string]any{"type": "string", "description": "Optional category folder."},
			"content":            map[string]any{"type": "string", "description": "Complete SKILL.md for create/edit."},
			"file_path":          map[string]any{"type": "string", "description": "Relative support path under references/, templates/, scripts/, or assets/."},
			"file_content":       map[string]any{"type": "string"},
			"find":               map[string]any{"type": "string", "description": "Exact text to replace for patch."},
			"replace":            map[string]any{"type": "string"},
			"archive_path":       map[string]any{"type": "string", "description": "Archive path for restore."},
			"bundle_skills":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"bundle_description": map[string]any{"type": "string"},
			"bundle_instruction": map[string]any{"type": "string"},
		},
		"required": []string{"action"},
	}
}

func (t *SkillManageTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if err := ctx.Err(); err != nil {
		return ErrorResult(err.Error())
	}
	action, _ := args["action"].(string)
	action = strings.ToLower(strings.TrimSpace(action))
	name, _ := args["name"].(string)
	if action == "health" {
		if strings.TrimSpace(name) == "" {
			data, err := json.MarshalIndent(t.manager.ListHealth(), "", "  ")
			if err != nil {
				return ErrorResult(err.Error())
			}
			return NewToolResult(string(data))
		}
		data, err := json.MarshalIndent(t.manager.Health(name), "", "  ")
		if err != nil {
			return ErrorResult(err.Error())
		}
		return NewToolResult(string(data))
	}
	if action == "archive" {
		path, err := t.manager.Archive(name)
		if err != nil {
			return ErrorResult(err.Error())
		}
		return SilentResult(fmt.Sprintf("Skill %q archived recoverably at %s", name, path))
	}
	if action == "restore" {
		path, _ := args["archive_path"].(string)
		if err := t.manager.Restore(path); err != nil {
			return ErrorResult(err.Error())
		}
		return SilentResult(fmt.Sprintf("Skill restored from %s", path))
	}
	if action == "curate" {
		items, err := t.manager.Curate()
		if err != nil {
			return ErrorResult(err.Error())
		}
		data, _ := json.Marshal(items)
		return SilentResult(fmt.Sprintf("Curator archived %d recoverable skill(s): %s", len(items), string(data)))
	}
	if action == "bundle_list" {
		items, err := t.manager.ListBundles()
		if err != nil {
			return ErrorResult(err.Error())
		}
		data, _ := json.MarshalIndent(items, "", "  ")
		return NewToolResult(string(data))
	}
	if action == "bundle_save" {
		raw, _ := args["bundle_skills"].([]any)
		skillNames := make([]string, 0, len(raw))
		for _, value := range raw {
			if name, ok := value.(string); ok {
				skillNames = append(skillNames, name)
			}
		}
		description, _ := args["bundle_description"].(string)
		instruction, _ := args["bundle_instruction"].(string)
		if err := t.manager.SaveBundle(skills.SkillBundle{Name: name, Description: description, Skills: skillNames, Instruction: instruction}); err != nil {
			return ErrorResult(err.Error())
		}
		return SilentResult(fmt.Sprintf("Skill bundle %q saved.", name))
	}

	category, _ := args["category"].(string)
	content, _ := args["content"].(string)
	filePath, _ := args["file_path"].(string)
	fileContent, _ := args["file_content"].(string)
	find, _ := args["find"].(string)
	replace, _ := args["replace"].(string)
	health, err := t.manager.Apply(skills.ManagedSkillRequest{Action: action, Name: name, Category: category, Content: content, FilePath: filePath, FileContent: fileContent, Find: find, Replace: replace})
	if err != nil {
		return ErrorResult(err.Error())
	}
	data, _ := json.MarshalIndent(health, "", "  ")
	return SilentResult(fmt.Sprintf("Skill %q updated successfully. Health:\n%s", health.Name, string(data)))
}
