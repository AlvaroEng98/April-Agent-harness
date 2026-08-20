# Session History

<!-- Append-only log. Most recent at the top. -->

## 2026-08-20 — Deepening: extraer classifyExistingEntries en scaffold.go

**Disparador**: `/improve-codebase-architecture` (reporte en `/tmp/.../architecture-review-1787253425.html`),
candidato #1, elegido y grillado con el humano.

**Cambio**: el loop inline de `planScaffold` que decidía si `absTarget` ya
es un harness existente (mira `AGENTS.md`/`feature_list.json`) y qué
directorio de agentes limpiar, se extrajo a `classifyExistingEntries(absTarget, entries)`
— pura, sin I/O. `planScaffold` delega en un solo call site
(`scaffold.go:90`). Sin cambio de comportamiento observable.

**Test**: `TestClassifyExistingEntries` nuevo en `scaffold_test.go`, tabla
de 5 casos (vacío, solo AGENTS.md, solo feature_list.json, ambos, otros
archivos no relacionados) — antes esta rama solo se verificaba vía
integración (`TestCmdInitExistingHarnessRegeneratesAgents`).

**Flujo**: `agent_developer` falló a mitad de su reporte final por un error
de conexión API (no del código); el diff aplicado se verificó igual —
`go build ./...` y `go test ./... -v` limpios, 13 tests previos + 5
sub-casos nuevos PASS. `reviewer_agent`: `APPROVED` sin objeción.

**Cierre**: humano aprobó explícitamente. No es feature de
`feature_list.json` (deepening puntual fuera del backlog de
`bootstrap_project`).

## 2026-08-20 — Fix crítico de release: `go:embed` pattern `AGENT.md` roto

**Disparador**: `go test ./...` fallaba en CI con
`scaffold.go:29:20: pattern AGENT.md: no matching files found`, bloqueando el release.

**Causa raíz**: rename previo de `AGENT.md` → `AGENTS.md` dejó 5 referencias
residuales al nombre viejo: `scaffold.go` (embed, detección de harness
existente, mensaje "next steps"), `scaffold_test.go` (fixtures/comentarios),
`init.sh` (chequeo en `BASE_FILES`, hacía fallar `./init.sh` siempre) y
`session-handoff.md` (texto embebido, se propagaba a cada proyecto
scaffoldeado con `harness init`).

**Flujo**: 2 rondas de `agent_developer` + 2 rondas de `reviewer_agent`.
Ronda 1: fix en `scaffold.go`/`scaffold_test.go`, build/test verdes pero
`reviewer_agent` dio `CHANGES_REQUESTED` — quedaban `init.sh` y
`session-handoff.md` sin corregir (mismo bug, ejecutable/embebido, no solo
Go). Ronda 2: corregidos ambos, `./init.sh` limpio, grep global sin
residuales en código/scripts/embed (solo quedan menciones en `README.md`/
`CHANGELOG.md`, documentación histórica fuera de alcance). Veredicto final
`APPROVED`.

**Cierre**: humano aprobó cierre explícitamente. No era feature de
`feature_list.json` (bug ad-hoc de release, fuera del backlog de
`bootstrap_project`), por eso no se creó entrada allí.
