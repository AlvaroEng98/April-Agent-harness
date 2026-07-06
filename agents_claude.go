package main

type claudeGenerator struct{}

func (g *claudeGenerator) Transform(data []byte) []byte {
	return data
}

func (g *claudeGenerator) GetSubdir() string {
	return "agents"
}
