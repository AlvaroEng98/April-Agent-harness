# CHECKPOINTS

Criterios objetivos usados por el `reviewer_agent` para evaluar una feature.

La columna **Flujo** dice en qué flujos aplica el checkpoint. Un checkpoint que
no aplica se marca `n/a`, no `[ ]`: en F2 no hay spec, así que exigir
trazabilidad `US<n>` o cobertura de `spec.md` sería rechazar por algo que
nunca debió existir.

| ID | Criterio | Flujo |
|----|----------|-------|
| C1 | Código compila/interpreta sin errores | F2 · F3 |
| C2 | `init.sh` pasa verde | F2 · F3 |
| C3 | Todos los tests pasan | F2 · F3 |
| C4 | Trazabilidad `US<n>` (Historias de usuario de `spec.md`) ↔ tests completa | **solo F3** |
| C5 | Todos los módulos de `## Decisiones de implementación` de `spec.md` fueron tocados | **solo F3** |
| C6 | Sin TODOs, debug prints, ni código comentado | F2 · F3 |
| C7 | La feature respeta `docs/architecture.md` y `docs/conventions.md` | F2 · F3 |
| C8 | Trazabilidad `A<n>` ↔ tests completa, y el mapa de `progress/plan_<name>.md` coincide con el código real | **solo F2** |
| C9 | Cobertura cumple el mínimo de `docs/verification.md` (60% código nuevo, 80%+ funciones críticas) | F2 · F3 |

## Eje sustancia (no es un checkpoint, es un juicio)

Los C1-C8 son mecánica: verificables sin criterio. Además de recorrerlos, el
`reviewer_agent` responde por escrito tres preguntas de sustancia y puede emitir
`APPROVED_WITH_OBJECTION` — mecánica verde, sustancia dudosa:

1. ¿La implementación resuelve el problema real, o solo satisface el contrato al
   pie de la letra dejando el problema en pie?
2. ¿Hay complejidad que ningún requirement ni criterio pide?
3. ¿Algún test verifica el mock o la propia implementación en vez del
   comportamiento observable?

Una objeción de sustancia **informa al humano, no bloquea**. La mecánica es lo
único que produce `CHANGES_REQUESTED`.
