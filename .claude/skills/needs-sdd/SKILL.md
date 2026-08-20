---
name: needs-sdd
description: Decide si una feature necesita sdd:true (spec_writer redacta specs/<name>/spec.md antes de implementar) o sdd:false (agent_developer va directo). Usar al clasificar el sdd de una feature nueva o de una ya en feature_list.json.
---

# needs-sdd

La pregunta real no es "¿es grande?" — es si el `cómo` de la feature es
una **puerta de una vía** (cara de deshacer si se elige mal) o una
**puerta de dos vías** (barata de deshacer). `sdd: true` compra tiempo
para decidir bien antes de cruzarla; `sdd: false` no lo necesita.

## Marca `sdd: true` si al menos una es cierta

- Crea o cambia un contrato entre módulos/servicios (API, esquema,
  formato de mensaje) que otro código va a depender — puerta de una vía.
- Hay más de un enfoque razonable para lograrla y la elección no es obvia
  sin decidirla explícitamente (arquitectura, algoritmo, modelo de datos).
- El `acceptance` que le darías a `agent_developer` alcanza para saber el
  **qué**, pero no el **cómo**.

## Marca `sdd: false` si TODAS son ciertas

- El `acceptance` ya deja claro el cómo (un archivo con tal contenido, un
  comando que sale en verde, un endpoint que responde tal cosa).
- No crea ni cambia contratos que otro módulo consuma.
- Si el detalle sale mal, deshacerlo es barato — puerta de dos vías.

## Empate

Si dudas entre `true` y `false`, marca `true`. El costo de una spec de
más es menor que el de cruzar una puerta de una vía sin plan.

## Salida

`sdd: true` o `sdd: false` + una razón de una línea citando qué señal de
arriba disparó la decisión — nunca un default sin justificar.
