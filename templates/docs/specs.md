# Contratos de implementación (F2 plan · F3 SDD)

> Ningún agente escribe código sin un **contrato**: el artefacto que dice qué
> debe quedar verdadero al final y contra el que revisa el `reviewer_agent`.
> Qué contrato aplica depende del flujo.

| Flujo | Contrato | Quién lo escribe | Puertas humanas |
|-------|----------|------------------|-----------------|
| **F1 Directo** | el `acceptance` de `feature_list.json` | nadie, se usa tal cual | 1 (cierre) |
| **F2 Delegado** | `progress/plan_<feature>.md` | `agent_developer`, antes de codear | 1 (cierre) |
| **F3 SDD** | `specs/<feature>/{requirements,design,tasks}.md` | `sdd_agent_author`, antes de codear | 2 (spec + cierre) |

Los tres flujos y la Puerta de Desafío están definidos en
`.claude/agents/orquestador.md` y resumidos en `AGENT.md` §4.

---

## F2 — el plan ligero

Para features **claras pero no triviales** (2-3 archivos). No hay spec, no hay
EARS, no hay `tasks.md`. El `agent_developer` escribe **un solo archivo antes de
tocar código**:

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

> Flujo al estilo Kiro: requirements → design → tasks → code.
> El código no se escribe hasta que el spec está aprobado por un humano.

Solo para features **ambiguas** (`"ambiguity": "vague"` o `acceptance` no
verificables). Si la feature es clara, F3 es sobrecoste puro.

### Estructura

Cada feature F3 (`"sdd": true` en `feature_list.json`) tiene una carpeta
dedicada en cuanto deja `pending`:

```
specs/<feature-name>/
├── requirements.md   # QUÉ se necesita (EARS notation)
├── design.md         # CÓMO se construirá (decisiones técnicas + ## Desafío)
└── tasks.md          # PASOS concretos a implementar
```

El `feature-name` coincide con el campo `name` de `feature_list.json`.

### Estados de una feature

| Estado         | Significado                                                    |
|----------------|----------------------------------------------------------------|
| `pending`      | Sin contrato. En F3, el `sdd_agent_author` es el primero en actuar. |
| `spec_ready`   | Spec drafted. Esperando aprobación humana. NO se toca código. Solo F3. |
| `in_progress`  | Contrato listo. `agent_developer` trabajando.                  |
| `done`         | Verde, revisado y **aprobado por el humano**. Lo escribe el orquestador. |
| `blocked`      | Atascado. Razón en `progress/current.md`.                      |

### Las dos puertas de aprobación

F3 se detiene **dos veces**:

1. **Puerta del spec.** El `sdd_agent_author` termina sus tres archivos, marca
   `spec_ready` y para. El humano lee `specs/<feature>/` —incluida la sección
   `## Desafío`— y dice "aprobado" o pide cambios. Solo entonces el orquestador
   transiciona `spec_ready → in_progress` y lanza el `agent_developer`.
2. **Puerta de cierre.** Tras el `reviewer_agent`, el orquestador para otra vez.
   El humano aprueba y **el orquestador** escribe `done`. Ningún subagente
   escribe `done`, nunca, ni aunque el reviewer haya aprobado.

```
pending → [sdd_agent_author] → spec_ready → ⏸ HUMANO → in_progress
        → [agent_developer → reviewer_agent] → ⏸ HUMANO → done
```

F1 y F2 tienen solo la puerta de cierre.

### requirements.md — EARS estricto

Las requirements se redactan en **EARS** (Easy Approach to Requirements
Syntax). Cada requirement es un párrafo numerado con uno de estos cinco
patrones:

| Patrón         | Plantilla                                                   |
|----------------|-------------------------------------------------------------|
| **Ubicuo**     | `El sistema DEBE <acción>.`                                 |
| **Evento**     | `CUANDO <disparador>, el sistema DEBE <acción>.`            |
| **Estado**     | `MIENTRAS <estado>, el sistema DEBE <acción>.`              |
| **Opcional**   | `DONDE <feature opcional>, el sistema DEBE <acción>.`       |
| **No deseado** | `SI <evento no deseado> ENTONCES el sistema DEBE <acción>.` |

