# ROADMAP — April con gentle-ai como frontera

> Plan de evolución del arnés, derivado de comparar April contra
> [gentle-ai](https://github.com/Gentleman-Programming/gentle-ai) v2.3.0.
> No es un plan de réplica: gentle-ai marca el destino, el camino es el de
> este repositorio.
>
> Fecha: 25/08/2026, actualizado 26/08/2026 · Estado: propuesta, backlog
> **no** escrito todavía.

---

## Procedencia de la evidencia

La comparación original (25/08) se hizo contra `../gentle-ai-dist`, que
contiene únicamente:

- `README.md` y `docs/review-integration.md`
- `contracts/review-integration/v1` (congelado) y `v2` — 22 schemas + 24
  fixtures deterministas
- `bin/gentle-ai` — ELF x86-64 **stripped**, sin código fuente

El vocabulario de SDD que se cita abajo (`sdd-status`, `nextRecommended`,
`sdd-attempt`, presupuestos, gates de archive) se extrajo con `strings`
sobre el binario: son prompts y mensajes embebidos, no código fuente
leído.

**Actualización 26/08:** se localizó el código fuente completo en
`/home/avalor/Proyectos/gentle-ai` (repo hermano, no el dist stripped).
Todo lo citado abajo como inferido por `strings` queda **confirmado por
código fuente real** (`AGENTS.md`, `openspec/config.yaml`, `contracts/`,
`internal/reviewtransaction/`, `docs/architecture/organic-rdd.md`,
`docs/architecture/guard-population.md`, `skills/`). Se agregan además tres
mecanismos que el binario stripped no dejó ver (sección nueva más abajo).

Tratar lo citado como evidencia fuerte de *vocabulario, contrato e
implementación* — ya no solo inferencia sobre binario.

---

## Tesis

Un solo cambio de fondo; todo lo demás es consecuencia:

> **Mover la autoridad del estado de la prosa al binario.**

April ya tiene los cinco actores correctos (planner, spec writer, ticket
writer, developer, reviewer) y ningún árbitro. Las reglas de proceso viven
en texto que el agente *debería* obedecer: `CLAUDE.md`, el bloque `rules`
de `feature_list.json`, `CHECKPOINTS.md`. Nada las hace cumplir.

`init.sh` ya cubre cerca del 60% de la validación necesaria — pero en
Python embebido en Bash, y solo responde "válido / inválido". Nunca
responde "en qué fase estás" ni "qué sigue".

Frase del README de gentle-ai que resume la brecha:

> *"Trust what the system can derive, not agent narration."*

April todavía confía en la narración del agente.

---

## Estado comparado

| Eje | April hoy | gentle-ai |
|---|---|---|
| Reglas de proceso | prosa en `CLAUDE.md` + `rules` de `feature_list.json` | máquina de estados en Go; el agente solo transporta tokens opacos |
| Contratos | `.claude/manifest.json` con `schemaVersion` | `contracts/v1` congelado + `v2` vivo, con fixtures deterministas |
| Sujeto del review | worktree vivo — puede mutar bajo el revisor | candidato congelado (`subject_hash`), árboles inmutables |
| Evidencia de tests | prosa en el reporte de `agent_developer` | evidencia derivada por el sistema, no narrada |
| Salud del entorno | `init.sh` (bash + python3) | `doctor`, backups comprimidos/dedup/prune, rollback |
| Runtimes soportados | Claude Code | 16 |

### Divergencia deliberada

En gentle-ai el review es **informativo** y nunca bloquea el delivery. En
April, `require_review_to_close` sí es puerta. **No se copia**: con un solo
humano operando, la versión de April es más honesta.

Confirmado con código fuente (26/08): gentle-ai además expone un kill
switch (`gentle-ai review mode disable`) para apagar el ceremonial cuando
no aplica — la ceremonia ahí es proporcional y desactivable, no un piso
fijo. Es coherente con la divergencia ya tomada, no la contradice: April
ya eligió el punto correcto del espectro para un solo operador; el matiz a
llevar a E5 es que la *profundidad* del review, no si ocurre, puede variar
según qué tocó el diff (ver mecanismo 8 arriba).

---

## Cómo maneja gentle-ai las "features"

No las llama features. La unidad es un **change**:
`openspec/changes/<change-name>/` con `proposal.md`, `spec.md`,
`design.md`, `tasks.md` y `verify-report.md`. Se cierra moviéndolo a
`openspec/changes/archive/`.

**No existe un `feature_list.json`.** No hay backlog global declarado: el
directorio existe ⇒ el change está activo; archivado ⇒ está cerrado. El
estado se **deriva del filesystem**, no se declara en un JSON que puede
mentir. Los tickets de April equivalen a los checkboxes de `tasks.md`.

### El ciclo

```
sdd-init → sdd-new / sdd-explore → sdd-propose → sdd-spec
        → sdd-design → sdd-tasks → sdd-apply → sdd-verify → sdd-archive
```

Más `sdd-ff` (fast-forward de planning), `sdd-continue`, `sdd-status`,
`sdd-attempt`, `sdd-research`, `sdd-onboard`.

### Mapeo contra April

| gentle-ai | April |
|---|---|
| `sdd-new` / `sdd-propose` | Fase Grill + `planner_agent` |
| `sdd-spec` + `sdd-design` | `spec_writer` |
| `sdd-tasks` | `ticket_writer` |
| `sdd-apply` | `agent_developer` |
| `sdd-verify` | `reviewer_agent` |
| `sdd-archive` | gate de cierre + `progress/history.md` |
| **`sdd-status` / `sdd-continue`** | **sin equivalente** |

Ese último renglón es todo el hueco. April tiene los cinco actores; le
falta el árbitro que determine objetivamente en qué fase está.

### Los cinco mecanismos ausentes en April

1. **Status calculado, no escrito.** `sdd-status <change> --json` lee los
   artefactos y devuelve `nextRecommended`, `blockedReasons`, `applyState`,
   estados de dependencias, `artifactPaths`, `actionContext`. Cuenta los
   checkboxes de `tasks.md` él mismo.

2. **Prohibido inferir la fase leyendo prosa.** Literal del binario:
   *"Native status is authoritative. Route by next_recommended and
   dependency state, not by prompt inference."* y *"Do not infer routing
   from free text."* April hace hoy exactamente lo contrario.

3. **Grafo de dependencias entre fases**, no lista lineal — con detección
   de ciclos. En April los `Blocked by` de tickets se evalúan a ojo.

4. **`apply` es una transacción con presupuesto congelado.**
   `sdd-attempt acquire/settle`, compare-and-swap, y errores como
   `"SDD runtime objective changed without an explicit reset"`. Si el
   implementador se va por la tangente, el sistema lo detecta.

5. **Escribir el veredicto requiere admisión previa.** *"run
   `gentle-ai sdd-verify-validate` … before any OpenSpec write. If the
   validator … denies admission, make zero writes."* Para archivar:
   `critical_findings must be zero`, más un gate contra checkboxes sin
   marcar. En April, `reviewer_agent` escribe su veredicto libremente.

### Tres mecanismos adicionales, confirmados con código fuente (26/08)

El dist stripped no dejó ver estos tres; el código fuente sí. Se integran
en E5/E6 más abajo, no cambian el orden de etapas.

6. **Ratchet de calidad progresiva.** gentle-ai congela el estado
   imperfecto actual en un archivo baseline (`.deadcode-baseline.txt`,
   `.refusal-ratchet-baseline.txt`, `.guard-population-baseline.txt`) y
   solo falla el build si la métrica **crece** frente a esa línea base —
   nunca exige llegar a cero de entrada. Justificación en el propio script:
   exigir cero antes de habilitar el guard significa que el guard nunca se
   habilita; congelar y prohibir crecimiento sí paga. Resuelve un problema
   que April no tiene resuelto: cómo meter un gate de calidad en código ya
   existente sin bloquear todo el desarrollo hasta limpiarlo entero.

7. **Kill switch del ceremonial.** `gentle-ai review mode disable` apaga
   todo el aparato de revisión transaccional cuando no aplica. Principio
   explícito del propio autor: una herramienta que fuerza ceremonia para
   cambiar una coma se desinstala en tres días. Refuerza — no contradice —
   la "Divergencia deliberada" ya registrada abajo: la ceremonia debe ser
   proporcional y desactivable, no un piso fijo.

8. **Revisión disparada por qué cambió, no por tipo de feature.** En
   gentle-ai, tocar áreas sensibles (ej. auth) exige más revisores/pasos
   que tocar docs o config; no es "toda feature pasa por el mismo
   reviewer". `reviewer_agent` en April hoy es parejo para toda feature
   `sdd: true` sin distinguir qué tocó el diff.

---

## El plan — 6 etapas

Cada etapa es una feature entregable e independiente. El orden E1 → E2 → E3
no es negociable: **E1 define el modelo de fase que E3 hace cumplir**.

### E0 — Cerrar `bootstrap_project` *(deuda, no mejora)*

`docs/architecture.md`, `docs/conventions.md` y `docs/verification.md`
siguen con secciones en `_pendiente_`. Es la feature 1, todavía `pending`.
Cualquier cosa construida encima valida contra documentos vacíos.

Costo: una sesión de Grill con el humano. Sin código.

---

### E1 — `april status` · el árbitro 🔑

El peldaño de mayor palanca. Todo lo demás depende de él.

Comando que **lee el disco y calcula** — no consulta el `status` escrito a
mano:

```bash
april status [feature] --json
```

Deriva el estado de: `feature_list.json`, existencia de
`specs/<name>/spec.md`, existencia y contenido de
`specs/<name>/tickets/*.md`, y sus campos `Status` y `Blocked by`.

Devuelve, adoptando el vocabulario que gentle-ai ya validó en producción:

| Campo | Significado |
|---|---|
| `phase` | `grill` \| `spec` \| `tickets` \| `implementation` \| `review` \| `closed` |
| `nextRecommended` | la única acción legal en este momento |
| `blockedReasons[]` | si no está vacío, no se avanza |
| `frontier[]` | tickets con todos sus `Blocked by` en `done` |
| `artifactPaths` | qué archivos leer para actuar |

Absorbe el bloque Python de `init.sh`, que pasa a invocar `april status`.

**Acceptance verificable**

- Feature `sdd: true` sin `spec.md` ⇒ `phase: spec`.
- Feature con spec aprobada y sin tickets ⇒ `phase: tickets`.
- Dos features en `in_progress` ⇒ `blockedReasons` no vacío.
- Ciclo en los `Blocked by` de tickets ⇒ detectado y reportado, no colgado.
- `go build ./...` y `go test ./...` en verde.

**Recomendación:** `sdd: true` — el modelo de fases merece spec escrita.

---

### E2 — `CLAUDE.md` obedece al árbitro

Solo documentación. Sin código.

Hoy el protocolo dice *"ejecuta la fase que toque según el estado de la
feature elegida"* — inferencia sobre prosa. Pasa a decir:

> Corre `april status --json`. Enruta **solo** por `nextRecommended` y
> `blockedReasons`. Nunca infieras la fase leyendo texto. Si
> `blockedReasons` no está vacío, repórtalos y detente.

Es literalmente la regla que gentle-ai embebe en su binario. Es barato, y
convierte E1 de "comando útil" en "protocolo".

---

### E3 — `april feature set-status` · transiciones validadas

```bash
april feature set-status <id> <estado>
```

Valida la transición contra el grafo
(`pending → spec_ready → in_progress → done`, más `blocked`), rechaza un
segundo `in_progress`, y rechaza `done` sin veredicto registrado.

`one_feature_at_a_time` deja de estar **prohibido** y pasa a ser
**imposible**. Lo mismo para `require_approved_spec_to_implement`.

Aquí es donde se materializa la decisión abierta (ver más abajo).

**Recomendación:** `sdd: false` — mecánico una vez que E1 fijó el modelo.

---

### E4 — Receipts · la evidencia deja de ser narración

Hoy `require_tests_to_close` se satisface con un párrafo de
`agent_developer`. Nadie corre nada de forma independiente.

```bash
april verify record --feature <id> -- go test ./...
```

Corre el comando, captura exit code, salida y hash del árbol, y hace
**append** a un ledger. `april status` lee ese ledger: sin receipt en verde
⇒ `blockedReasons: ["no_test_evidence"]`.

Mismo patrón para el veredicto de `reviewer_agent`: se **registra**, no se
narra.

Espejo directo de la regla de gentle-ai: *"if the validator denies
admission, make zero writes"*.

**Recomendación:** `sdd: true`.

---

### E5 — Candidato congelado para review

`april review start` ejecuta `git write-tree` y devuelve un `subject_hash`.
`reviewer_agent` emite su veredicto **contra ese hash**. Si el árbol cambió
mientras revisaba, el receipt no se admite.

Elimina el fallo silencioso "el revisor revisó otra cosa". Es la idea
central del RDD de gentle-ai, en ~80 líneas de Go en lugar de 85k.

**Extensión (26/08):** el mismo comando puede llevar el mecanismo 8 —
`reviewer_agent` recibe, junto al `subject_hash`, qué paquetes/rutas tocó
el diff, y ajusta profundidad (cuántas pasadas, qué exige explícitamente)
según si tocó áreas marcadas como sensibles en `docs/conventions.md` (a
definir en E0). No es una etapa nueva: es un parámetro más del veredicto
que E5 ya construye.

**Recomendación:** `sdd: true`.

---

### E6 — `april doctor` + snapshot en `init` + ratchet de deuda

- `april doctor`: read-only — manifiesto contra disco, drift, agentes
  presentes, `status` sano.
- `april init`: backup antes de `applyPlan`. El manifiesto ya existe; falta
  la red de seguridad.
- **Ratchet de calidad progresiva (26/08, mecanismo 6):** si este repo
  acumula deuda medible (código muerto, TODOs sin feature asociada — ver
  C3 en `CHECKPOINTS.md`), `april doctor` congela un baseline y falla solo
  si la métrica crece frente a él. No exige limpiar la deuda existente
  para empezar a exigir que no crezca — el mismo patrón que evitó que
  gentle-ai bloqueara su propio desarrollo al introducir el guard.

Higiene. Va al final a propósito.

**Recomendación:** `sdd: false`.

---

## Fuera de alcance, deliberadamente

- Soporte multi-runtime (16 agentes)
- Backends de persistencia intercambiables (`engram | openspec | hybrid`)
- Grafo de fases dinámico — las 5 fases fijas de April son suficientes
- Contratos negociados con transiciones opacas y `consent envelopes`
- Firma minisign de releases y canales stable/beta
- TUI

Ahí vive la mayor parte de esas 85k líneas y ninguna resuelve un problema
que April tenga hoy.

---

## Decisión abierta

**¿El binario es dueño del estado, o solo opina?** Cambia la forma de
E3 a E5.

**A · Advisory** — `april status` informa; el orquestador y el humano
siguen editando `feature_list.json` a mano. Retrocompatible, cero fricción,
pero el agente todavía *puede* mentir sobre el estado.

**B · Autoritativo** — `feature_list.json` solo se escribe vía
`april feature set-status`. Editarlo a mano queda fuera de protocolo. El
agente ya no puede mentir. Es el camino que tomó gentle-ai.

**Recomendación: B, llegando por A.** E1 y E2 son idénticos en ambos
caminos, así que la decisión se toma recién en E3, con el árbitro ya
funcionando y sin haber apostado nada.

**Estado: decidido (26/08/2026) — B, llegando por A.** El humano confirmó
la recomendación. Efecto concreto en el plan:

- E1 (`april status`) y E2 (`CLAUDE.md` enruta por él) se construyen y
  operan en modo **advisory**: el binario calcula y opina, pero
  `feature_list.json` se sigue editando a mano mientras se valida que el
  modelo de fases (`phase`, `nextRecommended`, `blockedReasons`) es
  correcto en uso real.
- Recién al llegar a E3 (`april feature set-status`), y solo si E1/E2
  demostraron ser confiables en la práctica, la escritura de
  `feature_list.json` pasa a ser exclusiva de ese comando — editarlo a
  mano queda fuera de protocolo desde ese punto en adelante.
- Esto no cambia el orden ni el contenido de E0-E6 ya descrito arriba;
  solo confirma que E3 es el punto de no retorno, no E1.

Con esto resuelto, el backlog puede escribirse.
