package main

// AgentGenerator define cómo transformar agentes para cada herramienta.
type AgentGenerator interface {
	Transform(data []byte) []byte
	GetSubdir() string
}

// generators registra los generadores disponibles por directorio de herramienta.
var generators = map[string]AgentGenerator{
	".claude":   &claudeGenerator{},
	".opencode": &opencodeGenerator{},
}