Reglas duras:

- Cada requirement tiene un id estable: `R1`, `R2`, ...
- Cada requirement DEBE ser verificable por al menos un test concreto.
- No mezcles varios `DEBE` en un mismo requirement. Si hay más de uno, parte.
- No uses verbos blandos ("podría", "puede", "soporta"). Solo `DEBE` / `NO DEBE`.

Ejemplo:

```markdown
## R1
CUANDO el usuario ejecuta `python -m src.cli recent`, el sistema DEBE
imprimir hasta 5 notas ordenadas por `created_at` descendente.

## R2
SI el flag `--limit` recibe un valor <= 0 ENTONCES el sistema DEBE
imprimir un mensaje de error en stderr y salir con código != 0.
```

### design.md — decisiones técnicas + desafío

Captura **antes** de tocar código:

- Qué archivos se crean / modifican.
- Qué firmas nuevas aparecen (funciones, clases, comandos).
- Qué excepciones se reutilizan o se añaden.
- La sección **`## Desafío`**, obligatoria, con tres bloques:
  - `### Alternativa descartada` — mínimo una, siempre, con su porqué.
  - `### Objeciones al planteamiento` — las que disparen los gatillos G1-G4, en
    formato `⚠️ OBJECIÓN [G<n>]` + `Evidencia:` + `Alternativa:`. Si no hay
    ninguna, se escribe explícitamente que los cuatro se revisaron sin disparo.
  - `### Riesgo asumido` — objeciones que el humano rechazó y se ejecutan igual.

NO es ingeniería desde primeros principios — apóyate en
`docs/architecture.md` y `docs/conventions.md`. El `design.md` documenta los
puntos donde tu feature roza la frontera de esas reglas.

### tasks.md — checklist ejecutable

Pasos discretos en orden, cada uno con checkbox. Cada task referencia al
menos un `R<n>` que cubre.

Ejemplo:

```markdown
- [ ] T1 — Añadir `cmd_recent` en `src/cli.py`. Cubre: R1, R3.
- [ ] T2 — Registrar subparser `recent` con flag `--limit`. Cubre: R1, R2.
- [ ] T3 — Añadir `test_recent_default_limit` en `tests/test_cli.py`. Cubre: R1.
- [ ] T4 — Añadir `test_recent_invalid_limit` en `tests/test_cli.py`. Cubre: R2.
```

El `agent_developer` marca `[x]` cada task al completarla. El `reviewer_agent`
rechaza si queda alguna `[ ]` sin justificación documentada.

---

## Trazabilidad (regla dura en los tres flujos)

Cambia el identificador, no el principio:

| Flujo | Regla |
|-------|-------|
| F1 | Cada criterio del `acceptance` tiene al menos un test concreto. |
| F2 | Cada `A<n>` tiene al menos un test concreto, y el mapa `A<n> → test` de `progress/plan_<name>.md` coincide con el código real. |
| F3 | Cada `R<n>` tiene al menos un test concreto, y cada test se mapea a un `R<n>`. |

El `reviewer_agent` comprueba esta correspondencia explícitamente y rechaza si
falta. El `agent_developer` documenta el mapa en `progress/impl_<name>.md`:

```markdown
## Trazabilidad
- R1 → `test_recent_default_limit`     ← F3
- A1 → `test_recent_default_limit`     ← F2
```

Un plan o spec que promete tests que no existen es rechazo, no advertencia.

## Cuándo NO aplica SDD

- Features con `"sdd": false`, o sin el campo `sdd`, o con
  `"ambiguity": "clear"` → van por **F1** o **F2** según su alcance real.
- `bootstrap_project` tiene su propio protocolo (Caso F del orquestador) y no
  pasa ni por la matriz de complejidad ni por SDD.

SDD solo se aplica hacia adelante: features previas al proceso no se
retro-especifican.
