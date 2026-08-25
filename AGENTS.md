# AGENTS.md — Mapa de navegación para agentes de IA

Este archivo es el punto de entrada para cualquier agente que trabaje en
este repositorio. NO es una biblia de reglas: es un mapa. Lee solo lo que
necesites cuando lo necesites (divulgación progresiva).

## El orquestador y los 5 subagentes

El **orquestador** no es un subagente aparte: es el hilo principal, con el
rol definido en `CLAUDE.md` (fuente única del protocolo completo — si
buscas "qué hago en tal fase", está ahí, no aquí). Coordina el ciclo
completo y es el único que escribe `feature_list.json`, `progress/` y
`session-handoff.md`. Nunca toca código — para eso lanza, en orden según la
fase, a los 5 subagentes:

1. **planner_agent** (`.claude/agents/planner_agent.md`) — descompone un
   objetivo en features atómicas y decide `acceptance` y `sdd` por
   feature.
2. **spec_writer** (`.claude/agents/spec_writer.md`) — redacta
   `specs/<name>/spec.md` para las features con `sdd: true`.
3. **ticket_writer** (`.claude/agents/ticket_writer.md`) — rompe una spec
   `sdd: true` ya aprobada en tickets tracer-bullet con `Blocked by`,
   guardados en `specs/<name>/tickets/`.
4. **agent_developer** (`.claude/agents/agent_developer.md`) — el único
   que toca `src/` y tests; implementa un ticket, subtarea o feature ya
   spec-eada o con `acceptance` claro.
5. **reviewer_agent** (`.claude/agents/reviewer_agent.md`) — tras la
   Implementación, demuestra que los tests cubren camino feliz y camino
   de error de cada criterio/historia, y emite el veredicto que condiciona
   el gate de cierre.

Ninguno de los cinco se lanza a sí mismo ni a otro.

## Invariante entre los cinco

Estado del proyecto (`feature_list.json`, `progress/*`,
`session-handoff.md`) lo escribe **solo** el orquestador. `planner_agent`,
`spec_writer`, `ticket_writer`, `agent_developer` y `reviewer_agent`
siempre devuelven su resultado — lista propuesta, spec, tickets, reporte,
veredicto —, nunca tocan ese estado ni marcan status (`ticket_writer` sí
escribe sus propios archivos en `specs/<name>/tickets/`, pero tampoco eso
cuenta como estado del proyecto). Si alguno de los cinco toca esos
archivos, es un bug del protocolo, no un atajo válido.

## Dónde está cada regla

- Ciclo completo, fases, gates de cierre → `CLAUDE.md`
- Reglas de estado (`valid_status`, `sdd_required_when`, gates) →
  `feature_list.json` → `rules`
- Convenciones de código y arquitectura → `docs/conventions.md`,
  `docs/architecture.md`
- Cómo verificar que el entorno está listo → `./init.sh`
