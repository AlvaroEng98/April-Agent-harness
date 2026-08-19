# Contratos de implementación (F2 plan · F3 SDD)

> Ningún agente escribe código sin un **contrato**: el artefacto que dice qué
> debe quedar verdadero al final y contra el que revisa el `reviewer_agent`.
> Qué contrato aplica depende del flujo.

| Flujo | Contrato | Quién lo escribe | Puertas humanas |
|-------|----------|------------------|-----------------|
| **F2 Delegado** | `progress/plan_<feature>.md` | `agent_developer`, antes de codear | 1 (cierre) |
| **F3 SDD** | `specs/<feature>/spec.md` | `sdd_agent_author`, antes de codear | 2 (spec + cierre) |

Los dos flujos y la Puerta de Desafío están definidos en
`.claude/agents/orquestador.md` — es la fuente única, no se resumen en ningún
otro archivo.

---

## F2 — el plan ligero

Para features **claras pero no triviales** (2-3 archivos). No hay spec. El
`agent_developer` escribe **un solo archivo antes de tocar código**:

```markdown
# Plan — <feature>

## Archivos
- src/x.go (modificar) — qué cambia
- tests/x_test.go (nuevo) — qué cubre

## Acceptance → test
- A1 → `test_foo_ok`
- A2 → `test_foo_invalid`

## Riesgo asumido
- <objeción que el humano rechazó, o "ninguno">
```

`A<n>` es el criterio en la posición `n` del array `acceptance` de la feature.

**Para qué sirve el plan**: obliga a mapear cada criterio a un test *antes* de
escribir 200 líneas. Si algún `A<n>` no se puede mapear a un test concreto, la
feature no era F2 — era F3, y el implementador para con `blocked`. Ese fallo
temprano es el valor del archivo, no la documentación.

**Lo que el plan NO es**: no es una puerta humana. No hay ronda de aprobación
antes de codear en F2 — para eso existe F3. El plan es la traza que hace
auditable un flujo sin spec.

---

## F3 — Spec Driven Development

> Metodología del skill `to-spec` (adaptada a April, ver
> `.claude/skills/to-spec/SKILL.md`): sintetiza el contexto ya disponible en
> un único `spec.md` — sin entrevista, sin issue tracker. El código no se
> escribe hasta que el spec está aprobado por un humano.

Solo para features **ambiguas** (`"ambiguity": "vague"` o `acceptance` no
verificables). Si la feature es clara, F3 es sobrecoste puro.

### Estructura

Cada feature F3 (`"sdd": true` en `feature_list.json`) tiene una carpeta
dedicada desde que nace — no hay estado previo sin carpeta:

```
specs/<feature-name>/
└── spec.md   # Enunciado del problema → Solución → Historias de usuario →
              # Decisiones de implementación → Decisiones de testing →
              # Fuera de alcance → Notas adicionales → ## Desafío
```

El `feature-name` coincide con el campo `name` de `feature_list.json`. No hay
`requirements.md`, `design.md` ni `tasks.md` — un único archivo.

### Estados de una feature

| Estado         | Significado                                                    |
|----------------|------------------------------------------------------------------|
| `pending`      | Sin contrato. Solo F2 — en F3 la feature no tiene fila hasta que el spec existe (ver nota abajo). |
| `spec_ready`   | Spec drafted. Esperando aprobación humana. NO se toca código. Solo F3. |
| `in_progress`  | Contrato listo. `agent_developer` trabajando.                  |
| `done`         | Verde, revisado y **aprobado por el humano**. Lo escribe el orquestador. |
| `blocked`      | Atascado. Razón en `progress/current.md`.                      |

**F3 no pasa por `pending`.** La fila de `feature_list.json` no existe hasta
que `sdd_agent_author` termina de escribir `spec.md`. `sdd_agent_author` solo
escribe el spec — no toca `feature_list.json`. Es `planner_agent` (modo
Spec-to-Feature) quien lee ese `spec.md` y crea la fila, directo en
`spec_ready`, sintetizando `title`/`description` del `Enunciado del
problema`/`Solución` y `acceptance` 1:1 de las `Historias de usuario`. No hay
ningún punto del flujo F3 donde exista una fila `pending` con `"sdd": true`.

### Las dos puertas de aprobación

F3 se detiene **dos veces**:

1. **Puerta del spec.** El `sdd_agent_author` termina `spec.md`, `planner_agent`
   crea la fila en `spec_ready`, y el orquestador para. El humano lee
   `specs/<feature>/spec.md` —incluida la sección `## Desafío`— y dice
   "aprobado" o pide cambios. Solo entonces el orquestador transiciona
   `spec_ready → in_progress` y lanza el `agent_developer`.
