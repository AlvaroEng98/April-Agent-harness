# Current Session

## Feature in progress

- Ninguna. Features 1-8 y 12 cerradas `done`. `april status --json` recomienda
  `feature 9` (`doctor_readonly_check`), `sdd: false`, `status: pending` —
  frontera de implementación.

## Plan

Sesión anterior (26-28/08/2026) cerró las features 1 a 7 del backlog
derivado de `ROADMAP.md` — ver `progress/history.md` para el detalle
completo de cada una (decisiones de diseño, objeciones de revisión,
lecciones de proceso) y `session-handoff.md` para el resumen ejecutivo y
los riesgos abiertos. No repetir esa historia acá — este archivo arranca
limpio para la sesión en curso.

## Progress Log

- `spec_writer`: redactada `specs/review_depth_by_diff_sensitivity/spec.md`
  para la feature 8 (`review_depth_by_diff_sensitivity`). Explorado
  `review.go`/`review_test.go` (feature 7, `computeSubjectHash`/
  `runReviewStart`), `status.go` (patrón pure-function + parseo de
  markdown con regex), `docs/conventions.md` (sección "Áreas sensibles",
  líneas 90-106, ya confirmada) y `ROADMAP.md` E5. Diseño: `review start`
  gana un flag `--json` opcional que reporta `touchedPaths`/
  `sensitiveAreasTouched`/`extraReviewRequired`, calculado reusando el
  `subject_hash` ya congelado por la feature 7 (`git diff --name-only
  HEAD subject_hash`, sin mecanismo de diff paralelo); el modo sin
  `--json` queda byte-a-byte idéntico al de la feature 7. Pendiente:
  aprobación humana de la spec antes de pasar a Tickets.
