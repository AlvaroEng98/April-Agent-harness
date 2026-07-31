package main

type Tool struct {
	Name   string
	Binary string
	Dir    string
}

type DetectedTool struct {
	Tool
	Found bool
}

var defaultTools = []Tool{
	{"Claude Code", "claude", ".claude"},
	{"OpenCode", "opencode", ".opencode"},
}

func DetectTools(tools []Tool, lookPath func(string) (string, error)) []DetectedTool {
	result := make([]DetectedTool, len(tools))
	for i, t := range tools {
		_, err := lookPath(t.Binary)
		result[i] = DetectedTool{Tool: t, Found: err == nil}
	}
	return result
}

func availableTools(detected []DetectedTool) []DetectedTool {
	var available []DetectedTool
	for _, d := range detected {
		if d.Found {
			available = append(available, d)
		}
	}
	return available
}
