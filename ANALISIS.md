# ANÁLISIS — april-harness

> Auditoría completa del proyecto. Fecha: 2026-07-29.
> Método: lectura de todos los archivos del harness + verificación empírica
> (scaffold real en directorio temporal, `git check-ignore`, `go build` medido,
> `./init.sh` ejecutado en repo y en proyecto generado).
>
> **Cómo usar este documento:** cada hallazgo tiene un ID estable, evidencia,
> fix y criterio de verificación. Se abordan **de uno en uno**, en el orden de
> la §7. Al cerrar uno, marca su checkbox y anota la fecha.

---

## Conclusión

El harness funciona **en su propio repo**, pero el producto que genera está roto
en 4 puntos verificados (§1). El problema del crecimiento de specs es real y su
causa raíz no es el redactor: **la puerta que decide "esto merece spec" nunca se
dispara**, porque hay 3 reglas contradictorias y ninguna la aplica una máquina (§2).

Patrón de fondo, repetido en todo el proyecto: **las reglas que solo viven en
prosa (`.md`) no se cumplen; las que verifica un script sí.**

---

## 1. Bloqueantes — rompen proyectos scaffoldeados

### [ ] B1 — `.gitignore` esconde el harness entero en cada proyecto generado

**Severidad:** crítica.

Patrones sin anclar en `.gitignore` (bloque "# Agent artifacts"):

```
architecture.md
conventions.md
verification.md
.claude/agents/
session-handoff.md
```

Sin `/` inicial, git los aplica **en cualquier profundidad**.

**Evidencia** (`git check-ignore` en scaffold nuevo):

| Archivo | Estado |
|---|---|
| `docs/architecture.md` | IGNORED |
| `docs/conventions.md` | IGNORED |
| `docs/verification.md` | IGNORED |
| `.claude/agents/orquestador.md` | IGNORED |
| `session-handoff.md` | IGNORED |

Consecuencia: usuario hace `harness init` + `git add .` + commit → repo **sin
agentes y sin docs**. El harness desaparece del control de versiones.

Y en **este** repo: `templates/docs/*.md` también IGNORED → un clone limpio
compila un CLI que genera proyectos sin `docs/`.

**Fix:** anclar o eliminar. Los archivos de agentes y los docs de proceso son
exactamente lo que **debe** estar trackeado; solo tiene sentido ignorar los
informes efímeros (`progress/impl_*.md`, `progress/review_*.md`, decisión en K1).

**Verificación:** en scaffold nuevo, los 5 archivos de la tabla dan `tracked`.

---

### [ ] B2 — 62 MB de `node_modules` embebidos en el binario

**Severidad:** crítica.

`main.go:20` → `//go:embed ... .opencode ...` incluye `.opencode/node_modules`
(62 MB en disco).

**Evidencia medida:**

| Binario | Tamaño |
|---|---|
| `harness` (jul 6, antes del embed de `.opencode`) | 2.9 MB |
| build actual | **57.6 MB** |

- `strings` sobre el binario → 42 rutas `node_modules/@ai-sdk`.
- `harness init` con OpenCode seleccionado copia los 62 MB **archivo por
  archivo**, imprimiendo miles de líneas `Created .opencode/node_modules/...`.
  Scaffold resultante: 62 MB (medido con `du -sh`).
- `.opencode/plugins/recap.js` **no importa** `@opencode-ai/plugin` → la
  dependencia declarada en `.opencode/package.json` es innecesaria.
- Goreleaser publica ese binario ×4 plataformas (linux/darwin × amd64/arm64).

**Fix:** sacar `node_modules` y `package*.json` del embed (embeber solo
`.opencode/plugins` + `opencode.json`). Nota: `go:embed` ya excluye los archivos
dot-prefijados de subdirectorios, por eso `.opencode/.gitignore` tampoco llega
al destino — ver B4b.

**Verificación:** `go build` produce binario <4 MB; scaffold con OpenCode pesa
<1 MB; `strings binario | grep -c node_modules` == 0.

---

### [ ] B3 — `docs/specs.md` no se embebe (el proceso SDD no llega al proyecto)

**Severidad:** crítica.

`templates/docs/` solo contiene `architecture.md`, `conventions.md`,
`verification.md`. `docs/specs.md` existe en la raíz del harness pero **no está
en `templates/`** ni en el embed.

**Evidencia:** en scaffold nuevo → `docs/specs.md` **MISSING**. Lo referencian 5
archivos que sí se copian: `AGENT.md:16`, `AGENT.md:28`, `orquestador.md:55`,
`sdd_agent_author.md:21`, `agent_developer.md:22`, `reviewer_agent.md:6`.