2. **Puerta de cierre.** Tras el `reviewer_agent`, el orquestador para otra vez.
   El humano aprueba y **el orquestador** escribe `done`. Ningún subagente
   escribe `done`, nunca, ni aunque el reviewer haya aprobado.

```
(sin fila) → [sdd_agent_author escribe spec.md] → [planner_agent crea la fila]
           → spec_ready → ⏸ HUMANO → in_progress
           → [agent_developer → reviewer_agent] → ⏸ HUMANO → done
```

F2 tiene solo la puerta de cierre.

### spec.md — plantilla `to-spec`

Secciones, en este orden. Las primeras siete son la plantilla original del
skill `to-spec`; la última es propia de April.

1. **`## Enunciado del problema`** — el problema desde la perspectiva del
   usuario.
2. **`## Solución`** — la solución, desde la perspectiva del usuario.
3. **`## Historias de usuario`** — lista larga y numerada (`US1`, `US2`, ...),
   formato `Como <actor>, quiero <feature>, para <beneficio>`. Exhaustiva.
   **Es la unidad de trazabilidad de F3**: cada `US<n>` DEBE ser verificable
   por al menos un test concreto. No mezcles varias historias en una — si
   hace falta, parte.
4. **`## Decisiones de implementación`** — módulos a construir/modificar,
   interfaces, decisiones de arquitectura, cambios de schema, contratos de
   API. Sin rutas de archivo ni snippets de código (quedan desactualizados
   rápido), salvo que un prototipo ya haya fijado una decisión con más
   precisión que la prosa (máquina de estados, schema) — ahí se cita, breve,
   marcado como proveniente de un prototipo.
5. **`## Decisiones de testing`** — qué hace un buen test (solo comportamiento
   externo), qué módulos se testean, prior art en el repo, y el sketch de
   seams de test (el punto más alto donde se puede verificar cada `US<n>`,
   prefiriendo un seam existente).
6. **`## Fuera de alcance`** — qué NO cubre esta feature.
7. **`## Notas adicionales`** — cualquier nota más.
8. **`## Desafío`** (obligatoria, no es parte del `to-spec` original) — tres
   bloques:
   - `### Alternativa descartada` — mínimo una, siempre, con su porqué.
   - `### Objeciones al planteamiento` — las que disparen los gatillos G1-G4, en
     formato `⚠️ OBJECIÓN [G<n>]` + `Evidencia:` + `Alternativa:`. Si no hay
     ninguna, se escribe explícitamente que los cuatro se revisaron sin disparo.
   - `### Riesgo asumido` — objeciones que el humano rechazó y se ejecutan igual.

NO es ingeniería desde primeros principios — apóyate en
`docs/architecture.md` y `docs/conventions.md`. `## Decisiones de
implementación` documenta los puntos donde tu feature roza la frontera de
esas reglas.

---

## Trazabilidad (regla dura en los dos flujos)

Cambia el identificador, no el principio:

| Flujo | Regla |
|-------|-------|
| F2 | Cada `A<n>` tiene al menos un test concreto, y el mapa `A<n> → test` de `progress/plan_<name>.md` coincide con el código real. |
| F3 | Cada `US<n>` (Historia de usuario de `spec.md`) tiene al menos un test concreto, y cada test se mapea a un `US<n>`. |

El `reviewer_agent` comprueba esta correspondencia explícitamente y rechaza si
falta. El `agent_developer` documenta el mapa en `progress/impl_<name>.md`:

```markdown
## Trazabilidad
- US1 → `test_recent_default_limit`    ← F3
- A1 → `test_recent_default_limit`     ← F2
```

Un plan o spec que promete tests que no existen es rechazo, no advertencia.

## Cuándo NO aplica SDD

- Features con `"sdd": false`, o sin el campo `sdd` → van por **F2**.
- `bootstrap_project` tiene su propio protocolo (Caso F del orquestador) y no
  pasa ni por la matriz de complejidad ni por SDD.

**`"sdd": true` fuerza F3, sin excepción.** No es una preferencia: `init.sh`
valida que toda feature `sdd:true` en estado `spec_ready`/`in_progress`/`done`
tenga su `spec.md` en `specs/<name>/`. Clasificarla F2 deja el build en
rojo. Por eso `planner_agent` mantiene la invariante
`sdd == (ambiguity == "vague")` en las features nuevas.

SDD solo se aplica hacia adelante: features previas al proceso no se
retro-especifican.
