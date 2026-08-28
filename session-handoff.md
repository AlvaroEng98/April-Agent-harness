# Session Handoff

## Current Objective

- Goal: implementar el backlog derivado de `ROADMAP.md` (E1-E6, "April vs
  gentle-ai") — el árbitro `april status`, las vías de escritura
  autoritativas, y el ledger de evidencia de tests/revisión.
- Current status: features 1-8 y 12 cerradas `done`. Frontera: feature 9
  (`doctor_readonly_check`), `sdd: false`, `status: pending` — `april
  status --json` recomienda implementar frontera de tickets (vacía para
  esta feature, es `sdd: false` sin tickets).
- Branch / commit: `main`, cambios sin commitear (pendiente de que el
  humano decida cuándo commitear — no se hizo ningún commit esta sesión).

## Completed This Session

- [x] Feature 1 `bootstrap_project` — Fase Grill completa (`docs/*.md`,
      `progress/project-definition.md`), backlog de 10 features aprobado.
- [x] Feature 2 `april_status_arbiter` — `april status --json` (árbitro
      advisory: `phase`/`nextRecommended`/`blockedReasons`/`frontier`).
- [x] Feature 3 `claude_md_routes_by_status` — `CLAUDE.md` enruta por
      `april status`, no por inferencia de prosa.
- [x] Feature 4 `set_status_authoritative_write` — `april feature
      set-status`, única vía de escritura de `feature_list.json` desde su
      cierre en adelante.
- [x] Feature 5 `verify_record_ledger` — `april verify record`, ledger
      append-only de evidencia de tests (`.claude/verify-ledger.jsonl`).
- [x] Feature 6 `review_verdict_recorded` — `april review record`,
      veredicto de `reviewer_agent` registrado en el mismo ledger
      (`kind: "review"`), no narrado.
- [x] Feature 7 `review_frozen_candidate` — `april review start` +
      `review record --subject-hash`, candidato congelado vía `git
      write-tree` sobre índice temporal (opt-in, no reemplaza el
      mecanismo de la feature 6).
- [x] Configuración: `git commit` agregado a
      `.claude/hooks/block-dangerous-git.sh` (a raíz de un incidente en
      otro proyecto).
- [x] Incidente: `progress/current.md` perdió ~600 líneas de historial
      durante la feature 6 (agentes con solo `Write`), reconstruido desde
      contexto de conversación.
- [x] Consolidación de sesión: `progress/history.md` actualizado,
      `progress/current.md` reseteado para la próxima sesión.

## Verification Evidence

| Check | Command | Result | Notes |
|---|---|---|---|
| Build | `go build ./...` | verde | corrido repetidamente por cada `agent_developer`/`reviewer_agent` |
| Vet | `go vet ./...` | verde | idem |
| Tests | `go test ./... -v` | verde, 180 tests | acumulado features 2-8 y 12 (165→180) |
| Init harness | `./init.sh` | exit 0 | `blockedReasons: []` al cierre de cada feature, incluido 12 |
| Ledger | `.claude/verify-ledger.jsonl` | receipts vigentes | `kind:test` y `kind:review` para features 5/6/7/8/12, treeHash `edf8d225...` para 12 verifica `no_test_evidence` estable tras `go build` |
| Manual | `april feature set-status`, `april verify record`, `april review record`/`review start` corridos en vivo contra este mismo repo | comportamiento correcto | dogfooding real, incluido `hashTree` respeta `.gitignore` (`HarnessInit` no invalida ledger) |

## Files Changed

- `status.go`/`status_test.go` (nuevo, feature 2, extendido en 5/6) —
  árbitro de fases, lectura del ledger.
- `set_status.go`/`set_status_test.go` (nuevo, feature 4) — escritura
  autoritativa de `feature_list.json`, escritura atómica.
- `verify.go`/`verify_test.go` (nuevo, feature 5, extendido en 6/7/12) —
  `hashTree`, ledger append-only, `verify record`, parser `.gitignore` (`parseGitignore`/`gitignoreMatches`/`loadGitignorePatterns`) + `hashTree` wiring.
- `review.go`/`review_test.go` (nuevo, feature 6, extendido en 7/8/12) —
  `review record`/`review start`, candidato congelado de git, `parseSensitiveAreas`/`computeTouchedPaths`/`matchSensitiveAreas` (feature 8), `fixedTreeExclusions` compartida (feature 12).
- `main.go` — nuevos casos `status`/`feature`/`verify`/`review` en el
  switch, `printUsage()` actualizado (`review start --feature <id> [--json]`).
- `init.sh` — heredoc Python reemplazado por invocación a `april status`.
- `CLAUDE.md` — paso 4 del ciclo por sesión reescrito (feature 3).
- `.claude/hooks/block-dangerous-git.sh` — `git commit` agregado a los
  patrones bloqueados.
- `feature_list.json` — features 1-8 y 12 → `done` (4-8 y 12 escritas
  exclusivamente vía `april feature set-status`, no edición manual).
- `specs/april_status_arbiter/`, `specs/verify_record_ledger/`,
  `specs/review_verdict_recorded/`, `specs/review_frozen_candidate/`,
  `specs/review_depth_by_diff_sensitivity/`, `specs/tree_hash_respects_gitignore/` —
  specs + tickets de las features `sdd: true`.
- `progress/current.md`, `progress/history.md` — bitácora y
  consolidación de esta sesión.
- `.claude/verify-ledger.jsonl` — ledger real con receipts de tests y
  veredictos de revisión (features 5-8, 12).

## Decisions Made

- "B llegando por A" (26/08/2026, `ROADMAP.md`): `april status`/`CLAUDE.md`
  operan advisory hasta confirmación humana explícita de uso real
  confiable (dada 27/08/2026, tras el ciclo completo de la feature 2) —
  recién ahí se activó `set-status` como escritura exclusiva.
- Mecanismo interino de veredicto en `set_status.go` (feature 4, flag
  `--verdict`) **coexiste deliberadamente** con el ledger real (features
  5/6) — no se unificaron; decisión explícita, documentada en ambas specs.
- Exclusiones fijas para hash de árbol (`.git/`, el ledger, `progress/`)
  en `hashTree` (feature 5) y replicadas en `computeSubjectHash` (feature
  7, vía índice temporal de git) — no configurables.
  `subject_hash` de la feature 7 queda **opt-in**, no reemplaza a
  `treeHash`/`no_review_verdict`.
- `set_status.go` nunca consulta el ledger (features 5/6/7 lo dejan
  fuera de alcance explícitamente) — el gate de cierre sigue siendo
  verificación humana leyendo `april status --json`, no un bloqueo
  automático de `set-status done`.
- Protocolo de secuencia aprendido (feature 5 y reconfirmado en 6/7): el
  receipt final de una feature (`verify record`/`review record`) se
  graba *después* de que el orquestador termine de tocar `specs/`
  (marcar tickets `done`), nunca antes — si no, se auto-invalida.
- Features `sdd: true` permanecen `pending` durante la Fase Spec (no
  `in_progress`); pasan a `spec_ready` recién cuando el humano aprueba la
  spec, vía `april feature set-status`.
- `git commit` bloqueado por hook en este repo (además de los patrones
  destructivos ya existentes) — decisión del humano tras discutir un
  incidente en otro proyecto.

## Blockers / Risks

- Ninguna feature bloqueada. La 8 ya está cerrada; la 12 (corrección
  transversal de `hashTree`/`computeSubjectHash`) también. Frontera
  actual: 9 (`doctor_readonly_check`), 10 (`init_backup_before_apply`),
  11 (`doctor_debt_ratchet`), todas `pending` sin bloqueos.
- `hashTree` respeta `.gitignore` desde la feature 12 — el riesgo previo
  (binario `HarnessInit` invalidando receipts tras `go build ./...`)
  queda corregido y verificado (tests en `fstest.MapFS` + integración con
  `recordVerify`/`computeStatus` en ambos órdenes + dogfooding en vivo).
- `progress/current.md` se reconstruyó desde contexto de conversación
  tras perder ~600 líneas (ver `progress/history.md`, entrada de
  incidente, feature 6). El contenido es fiel en sustancia pero no
  byte-exacto al original. Vigilar que `spec_writer`/`ticket_writer`
  (solo `Write`, sin `Edit`) no vuelvan a perder contenido en tareas
  largas.
- Nada commiteado todavía — el árbol de trabajo tiene todos los cambios
  de las features 1-8 y 12 sin `git commit` (bloqueado además por el hook
  nuevo salvo que el humano lo pida explícitamente).

## Next Session Startup

1. Leer `AGENTS.md`/`CLAUDE.md`.
2. Correr `./init.sh` — debe estar en verde (`blockedReasons: []`).
3. Correr `april status --json` (o `go build -o <tmp> . && <tmp> status
   --json`) — debe recomendar implementar `feature 9` (`doctor_readonly_check`).
4. Leer este handoff y `progress/history.md` (entrada de esta sesión) si
   hace falta contexto de las decisiones de diseño ya tomadas.

## Recommended Next Step

- Continuar con la feature 9 (`doctor_readonly_check`): `sdd: false`,
  implementación directa vía `agent_developer` (read-only, sin spec).
  Features 10 y 11 quedan en `pending` y pueden paralelizarse según
  `ROADMAP.md` E6.
- Considerar si conviene commitear el estado actual (features 1-8 y 12
  completas, sin commits) antes de seguir sumando trabajo — el humano no
  lo ha pedido todavía.
