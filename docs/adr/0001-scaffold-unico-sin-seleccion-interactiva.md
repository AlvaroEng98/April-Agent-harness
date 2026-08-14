# Scaffold único (solo Claude Code), sin detección ni selección interactiva de herramientas

`apil init` copia siempre el mismo árbol embebido (`.claude/agents/`, `AGENT.md`,
etc.), sin preguntar ni detectar qué agente de IA usa el usuario. Hasta
`8517803` (31/07/2026) existió una capa de detección (`detector.go`) y una UI
interactiva de selección multi-herramienta (`selector.go`, con soporte para
Claude y OpenCode); ambas se eliminaron en esa misma commit al simplificar el
proyecto a un único flujo Claude-Code-only. La alternativa (detectar/soportar
múltiples agentes) existió, se construyó y se descartó deliberadamente: este
repo se dogfoodea a sí mismo con Claude Code, y mantener N flujos de scaffold
no pagaba su costo frente a un único camino bien probado.

## Status

accepted

## Consequences

- `docs/architecture.md` describe hoy el módulo real (`main.go`, `config.go`,
  `update.go`); no vuelve a documentar `detector.go`/`selector.go`.
- Reintroducir selección multi-herramienta es una decisión nueva, no un bug:
  requiere abrir otro ADR, no "restaurar" código viejo.
