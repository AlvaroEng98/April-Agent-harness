# Session History

<!-- Append-only log. Most recent at the top. -->

## 2026-08-31 (continuación 2) — Bit de ejecución perdido en `.claude/hooks/`: feature 19 (done)

El humano corrió `april init` en un proyecto real
(`/home/avalor/Proyectos/Kada/CO-Backend`) y encontró que Claude Code
arrancaba con `Failed with non-blocking status code: ... Permiso
denegado` sobre `.claude/hooks/block-dangerous-git.sh` — el guardrail de
comandos git peligrosos nunca se ejecutaba.

### Feature 19 `scaffold_hooks_keep_exec_bit` (done, 31/08/2026)

Causa raíz: `go:embed` no preserva el bit de ejecución del árbol fuente;
`planScaffold` decidía el `mode` de cada archivo a mano vía un `switch
d.Name()` que solo cubría `"init.sh"` ⇒ `0755` — cualquier archivo bajo
`.claude/hooks/` quedaba en `0644` en el destino. Bug silencioso, no
específico de CO-Backend: **el guardrail nunca funcionó en ningún
proyecto scaffoldeado hasta ahora**. Fix con TDD rojo→verde: test nuevo
(`TestPlanScaffoldHooksQuedanEjecutables`) confirmado en rojo primero,
luego una condición por `relSlash` (`strings.HasPrefix(relSlash,
".claude/hooks/")` → `0755`, mismo estilo que el guard de dogfooding de
`.claude/manifest.json`) — por prefijo de ruta, no por nombre de archivo
puntual, para no repetir el bug con el próximo hook. `init.sh` y el
resto de archivos (`0644`) sin cambios. Revisión: `APPROVED` en la
primera pasada, con verificación activa de que el bloque nuevo no
reordena ni reemplaza el caso de `init.sh`.

**Fuera de alcance de esta feature, acción manual pendiente:** proyectos
ya scaffoldeados con el bug (`CO-Backend` incluido) necesitan
`chmod +x .claude/hooks/*.sh` a mano — el fix solo corrige corridas
futuras de `april init`.

## 2026-08-31 (continuación) — Fix de dos bugs reales de `.gitignore`: features 17, 18 (done)

El humano pegó un log real de CI de GitHub Actions mostrando que
`TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons`
(feature 14, ticket 01) fallaba con `no such file or directory` sobre
`specs/<name>/spec.md` para las 6 features `sdd:true` `done`. Investigando
la causa raíz se encontraron dos bugs hermanos en el `.gitignore` de la
**raíz** de este repo (no confundir con `templates/.gitignore`, que sí es
correcto tal cual está):

### Feature 17 `gitignore_root_tracks_specs` (done, 31/08/2026)

