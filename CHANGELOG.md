# Changelog

Registro de lo entregado. Este archivo es **producto** y va versionado.

El backlog vivo está en `feature_list.json`, que **no** se versiona (es estado
de desarrollo). El puente entre ambos es `./sync-changelog.sh`: cuando una
feature se aprueba como `done`, su entrada se vuelca aquí. Los releases leen
este archivo, nunca `feature_list.json`.

Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/).

## [Unreleased]

Rediseño de los flujos de trabajo del harness. No es una feature del backlog:
es el proceso que gobierna cómo se construyen las features.

### Añadido
- **Tres flujos de construcción explícitos** (F1 Directo · F2 Delegado · F3 SDD)
  con contrato, número de subagentes, puertas humanas y checkpoints propios.
  Documentados en `.claude/agents/orquestador.md`, `AGENT.md` §4 y
  `docs/specs.md`.
- **Puerta de Desafío** en los cinco agentes: cuatro gatillos (G1 contradicción ·
  G2 camino más simple · G3 no verificable · G4 coste >> valor), formato de
  objeción con `Evidencia` + `Alternativa` obligatorias, y reglas anti-teatro
  (sin gatillo → silencio; máximo 3; objeción rechazada = cerrada y anotada como
  riesgo asumido).
- **`progress/plan_<feature>.md`**: contrato ligero del flujo F2 (archivos +
  mapa `acceptance → test` + riesgo asumido). Lo escribe el `agent_developer`
  antes de codear; es lo que verifica el `reviewer_agent` cuando no hay spec.
- **Veredicto `APPROVED_WITH_OBJECTION`** en `reviewer_agent`, con eje de
  sustancia (¿resuelve el problema real? ¿complejidad no pedida? ¿tests que
  verifican mocks?) además del eje mecánico.
- `CHECKPOINTS.md` C8 para trazabilidad `A<n>` ↔ tests en F2.
- `templates/docs/specs.md`: el doc de contratos ahora se propaga a los
  proyectos scaffoldeados, y `docs/specs.md` pasa a ser archivo base requerido
  por `init.sh`.
- `rules.flows` y `rules.challenge_gates` en `feature_list.json`.

### Corregido
- El flujo F2 (MEDIO, sin SDD) era inejecutable: `agent_developer` y
  `reviewer_agent` exigían `specs/<name>/` de forma incondicional, así que toda
  feature delegada sin spec se autobloqueaba o se rechazaba. Ambos agentes tienen
  ahora modos F2/F3 explícitos y C4/C5 no aplican en F2.
- `orquestador.md` mandaba lanzar `implementer` / `spec_author` / `reviewer`:
  `subagent_type` inexistentes. Los reales son `agent_developer` /
  `sdd_agent_author` / `reviewer_agent`.
- Tres agentes intentaban leer `AGENTS.md`, que no existe. El archivo es
  `AGENT.md`.
- Faltaba la puerta de aprobación humana antes de `done` en el flujo F2.
- `agent_developer` se marcaba `done` a sí mismo tras la aprobación del reviewer,
  saltándose al humano. Ahora solo el orquestador escribe `done`.
- `docs/specs.md` estaba referenciado por cinco archivos embebidos pero nunca se
  copiaba a los proyectos generados.
- `sdd: true` + `ambiguity: "clear"` era un estado contradictorio: el
  orquestador podía clasificarlo F1/F2 mientras `init.sh` exigía los 3 specs,
  dejando el build en rojo. `sdd: true` fuerza F3 e `init.sh` documenta la
  invariante `sdd == (ambiguity == "vague")` para features nuevas.

## [0.3.1] - 2026-07-31
Release de infraestructura y documentación: no cambia el comportamiento del
scaffolding. Estas entradas no vienen de `feature_list.json` porque no son
features del backlog.

### Añadido
- `install.sh`: instalador vía `curl … | sh`. Detecta OS/arquitectura, resuelve
  la última release, verifica el checksum SHA-256 e instala **a nivel de
  usuario** en `~/.local/bin` (nunca usa `sudo`). Configurable con `VERSION` y
  `BIN_DIR`.
- `templates/README.md`: README del proyecto scaffoldeado, embebido vía
  `go:embed`.

### Cambiado
- README: la instalación recomendada pasa a ser el instalador; la descarga
  manual y la compilación desde fuente quedan como opciones 2 y 3, ya sin
  `sudo`.

### Corregido
- Workflow de release: las notas se capturan en una variable antes de abrir el
  here-doc de `$GITHUB_ENV`. Antes, si `release-notes.sh` fallaba, el
  delimitador de cierre no se escribía y el error real quedaba tapado por
  `Invalid value. Matching delimiter not found`.
- `release-notes.sh`: el mensaje de error distingue una sección ausente de una
  presente pero vacía, y apunta a `./sync-changelog.sh`.

## [0.3.0] - 2026-07-31
Primera release publicada desde este repositorio. Continúa la numeración de
la lineage anterior (`v0.2.10`), cuyo historial no está en este `main`.

- Centralize Configuration (`centralize_config`)
- Recap Automático al Iniciar Sesión (Claude + OpenCode) (`auto_recap_hook`)
