package commands

import (
	"context"
	"fmt"
	"strings"
)

func skillsCommand() Definition {
	return Definition{
		Name:        "skills",
		Aliases:     []string{"skill"},
		Description: "Manage active skills",
		SubCommands: []SubCommand{
			{
				Name:        "show",
				Description: "Show active skills",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					workspace := runtimeWorkspace(rt)
					if workspace == "" {
						return req.Reply(unavailableMsg)
					}
					skills := activeSkills(rt, workspace)
					if len(skills) == 0 {
						return req.Reply("No active skills. Use /skills add <skill> to enable one.")
					}
					return req.Reply(fmt.Sprintf("Active Skills:\n- %s", strings.Join(skills, "\n- ")))
				},
			},
			{
				Name:        "add",
				Description: "Add an active skill",
				ArgsUsage:   "<skill>",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					return updateSkillsFromCommand(req, rt, true)
				},
			},
			{
				Name:        "remove",
				Description: "Remove an active skill",
				ArgsUsage:   "<skill>",
				Handler: func(_ context.Context, req Request, rt *Runtime) error {
					return updateSkillsFromCommand(req, rt, false)
				},
			},
		},
	}
}

func updateSkillsFromCommand(req Request, rt *Runtime, add bool) error {
	workspace := runtimeWorkspace(rt)
	if workspace == "" {
		return req.Reply(unavailableMsg)
	}

	fields := strings.Fields(strings.TrimSpace(req.Text))
	if len(fields) < 3 {
		return req.Reply("Usage: /skills add <skill> or /skills remove <skill>")
	}

	skill := strings.TrimSpace(fields[2])
	if skill == "" {
		return req.Reply("Please provide a skill name.")
	}

	if rt != nil && rt.ListSkillNames != nil {
		known := make(map[string]struct{}, len(rt.ListSkillNames()))
		for _, name := range rt.ListSkillNames() {
			known[normalizeCommandName(name)] = struct{}{}
		}
		if len(known) > 0 {
			if _, ok := known[normalizeCommandName(skill)]; !ok {
				return req.Reply(fmt.Sprintf("Unknown skill: %s\nUse /list skills to see installed skills.", skill))
			}
		}
	}

	current := activeSkills(rt, workspace)
	index := make(map[string]int, len(current))
	for i, name := range current {
		index[normalizeCommandName(name)] = i
	}

	key := normalizeCommandName(skill)
	if add {
		if _, ok := index[key]; ok {
			return req.Reply(fmt.Sprintf("Skill %q is already active.", skill))
		}
		current = append(current, skill)
	} else {
		pos, ok := index[key]
		if !ok {
			return req.Reply(fmt.Sprintf("Skill %q is not active.", skill))
		}
		current = append(current[:pos], current[pos+1:]...)
	}

	current = normalizeStringList(current)
	if err := UpdateAgentSkills(workspace, current); err != nil {
		return req.Reply(fmt.Sprintf("Failed to update skills: %v", err))
	}
	if rt != nil && rt.SetActiveSkills != nil {
		if err := rt.SetActiveSkills(current); err != nil {
			return req.Reply(fmt.Sprintf("Saved skills, but could not update the running agent: %v", err))
		}
	}

	if add {
		return req.Reply(fmt.Sprintf("Added skill %q to the active set.", skill))
	}
	return req.Reply(fmt.Sprintf("Removed skill %q from the active set.", skill))
}

func activeSkills(rt *Runtime, workspace string) []string {
	if rt != nil && rt.GetActiveSkills != nil {
		return normalizeStringList(rt.GetActiveSkills())
	}
	return ReadAgentSkills(workspace)
}
