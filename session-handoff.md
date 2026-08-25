# Session Handoff

## Current Objective

- Goal: reemplazar el borrado ciego de `.claude/agents/` por un manifiesto
  general que permita a `april init` sincronizar sin perder trabajo del
  usuario en re-corridas.
- Current status: feature 2 `scaffold_manifest_sync` cerrada `done`.
- Branch / commit: `main`, cambios sin commitear todavía (pendiente de que
  el humano decida cuándo commitear).

## Completed This Session

- [x] `/improve-codebase-architecture` — 5 candidatos de fricción, reporte
      HTML generado en `/tmp/architecture-review-20260825-100438.html`.
- [x] Plan mode: diseño del manifiesto `.claude/manifest.json`
      (`/home/alvaro/.claude/plans/vamos-a-tratar-el-binary-diffie.md`),
      con 3 decisiones confirmadas por el humano (conservar-y-avisar en
      conflicto, adopción sin tocar nada en migración, manifiesto
      versionado en `.claude/manifest.json`).
- [x] Feature 2 `scaffold_manifest_sync`: implementada por `agent_developer`,
      corregida tras objeción de `reviewer_agent`, re-aprobada, cerrada por
      el humano.
- [x] Adoptada la skill `tdd` en el flujo: `spec_writer.md` (sección
      `## Testing Decisions`) y `agent_developer.md` (paso 2, ciclo
      red→green + mocking en fronteras).

## Verification Evidence

| Check | Command | Result | Notes |
|---|---|---|---|
| Build | `go build ./...` | verde | corrido por `agent_developer` y `reviewer_agent` independientemente |
| Vet | `go vet ./...` | verde | idem |
| Tests | `go test ./... -v` | verde | incluye los 14 tests nuevos de manifiesto |
| Init harness | `./init.sh` | exit 0 | corrido por `reviewer_agent` |
| Manual | `april init` dos veces sobre `/tmp/proyecto-demo` editando `feature_list.json` entre medio | edición sobrevive, manifiesto correcto | corrido por `agent_developer` |

## Files Changed

- `scaffold.go` — manifiesto, `planScaffoldFromFS`, `applyPlan` reescrito;
  `classifyExistingEntries`/`isExistingHarness`/`agentDirToClean` eliminados.
- `scaffold_test.go` — 2 tests eliminados, 1 ajustado, 14 nuevos.
- `.gitignore` (raíz) — excluye `/.claude/manifest.json` (dogfooding).
- `feature_list.json` — feature 2 agregada y cerrada `done`.
- `.claude/agents/spec_writer.md`, `.claude/agents/agent_developer.md` —
  principios de la skill `tdd` incorporados.
- `progress/current.md`, `progress/history.md` — este cierre de sesión.

## Decisions Made

- Manifiesto vive en `.claude/manifest.json`, versionado en git (no
  gitignorado en proyectos scaffoldeados).
- Conflicto real (usuario tocó + plantilla cambió) → conservar y avisar por
  consola, sin generar archivo `.april-new` al lado.
- Migración desde versiones sin manifiesto → modo adopción: primera corrida
  no toca nada existente, solo adopta hashes de línea base.
- `.gitignore` queda fuera del mecanismo de manifiesto — sigue con su merge
  ad-hoc actual.
- Feature 2 se registró y cerró sin pasar antes por `bootstrap_project`
  (feature 1, todavía `pending`) — decisión explícita del humano para no
  bloquear el fix de arquitectura en la ceremonia de Grill.

## Blockers / Risks

- Feature 1 `bootstrap_project` sigue `pending`, sin Grill completo
  (`docs/architecture.md`, `conventions.md`, `verification.md` en
  `_pendiente_`). No bloquea nada hoy, pero es deuda de proceso abierta.
- `TestProgressHistoryMdProtegidoEnConflictoRealSinCasoEspecial` cubre
  `progress/history.md`; no se agregó un test gemelo explícito para
  `progress/current.md` (el revisor lo consideró redundante porque el
  mecanismo no distingue por nombre de archivo, pero es asimetría de
  cobertura si alguien audita luego).

## Next Session Startup

1. Read `AGENTS.md`.
2. Read `feature_list.json` and `progress.md`.
3. Review this handoff.
4. Run `./init.sh` or the documented verification command before editing.

## Recommended Next Step

- Decidir si se retoma `bootstrap_project` (Grill de `docs/*.md`) o se
  registra otra feature de producto directamente, y si conviene commitear
  el estado actual antes de seguir.