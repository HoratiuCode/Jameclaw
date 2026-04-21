package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/jameclaw/pkg/fileutil"
	"gopkg.in/yaml.v3"
)

const (
	defaultAgentSignatureEmoji = "🦐"
	agentNameLinePrefix        = "Your name is JameClaw"
	personalityHeading         = "## Personality"
)

func runtimeWorkspace(rt *Runtime) string {
	if rt == nil || rt.Config == nil {
		return ""
	}
	return strings.TrimSpace(rt.Config.WorkspacePath())
}

func ReadAgentSignatureEmoji(workspace string) string {
	data, err := os.ReadFile(filepath.Join(workspace, "AGENT.md"))
	if err != nil {
		return defaultAgentSignatureEmoji
	}

	fields, body, _, err := splitMarkdownFrontmatter(string(data))
	if err == nil {
		if raw, ok := fields["emoji"]; ok {
			if emoji := strings.TrimSpace(fmt.Sprint(raw)); emoji != "" {
				return emoji
			}
		}
	}

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, agentNameLinePrefix) {
			continue
		}
		signature := strings.TrimSpace(strings.TrimPrefix(trimmed, agentNameLinePrefix))
		signature = strings.TrimSuffix(signature, ".")
		signature = strings.TrimSpace(signature)
		if signature == "" {
			return defaultAgentSignatureEmoji
		}
		return signature
	}

	return defaultAgentSignatureEmoji
}

func UpdateAgentSignatureEmoji(workspace, emoji string) error {
	path := filepath.Join(workspace, "AGENT.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	emoji = strings.TrimSpace(emoji)
	if emoji == "" {
		emoji = defaultAgentSignatureEmoji
	}

	fields, body, hasFrontmatter, err := splitMarkdownFrontmatter(string(data))
	if err != nil {
		return err
	}
	if fields == nil {
		fields = map[string]any{}
	}
	fields["emoji"] = emoji

	lines := strings.Split(body, "\n")
	replaced := false
	insertAfter := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "You are Jame, the default assistant for this workspace.") {
			insertAfter = i
		}
		if strings.HasPrefix(trimmed, agentNameLinePrefix) {
			lines[i] = agentNameLinePrefix + " " + emoji + "."
			replaced = true
			break
		}
	}

	if !replaced {
		replacement := agentNameLinePrefix + " " + emoji + "."
		if insertAfter >= 0 {
			lines = append(lines[:insertAfter+1], append([]string{replacement}, lines[insertAfter+1:]...)...)
		} else {
			lines = append([]string{replacement}, lines...)
		}
	}

	rendered, err := renderMarkdownFrontmatter(fields, strings.Join(lines, "\n"), hasFrontmatter)
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, []byte(rendered), 0o644)
}

func ReadAgentPersona(workspace string) string {
	data, err := os.ReadFile(filepath.Join(workspace, "SOUL.md"))
	if err != nil {
		return ""
	}
	return readMarkdownSection(string(data), personalityHeading)
}

func UpdateAgentPersona(workspace, persona string) error {
	path := filepath.Join(workspace, "SOUL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	rendered := writeMarkdownSection(string(data), personalityHeading, persona)
	return fileutil.WriteFileAtomic(path, []byte(rendered), 0o644)
}

func ReadAgentSkills(workspace string) []string {
	data, err := os.ReadFile(filepath.Join(workspace, "AGENT.md"))
	if err != nil {
		return nil
	}

	fields, _, _, err := splitMarkdownFrontmatter(string(data))
	if err != nil {
		return nil
	}

	raw, ok := fields["skills"]
	if !ok {
		return nil
	}

	switch values := raw.(type) {
	case []string:
		return normalizeStringList(values)
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s := strings.TrimSpace(fmt.Sprint(value)); s != "" {
				out = append(out, s)
			}
		}
		return normalizeStringList(out)
	default:
		if s := strings.TrimSpace(fmt.Sprint(raw)); s != "" {
			return []string{s}
		}
		return nil
	}
}

func UpdateAgentSkills(workspace string, skills []string) error {
	path := filepath.Join(workspace, "AGENT.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fields, body, hasFrontmatter, err := splitMarkdownFrontmatter(string(data))
	if err != nil {
		return err
	}
	if fields == nil {
		fields = map[string]any{}
	}

	skills = normalizeStringList(skills)
	if len(skills) == 0 {
		delete(fields, "skills")
	} else {
		fields["skills"] = skills
	}

	rendered, err := renderMarkdownFrontmatter(fields, body, hasFrontmatter)
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, []byte(rendered), 0o644)
}

func splitMarkdownFrontmatter(content string) (fields map[string]any, body string, hasFrontmatter bool, err error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return map[string]any{}, content, false, nil
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return map[string]any{}, content, false, nil
	}

	fields = map[string]any{}
	frontmatter := strings.Join(lines[1:end], "\n")
	if strings.TrimSpace(frontmatter) != "" {
		if err := yaml.Unmarshal([]byte(frontmatter), &fields); err != nil {
			return nil, "", false, err
		}
	}
	body = strings.TrimLeft(strings.Join(lines[end+1:], "\n"), "\n")
	return fields, body, true, nil
}

func renderMarkdownFrontmatter(fields map[string]any, body string, hadFrontmatter bool) (string, error) {
	body = strings.TrimLeft(body, "\n")
	if len(fields) == 0 && !hadFrontmatter {
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		return body, nil
	}

	if fields == nil {
		fields = map[string]any{}
	}
	frontmatter, err := yaml.Marshal(fields)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(strings.TrimRight(string(frontmatter), "\n"))
	sb.WriteString("\n---\n")
	sb.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func readMarkdownSection(content, heading string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	heading = strings.TrimSpace(heading)
	if heading == "" {
		return ""
	}

	inSection := false
	section := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inSection {
			if trimmed == heading {
				inSection = true
			}
			continue
		}
		if strings.HasPrefix(trimmed, "## ") && trimmed != heading {
			break
		}
		section = append(section, line)
	}

	return strings.TrimSpace(strings.Join(section, "\n"))
}

func writeMarkdownSection(content, heading, replacement string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	heading = strings.TrimSpace(heading)
	replacement = strings.TrimSpace(replacement)
	if heading == "" {
		return content
	}

	var out []string
	found := false

	for i := 0; i < len(lines); {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != heading {
			out = append(out, lines[i])
			i++
			continue
		}

		found = true
		out = append(out, lines[i])
		i++
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
		if replacement != "" {
			out = append(out, "")
			out = append(out, strings.Split(replacement, "\n")...)
		}
		for i < len(lines) {
			next := strings.TrimSpace(lines[i])
			if strings.HasPrefix(next, "## ") {
				break
			}
			i++
		}
	}

	if !found {
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, heading, "")
		if replacement != "" {
			out = append(out, strings.Split(replacement, "\n")...)
		}
	}

	rendered := strings.Join(out, "\n")
	rendered = strings.TrimRight(rendered, "\n")
	rendered += "\n"
	return rendered
}

func normalizeStringList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
