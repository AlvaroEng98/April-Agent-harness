# AGENTS.md — Mapa de navegación para agentes de IA

Este archivo es el punto de entrada para cualquier agente que trabaje en
este repositorio. NO es una biblia de reglas: es un mapa. Lee solo lo que
necesites cuando lo necesites (divulgación progresiva).

## Los 4 agentes y su orden

1. **orquestador** (`.claude/agents/orquestador.md`) — coordina el ciclo
   completo y es el único que escribe `feature_list.json`, `progress/` y
   `session-handoff.md`. Nunca toca código. Fuente única del protocolo
   completo: si buscas "qué hago en tal fase", está ahí, no aquí.
2. **planner_agent** (`.claude/agents/planner_agent.md`) — descompone un
   objetivo en features atómicas y decide `acceptance` y `sdd` por
   feature.
3. **spec_writer** (`.claude/agents/spec_writer.md`) — redacta
   `specs/<name>/spec.md` para las features con `sdd: true`.
4. **agent_developer** (`.claude/agents/agent_developer.md`) — el único
   que toca `src/` y tests; implementa una feature (o subtarea) ya
   spec-eada o con `acceptance` claro.

El orquestador lanza a los otros tres vía la herramienta `Agent`, en ese
orden según la fase. Ninguno de los tres se lanza a sí mismo ni a otro.

## Invariante entre los cuatro

Estado del proyecto (`feature_list.json`, `progress/*`,
`session-handoff.md`) lo escribe **solo** el orquestador. `planner_agent`,
`spec_writer` y `agent_developer` siempre devuelven su resultado — lista
propuesta, spec, reporte —, nunca lo escriben ellos ni marcan status. Si
alguno de los tres toca esos archivos, es un bug del protocolo, no un
atajo válido.

## Dónde está cada regla

- Ciclo completo, fases, gates de cierre → `.claude/agents/orquestador.md`
- Reglas de estado (`valid_status`, `sdd_required_when`, gates) →
  `feature_list.json` → `rules`
- Convenciones de código y arquitectura → `docs/conventions.md`,
  `docs/architecture.md`
- Cómo verificar que el entorno está listo → `./init.sh`
