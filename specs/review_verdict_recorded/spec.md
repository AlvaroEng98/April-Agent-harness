## Problem Statement

`require_review_to_close` hoy se satisface de dos formas, ninguna
verificable objetivamente por `april status`: (1) `reviewer_agent` narra su
veredicto en prosa dentro de `progress/current.md`, y (2) desde la feature
4, el orquestador lo repite a mano como flag `--verdict` en
`april feature set-status <id> done` — un mecanismo explícitamente
**interino** (`set_status.go:19-30`: "hasta que exista el ledger real de
veredictos (features 5/6, todavía pending)... el veredicto se pasa a mano
por flag en la misma invocación, no se lee de ningún registro"). Ninguna de
las dos vías dexa un registro consultable *durante* el ciclo de vida de la
feature (antes de llegar a `done`): si el humano o el orquestador quieren
saber, en medio de la Fase Revisión, si ya hay un veredicto vigente contra
el árbol actual — o si el último veredicto fue `CHANGES_REQUESTED` y por lo
tanto no habilita nada — tienen que leer `progress/current.md` línea por
línea y confiar en que nadie lo resumió mal. Es exactamente la misma brecha
que `ROADMAP.md` señala para tests ("evidencia derivada por el sistema, no
narrada"), aplicada ahora al veredicto de revisión.

## Solution

Se extiende el mismo ledger append-only que ya construyó la feature 5
(`.claude/verify-ledger.jsonl`, JSON Lines) con un segundo tipo de entrada,
`kind: "review"` — el campo que esa feature dejó reservado exactamente
para esto (`verify.go:104-106`, `specs/verify_record_ledger/spec.md`,
Out of Scope: "El campo `kind: review`... es la feature 6, que reusa este
mismo archivo pero define su propio flujo de escritura/lectura"). No se
crea un archivo nuevo.

Un comando nuevo, `april review record --feature <id> --verdict <valor>`
(`valor` ∈ `APPROVED`, `APPROVED_WITH_OBJECTION`, `CHANGES_REQUESTED` —
mismo vocabulario exacto que ya usa `reviewer_agent` en su salida, ver
`.claude/agents/reviewer_agent.md`), calcula el hash del árbol actual
(`hashTree`, reusada tal cual de la feature 5) y anexa una entrada
`kind: "review"` al ledger con `featureId`, `verdict`, `treeHash` y
`timestamp`. A diferencia de `verify record`, este comando **no corre
ningún subproceso** — el veredicto ya fue decidido por `reviewer_agent`
antes de invocarlo; `review record` solo lo deja registrado.

`april status <id> --json` (extensión de `computeBlockedReasons`, mismo
punto de extensión que usó la feature 5) busca, para la feature
`in_progress` consultada, la última entrada `kind: "review"` de esa
feature en el ledger. Si no hay ninguna, si su `verdict` es
`CHANGES_REQUESTED`, o si su `treeHash` no coincide con el árbol actual,
`blockedReasons` incluye una entrada con la substring literal
`"no_review_verdict"`. Con una entrada `kind: "review"` cuyo `verdict` sea
`APPROVED` o `APPROVED_WITH_OBJECTION` y cuyo `treeHash` coincida con el
árbol actual, no la incluye — mismo patrón exacto de tres casos
(ausente / valor que no habilita / hash desactualizado) que ya usa
`no_test_evidence` para `kind: "test"`.

**Decisión de diseño — el mecanismo interino de la feature 4 no se toca.**
El comentario de `set_status.go` que marca `--verdict` en `set-status done`
como interino ("destinado a ser reemplazado") podría leerse como mandato
de que esta feature debe además modificar `set_status.go` para que `done`
consulte el ledger en vez de aceptar el flag a mano. Se descarta
explícitamente esa lectura, seguida en su lugar de la ADR ya escrita por la
feature 5 (`specs/verify_record_ledger/spec.md`, Out of Scope: "Enforzar
`no_test_evidence` como bloqueo automático en `set-status done` —
`set_status.go` no se toca en esta feature... Automatizarlo... queda como
posible feature futura, no decidida acá"). Esta spec sigue el mismo
patrón, con el mismo razonamiento, para `no_review_verdict`: el ledger es
**consultivo** durante el ciclo de vida (vía `april status --json`, que el
humano/orquestador ya deben leer antes de aprobar cierre), y `set-status
done --verdict <valor>` sigue siendo la única escritura autoritativa de
`feature_list.json`, sin cambios. El resultado, reconocido explícitamente
como coexistencia deliberada y no como descuido: hay dos señales en
paralelo — `reviewVerdict` en `feature_list.json` (fijado en el instante de
cierre) y el historial completo de veredictos en el ledger (consultable en
cualquier momento del ciclo de vida, incluyendo antes de llegar a `done`).
Unificarlas (que `set-status done` lea el ledger en vez de/además del flag)
queda fuera de esta spec, con el mismo estatus que la automatización
equivalente de `no_test_evidence`: posible feature futura, no decidida
acá.

## User Stories

1. Como `reviewer_agent`, quiero registrar mi veredicto (`APPROVED`,
   `APPROVED_WITH_OBJECTION` o `CHANGES_REQUESTED`) con un comando, para
   dejar constancia objetiva sin depender de que mi resumen en
   `progress/current.md` sea leído con cuidado por el orquestador.
2. Como orquestador, quiero que `april status <id> --json` me diga si
   falta un veredicto de revisión vigente para la feature `in_progress`,
   para no avanzar al gate de cierre confiando solo en mi propia lectura
   de la bitácora.
3. Como orquestador, quiero que la ausencia total de cualquier entrada
   `kind: "review"` para la feature `in_progress` reporte
   `no_review_verdict`, para detectar el caso "nadie registró el
   veredicto todavía".
4. Como orquestador, quiero que la última entrada `kind: "review"` con
   `verdict: "CHANGES_REQUESTED"` reporte `no_review_verdict` — es decir,
   que `CHANGES_REQUESTED` NO habilite el cierre — para no confundir "se
   revisó y se rechazó" con "está aprobado".
5. Como orquestador, quiero que una entrada `kind: "review"` con `verdict`
   válido (`APPROVED`/`APPROVED_WITH_OBJECTION`) pero cuyo `treeHash` no
   coincide con el árbol actual reporte `no_review_verdict`, para detectar
   el caso "se aprobó una versión del código que ya no es la que está en
   el árbol" — mismo criterio de vigencia que ya usa `no_test_evidence`.
6. Como orquestador, quiero que una entrada `kind: "review"` con `verdict`
   `APPROVED` o `APPROVED_WITH_OBJECTION` y `treeHash` igual al árbol
   actual NO reporte `no_review_verdict`, para poder avanzar cuando el
   veredicto es real y vigente.
7. Como humano, quiero que una ronda `CHANGES_REQUESTED` seguida de una
   corrección y un nuevo `APPROVED` queden como dos entradas distintas en
   el ledger (nunca se pisa la anterior), para poder auditar cuántas
   rondas de revisión tomó cerrar una feature, no solo el resultado final.
8. Como humano, quiero que si el proceso muere a mitad de un `review
   record` (kill -9, corte de luz), el ledger existente quede intacto —
   con todas sus entradas previas legibles, tanto `kind: "test"` como
   `kind: "review"` — en vez de corrupto o truncado, por la misma garantía
   de `writeFileAtomic` que ya usa `verify record`.
9. Como desarrollador de April, quiero que `review record` reuse
   `appendToLedger`/`writeFileAtomic` tal cual, sin una segunda
   implementación del patrón temp-then-rename, porque el archivo y la
   garantía de atomicidad son exactamente los mismos que ya construyó la
   feature 5.
10. Como desarrollador de April, quiero que `review record` reuse
    `hashTree` tal cual (mismas exclusiones fijas: `.git/`, el propio
    ledger, `progress/`), para que el criterio de "qué cuenta como cambio
    en el árbol" sea idéntico para evidencia de tests y de revisión — dos
    hashes calculados con reglas distintas serían una fuente de confusión
    silenciosa.
11. Como humano, quiero que `review record` con un valor de `--verdict`
    fuera del vocabulario exacto de `reviewer_agent` (typo, sinónimo,
    minúsculas) sea un error de invocación explícito en stderr, sin
    escribir ninguna entrada al ledger — un veredicto mal escrito no es
    evidencia de nada.
12. Como humano, quiero que `review record` sin `--feature <id>` sea un
    error de invocación explícito, para no terminar con un veredicto
    huérfano sin feature asociada.
13. Como humano, quiero que `review record` sin `--verdict <valor>` sea un
    error de invocación explícito, para no registrar una entrada sin
    veredicto.
14. Como desarrollador de April, quiero que `kind: "review"` viva en el
    mismo archivo `.claude/verify-ledger.jsonl` que `kind: "test"`, sin
    crear un segundo ledger, porque el campo `kind` ya se reservó
    exactamente para esta extensión (feature 5) y ambos tipos comparten la
    misma garantía de escritura atómica.
15. Como desarrollador de April, quiero que la lectura del ledger para
    `no_review_verdict` reuse `readLedger` tal cual (la misma función que
    ya usa `no_test_evidence`), en vez de una segunda función que vuelva a
    parsear el archivo, para no duplicar el manejo de líneas corruptas.
16. Como humano, quiero que una línea corrupta en el ledger (JSON
    inválido) se siga reportando en `blockedReasons` exactamente como ya
    lo hace hoy, sin un mecanismo aparte para líneas corruptas de
    `kind: "review"` — la detección de corrupción no distingue `kind`,
    ocurre antes de saber qué tipo de entrada es.
17. Como orquestador, quiero que una entrada `kind: "test"` nunca cuente
    como evidencia de revisión (aunque tenga el mismo `featureId`), y que
    una entrada `kind: "review"` nunca cuente como evidencia de tests —
    misma separación estricta por `kind` que ya probó la feature 5 en el
    sentido inverso (`TestKindReviewEnLedgerNoSeConfundeConKindTest`).
18. Como orquestador, quiero que `no_review_verdict` solo se evalúe para
    la feature con `status: "in_progress"`, no para `pending`,
    `spec_ready`, `blocked` o `done` — mismo alcance exacto que
    `no_test_evidence` — para no llenar `blockedReasons` de ruido sobre
    trabajo que ni siquiera arrancó o que ya cerró.
19. Como humano, quiero que el exit code de `april review record` sea 0
    si el append al ledger fue exitoso, **incluso si el `verdict`
    registrado es `CHANGES_REQUESTED`** — registrar un rechazo
    correctamente no es un fallo del comando, es su función.
20. Como `reviewer_agent`, quiero que `review record` no intente correr
    ningún subproceso ni capturar stdout/stderr — a diferencia de `verify
    record`, el veredicto ya fue decidido por mí antes de invocar el
    comando; el comando solo lo persiste.
21. Como desarrollador de April, quiero que `review.go` siga el mismo
    patrón pure-function + wrapper-de-I/O que `verify.go`/`status.go`, con
    una función pura de orquestación (`recordReview`) separada del parseo
    de CLI (`runReviewRecord`) y del entry point que sí llama a
    `os.Exit` (`cmdReview`), para que el módulo nuevo se sienta consistente
    con el resto del código.
22. Como humano, quiero que `printUsage()` documente el subcomando nuevo
    (`review record --feature <id> --verdict <valor>`), para descubrirlo
    sin leer el código.
23. Como desarrollador de April, quiero que el esquema del ledger se
    extienda con el campo mínimo necesario (`verdict`), sin tocar la
    serialización exacta que la feature 5 ya fijó y testeó
    (`TestLedgerEntrySerializaAlEsquemaExactoDeLaSpec`) para entradas
    `kind: "test"` — el campo nuevo debe ser invisible para entradas
    existentes/futuras de `kind: "test"`.
24. Como orquestador, quiero poder consultar `april status --json` en
    cualquier punto de la Fase Revisión (no solo al final) y ver reflejado
    de inmediato si ya hay un veredicto vigente, sin esperar a que alguien
    actualice `progress/current.md`.
25. Como desarrollador de April, quiero que el mecanismo interino
    `--verdict` de `april feature set-status <id> done` (feature 4) siga
    funcionando exactamente igual después de esta feature — sin cambios en
    `set_status.go` — para no introducir una regresión ni un cambio de
    comportamiento no pedido por el `acceptance` de esta feature.
26. Como humano, quiero que el vocabulario de `--verdict` en `review
    record` sea idéntico —mismos tres literales, mismas mayúsculas— al que
    ya usa `set-status done --verdict` (feature 4) y al que emite
    `reviewer_agent`, para no tener tres vocabularios ligeramente distintos
    en el mismo sistema.
27. Como desarrollador de April, quiero que `review record` no valide que
    `--feature <id>` corresponda a una feature real de `feature_list.json`
    — mismo criterio permisivo que ya usa `verify record` — para no
    duplicar esa validación en dos lugares ni acoplar `review.go` a la
    lectura de `feature_list.json`.

## Implementation Decisions

**Módulo nuevo: `review.go` (+ `review_test.go`).** No `verify.go` — aunque
comparten el ledger y varias funciones de soporte, `review` es un comando
de primer nivel distinto de `verify` en `main.go`, y `docs/architecture.md`
ya anticipa `review.go` como archivo dedicado ("*(futuro, ROADMAP.md
E1-E6)* ... `review.go`"). Mismo criterio de "un archivo por
responsabilidad" que ya separa `set_status.go` de `status.go` pese a que
ambos leen `feature_list.json`.

**Extensión de `ledgerEntry` (en `verify.go`, no se duplica el tipo).**
Un campo nuevo:

```go
type ledgerEntry struct {
    Kind      string   `json:"kind"`
    FeatureID int      `json:"featureId"`
    Command   []string `json:"command"`
    ExitCode  int      `json:"exitCode"`
    TreeHash  string   `json:"treeHash"`
    Timestamp string   `json:"timestamp"`
    Stdout    string   `json:"stdout"`
    Stderr    string   `json:"stderr"`
    Verdict   string   `json:"verdict,omitempty"` // nuevo — solo kind:"review"
}
```

`Verdict` lleva `omitempty` porque las entradas `kind: "test"` nunca lo
usan — así la serialización exacta que la feature 5 ya fijó y testeó no
cambia ni un byte para esas entradas. Los campos que no aplican a
`kind: "review"` (`Command`, `ExitCode`, `Stdout`, `Stderr`) se dejan sin
`omitempty` — cambiarles la anotación arriesgaría alterar la
serialización congelada de `kind: "test"` sin ninguna necesidad real; una
entrada `kind: "review"` simplemente los serializa en su valor cero
(`"command":null,"exitCode":0,"stdout":"","stderr":""`) junto a
`kind`/`featureId`/`verdict`/`treeHash`/`timestamp`. Ejemplo de entrada
completa:

```json
{"kind":"review","featureId":6,"command":null,"exitCode":0,"treeHash":"3f8a...","timestamp":"2026-08-28T19:04:05Z","stdout":"","stderr":"","verdict":"APPROVED"}
```

**Vocabulario de veredictos — reusado, no redefinido.** `set_status.go` ya
define `verdictApproved`, `verdictApprovedWithObjection`,
`verdictChangesRequested` (mismo paquete `main`, visibles sin import).
`review.go` los reusa tal cual para validar `--verdict`; no se redeclara
un segundo conjunto de constantes con los mismos tres literales.

**`recordReview(featureID int, verdict string) (entry ledgerEntry, err
error)`** — función pura de orquestación en `review.go`, análoga a
`recordVerify` pero sin subproceso: valida que `verdict` sea uno de los
tres valores reconocidos (si no, devuelve error sin tocar el ledger — a
diferencia de `set-status done`, que distingue `CHANGES_REQUESTED` como
"reconocido pero no habilita", acá **sí** se acepta y registra
`CHANGES_REQUESTED` — es un veredicto real, registrarlo es la función del
comando; solo un valor *fuera* del vocabulario de los tres es error de
invocación); calcula `hashTree(os.DirFS("."))`; arma la `ledgerEntry{Kind:
"review", FeatureID: featureID, Verdict: verdict, TreeHash: treeHash,
Timestamp: time.Now().UTC().Format(time.RFC3339)}` (con `Command` nil,
`ExitCode` 0, `Stdout`/`Stderr` vacíos); hace `appendToLedger(entry)`
(reusada tal cual de `verify.go`). No corre ningún `exec.Command`.

**`runReviewRecord(args []string) int`** — parseo posicional simple, mismo
estilo estricto que `runVerifyRecord`: `args[0] == "--feature"`, `args[1]`
el id numérico, `args[2] == "--verdict"`, `args[3]` el valor. Cualquier
desvío (flag faltante, orden distinto, id no numérico, verdict fuera de
vocabulario, argumentos de más) es error explícito en stderr, exit
distinto de 0, sin invocar `recordReview`. El exit code de éxito es
siempre 0 — no hay comando externo cuyo exit code reflejar (US19/US20);
solo un fallo de invocación o un fallo de escritura del ledger producen
exit ≠ 0.

**`cmdReview()`** — entry point del CLI, mismo patrón que `cmdVerify`:
despacha `os.Args[2]` (hoy solo `"record"`, error explícito para cualquier
otro subcomando) y llama a `os.Exit(runReviewRecord(...))`.

**`main.go`.** Nuevo caso `"review"` en el `switch` de `main()` →
`cmdReview()`. `printUsage()` agrega la línea
`review record --feature <id> --verdict <valor>   Registra el veredicto de reviewer_agent en .claude/verify-ledger.jsonl`.

**Extensión de `status.go` — lectura, sin nueva pasada por el archivo.**
`computeStatusFromFS` ya invoca `readLedger(fsys)` una vez (feature 5); esa
misma lista de `ledgerEntry` (mezcla de `kind: "test"` y `kind: "review"`)
se pasa a `computeBlockedReasons` sin cambios de firma — el filtrado por
`kind` ocurre dentro de las funciones de chequeo, no en `readLedger`. Dos
funciones nuevas, mismo archivo, junto a sus análogas de la feature 5:

- `lastReviewEntryForFeature(entries []ledgerEntry, featureID int)
  (ledgerEntry, bool)` — igual que `lastTestEntryForFeature` pero
  filtrando `Kind == "review"`.
- `noReviewVerdictReason(f featureEntry, entries []ledgerEntry,
  currentTreeHash string) string` — igual estructura de tres casos que
  `noTestEvidenceReason`: sin ninguna entrada → mensaje con
  `no_review_verdict`; última entrada con `Verdict ==
  verdictChangesRequested` (o, por robustez, cualquier valor que no sea
  `verdictApproved`/`verdictApprovedWithObjection`) → mismo mensaje;
  última entrada con verdict válido pero `TreeHash != currentTreeHash` →
  mismo mensaje con el detalle de hashes; último caso (verdict válido y
  hash vigente) → `""`.

Dentro del loop ya existente de `computeBlockedReasons`, en la misma rama
`if f.Status == "in_progress"` que ya llama a `noTestEvidenceReason`, se
agrega la llamada a `noReviewVerdictReason` y su resultado (si no es `""`)
se agrega a `reasons` — mismo punto de extensión, sin nuevo loop sobre
`fl.Features`. Las líneas corruptas del ledger (`corruptLedgerLines`) no
cambian: ya se reportan una sola vez, sin depender de qué `kind` tuviera la
línea que falló al parsear.

`derivePhase`/`computeFrontier`/`nextRecommendedText` no cambian — esta
feature, igual que la 5, solo agrega contenido a `blockedReasons`.

**No se toca `set_status.go`.** Ver "Solution" arriba — decisión de
diseño explícita, siguiendo el precedente de Out of Scope de la feature 5.
El mecanismo `--verdict` de `set-status done` sigue exactamente igual.

## Testing Decisions

**Unitario, sobre el seam puro `recordReview`.** Mismo espíritu que
`verify_test.go` pero sin subproceso — no hace falta `t.TempDir()` con
binario real para la mayoría de los casos, aunque `appendToLedger`/
`hashTree` sí tocan el filesystem real (mismo precedente que
`TestAppendToLedgerDosLlamadasProducenDosLineasSinPisarNiDejarArchivosTemp`
en `verify_test.go`, que usa `chdirTemp`):

- `TestRecordReviewVerdictoValidoAnexaEntradaConTreeHashYTimestamp` —
  `chdirTemp`, corre `recordReview(6, "APPROVED")`, verifica la línea
  anexada al ledger real en disco (`kind: "review"`, `featureId: 6`,
  `verdict: "APPROVED"`, `treeHash` no vacío, `timestamp` parseable).
- `TestRecordReviewChangesRequestedSeRegistraConExitoNoEsError` — mismo
  patrón con `"CHANGES_REQUESTED"`: `err == nil`, la entrada queda en el
  ledger con ese `verdict` — literal conocido, verifica US19/US20
  explícitamente (registrar un rechazo no es un fallo del comando).
- `TestRecordReviewVerdictoFueraDeVocabularioNoEscribeLedger` — llama con
  `"aprobado"` (minúsculas) o `"LGTM"`, verifica `err != nil` y que el
  ledger no se creó (o su hash antes/después es idéntico) — mismo patrón
  que `TestRecordVerifyComandoInexistenteNoEscribeLedger`.
- `TestRecordReviewDosCorridasProducenDosEntradas` — dos llamadas
  sucesivas (`CHANGES_REQUESTED` y luego `APPROVED`, simulando una ronda de
  revisión real), verifica 2 líneas en el ledger, ambas parseables, en
  orden, ninguna pisa a la otra — literal conocido de US7.
- `TestRunReviewRecordFaltaFeatureEsErrorDeInvocacion`,
  `TestRunReviewRecordFaltaVerdictEsErrorDeInvocacion`,
  `TestRunReviewRecordFeatureNoNumericaEsErrorDeInvocacion`,
  `TestRunReviewRecordVerdictFueraDeVocabularioEsErrorDeInvocacion` — exit
  code ≠ 0, ledger sin tocar (hash del árbol antes/después idéntico) —
  mismo patrón que los tests de invocación de `runVerifyRecord`.
- `TestRunReviewRecordExitCodeCeroIndependienteDelVerdictRegistrado` —
  corre con los tres valores válidos (`APPROVED`,
  `APPROVED_WITH_OBJECTION`, `CHANGES_REQUESTED`), verifica exit code 0 en
  los tres — literal conocido de US19, no recalculado.

**Integración, sobre `computeStatusFromFS`/`runStatus` leyendo el ledger.**
Mismo patrón de fixtures que los tests de `no_test_evidence` en
`status_test.go` (`fstest.MapFS` + `ledgerLine` helper ya existente):

- `TestSinEntradaReviewParaFeatureInProgressReportaNoReviewVerdict` —
  fixture con feature `in_progress` sin ninguna entrada `kind: "review"`
  (puede tener entradas `kind: "test"` en verde, para aislar que el gap es
  específicamente de revisión) → `blockedReasons` contiene
  `no_review_verdict`.
- `TestUltimaEntradaReviewChangesRequestedReportaNoReviewVerdict` — última
  entrada `kind: "review"` con `verdict: "CHANGES_REQUESTED"` y `treeHash`
  igual al árbol actual → `no_review_verdict` presente (verifica US4: el
  hash vigente no alcanza si el verdict no habilita).
- `TestUltimaEntradaReviewApprovedVigenteNoReportaNoReviewVerdict` —
  `verdict: "APPROVED"`, `treeHash` igual al árbol actual →
  `no_review_verdict` ausente.
- `TestUltimaEntradaReviewApprovedWithObjectionVigenteNoReportaNoReviewVerdict`
  — mismo caso con `APPROVED_WITH_OBJECTION`, verificado aparte porque el
  `acceptance` de la feature nombra ambos valores explícitamente como
  habilitantes.
- `TestUltimaEntradaReviewApprovedConArbolDesactualizadoReportaNoReviewVerdict`
  — `verdict: "APPROVED"` pero `treeHash` de una versión vieja del árbol,
  fixture con un archivo no excluido modificado después → reaparece
  `no_review_verdict` (mismo patrón que
  `TestReceiptExitosoConArbolDesactualizadoReportaNoTestEvidence` de la
  feature 5, ahora para `kind: "review"`).
- `TestSecuenciaChangesRequestedLuegoApprovedResuelveNoReviewVerdict` —
  fixture con dos entradas `kind: "review"` para la misma feature, primero
  `CHANGES_REQUESTED` (hash viejo) y después `APPROVED` (hash actual) →
  `no_review_verdict` ausente, porque se evalúa la **última** entrada, no
  la primera — verifica US7 combinado con la lógica de "último" heredada
  de `no_test_evidence`.
- `TestEntradaKindTestNoCuentaComoEvidenciaDeReview` — fixture con una
  entrada `kind: "test"` en verde y vigente para la feature, sin ninguna
  entrada `kind: "review"` → `no_review_verdict` sigue presente (US17,
  espejo exacto de `TestKindReviewEnLedgerNoSeConfundeConKindTest` de la
  feature 5, ahora en el sentido inverso).
- `TestEntradaKindReviewNoCuentaComoEvidenciaDeTest` — fixture con una
  entrada `kind: "review"` `APPROVED` vigente, sin ninguna `kind: "test"`,
  para una feature `in_progress` → `no_test_evidence` sigue presente (misma
  aserción, sentido inverso, para cerrar el círculo por completo).
- `TestNoReviewVerdictSoloAplicaAFeatureInProgress` — fixture con una
  feature `pending` y otra `done`, ninguna con entradas `kind: "review"` →
  ninguna reporta `no_review_verdict`.
- `TestLineaCorruptaDeLedgerSeReportaSinImportarKind` — reuso directo del
  fixture/test ya existente de la feature 5
  (`TestLineaCorruptaDeLedgerSeReportaEnBlockedReasons`); se documenta acá
  que no hace falta un test nuevo porque `readLedger` no distingue `kind`
  al detectar corrupción — si el test existente sigue en verde tras esta
  feature, ya cubre el caso.

**Regresión — el mecanismo interino no cambia.** No se agrega ningún test
nuevo a `set_status_test.go`: los 21 tests existentes de esa suite deben
seguir en verde sin modificación, como evidencia de que `set_status.go` no
se tocó (US25). `go test ./...` completo cubre esto por construcción.

**Precedente para nombres/estilo.** Los nombres de test van en español
(casos de negocio específicos), mismo criterio ya confirmado en
`docs/conventions.md` y seguido por `verify_test.go`/`status_test.go`.

## Out of Scope

- Modificar `set_status.go` para que `set-status done` consulte el ledger
  en vez de (o además de) aceptar `--verdict` a mano — ver "Solution",
  decisión explícita de no tocarlo, siguiendo el mismo precedente que la
  feature 5 dejó para `no_test_evidence`. Queda como posible feature
  futura, no decidida acá.
- `subject_hash`/candidato congelado vía `git write-tree` — feature 7
  (`review_frozen_candidate`). Esta feature sigue registrando el veredicto
  contra el `treeHash` simple de `hashTree` (walk de filesystem con
  exclusiones fijas), no contra un candidato inmutable de git.
- Profundidad de revisión ajustada por sensibilidad del diff — feature 8
  (`review_depth_by_diff_sensitivity`). `review record` no reporta qué
  rutas tocó el diff ni exige pasos adicionales.
- Rotación, compactación o límite de tamaño del ledger — misma deuda
  reconocida ya por la feature 5, no crece el alcance acá.
- Validar que `--feature <id>` corresponda a una feature real de
  `feature_list.json` — mismo criterio permisivo que ya usa `verify
  record` (US27).
- Un kill switch o forma de desactivar el requisito de veredicto vigente
  — la divergencia deliberada de April frente a gentle-ai
  (`ROADMAP.md`: "en April, `require_review_to_close` sí es puerta") se
  mantiene sin excepción configurable.

## Further Notes

- Esta feature es, en estructura, un espejo casi exacto de la feature 5:
  mismo archivo de ledger, mismo `hashTree`, mismo `writeFileAtomic`, mismo
  punto de extensión en `computeBlockedReasons`, mismo patrón de "último
  registro gana" con tres casos (ausente / valor que no habilita / hash
  desactualizado). La diferencia central es que no hay subproceso que
  correr — el veredicto ya existe cuando se invoca el comando — por eso
  `review.go` es más chico que `verify.go` y no necesita distinguir
  "invocación fallida" de "corrida con exit code distinto de 0".
- La coexistencia de dos señales de veredicto (`reviewVerdict` en
  `feature_list.json`, fijado al cerrar; y el historial completo en el
  ledger, consultable durante todo el ciclo) es una decisión consciente,
  no una inconsistencia a resolver con urgencia. Si en el futuro se decide
  unificarlas, el candidato natural es que `set-status done` dejara de
  aceptar `--verdict` como input libre y en su lugar leyera la última
  entrada `kind: "review"` vigente del ledger — pero eso es una feature
  aparte, con su propio spec y su propia decisión humana explícita, no un
  efecto colateral de esta.
- `docs/architecture.md` ya nombra `review.go` como módulo futuro antes de
  que esta spec exista — esta feature no introduce esa decisión de
  ubicación, la implementa.
