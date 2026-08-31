## Problem Statement

Hoy `require_tests_to_close` se satisface con un párrafo narrado por
`agent_developer` en su reporte a `progress/current.md`: "corrí `go test
./...`, salió verde". Nadie más lo corre de forma independiente. El
orquestador, el humano y `reviewer_agent` confían en la prosa. Un agente
puede narrar un resultado que no corresponde al estado real del árbol de
trabajo — por accidente (probó una versión vieja, olvidó re-correr tras el
último cambio) o porque simplemente no corrió nada. `april status` (feature
2) ya calcula `blockedReasons` a partir de lo que hay en disco, pero no
tiene ninguna señal objetiva de "se corrieron tests y pasaron sobre el
código tal como está ahora mismo" — solo puede confiar en la narración,
exactamente el problema que el `ROADMAP.md` señala como brecha frente a
gentle-ai ("evidencia derivada por el sistema, no narrada").

## Solution

Un comando nuevo, `april verify record --feature <id> -- <comando>`, que
corre `<comando>` como subproceso, captura su exit code, su
salida (stdout/stderr) y un hash determinístico del árbol de trabajo en
ese momento, y **anexa** (nunca sobrescribe) una entrada a un ledger en
`.claude/verify-ledger.jsonl` (JSON Lines: un objeto JSON por línea).
`april status <id> --json` (extendiendo `computeBlockedReasons` de la
feature 2) lee ese ledger: si la feature consultada está `in_progress` y
su último receipt de tipo `test` tiene `exitCode != 0`, no existe, o su
`treeHash` no coincide con el hash del árbol actual, `blockedReasons`
incluye una entrada con la substring `"no_test_evidence"`. Con un receipt
de tipo `test` en verde y vigente (mismo hash), no la incluye.

El ledger es append-only por diseño de escritura (mismo patrón
write-temp-then-rename que ya prescribe `docs/architecture.md` para
"estado crítico", reusando `writeFileAtomic` de `set_status.go`): un
append fallido a mitad de camino nunca deja el archivo a medio escribir ni
pierde entradas previas — o se escribe la entrada completa, o el archivo
queda exactamente como estaba antes.

El campo `kind` del ledger distingue el tipo de entrada (`"test"` para
esta feature); queda reservado para que la feature 6
(`review_verdict_recorded`, todavía `pending`) agregue entradas
`kind: "review"` al mismo archivo en vez de crear un ledger separado — esa
extensión es trabajo de la feature 6, no de esta spec.

## User Stories

1. Como `agent_developer`, quiero correr
   `april verify record --feature <id> -- go test ./...` al terminar de
   implementar, para dejar evidencia registrada de que los tests pasaron
   sobre el código tal como quedó, sin depender de que mi reporte narrado
   sea suficiente.
2. Como orquestador, quiero que `april status <id> --json` me diga
   objetivamente si falta evidencia de tests vigente para la feature
   `in_progress`, para no avanzar a revisión/cierre confiando solo en la
   palabra de `agent_developer`.
3. Como orquestador, quiero que la ausencia total de cualquier receipt
   para la feature `in_progress` reporte `no_test_evidence`, para
   detectar el caso "nadie corrió `verify record` todavía".
4. Como orquestador, quiero que un receipt existente pero con
   `exitCode != 0` (el comando falló) reporte `no_test_evidence`, para no
   confundir "se intentó y falló" con "está verde".
5. Como orquestador, quiero que un receipt con `exitCode == 0` pero cuyo
   `treeHash` no coincide con el árbol actual reporte `no_test_evidence`,
   para detectar el caso "los tests pasaron, pero el código cambió
   después y ya no sé si sigue pasando".
6. Como orquestador, quiero que un receipt con `exitCode == 0` y
   `treeHash` igual al árbol actual NO reporte `no_test_evidence`, para
   poder avanzar cuando la evidencia es real y vigente.
7. Como humano, quiero que dos corridas de `verify record` sobre la misma
   feature produzcan dos entradas distintas en el ledger (nunca se pisa la
   anterior), para conservar el historial completo de intentos, no solo el
   último.
8. Como humano, quiero que si el proceso muere a mitad de un `verify
   record` (kill -9, corte de luz), el ledger existente quede intacto —
   con todas sus entradas previas legibles — en vez de corrupto o
   truncado a la mitad de una línea JSON.
9. Como `agent_developer`, quiero que `verify record` capture stdout y
   stderr del comando corrido, para poder diagnosticar por qué falló un
   test sin tener que volver a correrlo manualmente.
10. Como `agent_developer`, quiero que el exit code de `april verify
    record` en sí mismo refleje el exit code del comando que corrí, para
    poder encadenarlo en un script (`april verify record ... || exit 1`)
    sin parsear el ledger.
11. Como humano, quiero que pedir `verify record` con un comando que no
    existe (typo, binario no instalado) sea un error de invocación claro
    en stderr, distinto de "el comando corrió y falló", y que en ese caso
    NO se escriba ninguna entrada al ledger — un intento que ni siquiera
    arrancó no es evidencia de nada.
12. Como humano, quiero que `verify record` sin `--feature <id>` sea un
    error de invocación explícito, para no terminar con un receipt
    huérfano sin feature asociada.
13. Como humano, quiero que `verify record` sin el separador `--` (o sin
    ningún comando después de él) sea un error de invocación explícito,
    para no correr el binario `april` mismo por accidente ni confundir
    flags propios con el comando a ejecutar.
14. Como desarrollador de April, quiero que el hash del árbol excluya
    `.git/`, para que operaciones normales de git (commit, checkout,
    fetch) no invaliden un receipt vigente sin que el código haya
    cambiado.
15. Como desarrollador de April, quiero que el hash del árbol excluya el
    propio archivo `.claude/verify-ledger.jsonl`, para que el acto mismo
    de anexar una entrada no invalide el `treeHash` que esa entrada acaba
    de registrar.
16. Como agente que escribe su bitácora, quiero que el hash del árbol
    excluya `progress/`, para que escribir mi entrada obligatoria de fin
    de tarea (parte del protocolo de `CLAUDE.md`) no invalide, en el mismo
    ciclo, el receipt que ese mismo agente acaba de grabar.
17. Como orquestador, quiero que editar `feature_list.json`, `docs/` o
    `specs/` (rutas no excluidas explícitamente) SÍ invalide un receipt
    previo, porque esos cambios sí pueden afectar si el código bajo test
    sigue siendo válido — la exclusión es deliberadamente angosta
    (`.git/`, el ledger, `progress/`), no "todo lo que no sea código Go".
18. Como orquestador, quiero que modificar cualquier archivo de código NO
    excluido (`*.go`, `go.mod`, `templates/`, `install.sh`, etc.) después
    de un receipt en verde invalide ese receipt, para que la evidencia
    nunca quede desactualizada respecto al código real.
19. Como humano, quiero que el hash del árbol sea determinístico —mismo
    contenido, mismo hash, sin importar el orden en que el filesystem
    devuelva las entradas—, para poder comparar hashes de corridas
    distintas con confianza.
20. Como orquestador, quiero que `no_test_evidence` solo se evalúe para la
    feature con `status: "in_progress"`, y no para features `pending`,
    `spec_ready`, `blocked` o `done`, para no llenar `blockedReasons` de
    ruido sobre trabajo que ni siquiera arrancó o que ya cerró.
21. Como humano, quiero que una línea corrupta en
    `.claude/verify-ledger.jsonl` (JSON inválido, ej. por edición manual
    accidental) se reporte explícitamente en `blockedReasons` en vez de
    hacer que `april status` falle con un error opaco o ignore el ledger
    entero en silencio.
22. Como desarrollador de April, quiero que el campo `kind` del ledger
    quede desde ya en el esquema (aunque esta feature solo escriba
    `"test"`), para que la feature 6 pueda agregar `"review"` al mismo
    archivo sin migrar el formato.
23. Como desarrollador de April, quiero que `verify.go` siga el mismo
    patrón pure-function-+ wrapper-de-I/O que `status.go`
    (`computeStatusFromFS`/`computeStatus`) y `set_status.go`
    (`computeSetStatus`/`setStatus`), para que el módulo nuevo se sienta
    consistente con el resto del código.
24. Como humano, quiero que `verify record` NO interprete el comando
    después de `--` a través de una shell (sin pipes/redirects/`&&`
    implícitos) — se ejecuta directo vía `exec.Command`, argv a argv —
    para que el comportamiento sea predecible y no dependa de que exista
    `sh` en el sistema; si necesito shell, lo pido explícito
    (`-- sh -c "..."`).
25. Como CI/script, quiero poder invocar `april verify record` con
    cualquier comando arbitrario (no solo `go test`), para reusarlo con
    linters, builds, o cualquier otra verificación que quiera dejar
    registrada.
26. Como humano, quiero que la salida (`stdout`/`stderr`) capturada quede
    dentro de la misma entrada del ledger (no en un archivo aparte), para
    tener todo el contexto de un receipt en un solo lugar consultable.

## Implementation Decisions

**Módulo nuevo:** `verify.go` (+ `verify_test.go`), mapeo ya anticipado en
`docs/architecture.md` ("*(futuro, ROADMAP.md E1-E6)* ... `verify.go`").
Sin dependencias nuevas — `os/exec` (mismo paquete que ya usa
`update.go`), `encoding/json`, `crypto/sha256`, `encoding/hex`, `io/fs`,
`os`, `path/filepath`, `time`, `strings`.

**Esquema del ledger — `.claude/verify-ledger.jsonl`.** JSON Lines: una
línea, un objeto JSON, sin pretty-print (consistente con que el archivo
crece por líneas, no se reformatea entero en cada escritura). Campos por
entrada:

```json
{"kind":"test","featureId":5,"command":["go","test","./..."],"exitCode":0,"treeHash":"3f8a...","timestamp":"2026-08-27T18:04:05Z","stdout":"ok\n","stderr":""}
```

- `kind` (string) — `"test"` para toda entrada que escribe esta feature.
  Reservado para que la feature 6 agregue `"review"` sin cambiar el
  formato ni el archivo.
- `featureId` (int) — el `id` de `feature_list.json` pasado con
  `--feature`.
- `command` ([]string) — el argv completo del comando corrido, ya
  separado por la CLI (no una cadena única con espacios, para no
  reintroducir ambigüedad de quoting al leerlo de vuelta).
- `exitCode` (int) — el exit code real del subproceso.
- `treeHash` (string) — hex de sha256, el hash agregado del árbol de
  trabajo calculado **después** de que el comando terminó de correr
  (captura cualquier efecto lateral del propio comando sobre el árbol,
  ej. archivos generados que no estén en las exclusiones).
- `timestamp` (string) — `time.Now().UTC().Format(time.RFC3339)`.
- `stdout`, `stderr` (string) — capturados por separado (dos buffers, no
  uno combinado), sin límite de tamaño en esta feature (ver Further
  Notes).

**Cálculo del hash del árbol — `hashTree(fsys fs.FS) (string, error)`.**
Extrae a producción el mismo algoritmo que ya usa `hashDirTree` (hoy solo
un test helper en `status_test.go`): recorre todo el árbol bajo `fsys`,
para cada archivo calcula `sha256(contenido)`, arma pares
`ruta-relativa:hash`, los ordena por ruta (determinismo, sin depender del
orden que devuelva el filesystem), concatena con `\n` y calcula
`sha256` del agregado completo. Precedente exacto: `hashContent` en
`scaffold.go`.

Exclusiones (confirmadas con el humano el 27/08/2026, resultado directo de
la discusión sobre auto-invalidación durante la fase de spec de esta
feature):

- Cualquier ruta bajo `.git/` (prefijo) — evita que operaciones normales
  de git invaliden receipts sin que el código haya cambiado.
- `.claude/verify-ledger.jsonl` exacto — el propio ledger, para que
  anexarle una entrada no invalide el hash que esa misma entrada acaba de
  registrar.
- Cualquier ruta bajo `progress/` (prefijo) — la bitácora obligatoria que
  cada subagente escribe al terminar su tarea (`CLAUDE.md`, "Bitácora en
  progress/") no debe invalidar, en el mismo ciclo, el receipt que ese
  mismo agente acaba de grabar.

Nada más queda excluido: `feature_list.json`, `docs/`, `specs/`, y todo el
código fuente cuentan para el hash — cambiarlos invalida un receipt
previo (US17/US18). No hay lista configurable ni flag para ampliar las
exclusiones en esta feature (decisión B rechazada explícitamente, ver
pregunta trasladada al humano).

**Escritura atómica — reuso directo, sin patrón nuevo.**
`docs/architecture.md` ya prescribe write-temp-then-rename para "el
ledger de receipts" como estado crítico. `writeFileAtomic` en
`set_status.go` ya es genérica
(`func writeFileAtomic(path string, data []byte, mode os.FileMode) error`,
no atada a `feature_list.json`) — `verify.go` la reusa tal cual, sin
duplicar. El append se implementa como: leer el contenido existente del
ledger (`os.ReadFile`; archivo inexistente se trata como contenido vacío,
no como error — mismo criterio de "adopción" que `loadManifest` en
`scaffold.go`), concatenar la nueva línea JSON al final, y escribir el
resultado completo con `writeFileAtomic`. Esto evita depender de la
atomicidad de una única syscall `write()` bajo `O_APPEND` (que se rompe
si una entrada individual —por `stdout`/`stderr` grandes— supera
`PIPE_BUF`); con temp+rename, el tamaño de la entrada no importa para la
garantía de atomicidad.

**Orquestación — `recordVerify(featureID int, cmdArgs []string) (entry
ledgerEntry, exitCode int, err error)`.** Corre `exec.Command(cmdArgs[0],
cmdArgs[1:]...)` directo (sin `sh -c`, ver US24), con `Stdout`/`Stderr`
apuntando a dos `bytes.Buffer` (y también a `os.Stdout`/`os.Stderr` vía
`io.MultiWriter`, para que quien corre el comando en terminal vea la
salida en vivo, no solo al final). Distingue dos clases de fallo:

- El comando **no llegó a arrancar** (binario inexistente, permiso
  denegado — `err` no es `*exec.ExitError`): `recordVerify` devuelve error
  de invocación, no escribe ninguna entrada al ledger (US11).
- El comando arrancó y terminó con **exit code != 0**
  (`*exec.ExitError`): esto SÍ es una corrida válida — se registra
  normalmente con ese `exitCode`, no es un error de Go.

Tras correr el comando (exitosa o no, siempre que haya arrancado), calcula
`hashTree(os.DirFS("."))`, arma la entrada, y hace el append atómico
descrito arriba. El exit code del propio proceso `april verify record`
(`runVerifyRecord`/`cmdVerify`) es el mismo que el del comando corrido —
si el comando falló, `verify record` también sale con ese código, para
poder encadenarse en scripts (US10).

**CLI.** Nuevo caso `"verify"` en el `switch` de `main.go` → `cmdVerify()`,
que despacha en `os.Args[2]` (hoy solo `"record"`, error explícito para
cualquier otro subcomando). `runVerifyRecord(args []string) int` parsea
`--feature <id>` (obligatorio, `id` numérico) seguido de `--` (obligatorio)
y el resto de los argumentos como el comando a correr (al menos uno,
obligatorio) — mismo estilo de parseo simple que `runSetStatus`. Falta de
`--feature`, de `--`, `id` no numérico, o comando vacío: error explícito en
stderr, exit distinto de 0, sin tocar el ledger. `printUsage()` se
actualiza con
`verify record --feature <id> -- <comando>   Corre <comando>, anexa evidencia a .claude/verify-ledger.jsonl`.

**Lectura del ledger — extensión de `status.go`.** Nueva función
`readLedger(fsys fs.FS) (entries []ledgerEntry, corruptLines []string, err error)`:
lee `.claude/verify-ledger.jsonl` línea por línea (líneas vacías
ignoradas), intenta `json.Unmarshal` cada una; si falla, la línea (con su
número) va a `corruptLines` en vez de abortar la lectura completa —
mismo espíritu que `parseBlockedBy`/`ticketBlockedByReasons`: nunca fallar
en silencio, nunca tirar todo el cálculo por una entrada mala. Archivo
inexistente (nadie corrió `verify record` todavía) no es error: entries
vacío. `computeStatusFromFS` invoca `readLedger` una vez (mismo momento
que ya precarga `specExistsByFeature`/`ticketsByFeature`) y calcula
`hashTree(fsys)` una vez (el hash del árbol actual, para comparar contra
`treeHash` de cada receipt).

**Extensión de `computeBlockedReasons`.** Nuevo chequeo, dentro del mismo
loop existente sobre `fl.Features` (punto 7 de la lista ya documentada en
`specs/april_status_arbiter/spec.md`): para cada feature con
`status == "in_progress"` (y solo esa — ver US20), busca la última entrada
del ledger con `kind == "test"` y `featureId == f.ID` (última por orden de
aparición en el archivo, que ya es cronológico por construcción
append-only). Si no hay ninguna, si `exitCode != 0`, o si `treeHash` no
coincide con el hash del árbol actual, agrega a `reasons` un string que
contiene literalmente `"no_test_evidence"` junto con detalle legible (cuál
de los tres casos aplicó). Las líneas corruptas del ledger (si las hay) se
reportan aparte, como una razón adicional por línea, sin importar qué
feature esté `in_progress`. `derivePhase`/`frontier`/`nextRecommended` no
cambian — esta feature solo agrega contenido a `blockedReasons`, no
introduce un valor nuevo de `phase` (consistente con el límite ya
reconocido en `specs/april_status_arbiter/spec.md`, sección Out of
Scope, que anticipó exactamente esta extensión).

**No se toca `set_status.go`.** El gate de `require_tests_to_close` al
momento de `set-status <id> done` sigue sin aplicarse automáticamente —
sigue siendo el humano/orquestador quien lo verifica leyendo
`april status --json` antes de aprobar el cierre (ver Out of Scope).

## Testing Decisions

**Unitario, sobre los seams puros.** `fstest.MapFS` sintético, mismo
patrón que `status_test.go`/`scaffold_test.go`:

- `TestHashTreeExcluyeGitProgressYElPropioLedger` — arma un `fstest.MapFS`
  con archivos bajo `.git/`, `progress/` y `.claude/verify-ledger.jsonl`,
  calcula el hash, modifica solo esos archivos excluidos, recalcula, y
  verifica que el hash NO cambió.
- `TestHashTreeCambiaSiUnArchivoNoExcluidoCambia` — modifica un archivo
  fuera de las exclusiones (ej. `status.go` sintético) y verifica que el
  hash SÍ cambia.
- `TestHashTreeEsDeterministicoSinImportarOrden` — dos `fstest.MapFS` con
  el mismo contenido, construidos con claves insertadas en distinto orden
  (el orden de iteración de un mapa Go no es estable) producen el mismo
  hash.
- `TestAppendLedgerEsAppendOnlyNoPisaEntradasPrevias` — dos llamadas
  sucesivas a la función pura de append sobre el mismo contenido inicial
  producen un resultado con ambas líneas, en orden, la primera intacta.
- `TestAppendLedgerSobreArchivoInexistenteEmpiezaLimpio` — contenido
  inicial vacío/inexistente produce un archivo con una sola línea válida.

**Integración, sobre `recordVerify`/`runVerifyRecord` con subproceso
real.** `t.TempDir()` + comandos portables (`sh -c "..."`, mismo binario
que ya asume `update.go`):

- `TestRecordVerifyComandoExitosoRegistraExitCeroYTreeHash` — corre
  `sh -c "exit 0"`, verifica la entrada anexada al ledger real en disco
  (`exitCode == 0`, `featureId` correcto, `treeHash` no vacío).
- `TestRecordVerifyComandoFallidoRegistraExitCodeReal` — `sh -c "exit 3"`,
  verifica `exitCode == 3` en el ledger (no un booleano genérico de
  "falló").
- `TestRecordVerifyCapturaStdoutYStderrPorSeparado` —
  `sh -c "echo hola-stdout; echo hola-stderr >&2"`, verifica con
  literales conocidos (`"hola-stdout"`/`"hola-stderr"`, no recalculados)
  que cada campo tiene lo que le corresponde.
- `TestRecordVerifyComandoInexistenteNoEscribeLedger` — comando con un
  nombre de binario inventado, verifica `err != nil` y que
  `.claude/verify-ledger.jsonl` no se creó (o, si ya existía, hash del
  archivo antes/después idéntico).
- `TestRecordVerifyDosCorridasProducenDosEntradas` — corre dos veces sobre
  el mismo `featureId`, verifica 2 líneas en el archivo resultante, ambas
  parseables, ninguna sobrescribe a la otra.
- `TestRunVerifyRecordFaltaFeatureEsErrorDeInvocacion`,
  `TestRunVerifyRecordFaltaDobleGuionEsErrorDeInvocacion`,
  `TestRunVerifyRecordSinComandoTrasDobleGuionEsErrorDeInvocacion` — exit
  code != 0, sin tocar el ledger, mismo patrón que
  `TestRunSetStatusTransicionInvalidaNoEscribeNada` (hash del árbol
  antes/después idéntico).

**Integración, sobre `computeStatusFromFS`/`runStatus` leyendo el
ledger.** Mismo patrón de fixtures que `status_test.go`
(`fstest.MapFS` para el cálculo puro, `t.TempDir()` + `runStatusCaptured`
para el binario):

- `TestSinReceiptParaFeatureInProgressReportaNoTestEvidence`
- `TestReceiptConExitDistintoDeCeroReportaNoTestEvidence`
- `TestReceiptExitosoSobreArbolActualNoReportaNoTestEvidence`
- `TestReceiptExitosoConArbolDesactualizadoReportaNoTestEvidence` — arma
  un `treeHash` de una versión "vieja" del árbol en el receipt, cambia un
  archivo no excluido en el fixture, verifica que reaparece
  `no_test_evidence`.
- `TestEscribirEnProgressNoInvalidaReceiptVigente` — regresión directa del
  problema encontrado durante esta spec: fixture con receipt vigente,
  agrega contenido a `progress/current.md` en el fixture, recalcula
  `blockedReasons`, verifica que `no_test_evidence` NO aparece.
- `TestNoTestEvidenceSoloAplicaAFeatureInProgress` — fixture con una
  feature `pending` y otra `done`, ninguna con receipt, verifica que
  ninguna reporta `no_test_evidence` (solo se evalúa para `in_progress`).
- `TestLineaCorruptaDeLedgerSeReportaEnBlockedReasons` — una línea con
  JSON inválido en el fixture del ledger, verifica que aparece un
  `blockedReasons` que la señala, sin que el resto del cálculo se rompa.
- `TestKindReviewEnLedgerNoSeConfundeConKindTest` — una entrada
  `kind: "review"` (simulando lo que hará la feature 6) para la misma
  feature no cuenta como evidencia de tests — solo `kind: "test"` importa
  para `no_test_evidence`.

**Precedente de subproceso real en tests.** `update_test.go` no ejecuta
subprocesos reales (testea `buildUpdateCmd` puro, sin correr `sh`); esta
feature es la primera que sí necesita correr un subproceso real dentro de
un test para verificar exit code/captura de salida — se documenta acá
porque no hay precedente exacto en el repo, y se elige `sh -c` por ser lo
mismo que ya asume disponible `update.go` en producción (no agrega una
dependencia de entorno nueva).

## Out of Scope

- Enforzar `no_test_evidence` como bloqueo automático en
  `april feature set-status <id> done` — `set_status.go` no se toca en
  esta feature; el gate sigue siendo que el humano/orquestador lea
  `april status --json` antes de aprobar el cierre. Automatizarlo (que
  `set-status` consulte el ledger él mismo) queda como posible feature
  futura, no decidida acá.
- Rotación, compactación o límite de tamaño del ledger — crece sin límite
  mientras el proyecto viva. No es parte de esta feature (ver Further
  Notes).
- Truncar o limitar el tamaño de `stdout`/`stderr` capturados — se
  guardan completos.
- El campo `kind: "review"` y su lógica asociada — es la feature 6
  (`review_verdict_recorded`), que reusa este mismo archivo pero define
  su propio flujo de escritura/lectura.
- `subject_hash`/candidato congelado vía `git write-tree` — feature 7
  (`review_frozen_candidate`). El hash de esta feature es deliberadamente
  más simple (walk de filesystem con exclusiones fijas, stdlib puro, sin
  requerir que el directorio sea un repo git).
- Interpretación de shell del comando después de `--` (pipes, `&&`,
  redirects) — se ejecuta directo vía `exec.Command`, sin `sh -c`
  implícito (ver US24). Quien lo necesite lo pide explícito.
- Exclusiones configurables del hash del árbol (opción C descartada
  explícitamente por el humano el 27/08/2026) — la lista de exclusiones
  (`.git/`, el ledger, `progress/`) es fija en el código, no un flag ni
  un archivo de configuración.
- Locking o coordinación entre corridas concurrentes de `verify record`
  sobre el mismo ledger — `one_feature_at_a_time` ya limita en la
  práctica a una feature activa a la vez; no se agrega ningún mecanismo
  de lock de archivo más allá de la atomicidad de `writeFileAtomic`.

## Further Notes

- La decisión de exclusiones (`.git/`, el ledger, `progress/`) surgió
  directamente de un problema concreto detectado durante la fase de
  spec de esta feature: con un hash de árbol completo sin exclusiones, la
  bitácora obligatoria que cada subagente escribe en `progress/` al
  terminar su tarea (`CLAUDE.md`, sección "Bitácora en progress/")
  invalidaría el receipt que ese mismo agente acababa de grabar, en el
  mismo ciclo. El humano confirmó explícitamente (27/08/2026) la opción A
  (walk completo con exclusiones fijas) sobre B (`git ls-files`) y C
  (alcance configurable).
- `docs/architecture.md` ya mencionaba "el ledger de receipts" como
  estado crítico con escritura atómica obligatoria antes de que esta spec
  existiera — esta feature no introduce esa regla, la implementa.
- `specs/april_status_arbiter/spec.md` (feature 2) ya anticipó
  explícitamente esta extensión en su sección "Out of Scope" ("El ledger
  de evidencia de tests... hasta que existan, esta feature no puede
  distinguir objetivamente 'implementation' de 'review'"). Esta spec no
  contradice esa spec: no cambia la tabla de derivación de `phase` ni el
  contrato de `frontier`/`artifactPaths`, solo agrega contenido posible a
  `blockedReasons`, que la spec de la feature 2 nunca declaró cerrado.
- Crecimiento sin límite del ledger (`stdout`/`stderr` completos, sin
  rotación) es una deuda reconocida desde ya, candidata natural para el
  ratchet de calidad de la feature 11 (`doctor_debt_ratchet`) más
  adelante — no se resuelve acá para no sobrealcanzar esta feature.
- Si en el futuro se necesita ejecutar el comando a través de una shell
  (pipes, variables de entorno del shell), la vía es que quien invoque
  `verify record` pase explícitamente `-- sh -c "..."` — no hace falta
  que `april` lo soporte de forma especial.