Consecuencia: el agente que redacta specs arranca sin su manual (EARS, los 3
archivos, la puerta de aprobación) → improvisa. `init.sh` §1 no lo verifica, así
que la falla es silenciosa.

**Fix:** `templates/docs/specs.md` + añadirlo a `BASE_FILES` de `init.sh`.

**Verificación:** scaffold nuevo tiene `docs/specs.md`; `./init.sh` en un
proyecto sin él da `[FAIL]`.

---

### [ ] B4 — `README.md` no embebido + README raíz con doble identidad

**Severidad:** alta.

`README.md` no está en la directiva `//go:embed` → **MISSING** en el scaffold
(verificado). Además su contenido describe *el proyecto generado* ("Project
Name", "Getting started: run ./init.sh"), no el CLI: cero instrucciones de
instalar o usar `harness init`, ningún flag documentado.

**Fix:** separar. `README.md` (raíz) = documentación del CLI `harness`.
`templates/README.md` = README del proyecto generado, embebido.

**Verificación:** scaffold nuevo tiene `README.md`; el README raíz documenta
instalación + `harness init|version|help`.

#### [ ] B4b — `.opencode/.gitignore` tampoco llega

`go:embed` excluye archivos dot-prefijados dentro de subdirectorios → el
`.gitignore` de `.opencode/` no se copia (verificado: MISSING). Hoy queda
cubierto por el patrón `.opencode/node_modules/` del `.gitignore` raíz, pero es
una dependencia frágil. Se resuelve junto con B2 (si no hay `node_modules`, no
hay nada que ignorar).

---

### [ ] B5 — nada commiteado desde el refactor

**Severidad:** alta.

Untracked: `config.go`, `config_test.go`, `detector.go`, `detector_test.go`,
`recap.sh`, `recap_test.go`, `templates/`, `specs/`, `.claude/settings.json`,
`.claude/hooks/`, `.opencode/`, `opencode.json`. Modificados sin commit:
`main.go`, `selector.go`, `init.sh`, `CLAUDE.md`, `docs/*`, `feature_list.json`,
`progress/*`, agentes.

Combinado con B1 (`templates/docs/*` ignorado): **el clone actual no reproduce
el repo local**.

**Fix:** commitear en bloques temáticos después de B1 (si se commitea antes,
`templates/docs/` se queda fuera).

**Verificación:** `git status` limpio; clone en directorio temporal → `go build`
+ `harness init` producen scaffold completo.

---

## 2. Problema principal — specs que crecen

### Evidencia dura

`auto_recap_hook` entregó ~150 líneas de producto (bash de 57 + hook de 8 +
plugin de 40 + tests). Su spec:

| Archivo | Líneas | Contenido |
|---|---|---|
| `requirements.md` | 138 | **R1–R18** |
| `design.md` | 270 | incluye "Investigación previa" (~40 líneas) y "Riesgos a validar" |
| `tasks.md` | 59 | T1–T14; T11/T12/T13 = "corre build / test / init.sh" |

**467 líneas de spec por ~150 de código. Ratio 3:1.** Y la feature estaba
marcada `"ambiguity": "clear"`.

Comparación: `centralize_config` → 37 + 74 + 11 = 122 líneas. La inflación es
creciente, no constante.

### [ ] S1 — Causa raíz: la puerta SDD no se dispara (3 reglas en conflicto)

**Severidad:** crítica (es el hallazgo que pediste).

| Fuente | Cuándo aplica SDD |
|---|---|
| `docs/specs.md:8` | feature con `"sdd": true` |
| `feature_list.json:15` (`sdd_required_when`) | `sdd: true` **AND** `ambiguity == "vague"` |
| `orquestador.md:33` (matriz) | solo clasificación **AMBIGUO** |
| `sdd_agent_author.md:19` | primera `pending` con `sdd: true` — **ignora `ambiguity`** |

El `planner_agent` pone `sdd: true` a todas las features (formato fijo en su
plantilla) → el `sdd_agent_author` no consulta `ambiguity` → **todo pasa por SDD
completo**. Las 2 features del repo son `sdd:true` + `clear` y ambas recibieron
spec completa. La matriz SIMPLE/MEDIO/AMBIGUO existe en papel y **no se ejecuta
nunca**.

**Fix:** una sola regla, un solo campo. Eliminar el booleano `sdd`; conservar
`ambiguity` (`clear` | `vague`). SDD ⟺ `ambiguity == "vague"` **o** ≥4 archivos
tocados. `clear` → ruta MEDIO: `agent_developer` + `reviewer_agent` trabajando
contra `acceptance`, sin spec. Actualizar las 4 fuentes de la tabla para que
digan lo mismo.

**Verificación:** una feature `clear` recorre el flujo entero sin crear
`specs/<name>/`; `init.sh` no la marca como incompleta.

---

### [ ] S2 — Cero techos en el redactor de specs

**Severidad:** alta.

`sdd_agent_author.md` no limita número de `R<n>`, largo de `design.md`, ni
número de tasks. Sin techo, el modelo maximiza cobertura → R18.

**Fix:** techos duros en `sdd_agent_author.md`:

- `R<n>` ≤ número de criterios de `acceptance` (máximo absoluto 8).
- `design.md` ≤ 80 líneas, secciones fijas: *archivos afectados* / *firmas
  nuevas* / *1 alternativa descartada*.
- `tasks.md` ≤ 10 tasks.
- **Prohibido** crear tasks de verificación (build/test/init.sh): ya están en
  `init.sh` y `CHECKPOINTS.md` C1–C3.
- Investigación previa y riesgos → `progress/spec_<name>_notes.md`, no al spec.

**Verificación:** S5 (enforcement en `init.sh`) falla si se incumple.

---

### [ ] S3 — Requirements contaminadas de diseño → el mismo hecho escrito 3 veces

**Severidad:** alta.

`specs/auto_recap_hook/requirements.md` R7 nombra `.claude/settings.json`, el
hook `SessionStart` y `type: "command"` — eso es **design**, no requirement. El
mismo hecho aparece en `requirements.md`, `design.md` y `tasks.md` → triple
mantenimiento y triple volumen.

**Fix:** regla en `docs/specs.md`: requirements = comportamiento observable;
**prohibido nombrar rutas y archivos**. `design.md` es el único lugar con paths.
Flujo de una sola dirección: `acceptance` → `R<n>` → `T<n>`; `design.md`
referencia `R<n>`, no los reescribe.

**Verificación:** grep de `.md`/`.json`/`.sh`/`.go` en `requirements.md` → 0
resultados (fuera de bloques de código de ejemplo).

---

### [ ] S4 — Los specs no tienen ciclo de vida (crecimiento acumulativo)

**Severidad:** alta.

Nada compacta ni archiva `specs/<feature>/` al pasar a `done`. `specs/` crece
lineal y para siempre. En paralelo, `agent_developer.md:22` y
`reviewer_agent.md:6` ordenan leer 4 docs + el spec completo en cada arranque →
**el coste de contexto por sesión crece con la edad del proyecto**. Ese es el
síntoma que estás notando.

**Fix:** al marcar `done`, colapsar `specs/<feature>/{requirements,design,tasks}.md`
en un único `specs/_done/<feature>.md` de ≤30 líneas: requirements finales +
mapa `R<n> → test`. Los 3 archivos de proceso mueren con la feature. Añadir el
paso al cierre de `AGENT.md` §5 y a `agent_developer.md` paso 8.

**Verificación:** tras cerrar una feature, `specs/<feature>/` no existe y
`specs/_done/<feature>.md` ≤30 líneas.

---

### [ ] S5 — Enforcement mecánico de los techos

**Severidad:** alta (sin esto, S2/S3/S4 son prosa más y se ignoran como el resto).

**Fix:** nueva sección en `init.sh`: `[FAIL]` si algún `design.md` >80 líneas, si
un `requirements.md` tiene >8 `R<n>`, o si un `tasks.md` tiene >10 tasks.

**Verificación:** spec inflado a mano → `./init.sh` sale en rojo.

---

## 3. Incoherencias que confunden a los agentes

### [ ] C1 — Tres vocabularios para los mismos 5 agentes

**Severidad:** alta.

| Fuente | Nombres |
|---|---|
| Tipos reales / `CLAUDE.md` | `sdd_agent_author`, `agent_developer`, `reviewer_agent` |
| `orquestador.md` (líneas 32, 33, 59, 72, 73, 76, 95, 97, 103, 104, 106, 132) | `spec_author`, `implementer`, `reviewer` |
| `opencode.json` (`agent`) | `spec-author`, `developer`, `reviewer`, `planner` |

El orquestador solo funciona porque `CLAUDE.md` trae el mapeo correcto; siguiendo
su propia definición invocaría `subagent_type` inexistentes.

**Fix:** unificar a los nombres reales en `orquestador.md` y `opencode.json`.

**Verificación:** `grep -rn "implementer\|spec_author\|spec-author" .claude opencode.json` → 0.

---

### [ ] C2 — `AGENTS.md` no existe (21 días abierto)

**Severidad:** media (fricción en cada arranque).

El archivo es `AGENT.md`. Lo mandan leer: `orquestador.md:14`,
`sdd_agent_author.md:21`, `agent_developer.md:22`. Detectado como **crítico** en
`mejoras.md:32` (2026-07-05) y de nuevo en `RECAP_2026-07-14.md:35,55,66`. Sigue
sin corregir → evidencia de que los planes de mejora en `.md` no se ejecutan.

**Fix:** `AGENTS.md` → `AGENT.md` en los 3 agentes.

---

### [x] C3 — `planner_agent` se contradice y se lanza siempre

**Severidad:** media.

- Dice "Solo editas `feature_list.json` ningún otro archivo" y dos secciones más
  abajo "Guarda las respuestas en `progress/project-definition.md`".
- `AGENT.md` §4 paso 0 y `orquestador.md` paso 5 lo lanzan **siempre** al inicio
  → interrogatorio forzado en cada sesión, incluso cuando no hay nada que
  planificar.

**Fix:** permitir escritura en `progress/project-definition.md` explícitamente; y
lanzarlo solo si el `feature_list.json` está en estado template o el usuario lo
pide.

**Estado 31/07/2026 — partes (a) y (b) cerradas.**

- ✅ Contradicción de escritura resuelta: el `planner_agent` escribe **solo**
  `feature_list.json`. `progress/project-definition.md` lo mantiene ahora el
  orquestador, que es quien conduce el Grill.
- ✅ Parte (b) — disparo condicional. La planificación corre solo si:
  `feature_list.json` en estado template, backlog agotado (nada en `pending`
  ni `in_progress`), o el usuario la pide. En cualquier otro caso el
  orquestador salta directo a los Casos A-D sin preguntar nada.
- ✅ **Defecto extra encontrado al revisar (no estaba en el hallazgo original):**
  el Grill era inejecutable. `planner_agent.md` declaraba tool `Question` y un
  protocolo interactivo ("una pregunta a la vez", "pide confirmación
  explícita"), pero un subagente no tiene canal con el usuario. Fix: la FASE
  Grill se movió al hilo principal (`orquestador.md`), el `planner_agent` queda
  reducido a Decomposer puro y se le quitó el tool `Question`.
- ✅ Grill recortado de 5 preguntas a 2 (objetivo, tech stack) por decisión del
  usuario: nombre, módulos, flujo crítico y restricciones no cambian cómo se
  orquesta. Esas 3 últimas secciones nacen `_pendiente_` en
  `project-definition.md` y se rellenan al implementar, cuando una feature
  revela un módulo o restricción real.
- ⚠️ Nota histórica: `progress/planning_session.md` (21/07/2026) fue un log
  improvisado por el planner antes de que existiera el contrato. Queda en disco
  como evidencia; el contrato prohíbe más logs por sesión (la bitácora va dentro
  de `project-definition.md`).

Archivos tocados: `orquestador.md` (paso 5 + sección FASE Grill nueva),
`planner_agent.md` (reescrito), `AGENT.md` §4, `CLAUDE.md` (paso 5 + entrada
`planner_agent` en la lista de subagentes), `opencode.json`.
- ⚠️ Pendiente: `april-harness` mismo no tiene `progress/project-definition.md`
  — requiere pasar el Grill con el humano.

---

### [ ] C4 — Agentes OpenCode ven menos reglas que los de Claude

**Severidad:** media.

`opencode.json.instructions` = `AGENT.md`, `docs/architecture.md`,
`docs/conventions.md`, `docs/verification.md`, `CHECKPOINTS.md`. **Falta
`CLAUDE.md`** (reglas de rol, mapeo de subagentes) y **`docs/specs.md`** (proceso
SDD).

**Fix:** añadir ambos a `instructions`.

---

### [ ] C5 — Guardarraíl que no ata: `src/` y `tests/`

**Severidad:** baja, pero conceptual.

`CLAUDE.md` y `orquestador.md` prohíben editar `src/` y `tests/` en tareas
MEDIO/AMBIGUO. Este repo no tiene ninguno de los dos (Go en raíz, `*_test.go` al
lado del código, tal como manda `docs/conventions.md`). `main.go` crea
`src/ tests/ specs/` siempre. En cualquier proyecto Go/Rust idiomático la regla
es letra muerta.

**Fix:** redefinir el guardarraíl por *tipo de archivo* ("código de aplicación y
tests, donde vivan") en lugar de por directorio; y no crear `src/`+`tests/`
cuando el lenguaje no los usa.

---

## 4. Verificación más débil de lo que se afirma

### [ ] V1 — `init.sh` nunca corre tests

**Severidad:** crítica (invalida el criterio de cierre de todo el flujo).

`init.sh` §4 solo hace `go build -o /dev/null .`. No hay ninguna invocación de
tests. Pero:

- `AGENT.md:39-40` — "No declares una tarea `done` sin pruebas verdes. Ejecuta
  `./init.sh` y asegúrate de que **el bloque de tests** pasa al 100%".
- `CHECKPOINTS.md` C3 — "Todos los tests pasan".
- `reviewer_agent.md` paso 6 + regla dura — "Nunca apruebes con tests rojos",
  apoyándose en `./init.sh`.

**Ese bloque de tests no existe.** Tres documentos firman sobre una verificación
inexistente.

**Fix:** sección de tests en `init.sh` con detección de stack (`go test ./...`,
`pytest`, `npm test`) y `[FAIL]` en rojo.

**Verificación:** romper un test a propósito → `./init.sh` sale != 0.

---

### [ ] V2 — `init.sh` es Go-only y python3-only pero se envía a cualquier proyecto

**Severidad:** alta.

- §2 (validación de `feature_list.json` y specs) exige `python3`. Si falta, el
  heredoc falla y el mensaje es confuso. Requisito real no declarado en `init.sh`
  (solo aparece de pasada en el README que además no se copia — B4).
- §4 solo entiende `go.mod`. Proyecto Node/Python/Rust → `[WARN]` y sigue: cero
  verificación de compilación.
- `recap.sh` también depende de `python3` inline, con `except:` desnudo.

**Fix:** validación de JSON en Go (subcomando `harness check`) o fallback sin
python3; detección de stack para el paso de build.

---

### [ ] V3 — CI no verifica nada

**Severidad:** alta.

`.gitlab-ci.yml` tiene un único stage `release` que corre goreleaser en `main`.
Sin `go build`, sin `go vet`, sin `go test`. Cada push a `main` publica release
(hoy, con el binario de 57 MB de B2) sin puerta de calidad.

Un test de scaffold golden (`harness init` en temp → comparar árbol de archivos
esperado) habría cazado **B1, B2, B3 y B4** el primer día.

Bloqueador para automatizarlo: `harness init` es interactivo. No hay
`--tools claude --yes`. (El fallback `selectNumbered` acepta stdin por pipe, que
es lo que usé para auditar, pero no es una interfaz declarada.)

**Fix:** stages `test` (build+vet+test) antes de `release`; flags
`--tools`/`--yes`; test de scaffold golden.

---

### [ ] V4 — `centralize_config` está `done` con acceptance falso

**Severidad:** alta (contamina el contexto de los agentes).

Feature ID 1, `status: "done"`. Estado real hoy:

| Acceptance | Realidad |
|---|---|
| "La versión se define únicamente en `config.go`" | ✅ (`config.go` = 2 líneas) |
| "La lista de archivos requeridos se define únicamente en `config.go`" | ❌ no existe `RequiredFiles` |
| "`init.sh` consume la lista desde `config.go`" | ❌ `init.sh:23` rehardcodea `BASE_FILES` |
| "No hay valores duplicados entre `main.go` e `init.sh`" | ❌ |

No existen `gen_required.go` ni `required_files.txt`, pero
`progress/history.md` (entrada 2026-07-16) sigue afirmando que se crearon. Un
agente que lea la bitácora arranca con contexto falso — **ya ocurrió** en una
sesión anterior. `docs/architecture.md` ("Config | config.go | Versión, rutas,
archivos requeridos") y `docs/conventions.md` ("config.go — futuro") también
quedaron stale y se contradicen entre sí.

Origen: una simplificación posterior ("sin ruido en los proyectos generados")
revirtió la implementación sin actualizar feature, spec, historia ni docs.

**Fix:** decidir. O reabrir la feature, o corregir `acceptance` + entrada de
`history.md` + los 2 docs para que describan la decisión real. Regla nueva: si
una decisión revierte una feature `done`, la entrada de `history.md` se
enmienda en el mismo commit.

---

## 5. Ruido y deuda menor

### [ ] K1 — Informes de auditoría invisibles y sin límite

`.gitignore` ignora `progress/*.md` salvo `current.md` y `history.md` → los
`impl_<feature>.md` y `review_<feature>.md` que el harness **obliga** a producir
(regla anti-teléfono-descompuesto) no se versionan y se acumulan en disco.
Decidir: trackearlos (son la traza de auditoría) o borrarlos al cerrar la feature.

### [ ] K2 — Cinco artefactos de estado solapados

`progress/current.md`, `progress/history.md`, `session-handoff.md`,
`RECAP_2026-07-14.md`, `mejoras.md`. Los dos últimos son one-offs de julio,
stale, en la raíz (`mejoras.md` además trackeado). `session-handoff.md` solo lo
usa el Caso G del orquestador. Consolidar en: *ahora* (`current.md`) + *pasado*
(`history.md`); el resto a `docs/notes/` o borrar. **Este `ANALISIS.md` corre el
mismo riesgo** — es un one-off de raíz: cerrarlo o convertirlo en features.

### [ ] K3 — `CHECKPOINTS.md` no cubre la ruta sin spec

C4 (trazabilidad `R<n>` ↔ test) y C5 (`tasks.md` en `[x]`) solo aplican a la ruta
SDD. Con S1 aplicado, la ruta MEDIO pasa a ser la mayoritaria y se queda **sin
criterios de cierre**. Añadir equivalentes contra `acceptance`.

### [ ] K4 — Cosmética de `init.sh`

§2 imprime `[OK]` sin los códigos de color que usan las demás secciones (el
heredoc de python no pasa por `ok()`).

---

## 6. Lo que está bien (no tocar)

- `recap.sh` como fuente única de verdad, consumido por `init.sh` §5, el hook
  `SessionStart` de Claude y el plugin de OpenCode. Sin lógica duplicada.
- `mergeGitignore` — no destruye el `.gitignore` del usuario, solo añade lo que
  falta.
- `detector.go` — lógica pura, `lookPath` inyectado, testeable. Respeta las
  reglas de capas de `docs/architecture.md`.
- Regla anti-teléfono-descompuesto: subagentes devuelven una línea + referencia a
  archivo. Es la decisión de diseño más valiosa del harness.
- `opencodeGenerator.Transform` — traduce frontmatter de Claude a OpenCode sin
  mantener dos juegos de definiciones de agente.

---

## 7. Orden de ejecución

Uno a uno. Cada bloque cierra con `./init.sh` verde y commit.

### Sprint 1 — Bloqueantes (~1 día)

1. **B1** `.gitignore` anclado ← primero, o B5 commitea sin `templates/docs/`
2. **B2** sacar `node_modules` del embed
3. **B3** `templates/docs/specs.md` + `BASE_FILES`
4. **B4** + **B4b** README del CLI / README del template
5. **B5** commit de todo lo pendiente

### Sprint 2 — Crecimiento de specs (~1 día)

6. **S1** unificar la regla SDD en un solo campo ← el fix de raíz
7. **S2** techos duros en `sdd_agent_author.md`
8. **S3** separación requirements / design
9. **S4** compactación a `specs/_done/`
10. **S5** enforcement en `init.sh` ← sin esto, 6–9 se ignoran

### Sprint 3 — Coherencia y verificación (~medio día)

11. **C1** nombres de subagente unificados
12. **C2** `AGENTS.md` → `AGENT.md`
13. **V1** bloque de tests en `init.sh`
14. **V4** reabrir `centralize_config` o enmendar acceptance + history + docs
15. **C4** `instructions` de OpenCode
16. **V3** CI con test + flags `--tools`/`--yes` + scaffold golden

### Backlog

**C3**, **C5**, **V2**, **K1**, **K2**, **K3**, **K4**.

---

## Registro de cierre

| ID | Fecha | Notas |
|---|---|---|
| C3 (a) | 31/07/2026 | Contradicción de escritura del `planner_agent` resuelta: 2 archivos escribibles + schema de `progress/project-definition.md`. Parte (b) — "se lanza siempre" — sigue abierta. |
| C3 (b) | 31/07/2026 | Cerrada. Disparo condicional (template / backlog agotado / petición del usuario). Además: FASE Grill movida al hilo principal del orquestador — el subagente no tenía canal con el usuario, el Grill era inejecutable; tool `Question` eliminado. Grill recortado a 2 preguntas; Módulos/Flujo/Restricciones pasan a incrementales. |
