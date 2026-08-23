package commands

// learnCommand is intentionally passthrough: AgentLoop rewrites /learn into a
// normal model turn so the agent can inspect sources and call skill_manage.
func learnCommand() Definition {
	return Definition{
		Name:        "learn",
		Description: "Learn a reusable skill from a workflow, files, URL, or conversation",
		Usage:       "/learn [workflow, files, URL, or instructions]",
	}
}
