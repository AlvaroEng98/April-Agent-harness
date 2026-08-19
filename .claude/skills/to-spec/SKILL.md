---
name: to-spec
description: Sintetiza el contexto ya disponible (tarea cruda, project-definition.md, docs/) en un único spec.md — sin entrevista. La invoca sdd_agent_author al escribir specs/<name>/spec.md.
disable-model-invocation: true
---

# to-spec (adaptado a April)

Adaptación del skill `to-spec` (Matt Pocock) al flujo F3 de este harness. La
**única** diferencia con el original: en vez de publicar el spec a un issue
tracker, se escribe a disco en `specs/<name>/spec.md`. Todo lo demás —
proceso y plantilla— se conserva tal cual.

Solo la invoca `sdd_agent_author`, después de leer `docs/architecture.md`,
`docs/conventions.md` y la tarea o bullet de origen (nunca una fila de
`feature_list.json`: en F3 esa fila no existe hasta que el spec está
terminado y `planner_agent` la crea a partir de él).

## Proceso

1. **No entrevistes, sintetiza.** El contexto ya está en la tarea/bullet de
   origen, `progress/project-definition.md` y `docs/`. Explora el repo para
   entender el estado actual si no lo has hecho ya, usa el glosario de
   dominio del proyecto y respeta cualquier ADR en la zona que toques.
2. **Sketch de seams de test**, antes de escribir el spec: por cada pieza de
   comportamiento, ubica el punto más alto del código donde se puede
   verificar. Prefiere un seam existente sobre uno nuevo; si hace falta uno
   nuevo, el número ideal de seams nuevos es el mínimo posible. El original
   pide confirmar los seams con el usuario en ese momento — aquí no se abre
   una pausa nueva: el seam queda documentado en `## Decisiones de testing`
   y el humano lo revisa en la puerta de aprobación del spec que ya existe.
3. Escribe `specs/<name>/spec.md` con esta plantilla, en este orden:

<spec-template>

## Enunciado del problema

El problema que enfrenta el usuario, desde su perspectiva.

## Solución

La solución al problema, desde la perspectiva del usuario.

## Historias de usuario

Una lista LARGA y numerada de historias de usuario (`US1`, `US2`, ...). Cada
una en formato:

`US<n>. Como <actor>, quiero <feature>, para <beneficio>`

Ejemplo: `US1. Como cliente de banca móvil, quiero ver el saldo de mis
cuentas, para poder decidir mejor sobre mis gastos.`

Esta lista debe ser extremadamente exhaustiva y cubrir todos los aspectos de
la feature. **Sustituye a los `R<n>` de EARS como unidad de trazabilidad**:
cada `US<n>` DEBE ser verificable por un test concreto — si no lo es, parte
la historia o márcala como blocker. No mezcles varias historias en una.

## Decisiones de implementación

Una lista de las decisiones de implementación tomadas. Puede incluir:

- Los módulos que se van a construir/modificar.
- Las interfaces de esos módulos que van a cambiar.
- Clarificaciones técnicas.
- Decisiones de arquitectura.
- Cambios de schema.
- Contratos de API.
- Interacciones específicas.

NO incluyas rutas de archivo concretas ni snippets de código — quedan
desactualizados rápido. Excepción: si un prototipo ya produjo un snippet que
fija una decisión con más precisión que la prosa (máquina de estados,
reducer, schema, forma de un tipo), inclúyelo dentro de la decisión
correspondiente, señalando brevemente que viene de un prototipo. Recorta a
la parte que fija la decisión — no una demo funcionando, solo lo importante.

## Decisiones de testing

Una lista de las decisiones de testing tomadas. Incluye:

- Qué hace un buen test (solo comportamiento externo, no detalles de
  implementación).
- Qué módulos se van a testear.
- Prior art para los tests (tests similares ya existentes en el repo).
- El sketch de seams del paso 2: por cada `US<n>`, el seam elegido.

## Fuera de alcance

Qué NO cubre este spec.

## Notas adicionales

Cualquier nota adicional sobre la feature.

</spec-template>

4. Con el spec ya escrito, `sdd_agent_author` añade su propia sección
   `## Desafío` (no forma parte de la plantilla original — es invariante de
   April, ver su protocolo). El archivo completo queda en
   `specs/<name>/spec.md`, un único archivo.

## Lo que NO se importa de la versión original

- Nada de issue tracker ni triage label (`ready-for-agent`) — el spec vive en
  disco, el estado en `feature_list.json` lo pone `planner_agent` después.
- Nada de entrevista al usuario — si la tarea viene del backlog, la
  entrevista ya la hizo `planner_agent` en el Grill; si es una tarea nueva
  clasificada `SDD` directo, tampoco se entrevista: se sintetiza lo que el
  humano ya escribió.
