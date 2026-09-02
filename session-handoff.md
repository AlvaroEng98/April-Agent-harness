# Session Handoff

## Current Objective

- Goal (01-02/09/2026): revisar y corregir que `session-handoff.md` no
  se propague a proyectos scaffoldeados (feature 20); cerrar la última
  feature pendiente del backlog, `blocked_reasons_remedy_commands`
  (feature 15); y, al intentar commitear, se encontró y corrigió un
  cuarto hallazgo hermano de `.gitignore`: `feature_list.json` de la raíz
  nunca había estado trackeado en git (feature 21).
- Current status: **backlog completo — features 1-21 `done`**.
  `april status --json` reporta `phase: closed`, `nextRecommended: "nada
  — no hay features pendientes"`. Ver `progress/history.md` (secciones
  "2026-09-01", "2026-09-02" y "2026-09-02 (continuación)") para el
  detalle completo.
- Branch / commit: `main`. **Sin commitear todavía**, todo el trabajo de
  esta sesión (features 20, 15 y 21):
  - `scaffold.go`, `scaffold_test.go` (feature 20)
  - `status.go`, `status_test.go` (feature 15)
  - `.gitignore`, `docs/conventions.md` (feature 21 — nota: `docs/conventions.md`
    también fue tocado por la feature 20, ambos cambios conviven en el
    mismo archivo)
  - `feature_list.json` — **por primera vez puede commitearse**, la
    feature 21 sacó la línea que lo gitignoreaba
  - `progress/current.md`, `progress/history.md`, este `session-handoff.md`
  - `specs/blocked_reasons_remedy_commands/` completo (spec, 2 tickets,
    `verify-report.md`)
  - `.claude/verify-ledger.jsonl`
  - **Importante:** `templates/session-handoff.md` (feature 20) sigue
    *staged* con `git add -f` — colisiona con `templates/.gitignore:9`,
    así que un `git add templates/` plano sin `-f` no lo agrega.

  Comando sugerido (reemplaza el intento anterior que falló por el
  `.gitignore` de `feature_list.json`, ya corregido):
  ```
  git add -f templates/session-handoff.md && git add scaffold.go \
    scaffold_test.go status.go status_test.go .gitignore \
    docs/conventions.md feature_list.json \
    specs/blocked_reasons_remedy_commands/ .claude/verify-ledger.jsonl \
    progress/ session-handoff.md && git commit
  ```

## Completed This Session

- [x] **Feature 20** (`scaffold_session_handoff_placeholder`) — done.
      `session-handoff.md` ya no se propaga desde la raíz de este repo a
      proyectos scaffoldeados.
- [x] **Feature 15** (`blocked_reasons_remedy_commands`) — done. Los 10
      mensajes de `blockedReasons` ahora recetan el comando `april ...`
      exacto o la acción de archivo concreta, no solo diagnostican.
- [x] **Feature 21** (`gitignore_root_tracks_feature_list`) — done. El
      `.gitignore` de la raíz ya no ignora `feature_list.json` — el
      backlog vivo (21 features) queda versionado por primera vez desde
      el commit `8517803` (31/07/2026). Cuarto hallazgo hermano de la
      misma familia que las features 17/18. `CHANGELOG.md`/
      `sync-changelog.sh` quedaron señalados como arquitectura de
      respaldo obsoleta, pero sin tocar — fuera de alcance a propósito,
      es una decisión aparte si el humano quiere retomarlos.

Detalle completo de las tres en `progress/history.md`.

## Verification Evidence

| Check | Resultado |
|---|---|
| Feature 20/15: ver tablas de sesiones previas en `progress/history.md` | verde |
| Feature 21: `git check-ignore -v feature_list.json` (antes/después) | matcheaba `.gitignore:27` → sin match tras el fix |
| Feature 21: `git diff .gitignore` | acotado a 4 líneas (bloque de comentario + `/feature_list.json`) |
| Feature 21: `git diff templates/.gitignore` | vacío — template no tocado |
| Feature 21: 1ª revisión → `CHANGES_REQUESTED` (mecánico, faltaba receipt en ledger) → registrado → 2ª revisión → `APPROVED` | resuelto |
| `april verify record`/`april review record` (features 20, 15, 21) | registrados en `.claude/verify-ledger.jsonl` |
| `april status --json` (final) | `phase: closed`, `blockedReasons: []` |

## Decisions Made

- **`feature_list.json` de la raíz se trackea en git**, mismo criterio
  que specs/docs (features 17/18): es contenido vinculante (backlog real
  con `acceptance`/`status`), no estado descartable. `CHANGELOG.md`/
  `sync-changelog.sh` quedan como resumen curado aparte, no como único
  respaldo — su staleness (describen una arquitectura de flujos F1/F2/F3
  y un `orquestador.md` que ya no existen) queda señalada pero sin
  arreglar, decisión explícita de alcance.
- **`templates/session-handoff.md` necesita `git add -f`** (colisión con
  `templates/.gitignore:9`) — feature 20, sigue vigente.
- La lección de sesiones previas sobre registrar en el ledger real antes
  de `done` se reafirmó con la feature 21: un `CHANGES_REQUESTED`
  formal, no solo una nota — el hábito de registrar el ledger en cada
  cierre evita este tipo de rechazo mecánico.

## Blockers / Risks

- Ninguno técnico — backlog completo (1-21) `done`, `phase: closed`.
- **Acción pendiente del humano:** commitear lo de esta sesión (ver
  "Current Objective" arriba — el intento anterior falló por
  `feature_list.json` gitignoreado, ya corregido por la feature 21;
  sigue pendiente el `git add -f` de `templates/session-handoff.md`).

## Next Session Startup

1. Verificar si el humano ya corrió el commit pendiente.
2. Correr `./init.sh` — debe estar en verde.
3. Correr `april status --json` — debe reportar `phase: closed`.
4. Preguntar al humano: ¿hay features nuevas que agregar al backlog (vía
   `planner_agent`), o algún otro objetivo? (Posible candidato pendiente
   de decisión, no abierto como feature: revisar si vale la pena
   modernizar o retirar `CHANGELOG.md`/`sync-changelog.sh`, ahora que su
   premisa de diseño quedó obsoleta.)

## Recommended Next Step

- No hay trabajo de backlog pendiente. Si el humano no tiene un objetivo
  nuevo, la sesión puede cerrarse aquí.
