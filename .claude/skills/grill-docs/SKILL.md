---
name: grill-docs
description: Rellena docs/architecture.md, docs/conventions.md, docs/verification.md y docs/specs.md durante la Fase Grill de bootstrap_project, entrevistando al humano por lo que falte. Usar cuando alguno de esos 4 archivos no existe, o tiene una sección que el humano ya respondió pero sigue en `_pendiente_`.
---

# grill-docs

El cierre de `bootstrap_project` depende de esto: su `acceptance` en
`feature_list.json` exige `docs/architecture.md` y `docs/conventions.md`
con secciones rellenas. Los 4 archivos comparten una regla: sección sin
respuesta del humano queda literalmente `_pendiente_` — nunca se omite, y
nunca se inventa una respuesta plausible en su lugar.

## Antes de preguntar

Lee lo que ya existe — no reabras entrevista sobre algo ya respondido:

- `progress/project-definition.md` — Objetivo y Tech stack ya están ahí.
- `.claude/agents/orquestador.md`, `.claude/agents/spec_writer.md` y la
  skill `needs-sdd` — la Fase Spec y su plantilla ya están definidas ahí;
  `docs/specs.md` las cita, no las repite.
- Cualquier sección ya rellena en un `docs/*.md` parcial — completa los
  huecos, no reescribas lo que ya está.

## Los 4 archivos

### docs/architecture.md

Pregunta: capas/módulos del proyecto y qué depende de qué, dependencias
externas permitidas, manejo de errores (excepciones nombradas vs. valores
de retorno), invariantes de persistencia si aplica (atomicidad).

Secciones: `## Principios` (lista numerada), `## Capas`/`## Módulos`
(tabla módulo → responsabilidad), `## Flujo de datos` (diagrama de texto),
`## Qué NO hacer` (anti-patrones concretos de este proyecto, no genéricos).

### docs/conventions.md

Pregunta: lenguaje/versión, formato/linter, convención de nombres por tipo
(módulo, tipo, función, constante), estructura estándar de archivo, dónde
viven los errores de dominio, convención de tests (ubicación, nombres).

Secciones: `## Estilo`, `## Nombres` (tabla tipo → convención → ejemplo),
`## Estructura de archivo`, `## Tests`, `## Manejo de errores`,
`## Comentarios` — por defecto sin comentarios salvo *por qué* no obvio
(mismo criterio que las instrucciones globales de la sesión; no lo
contradigas).

### docs/verification.md

Pregunta: comando real para correr los tests (`go test ./...`, `pytest`,
etc.), umbral de cobertura solo si el humano quiere uno explícito — no
asumas un porcentaje que `CHECKPOINTS.md` no pide, qué cuenta como test de
integración en este proyecto.

Secciones: `## Niveles de verificación` (unitario obligatorio, integración
si aplica, smoke test manual opcional), `## Comando` (bloque ejecutable
real, no pseudo-código), `## Anti-patrones` (afirmar sin ejecutar, mockear
lo que debería ser real), `## Verificación final antes de cerrar` → cita
`./init.sh`.

### docs/specs.md

No es una plantilla nueva — es el mapa de la Fase Spec que ya vive en
`.claude/agents/orquestador.md` y `.claude/agents/spec_writer.md`. Escribe:

- Cuándo aplica (`sdd: true`) y quién decide (skill `needs-sdd`).
- Quién escribe el spec (`spec_writer`) y dónde
  (`specs/<name>/spec.md`) — cita la plantilla de `spec_writer.md`, no la
  copies aquí: si cambia allá, esta copia quedaría desactualizada y
  habría dos fuentes de verdad para el mismo contrato.
- Las dos puertas humanas: aprobar el spec y aprobar el cierre
  (`require_approved_spec_to_implement`,
  `human_approval_required_to_close` en `feature_list.json`).

No documentes flujos que ya no existen (p. ej. F2/F3, `sdd_agent_author`,
`reviewer_agent`) aunque aparezcan en commits antiguos — es protocolo
retirado, no el actual.

## Cierre

Los 4 archivos existen y ninguna sección que el humano ya respondió sigue
en `_pendiente_`. Si preguntaste algo y el humano no supo o no quiso
responder todavía, el `_pendiente_` se queda explícito ahí — nunca se
borra la sección ni se rellena con una suposición.
