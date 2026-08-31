# Specs

> Mapa de la Fase Spec. No es una plantilla nueva — cita
> `CLAUDE.md` y `.claude/agents/spec_writer.md` como
> fuente única; si el protocolo cambia allá, actualiza la cita aquí, no
> dupliques el contenido.

## Cuándo aplica

Una feature necesita spec (`sdd: true`) cuando su implementación no es
obvia a partir del `acceptance` — hay más de un enfoque razonable, o toca
un contrato entre módulos que otro código va a depender. `sdd: false`
cuando el `acceptance` ya deja claro el cómo y deshacerlo, si sale mal, es
barato.

## Quién decide

El humano, directamente, feature por feature, junto al orquestador —
`planner_agent` propone la lista de features pero nunca resuelve `sdd`
por su cuenta. No hay heurística automática de por medio.

## Quién escribe la spec

`spec_writer` redacta `specs/<name>/spec.md` — un único archivo por
feature, con la plantilla definida en `.claude/agents/spec_writer.md`
(Problem Statement, Solution, User Stories, Implementation Decisions,
Testing Decisions, Out of Scope, Further Notes). No se generan
`requirements.md`/`design.md`/`tasks.md` por separado.

## De spec a tickets

Con la spec aprobada, `ticket_writer` la rompe en tickets tracer-bullet
(vertical slices, con `Blocked by`), uno por archivo en
`specs/<name>/tickets/<NN>-<slug>.md` — plantilla en
`.claude/agents/ticket_writer.md`. Nunca se publican en un tracker externo,
este proyecto no usa uno. `agent_developer` implementa la frontera —
tickets cuyos bloqueadores ya están en `done` — en vez de la feature
entera de una sola vez.

## Las tres puertas humanas

- **Aprobar el spec** antes de que exista una línea de código —
  `require_approved_spec_to_implement` en `feature_list.json`.
- **Aprobar el desglose de tickets** (granularidad, `Blocked by`) antes de
  que `ticket_writer` escriba los archivos — ver `CLAUDE.md`, Fase
  Tickets.
- **Aprobar el cierre** de la feature, tras el veredicto de
  `reviewer_agent` — `human_approval_required_to_close` en
  `feature_list.json`.

Ninguna de las tres la salta el agente, nunca.
