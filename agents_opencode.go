package main

import (
	"fmt"
	"strings"
)

type opencodeGenerator struct{}

func (g *opencodeGenerator) Transform(data []byte) []byte {
	content := string(data)

	if !strings.HasPrefix(content, "---") {
		return data
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return data
	}

	frontmatter := parts[1]
	body := parts[2]

	lines := strings.Split(frontmatter, "\n")

	description := ""
	tools := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "description:") {
			description = strings.TrimPrefix(trimmed, "description:")
			description = strings.TrimSpace(description)
		} else if strings.HasPrefix(trimmed, "tools:") {
			tools = strings.TrimPrefix(trimmed, "tools:")
			tools = strings.TrimSpace(tools)
		}
	}

	var sb strings.Builder
	sb.WriteString("---\n")

	if description != "" {
		sb.WriteString(fmt.Sprintf("description: %s\n", description))
	}

	sb.WriteString("mode: subagent\n")
	sb.WriteString("temperature: 0.1\n")

	if tools != "" {
		permissions := mapToolsToPermissions(tools)
		if len(permissions) > 0 {
			sb.WriteString("permission:\n")
			for _, p := range permissions {
				sb.WriteString(fmt.Sprintf("  %s\n", p))
			}
		}
	}

	sb.WriteString("---\n")
	sb.WriteString(body)

	return []byte(sb.String())
}

func (g *opencodeGenerator) GetSubdir() string {
	return "agents"
}

func mapToolsToPermissions(tools string) []string {
	var result []string
	seen := map[string]bool{}

	toolList := strings.Split(tools, ",")
	for _, t := range toolList {
		t = strings.TrimSpace(t)

		switch t {
		case "Write", "Edit":
			if !seen["edit: allow"] {
				result = append(result, "edit: allow")
				seen["edit: allow"] = true
			}
		case "Bash":
			if !seen["bash: allow"] {
				result = append(result, "bash: allow")
				seen["bash: allow"] = true
			}
		case "Agent":
			if !seen["task: allow"] {
				result = append(result, "task: allow")
				seen["task: allow"] = true
			}
		case "Question":
			if !seen["question: allow"] {
				result = append(result, "question: allow")
				seen["question: allow"] = true
			}
		}
	}

	return result
}
