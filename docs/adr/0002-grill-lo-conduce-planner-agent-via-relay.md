# El Grill lo conduce `planner_agent`, no el orquestador — vía relay

Hasta ahora el orquestador conducía el Grill (preguntar, investigar,
proponer) directamente en el hilo principal, porque los subagentes no tienen
canal con el usuario. El 19/08/2026 se decidió mover esa fase completa a
`planner_agent`: es quien investiga y decide qué preguntar, dejando al
orquestador solo con el rol de clasificar el flujo (vía una skill aún por
construir) y hacer de relay de mensajes cuando `planner_agent` necesita una
respuesta del humano — trasladando la pregunta tal cual y reanudando al
subagente con `SendMessage` cuando llega la respuesta.

Se planteó como alternativa mantener el Grill en el orquestador y limitar
`planner_agent` a investigación/propuesta sin interacción directa, por ser
la arquitectura de subagentes la que originalmente motivó ese diseño (un
subagente lanzado vía `Agent` no tiene diálogo en vivo con el usuario). Se
rechazó explícitamente: el objetivo es que sea `planner_agent` quien conduce
el interrogatorio, apoyándose en el mecanismo de relay/`SendMessage` para
resolver la limitación de canal en lugar de evitarla.

## Status

accepted

## Consequences

- `orquestador.md` y `planner_agent.md` documentan el patrón de relay:
  `planner_agent` termina su turno con una pregunta como salida de texto, el
  orquestador la traslada al humano y responde con
  `SendMessage(to: "<agente>", message: "<respuesta>")` para reanudarlo.
- `planner_agent` ya no está limitado a `feature_list.json`: ahora también
  escribe `progress/project-definition.md`.
- Si el mecanismo de relay resulta poco fiable en la práctica (turnos
  perdidos, contexto no preservado entre `SendMessage`), revertir a que el
  orquestador conduzca el Grill directamente es la alternativa ya evaluada
  arriba — no haría falta reabrir el diseño desde cero.