Causa raíz: la línea `specs/` del `.gitignore` de la raíz era una copia
literal de `templates/.gitignore` (correcta ahí — un proyecto
scaffoldeado nuevo trata sus specs como estado de trabajo descartable),
pero incorrecta en este propio repo, donde `specs/*.md` es la
documentación SDD real que `CLAUDE.md` exige
(`require_approved_spec_to_implement`). Confirmado con `git ls-files
specs/` (vacío): ninguna de las 7 specs existentes (6 de features `done`
+ `spec_gwt_mechanical_check` de esta sesión) estuvo nunca en git — de
ahí la falla en CI (checkout limpio sin esos archivos) que no se veía en
local (archivos presentes en el filesystem de cada desarrollador). Fix:
se sacó la línea `specs/` del `.gitignore` de la raíz (`templates/.gitignore`
sin tocar); verificado con `git check-ignore -v` contra las 7 specs
reales, ninguna matchea ya. Nota nueva en `docs/conventions.md`
("`templates/.gitignore` vs `.gitignore` de la raíz — no son el mismo
target"). Revisión: `APPROVED` en la primera pasada.

### Feature 18 `gitignore_root_tracks_docs` (done, 31/08/2026)

Hallazgo hermano, encontrado por `reviewer_agent` al revisar la feature
17: `/docs/` también estaba en el `.gitignore` de la raíz — pero a
diferencia del bug de `specs/`, esta regla fue agregada **a propósito**
en el commit `3c24d6b` (24/08/2026) bajo el razonamiento "`docs/` es
estado de trabajo... igual que `/feature_list.json`". El humano
determinó que ese razonamiento no se sostiene: `docs/conventions.md`/
`docs/verification.md` son documentación acumulada citada repetidamente
como fuente de verdad vinculante (no estado operativo descartable como
`feature_list.json`) — misma naturaleza que `specs/*.md`. Verificado con
`git log --diff-filter=D -- "docs/*.md"`: esos archivos SÍ estuvieron
trackeados hasta el 24/08, cuando se sacaron del tracking junto con
agregar la regla. Fix: mismo tratamiento que la feature 17, con nota
"hermana" en `docs/conventions.md` documentando que la causa esta vez
fue un razonamiento que dejó de aplicar, no un copy-paste. Revisión:
`APPROVED` en la primera pasada.

**Pendiente del humano tras ambas features (ninguna la ejecuta un
agente, regla dura de este repo):** `git add specs/ docs/ && git commit`
para que las specs y la documentación por fin queden versionadas y CI
deje de fallar.

## 2026-08-31 — Segunda comparación contra `gentle-ai` (C1-C13) + features 13, 14, 16 (done)

Segunda comparación independiente contra `/home/avalor/Proyectos/gentle-ai`
(dos análisis sin contexto compartido) sumó 13 candidatos de metodología
(C1-C13) a `ROADMAP.md`, atacados uno a uno con el humano. Resueltos
todos:

- **C1** → partido en features 13 y 14 (detalle abajo).
- **C2** → adoptado como disciplina de test de caracterización (no el
  shadow-flag de runtime de gentle-ai, que no traduce a un CLI
  recompilado) — `docs/conventions.md`, criterio de acceptance de la
  feature 14.
- **C3** → convención + feature 15 (`blocked_reasons_remedy_commands`,
  `pending`, sin ejecutar).
- **C4** → disciplina anti-sobre-ingeniería, `CLAUDE.md`.
- **C5** → corrección factual (ningún commit real del repo llevaba
  `Co-Authored-By`, contra lo que afirmaba `ROADMAP.md` sin verificar) +
  regla dura nueva: ningún agente ejecuta `git commit`, solo el humano.
- **C6** → `reviewer_agent` persiste su "Formato de salida" en
  `specs/<name>/verify-report.md` (solo `sdd: true`).
- **C7** → límite de rollback en el reporte de `agent_developer`.
- **C8, C10, C11** → documentados en `CLAUDE.md` ("Mecanismos
  incorporados de April"), no en `docs/*.md` de la raíz — corregido un
  desvío real detectado por el humano a mitad de sesión: esos tres
  describen mecanismos que todo proyecto scaffoldeado hereda (ledger,
  backups, `.claude/agents/*.md`), y `docs/*.md` de la raíz no está en
  el `go:embed` de `scaffold.go`, así que nunca habría propagado. C2/C12
  sí quedaron en `docs/conventions.md` (desarrollo del código Go de
  April, nada que un proyecto scaffoldeado herede).
- **C9** → descartado: los tests existentes (`TestCmdInitScaffoldsEmptyDir`)
  ya cubren el mecanismo de embed que un golden file protegería; el
  contenido vivo (`CLAUDE.md`) cambia demasiado seguido para que valga
  la pena congelarlo.
- **C12** → documentado sin tocar código: `doctor.go` ya tenía el patrón
  de ratchet genérico (`Metrics map[string]int`, `evaluateDebtRatchet`
  puro) — la premisa de "bespoke" no se sostuvo.
- **C13** → descartado: su premisa (guiar el commit de `agent_developer`
  por ticket) quedó contradicha por C5 en la misma sesión.

**Hallazgo colateral, corregido:** verificando el `go:embed` para
C8/C10/C11 se encontró que `.claude/verify-ledger.jsonl` (trackeado en
git) no tenía guard de exclusión en `scaffold.go` — `april init` sobre
un proyecto nuevo hubiera copiado el historial real de verify/review de
este repo. Feature 16 lo corrigió (detalle abajo).

### Feature 13 `spec_template_gwt_rfc2119` (done, 31/08/2026)

Edición directa del orquestador en `.claude/agents/spec_writer.md`
(mismo precedente que la feature 3: config/documentación de agente, sin
`agent_developer`/`reviewer_agent`, aprobado por el humano contra el
diff real). Agregó bloque Given/When/Then de ejemplo junto al formato
Como/quiero/para en `## User Stories` (con la aclaración de que no se
fuerza en historias sin rama verificable), e instrucción RFC 2119
(MUST/SHOULD/MAY, con definición de qué rompe el acceptance) en
`## Implementation Decisions`. Sin ledger — mismo patrón sin evidencia
que la feature 3 (edición directa sin agente).

### Feature 14 `spec_gwt_mechanical_check` (done, 31/08/2026)

`computeBlockedReasons` (`status.go`) gana el chequeo `no_gwt_coverage`:
para una feature `sdd:true` con spec existente, sin tickets todavía y
`status != done`, exige al menos un bloque Given/When/Then (líneas que
empiecen con `Given`/`When`/`Then`) o el marcador de opt-out
`<!-- gwt: no aplica -->`. `doctor.go` hereda la señal sin código propio
(ya copia `BlockedReasons` de `computeStatus(nil)`). 2 tickets: 01 (test
de caracterización de las 6 features `sdd:true` `done` reales, sobre un
fixture `fstest.MapFS` aislado por feature para evitar contaminación
cruzada de `blockedReasons` global) y 02 (el chequeo real, TDD
rojo→verde, 8 tests nuevos). Ajustó el fixture — no la aserción — de un
test preexistente (`TestFeatureConSpecYSinTicketsEsFaseTickets`) para
eliminar un confound no relacionado; `reviewer_agent` lo verificó
reproduciendo el fallo real antes de aprobar. Revisión: `APPROVED` en la
primera pasada sobre la feature completa (no ticket por ticket). 217
tests.

### Feature 16 `scaffold_excludes_verify_ledger` (done, 31/08/2026)

Corrige el hallazgo colateral de la comparación contra gentle-ai: guard
nuevo en `scaffold.go` (junto al ya existente para
`.claude/manifest.json`) que excluye `.claude/verify-ledger.jsonl`
(constante `verifyLedgerPath` de `verify.go`) de lo que `april init`
propaga a un proyecto nuevo. Test `TestVerifyLedgerEmbebidoNuncaSePropaga`,
análogo a `TestManifestJsonEmbebidoNuncaSePropaga`. Revisión: `APPROVED`
en la primera pasada.

### Lección de proceso — ledger no registrado antes de cerrar (31/08/2026)

Las features 13 y 16 se marcaron `done` sin correr `april verify
record`/`april review record` — se notó recién al cerrar la feature 14
(primera vez en la sesión que se siguió el gate de cierre completo hasta
el final). Para la 13 no aplica (edición directa, mismo patrón sin
ledger que la feature 3, previa a que existiera el mecanismo). Para la
16 sí era un gap real — se backfilleó con el `go test`/veredicto reales
de esta sesión. Regla adoptada: registrar en el ledger real (binario
reconstruido, no el de PATH) inmediatamente después del veredicto de
`reviewer_agent`, antes de mover `feature_list.json` a `done`.

## 2026-08-28 — Sesión continuación (features 8-12, todas `done`, backlog cerrado)

Continuación directa de la sesión anterior. Cerró las 5 features restantes
del backlog derivado de `ROADMAP.md` — con esto, `feature_list.json` queda
completo (1-12, todas `done`) y `april status --json` reporta `phase:
closed`, `nextRecommended: "nada — no hay features pendientes"`.

### Feature 8 `review_depth_by_diff_sensitivity` (done, 28/08/2026)

`review start` gana flag `--json` opcional: reporta `touchedPaths`/
`sensitiveAreasTouched`/`extraReviewRequired`, reusando el `subject_hash`
de la feature 7 (`git diff --name-only` contra el árbol base). Sin
`--json`, comportamiento byte-a-byte idéntico a la feature 7. 3 tickets
(parseo de "Áreas sensibles" de `docs/conventions.md` / `computeTouchedPaths`
vía diff de árbol congelado / ensamblaje `--json`). Revisión: `APPROVED` en
la primera pasada. Hallazgo de proceso no bloqueante: `go build ./...`
reescribe el binario `HarnessInit` en la raíz del repo, invalidando el
ledger si se corre después de `verify record` — resuelto después por la
feature 12. 165 tests.

### Feature 12 `tree_hash_respects_gitignore` (done, 28/08/2026)

Corrige el hallazgo de la feature 8: `hashTree` ahora respeta `.gitignore`
(parser puro `parseGitignore`/`gitignoreMatches`/`loadGitignorePatterns` en
`verify.go`), evitando que un `go build ./...` que regenera binarios
gitignoreados invalide receipts vigentes. `computeSubjectHash` ya respetaba
`.gitignore` nativamente (vía `git add -A`) — se confirmó con un test de
regresión y se compartió la lista fija de exclusiones (`fixedTreeExclusions`)
entre ambos mecanismos. Se descartó explícitamente unificar `hashTree`/
`computeSubjectHash` en un solo mecanismo. 3 tickets. Revisión: `APPROVED`
en la primera pasada, con prueba de fuego en vivo (`go build ./...` repetido
no invalida el ledger). 180 tests.

### Feature 9 `doctor_readonly_check` (done, 28/08/2026)

`april doctor`: compara `.claude/manifest.json` contra disco (drift
`missing`/`modified`), lista agentes en `.claude/agents/`, consulta
`computeStatus` — 100% read-only. Revisión: `APPROVED_WITH_OBJECTION` — el
chequeo de agentes (`strings.Contains("#")`) diverge de `init.sh`
(`grep -q "^#"`, anclado a inicio de línea); el humano decidió cerrar con
la objeción documentada (edge-case improbable: los agentes reales del
repo ya empiezan con `#`) en vez de pedir el fix. Queda como mejora futura
si aparece un caso real. 186 tests.

### Feature 10 `init_backup_before_apply` (done, 28/08/2026)

`applyPlan` hace backup en `.claude/backups/<timestamp>/` de todo lo que
va a tocar (create/update/delete) antes de escribir nada. Rollback MANUAL
por diseño, documentado explícitamente. Revisión en dos rondas: 1ra
`APPROVED_WITH_OBJECTION` (faltaba test explícito de I/O real para
`actionCreate` sobre un archivo ya existente en disco); el humano pidió
cerrar el gap — se agregó el test, confirmando que era solo falta de
cobertura, no un bug real; 2da ronda `APPROVED` sin objeciones. 202 tests.

### Feature 11 `doctor_debt_ratchet` (done, 28/08/2026)

Extiende `april doctor` con un ratchet de deuda: métrica de TODOs sin
feature asociada (`CHECKPOINTS.md` C3), tokenizada con `go/scanner`
(distingue `COMMENT` de `STRING` — evitó falsos positivos reales del
propio repo, incluidos fixtures de test que contienen el texto literal
`// TODO`). Baseline persistido en `.claude/doctor-baseline.json`
(gitignoreado, excluido del hash de árbol vía la feature 12 sin tocar
código), congelado solo mediante flag explícito `--freeze-baseline` que
se niega a sobreescribir — preserva el contrato read-only de la feature 9
por defecto. Revisión en dos rondas: 1ra `APPROVED_WITH_OBJECTION` (bloque
de test muerto sin aserción + falta de test aislado para el mecanismo
comentario-vs-string); el humano pidió cerrar ambas antes de aprobar —
cerradas sin tocar código de producción; 2da ronda `APPROVED` sin
objeciones. 207 tests.

### Estado al cierre de esta sesión

Backlog completo (features 1-12) en `done`. `april status --json`:
`phase: closed`. Nada commiteado a git todavía — decisión explícita del
humano de manejar el commit él mismo en el siguiente paso, no bloqueante
para el estado del backlog.

## 2026-08-26/28 — Sesión "árbitro `april status`" (features 1-7, todas `done`)

Backlog derivado de `ROADMAP.md` (comparación April vs. gentle-ai, E0-E6):
`planner_agent` propuso 10 features atómicas (ids 2-11), el humano confirmó
`sdd` para cada una y se escribió `feature_list.json` con las 11 (1 más
2-11). Esta sesión cerró la 1 a la 7. Decisión de fondo tomada el
26/08/2026 ("B llegando por A", `ROADMAP.md`): `april status`/`CLAUDE.md`
(features 2/3) operan primero en modo **advisory** — informan pero no
escriben `feature_list.json` — hasta que el humano confirme en uso real
que el modelo de fases es confiable; recién ahí se activa `set-status`
(feature 4) como única vía de escritura.

### Feature 1 `bootstrap_project` (done, 26/08/2026)

Fase Grill vía skill `grill-docs`. Decisiones fijadas en
`docs/architecture.md`/`docs/conventions.md`: monolito plano (`package
main`, sin `internal/`), sin dependencias externas (cualquier necesidad de
librería de terceros se resuelve invocando el binario), nombres de test en
español para casos de negocio, escritura atómica obligatoria para estado
crítico, áreas sensibles = `scaffold.go` + `init.sh` +
`.github/workflows/`. `progress/project-definition.md`,
`docs/verification.md`, `docs/specs.md` completados. `./init.sh` verde.

### Feature 2 `april_status_arbiter` (done, 27/08/2026)

`april status [id] --json` — árbitro que deriva `phase`/`nextRecommended`/
`blockedReasons`/`frontier`/`artifactPaths` leyendo disco
(`feature_list.json`, `specs/*/spec.md`, `specs/*/tickets/*.md`), modo
advisory (nunca escribe). Seam: `computeStatusFromFS(fsys fs.FS, targetID
*int)`, mismo patrón pure-function-over-`fs.FS` que `planScaffoldFromFS`
de `scaffold.go`. Absorbió las 4 validaciones del heredoc Python de
`init.sh` (ticket 03) más detección de ciclos en `Blocked by` (DFS,
ticket 02). 3 tickets (núcleo/frontier-ciclos/init.sh). Revisión: 1ra
ronda `CHANGES_REQUESTED` (4 huecos de trazabilidad de test, no bugs), 2da
`APPROVED_WITH_OBJECTION` (nextRecommended sin aserción de contenido para
3 fases), humano pidió cerrar la objeción, 3ra `APPROVED`. 53 tests.

### Feature 3 `claude_md_routes_by_status` (done, 27/08/2026)

Edición directa de `CLAUDE.md` por el orquestador (excepción aprobada por
el humano, sin `agent_developer`/`reviewer_agent`): paso 4 del ciclo por
sesión pasa a exigir correr `april status --json` y enrutar solo por
`nextRecommended`/`blockedReasons`, con prohibición explícita de inferir
la fase leyendo prosa. Diff acotado, revisado y aprobado por el humano.

### Feature 4 `set_status_authoritative_write` (done, 27/08/2026)

`april feature set-status <id> <estado>` — única vía de escritura válida
de `feature_list.json` de ahí en más. Grafo `pending → spec_ready →
in_progress → done (+blocked)` (`spec_ready` solo alcanzable para
`sdd:true`). Primer patrón de escritura atómica del repo
(`writeFileAtomic`, temp+rename). Mecanismo interino de veredicto (el
ledger real todavía no existía): flag `--verdict` en `set-status done`,
guardado como campo `reviewVerdict` en `feature_list.json` — destinado a
convivir con, no ser reemplazado por, el ledger de las features 5/6.
Revisión: `APPROVED_WITH_OBJECTION` (reordena alfabéticamente las claves
de la feature tocada al serializar — decodifica a
`map[string]interface{}`; ensucia el diff pero el dato es correcto).
Humano decidió cerrar aceptando la objeción tal cual. 74 tests.

### Feature 5 `verify_record_ledger` (done, 27/08/2026)

`april verify record --feature <id> -- <comando>` — corre el comando,
captura exit code/stdout/stderr/hash del árbol, anexa (append-only,
`writeFileAtomic`) a `.claude/verify-ledger.jsonl` (JSON Lines). `april
status` extiende `computeBlockedReasons` con `no_test_evidence`. Durante
la Fase Spec se encontró y resolvió una ambigüedad real: hashear el árbol
completo sin exclusiones auto-invalida el receipt, porque la bitácora
obligatoria en `progress/` que cada subagente escribe al terminar cambia
el árbol en el mismo ciclo. Exclusiones fijas confirmadas por el humano:
`.git/`, el propio ledger, `progress/`. 4 tickets (`hashTree` puro /
esquema+append del ledger / comando completo con subproceso real / lectura
en `status.go`). Revisión: `CHANGES_REQUESTED` dos veces — primera por
`gofmt` + falta de receipt real sobre el árbol; segunda por secuencia (el
receipt se grabó antes de marcar los tickets `done` en `specs/`, que
cuenta para el hash). Lección de proceso adoptada: el receipt final se
graba *después* de que el orquestador termine de tocar `specs/`. 137
tests. Hallazgo colateral no bloqueante: `hashTree` no respeta
`.gitignore` (un binario compilado con `go build ./...` sin `-o` dentro
del repo puede invalidar un receipt vigente).

### Feature 6 `review_verdict_recorded` (done, 28/08/2026)

Extiende el mismo ledger con `kind: "review"`. `april review record
--feature <id> --verdict <valor>` (sin subproceso — el veredicto ya viene
decidido, solo se persiste; `CHANGES_REQUESTED` se registra con éxito,
exit 0, registrar un rechazo no es un fallo). `april status` extiende
`computeBlockedReasons` con `no_review_verdict` (mismo patrón de 3 casos
que `no_test_evidence`). Decisión de diseño resuelta sin bloquear
(siguiendo precedente de la feature 5): no toca `set_status.go` — el
mecanismo interino de la feature 4 sigue igual, coexistiendo
deliberadamente con el ledger. 2 tickets. Revisión: `APPROVED` en la
primera pasada (tras un intento fallido por timeout de infraestructura,
sin rastro). `reviewer_agent` registró su propio veredicto con el comando
recién construido (dogfooding). 118 tests.

### Feature 7 `review_frozen_candidate` (done, 28/08/2026)

`april review start --feature <id>` calcula un candidato congelado real
de git: `git write-tree` sobre un **índice temporal** (`GIT_INDEX_FILE`,
nunca el índice real del usuario), con `git add -A` + `git rm --cached`
excluyendo las mismas rutas que `hashTree` (ledger, `progress/`) —
confiar solo en `.gitignore` no alcanza (ni este repo ni
`templates/.gitignore` excluyen esas rutas). `review record` gana
`--subject-hash <hash>` **opcional**: si se pasa y quedó obsoleto,
rechaza explícito ("stale subject_hash") sin tocar el ledger; sin el
flag, cero cambio de comportamiento (11 tests de la feature 6 intactos).
Primera dependencia dura de `git` del repo, acotada a `review
start`/`--subject-hash`. Decisiones confirmadas por el humano: flag
opcional (no obligatorio, no automático vía ledger), y falla explícita
sin fallback a `hashTree` fuera de un repo git. 3 tickets
(`computeSubjectHash` building-block / `review start` / `review record
--subject-hash`). Revisión: `APPROVED` en la primera pasada, con
dogfooding completo (`review start` → `review record --subject-hash`).
152 tests.

### Incidente — pérdida y reconstrucción de `progress/current.md` (28/08/2026)

Durante la Fase Spec/Tickets de la feature 6, `progress/current.md`
perdió su encabezado y ~600 líneas de historial (features 1-5). Causa:
`spec_writer`/`ticket_writer` solo tienen la herramienta `Write` (no
`Edit`), así que "agregar un bullet" implica leer el archivo entero y
reescribirlo completo — en algún punto de ese ciclo se perdió contenido
previo. Reconstruido por el orquestador desde el contexto de la
conversación (fiel en sustancia, no byte-exacto). Riesgo a vigilar:
agentes con solo `Write` en tareas largas que tocan archivos de bitácora
compartidos.

### Configuración — hook de git endurecido (28/08/2026)

A raíz de un incidente reportado en otro proyecto (un subagente
`agent_developer` corrió `git commit` sin autorización porque ni su
definición ni el hook de ese proyecto lo prohibían — los hooks de
guardrail típicos solo cubren comandos destructivos como `push
--force`/`reset --hard`, no `commit`, que es local y reversible), se
agregó `git commit` a `.claude/hooks/block-dangerous-git.sh` de este
repo. Confirmado que el hook intercepta la llamada Bash igual para el
hilo principal y para subagentes.

## 2026-08-25 — Feature 2 `scaffold_manifest_sync` (done)

`/improve-codebase-architecture` detectó que `april init` sobreescribía
siempre `feature_list.json`/`progress/*.md` en un segundo `init` (solo
`.claude/agents/` se limpiaba con `RemoveAll`, todo lo demás se pisaba sin
condición). Se diseñó vía plan mode un manifiesto `.claude/manifest.json`
(sha256 por archivo, patrón "last-applied-configuration" tipo `kubectl
apply`) que distingue archivo por archivo: nuevo → crea; no tocado por el
usuario → actualiza con la plantilla nueva; tocado y plantilla sin cambios →
deja intacto en silencio; tocado y plantilla también cambió (conflicto
real) → deja intacto y avisa; obsoleto en la plantilla nueva → borra solo
si el usuario no lo tocó. Manifiesto ausente o corrupto → modo adopción (no
toca nada existente, solo adopta hashes de línea base).

`classifyExistingEntries`/`isExistingHarness`/`agentDirToClean` eliminados
por completo — la limpieza de agentes ya no es un caso especial, es una
instancia más de la regla general. `feature_list.json`/`progress/*.md`
quedan protegidos sin ningún `if relPath == "..."` hardcodeado: en cuanto
divergen del hash registrado, la regla general los protege sola.

`agent_developer` implementó en 6 pasos incrementales (`go build`/`go test`
verde en cada uno). `reviewer_agent` primera pasada: `APPROVED_WITH_OBJECTION`
(el criterio de "sobrevive sin caso especial" solo tenía test para
`feature_list.json` en raíz, no para `progress/*.md` en subdirectorio).
Corregido con `TestProgressHistoryMdProtegidoEnConflictoRealSinCasoEspecial`.
Segunda pasada: `APPROVED` sin objeciones. Humano aprobó cierre.

Archivos tocados: `scaffold.go`, `scaffold_test.go`, `.gitignore` (raíz,
excluye `/.claude/manifest.json` para no propagar el manifiesto del propio
repo vía dogfooding).
