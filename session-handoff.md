# Session Handoff

## Current Objective

- Goal: implementar el backlog derivado de `ROADMAP.md` (E1-E6, "April vs
  gentle-ai") — el árbitro `april status`, las vías de escritura
  autoritativas, el ledger de evidencia de tests/revisión, y las
  extensiones de `april doctor`.
- Current status: **backlog completo — features 1-12, todas `done`**.
  `april status --json` reporta `phase: closed`, `nextRecommended: "nada —
  no hay features pendientes"`.
- Branch / commit: `main`, cambios sin commitear (el humano pidió
  explícitamente no commitear al cierre de esta sesión — lo maneja él
  mismo en el siguiente paso).

## Completed This Session (continuación 28/08/2026)

- [x] Feature 8 `review_depth_by_diff_sensitivity` — `review start
      --json` reporta rutas tocadas y áreas sensibles.
- [x] Feature 12 `tree_hash_respects_gitignore` — `hashTree` respeta
      `.gitignore`, corrige la auto-invalidación de receipts por
      binarios regenerados.
- [x] Feature 9 `doctor_readonly_check` — `april doctor`, chequeo
      read-only de salud (drift de manifiesto + agentes).
      `APPROVED_WITH_OBJECTION` (objeción documentada, no corregida a
      pedido del humano).
- [x] Feature 10 `init_backup_before_apply` — backup automático antes de
      `applyPlan`, rollback manual. `APPROVED` (objeción de cobertura de
      la 1ra ronda cerrada antes del cierre).
- [x] Feature 11 `doctor_debt_ratchet` — ratchet de deuda (TODOs sin
      feature asociada) sobre `april doctor`. `APPROVED` (dos objeciones
      de la 1ra ronda cerradas antes del cierre).
- [x] Consolidación de sesión: `progress/history.md` actualizado,
      `progress/current.md` reseteado para la próxima sesión.

Ver `progress/history.md` (sección "2026-08-28 — Sesión continuación") y
la sección anterior para el detalle completo de features 1-7.

## Verification Evidence

| Check | Command | Result | Notes |
|---|---|---|---|
| Build | `go build ./...` | verde | corrido repetidamente por cada `agent_developer`/`reviewer_agent` |
| Vet | `go vet ./...` | verde | idem |
| Tests | `go test ./... -v` | verde, 207 tests | acumulado features 1-12 (180→207 esta continuación) |
| Init harness | `./init.sh` | exit 0 | `blockedReasons: []` al cierre de cada feature |
| Ledger | `.claude/verify-ledger.jsonl` | receipts vigentes | `kind:test`/`kind:review` para todas las features 1-12 |
| Manual | `april feature set-status`, `april verify record`, `april review record`, `april doctor` corridos en vivo contra este mismo repo | comportamiento correcto | dogfooding real |

## Files Changed (esta continuación, sobre lo ya descrito en `progress/history.md`)

- `review.go`/`review_test.go` — feature 8 (`--json` en `review start`,
  `matchSensitiveAreas`/`computeTouchedPaths`), feature 12
  (`fixedTreeExclusions` compartida).
- `verify.go`/`verify_test.go` — feature 12 (`parseGitignore`/
  `gitignoreMatches`/`loadGitignorePatterns`, wiring en `hashTree`).
- `doctor.go`/`doctor_test.go` (nuevo, feature 9, extendido en feature 11)
  — chequeo read-only + ratchet de deuda (`--freeze-baseline`).
- `scaffold.go`/`scaffold_test.go` — feature 10 (`backupCandidates`/
  `backupBeforeApply`, llamado desde `applyPlan`).
- `.gitignore` — feature 11 (excluye `/.claude/doctor-baseline.json`).
- `main.go` — nuevos casos `doctor`, flags `--freeze-baseline`,
  `printUsage()` actualizado.
- `feature_list.json` — features 8-12 → `done` (todas vía `april feature
  set-status`).
- `specs/review_depth_by_diff_sensitivity/`,
  `specs/tree_hash_respects_gitignore/` — specs + tickets (features
  `sdd: true` de esta continuación).
- `progress/current.md`, `progress/history.md` — bitácora y
  consolidación de esta sesión.
- `.claude/verify-ledger.jsonl` — ledger real con receipts nuevos
  (features 8-12).

## Decisions Made

- Todas las decisiones de fondo de la sesión anterior (advisory → única
  vía de escritura, exclusiones fijas de hash, veredicto interino en
  `set_status.go` coexistiendo con el ledger real) siguen vigentes — ver
  `progress/history.md`.
- `hashTree` y `computeSubjectHash` comparten `fixedTreeExclusions`
  (feature 12) pero permanecen mecanismos separados a propósito — uno
  agnóstico de `fs.FS`/sin git, el otro delegando en `git` real. No se
  unificaron.
- `.claude/doctor-baseline.json` (feature 11) se excluye del hash de
  árbol vía `.gitignore`, no vía código — decisión explícita porque, a
  diferencia del ledger (que se reescribe en cada `record`,
  autoinvalidándose), este archivo solo cambia por una acción explícita y
  rara (`--freeze-baseline`).
- El contrato read-only de `april doctor` (feature 9) se preserva en la
  feature 11: la única escritura vive detrás del flag explícito
  `--freeze-baseline`, que además rechaza sobreescribir un baseline
  existente.
- Objeción de la feature 9 (chequeo de agentes con `strings.Contains("#")`
  en vez de `grep -q "^#"` anclado) cerrada por decisión humana explícita
  — se documenta, no se corrige, por ser edge-case improbable.
- Ambas objeciones de las features 10 y 11 (huecos de cobertura de test)
  sí se pidieron cerrar antes de aprobar — patrón distinto al de la
  feature 9, decidido caso por caso por el humano, no automático.

## Blockers / Risks

- Ninguna feature bloqueada ni pendiente — backlog completo.
- Nada commiteado — el árbol de trabajo tiene todos los cambios de las
  features 1-12 sin `git commit` (bloqueado además por el hook existente
  salvo pedido explícito). El humano pidió expresamente NO commitear al
  cierre de esta sesión — lo maneja él mismo en el siguiente paso, no
  tocar `git commit`/`git add` de forma proactiva.
- `april review start` puede reportar `extraReviewRequired: true` por
  arrastre de diffs de features ya cerradas y sin commitear (ej. la
  10 tocando `scaffold.go`, área sensible) — ruido informativo, no
  bloqueante, desaparece en cuanto se comitee.

## Next Session Startup

1. Leer `AGENTS.md`/`CLAUDE.md`.
2. Correr `./init.sh` — debe estar en verde (`blockedReasons: []`).
3. Correr `april status --json` — debe reportar `phase: closed`,
   `nextRecommended: "nada — no hay features pendientes"`.
4. Preguntar al humano si el trabajo ya se commiteó y si hay backlog
   nuevo que sumar (vía `planner_agent`) — no hay features `pending` que
   retomar automáticamente.

## Recommended Next Step

- El humano decide cuándo y cómo commitear el trabajo acumulado
  (features 1-12, sin commits).
- Si aparece nuevo trabajo, lanzar `planner_agent` con el objetivo
  acordado para sumar features al backlog — el ciclo de Grill/Spec/
  Tickets/Implementación/Revisión de `CLAUDE.md` sigue aplicando igual.