- `ticket_writer`: desglosada la spec aprobada de la feature 8
  (`review_depth_by_diff_sensitivity`) en 3 tickets tracer-bullet,
  publicados en `specs/review_depth_by_diff_sensitivity/tickets/`
  (`01-sensitive-areas-parseo-docs-conventions.md`,
  `02-computeTouchedPaths-diff-arbol-congelado.md`,
  `03-review-start-json-sensibilidad.md`), siguiendo el formato/
  granularidad de `specs/review_frozen_candidate/tickets/` (feature 7):
  01 y 02 son building blocks puros sin bloqueadores (parseo de "Áreas
  sensibles" y diff de árbol vía `runGit`, respectivamente); 03 ensambla
  ambos detrás de `review start --json` y depende de los dos. Pendiente:
  aprobación humana del desglose (granularidad y blocking edges) antes
  de darlo por cerrado — ver mensaje al humano en esta misma entrega.
- `agent_developer`: implementado el ticket 01
  (`01-sensitive-areas-parseo-docs-conventions.md`) de la feature 8
  (`review_depth_by_diff_sensitivity`). Agregadas `parseSensitiveAreas`/
  `readSensitiveAreas` a `review.go` (patrón de dos capas de
  `status.go`: función pura sobre `string` + wrapper de I/O sobre
  `fs.FS`), replicando casi literal el código de
  `specs/review_depth_by_diff_sensitivity/spec.md` (sección "Áreas
  sensibles"). Tests nuevos en `review_test.go`: 4 puros sobre literales
  de `string` (incluye el contenido literal real de `docs/conventions.md`,
  líneas 90-106) y 2 sobre `fstest.MapFS` — los 6 puntos del `acceptance`
  del ticket. Comandos: `go build ./...` (limpio), `go test ./...` (ok,
  0.626s), `go vet ./...` (limpio). No se tocó `computeTouchedPaths`/
  `baseTreeForDiff` (ticket 02) ni el cableado de `--json` en
  `runReviewStart` (ticket 03), ni `feature_list.json`/`Status` de otros
  tickets.
- `agent_developer`: implementado el ticket 02
  (`02-computeTouchedPaths-diff-arbol-congelado.md`) de la feature 8
  (`review_depth_by_diff_sensitivity`). Agregadas en `review.go` (línea
  ~330 en adelante, junto a `computeSubjectHash`): constante
  `gitEmptyTreeHash` (SHA-1 fijo del árbol vacío de git), `baseTreeForDiff()`
  (corre `git rev-parse --verify -q HEAD^{tree}` vía `runGit`; sin commits
  previos devuelve `gitEmptyTreeHash` sin propagar error) y
  `computeTouchedPaths(subjectHash string) ([]string, error)` (corre
  `git diff --name-only <base> <subjectHash>` vía el mismo `runGit`, filtra
  `verifyLedgerPath` y rutas bajo `progress/`, normaliza a `[]string{}`
  nunca `nil`). En `review_test.go`: nuevo helper
  `gitRepoWithCommitTestDir(t, files)` (extiende `gitRepoTestDir`, hace un
  commit real con autor fijo inline `-c user.email=test@april.dev -c
  user.name=test`, sin tocar `git config --global`) y 7 tests nuevos con
  git real: 2 para `baseTreeForDiff` (con/sin commits) y 5 para
  `computeTouchedPaths` (archivo modificado, sin cambios, sin commits
  previos, exclusión de ledger/progress/, archivo nuevo no trackeado) — los
  9 puntos de `acceptance` del ticket cubiertos. Comandos corridos: `go
  build ./...` (limpio), `go vet ./...` (limpio), `go test ./...` (ok,
  0.927s, incluye los tests del ticket 01 sin modificarlos y en verde). No
  se tocó el parseo de "Áreas sensibles" (ticket 01, ya mergeado) ni el
  cableado de `--json`/`matchSensitiveAreas`/`reviewStartReport` en
  `runReviewStart` (ticket 03, bloqueado por este), ni `feature_list.json`/
  `Status` de otros tickets.
- `agent_developer`: implementado el ticket 03
  (`03-review-start-json-sensibilidad.md`, ensamblaje final) de la feature 8
  (`review_depth_by_diff_sensitivity`). Agregadas en `review.go`:
  `matchSensitiveAreas(touched, sensitive []string) []string` (cruce
  prefijo-de-directorio si el área termina en `/`, exacto si no,
  `[]string{}` nunca `nil`), struct `reviewStartReport` (camelCase:
  `subjectHash`/`touchedPaths`/`sensitiveAreasTouched`/
  `extraReviewRequired`), y extendido `runReviewStart(args []string) int`
  para interpretar `args[2:]`: vacío → camino idéntico a la feature 7 (no
  llama a `computeTouchedPaths`/`readSensitiveAreas`); exactamente
  `["--json"]` → arma el reporte completo con
  `json.MarshalIndent(report, "", "  ")` vía `fmt.Println`; cualquier otra
  cosa → error de invocación explícito, exit 1. `main.go`: `printUsage()`
  actualizado con `review start --feature <id> [--json]`. Tests nuevos en
  `review_test.go`: 4 puros para `matchSensitiveAreas` y 9 de integración
  para `runReviewStart` extendido (sin `--json` idéntico a feature 7,
  `--json` con/sin área sensible tocada, sin sección de "Áreas sensibles",
  campos nunca `null`, fuera de repo git, argumentos de más/typo de flag).
  Comandos corridos: `go build ./...` (limpio), `go vet ./...` (limpio),
  `go test ./... -v` (165 tests, 0 failures, 1.047s) — incluye sin
  modificación los 11 tests previos de `runReviewStart`/`recordReview`
  (feature 6/7) y los de los tickets 01/02, todos en verde. No se tocó
  `feature_list.json`, `Status` de tickets, ni
  `.claude/agents/reviewer_agent.md`.
- `reviewer_agent`: revisada la feature 8
  (`review_depth_by_diff_sensitivity`, 3 tickets `done`) contra
  `specs/review_depth_by_diff_sensitivity/spec.md` (24 US) y el
  `acceptance` de `feature_list.json`. Verificado punto por punto en
  `review.go`/`review_test.go`: cada US tiene test citado (camino feliz y
  de error), incluidas las exclusiones `verify-ledger`/`progress/` (US8),
  coincidencia exacta vs. prefijo `/` (US10), campos JSON nunca `null`
  (US14), `extraReviewRequired` sin afectar exit code (US18), y ausencia
  de cambios en `ledgerEntry`/`computeBlockedReasons`/
  `.claude/agents/reviewer_agent.md` (US19/US20) confirmada por
  inspección directa. Corridos de forma independiente: `go build ./...`,
  `go vet ./...`, `go test ./... -v` (165/165 verde), `gofmt -l .` (solo
  señala `config.go`/`config_test.go`, ajenos a esta feature). Hallazgo
  de proceso (no atribuible al código de la feature): `./init.sh` dio
  `no_test_evidence` porque `go build ./...` reescribe el binario
  `HarnessInit` en la raíz del repo (único paquete `main`, sin
  subdirectorios ⇒ `./...` resuelve a un solo paquete ejecutable) y
  `hashTree` no lo excluye — cualquier `go build ./...` corrido después
  de `verify record` invalida la evidencia al instante. Confirmado con un
  re-registro (`verify record --feature 8 -- go test ./...`) sin volver a
  correr `go build ./...` después: `blockedReasons` quedó limpio de
  `no_test_evidence`. Vale la pena que el humano decida si excluir el
  binario de `hashTree` o reordenar `docs/verification.md`, pero no
  bloquea esta feature. **Veredicto: APPROVED** — registrado en el ledger
  con `review record --feature 8 --verdict APPROVED`.
- `planner_agent`: recibido el objetivo de convertir en feature de backlog
  el hallazgo de `reviewer_agent` sobre `isExcludedFromTreeHash`/
  `computeSubjectHash` (auto-invalidación del ledger por `go build ./...`
  reescribiendo binarios gitignorados no excluidos del hash). Explorados
  `verify.go` (`hashTree`/`isExcludedFromTreeHash`), `review.go`
  (`computeSubjectHash`), `.gitignore` y `feature_list.json` (features 5,
  7, 9-11). Propuesta 1 feature atómica nueva (`tree_hash_respects_gitignore`,
  candidata a `id: 12`) al orquestador, con `sdd` explícitamente pendiente
  de decisión humana (recomendación: `true`, por tocar dos mecanismos de
  hash compartidos con superficie de regresión sobre features 5-8 ya
  cerradas).
- `spec_writer`: redactada `specs/tree_hash_respects_gitignore/spec.md`
  para la feature 12 (`tree_hash_respects_gitignore`). Explorados
  `verify.go` (`hashTree`/`isExcludedFromTreeHash`), `review.go`
  (`computeSubjectHash`, `computeTouchedPaths`), `.gitignore`,
  `verify_test.go`/`review_test.go` (tests existentes, ninguno testea
  `isExcludedFromTreeHash` en aislamiento) y las specs previas de la
  misma zona (`specs/verify_record_ledger/spec.md`,
  `specs/review_frozen_candidate/spec.md` — ambas citadas y no
  contradichas, ver "Further Notes" de la spec nueva). Hallazgo clave:
  el bug de auto-invalidación (`go build ./...` invalidando receipts)
  vive solo en `hashTree`/`treeHash`; `computeSubjectHash` ya respeta
  `.gitignore` de forma nativa vía `git add -A` sobre archivos untracked
  — no necesita corrección de comportamiento, solo un test de regresión
  que lo confirme y un refactor menor (lista compartida de exclusiones
  fijas con `isExcludedFromTreeHash`). Diseño: `hashTree(fsys fs.FS)`
  mantiene su firma; `isExcludedFromTreeHash` gana un segundo parámetro
  (`[]gitignorePattern`, sin tests que la llamen en aislamiento, libre de
  romper); parser de `.gitignore` en Go puro (stdlib, sin dependencias
  nuevas — `docs/architecture.md` Principio 2), leído del mismo `fsys`
  que recibe `hashTree`, compatible con `fstest.MapFS`. Se descartó
  explícitamente unificar `hashTree`/`computeSubjectHash` en un solo
  mecanismo (uno necesita ser agnóstico de `fs.FS`/sin git, el otro ya
  delega correctamente en `git` real) y descartó delegar en
  `git check-ignore`/`git ls-files` para `hashTree` (rompería los tests
  existentes sobre `fstest.MapFS` y la garantía "no requiere ser un repo
  git" ya fijada en la feature 5). Pendiente: aprobación humana de la
  spec (y de los seams de test propuestos) antes de pasar a Tickets.
- `ticket_writer`: desglosada la spec aprobada de la feature 12
  (`tree_hash_respects_gitignore`) en 3 tickets tracer-bullet, publicados
  en `specs/tree_hash_respects_gitignore/tickets/`
  (`01-parser-gitignore-puro.md`,
  `02-fixed-tree-exclusions-compartidas.md`,
  `03-hashtree-respeta-gitignore-end-to-end.md`), siguiendo el formato/
  granularidad de `specs/review_depth_by_diff_sensitivity/tickets/`
  (feature 8): 01 (parser puro `parseGitignore`/`gitignoreMatches`/
  `loadGitignorePatterns`) y 02 (`fixedTreeExclusions` compartida en
  `review.go` + test de confirmación sobre `computeSubjectHash`) son
  building blocks independientes entre sí, sin bloqueadores; 03 (wiring
  real de `hashTree`/`isExcludedFromTreeHash` + tests unitarios sobre
  `fstest.MapFS` + integración extremo-a-extremo con
  `recordVerify`/`computeStatus` en ambos órdenes) ensambla ambos y
  depende de los dos. Desglose (granularidad y blocking edges) aprobado
  explícitamente por el humano antes de escribir los archivos.
- `agent_developer`: implementado el ticket 01
  (`01-parser-gitignore-puro.md`) de la feature 12
  (`tree_hash_respects_gitignore`). Agregados a `verify.go` (tras
  `isExcludedFromTreeHash`, `verify.go:100` en adelante): tipo
  `gitignorePattern` (`anchored`/`dirOnly`/`glob`) y las funciones puras
  `parseGitignore(content string) []gitignorePattern`,
  `gitignoreMatches(rel string, patterns []gitignorePattern) bool`,
  `gitignorePatternMatches` (helper interno), y el wrapper de I/O
  `loadGitignorePatterns(fsys fs.FS) ([]gitignorePattern, error)` —
  siguiendo el patrón de dos capas de `parseSensitiveAreas`/
  `readSensitiveAreas` (`review.go`), código casi literal de la sección
  "Implementation Decisions" de la spec. No se tocó `hashTree`/
  `isExcludedFromTreeHash` (siguen sin llamar a este parser — eso es el
  ticket 03) ni `review.go`/`fixedTreeExclusions` (ticket 02). Ciclo
  red→green: escritos primero los 9 tests nuevos en `verify_test.go`
  (confirmado rojo: `go test ./...` fallaba con `undefined: parseGitignore`
  etc.), luego la implementación mínima. Los 9 tests cubren uno a uno los
  puntos del `acceptance` del ticket (clases básicas de patrón campo a
  campo, negación ignorada, anclaje a raíz, sin ancla a cualquier
  profundidad, `dirOnly` sobre contenido completo, `/` intermedio ancla,
  `loadGitignorePatterns` sin archivo → `nil,nil`, con archivo → mismos
  patrones que `parseGitignore`). Comandos corridos: `go build ./...`
  (limpio), `go test ./...` (verde, incluye los 9 tests nuevos y todos los
  previos sin modificar), `go vet ./...` (limpio), `gofmt -l .` (solo
  señala `config.go`/`config_test.go`, estado previo ya conocido de la
  feature 8, ajeno a este ticket). No se tocó `feature_list.json` ni
  `Status` de otros tickets.
- `agent_developer`: implementado el ticket 02
  (`02-fixed-tree-exclusions-compartidas.md`) de la feature 12
  (`tree_hash_respects_gitignore`). En `review.go`: nueva variable de
  paquete `fixedTreeExclusions = []string{verifyLedgerPath, "progress"}`
  (comentada: por qué no dependen de `.gitignore`, por qué `.git/` no
  entra en la lista), y el `git rm --cached -r --ignore-unmatch --` de
  `computeSubjectHash` ahora arma sus argumentos con
  `append([]string{"rm", "--cached", "-r", "--ignore-unmatch", "--"},
  fixedTreeExclusions...)` en vez de los dos literales inline
  (`review.go:59-60`). Cero cambio de mecanismo. Ciclo red→green: escrito
  primero el test nuevo `TestComputeSubjectHashYaRespetaGitignoreParaArchivosNoTrackeados`
  (`review_test.go`, tras `TestComputeSubjectHashExcluyeLedgerYProgress`)
  con `gitRepoTestDir` — repo git real, `.gitignore` con `/HarnessInit`,
  escribe `HarnessInit` sin `git add`, calcula `computeSubjectHash()`,
  sobrescribe con contenido distinto, recalcula, verifica mismo hash —
  corrido en verde ANTES del refactor (confirma, no corrige, que
  `computeSubjectHash` ya respetaba `.gitignore` vía `git add -A` nativo)
  y de nuevo después del refactor de `fixedTreeExclusions`, sigue verde.
  No se tocó `isExcludedFromTreeHash`/`hashTree` (ticket 03, bloqueado por
  este y por el 01) ni el parser de `.gitignore` del ticket 01, ni
  `feature_list.json`/`Status` de otros tickets. Comandos corridos: `go
  build ./...` (limpio), `go test ./...` (verde), `go test ./... -run
  TestComputeSubjectHash -v` (8/8 tests en verde, incluidos los 6
  existentes sin editar), `go vet ./...` (limpio).
- `agent_developer`: implementado el ticket 03
  (`03-hashtree-respeta-gitignore-end-to-end.md`, ensamblaje final) de la
  feature 12 (`tree_hash_respects_gitignore`). En `verify.go`: `hashTree`
  ahora llama a `loadGitignorePatterns(fsys)` una sola vez antes de
  `fs.WalkDir` (`verify.go:44`); `isExcludedFromTreeHash` gana el segundo
  parámetro `patterns []gitignorePattern` (`verify.go:96`) y referencia
  `fixedTreeExclusions[0]`/`[1]` (ticket 02, `review.go`) en vez de los
  literales `verifyLedgerPath`/`"progress"` sueltos, con `.git/` incondicional
  antes y `gitignoreMatches` como último chequeo. Los tres tests existentes
  de `hashTree` (`TestHashTreeExcluyeGitProgressYElPropioLedger`,
  `TestHashTreeCambiaSiUnArchivoNoExcluidoCambia`,
  `TestHashTreeEsDeterministicoSinImportarOrden`) no se tocaron y siguen en
  verde (US10). Tests nuevos en `verify_test.go`: 3 unitarios sobre
  `fstest.MapFS` (`TestHashTreeExcluyeArchivoGitignoreadoAunSinListaFija` —
  reproduce el bug real de `/HarnessInit` regenerado por un rebuild;
  `TestHashTreeSigueExcluyendoLasTresFijasAunNoEstenEnGitignore`;
  `TestHashTreeArchivoNoGitignoreadoSigueCambiandoElHashConGitignorePresente`)
  y 3 de integración extremo a extremo sobre disco real (`chdirTemp` +
  fixture con `feature_list.json` in_progress/sdd:false + `.gitignore` con
  `/build-artifact`): `TestRecordVerifyLuegoRegenerarArtefactoGitignoreadoNoProduceNoTestEvidence`,
  `TestRegenerarArtefactoGitignoreadoLuegoRecordVerifyNoProduceNoTestEvidence`
  (orden inverso) y el control
  `TestModificarArchivoNoGitignoreadoLuegoRecordVerifySigueDetectandoNoTestEvidence`
  (`recordVerify` + `computeStatus`, verificando `blockedReasons` vía
  `strings.Contains` sobre `no_test_evidence`). Comandos corridos: `go build
  ./...` (limpio), `go vet ./...` (limpio), `go test ./... -v` (180
  `--- PASS`, 0 `FAIL`, `ok ... 1.095s`) — incluye sin modificación todos los
  tests previos de las features 2-8 y de los tickets 01/02 de esta feature,
  todos en verde. `gofmt -l` sobre `verify.go`/`verify_test.go`/`review.go`
  sin salida (limpio). No se tocó `feature_list.json` ni `Status` de ningún
  ticket.
- `reviewer_agent`: revisada la feature 12 (`tree_hash_respects_gitignore`),
  veredicto **APPROVED**. Trazabilidad completa: los 7 puntos del
  `acceptance` (`feature_list.json`) tienen test citado por nombre, incluidos
  ambos órdenes (US13/US14) y el control de no-regresión (US7). Completitud
  confirmada: solo `verify.go`/`verify_test.go`/`review.go`/`review_test.go`
  tocados, igual que lo reportado por `agent_developer` en las tres
  entradas de tickets 01/02/03 de arriba. Verificación independiente:
  `go build ./...`, `go vet ./...` limpios; `go test ./... -v` → 180
  `--- PASS`, 0 `FAIL`; `gofmt -l .` solo señala `config.go`/`config_test.go`
  (estado previo ya conocido de la feature 8, ajeno a esta). Prueba de
  fuego reproducida en vivo sobre el repo real: tras el `verify record`
  de `agent_developer` (`treeHash edf8d225...`), corrí `go build ./...`
  varias veces (incluyendo sobrescribir `/HarnessInit` con contenido
  arbitrario) y `april status 12 --json` nunca reportó `no_test_evidence`
  — sí encontré que el binario `april` ya instalado en
  `~/.local/bin/april` estaba desactualizado (build previo a este fix) y
  daba falso positivo; lo reconstruí desde el fuente actual
  (`go build -o ~/.local/bin/april .`) y confirmé limpio, hallazgo de
  higiene de entorno, no del código. También confirmé por lectura y test
  en vivo que `hashTree`/`computeSubjectHash` siguen siendo consistentes
  en efecto para archivos gitignoreados-pero-trackeados de este propio
  repo (`session-handoff.md`): ninguno de los dos los cuenta, sin
  divergencia entre mecanismos. Mi propio `review record` quedó con el
  mismo `treeHash` (`edf8d225...`) que el `verify record` previo, prueba
  adicional de que el mecanismo no se autoinvalida. Sustancia limpia: sin
  complejidad no pedida, sin unificación de mecanismos (respetada la
  decisión explícita en contra de la spec), sin tests que verifiquen el
  mock en vez del comportamiento observable. Sin objeciones.
- `orquestador`: cerrada la feature 12 (`tree_hash_respects_gitignore`) —
  `april feature set-status 12 done --verdict APPROVED` (única vía
  autoritativa, `set_status.go:validTransition` `in_progress → done` con
  veredicto). Gates verificados: 3 tickets `done`, spec aprobada, `april
  status 12 --json` `phase: review` `blockedReasons: []`, `april status
  --json` global sin bloqueos, 180 tests `go test ./...` verde, `go build`
 /`go vet` limpios, ledger con `treeHash edf8d225...` para `verify` y
  `review` (APPROVED) vigente al momento del cierre. Humano aprobó el
  cierre explícitamente. `april status --json` post-cierre: `phase:
  implementation` recomienda `feature 9` (`doctor_readonly_check`).
