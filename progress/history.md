# Session History

<!-- Append-only log. Most recent at the top. -->

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
