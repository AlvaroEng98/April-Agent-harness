# Verificación post `april init`

Checklist de qué revisar en un proyecto recién scaffoldeado por el CLI
`april` (ver `scaffold.go:29`, directiva `go:embed`). Fuente: dos orígenes
embebidos, copiados sin transformación salvo `.gitignore` (merge si el
destino ya tenía uno).

## Copiados directo a la raíz del destino

- `AGENT.md`
- `CLAUDE.md`
- `init.sh`
- `session-handoff.md`
- `CHECKPOINTS.md`
- `.claude/agents/` — 5 agentes: `agent_developer.md`, `orquestador.md`,
  `planner_agent.md`, `reviewer_agent.md`, `sdd_agent_author.md`
- `.claude/hooks/` — `block-dangerous-git.sh`, `recap.sh`
- `.claude/settings.json`
- `.claude/skills/` — `codebase-design`, `domain-modeling`,
  `git-guardrails-claude-code`, `grilling`, `grill-with-docs`,
  `writing-for-agents`

## Copiados desde `templates/` (prefijo eliminado, lienzo limpio con estado)

- `config.json`
- `feature_list.json`
- `.gitignore`
- `docs/architecture.md`
- `docs/conventions.md`
- `docs/specs.md`
- `docs/verification.md`
- `progress/current.md`
- `progress/history.md`

## Directorio vacío creado aparte

- `specs/`

## Qué NO se scaffoldea (propio de este repo, no del template)

- `docs/adr/`
- `docs/skills-adoptadas.md`
- `release-notes.sh`, `sync-changelog.sh`, `.goreleaser.yaml`, `CHANGELOG.md`

## Cómo verificar un init real

1. Revisar que `config.json` y `feature_list.json` no tengan placeholders
   (`__YOUR_PROJECT_NAME__`, `__ONE_LINE_DESCRIPTION_OF_YOUR_PROJECT__`).
2. Revisar que `docs/architecture.md`, `docs/conventions.md` y
   `docs/verification.md` tampoco tengan `__YOUR_PROJECT_NAME__`.
3. Ejecutar `./init.sh` dentro del proyecto scaffoldeado y confirmar que las
   5 secciones terminan en `[OK]`.
