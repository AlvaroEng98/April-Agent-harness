## Problem Statement

`computeBlockedReasons` (`status.go`) y sus helpers
(`noTestEvidenceReason`, `noReviewVerdictReason`, `ticketBlockedByReasons`,
`detectBlockedByCycle`) ya diagnostican con precisión qué está mal — pero
se quedan ahí. El orquestador (humano o agente) que lee `blockedReasons`
sabe *qué* pasa (dos features `in_progress`, falta evidencia de tests,
falta veredicto de revisión, un `Blocked by` ilegible, un ciclo de
dependencias entre tickets) pero tiene que reconstruir de memoria, cada
vez, cuál es el comando `april ...` exacto o la acción de archivo que lo
resuelve. Eso ya generó fricción real y evitable: la sintaxis exacta de
`april verify record`/`april review record`/`april feature set-status`
vive dispersa en `verify.go`/`review.go`/`set_status.go`, no al lado del
mensaje que la necesita.

## Solution

Cada mensaje que hoy produce `computeBlockedReasons` (directamente o vía
sus cinco helpers) gana, además del diagnóstico que ya tiene, una receta
concreta: el comando `april ...` exacto (con el id real de la feature
sustituido, nunca un placeholder cuando el id se conoce) o la acción de
archivo concreta (qué archivo de ticket editar y con qué formato). El
diagnóstico existente **no se reescribe** — se preserva carácter por
carácter como prefijo del mensaje nuevo, y la receta se agrega después
como contenido añadido. Esto es deliberado: permite que el test de
caracterización obligatorio (`docs/conventions.md`, "Cambios a la lógica
de derivación de fase") demuestre la propiedad exacta que pide esta
feature — que lo único que cambió es la adición, nunca el texto
preexistente — con una simple comprobación de prefijo
(`strings.HasPrefix`) contra el literal congelado antes del cambio, en
vez de tener que re-litigar manualmente si el resto del mensaje sigue
igual.

No todos los mensajes de `blockedReasons` reciben receta: revisado
`status.go` completo (ver "Inventario completo de mensajes" en
Implementation Decisions), algunos ya son autoexplicativos o no tienen un
único comando `april` que los resuelva (ej. una corrupción de
`feature_list.json` o de una línea del ledger exige edición manual
puntual, no una receta genérica) — esos quedan fuera de alcance con
justificación explícita, no por omisión.

### Spec previa en la misma zona — sigo la que ya delegó esto explícitamente

`specs/spec_gwt_mechanical_check/spec.md` (feature 14, `done`) es la
última spec que tocó `computeBlockedReasons` antes que esta. Su sección
"Out of Scope" dice literalmente: *"El comando de remedio concreto para
el mensaje `no_gwt_coverage` [...] queda para la feature 15
(`blocked_reasons_remedy_commands`) cuando se implemente, esta feature
solo deja la substring de contrato estable para que esa feature la
extienda."* Esta spec **sigue** esa decisión explícitamente: aunque la
`description`/`acceptance` de la feature 15 en `feature_list.json` no
menciona `no_gwt_coverage` por nombre (enumera puntualmente las líneas de
`in_progress` duplicado, `noTestEvidenceReason`, `noReviewVerdictReason`,
`ticketBlockedByReasons` y `detectBlockedByCycle`), el punto 7 de su
`acceptance` exige revisar *todo* `status.go` y decidir explícitamente
para cada mensaje — y la spec previa de la misma zona ya decidió, por
adelantado, que `no_gwt_coverage` le toca a esta feature. Desviarse de
eso sin señalarlo sería contradecir un ADR ya escrito; esta spec lo honra
y lo suma como alcance adicional (ver Implementation Decisions), no
como sustituto de los cinco casos explícitamente enumerados en
`feature_list.json`.

## User Stories

1. Como orquestador, quiero que el mensaje de más de una feature
   `in_progress` me diga exactamente qué comando correr para resolverlo,
   para no tener que recordar la sintaxis de `april feature set-status`
   de memoria.

   Given `feature_list.json` con dos o más features en status
   `in_progress`
   When corro `april status --json`
   Then `blockedReasons` contiene una entrada que preserva el texto
   diagnóstico actual ("hay N features en in_progress a la vez...") como
   prefijo, y agrega los ids de esas features y el comando
   `april feature set-status <id> pending`

2. Como orquestador, quiero que ese mismo mensaje liste los ids reales de
   las features en conflicto, para saber cuál de ellas bajar sin tener
   que cruzar el JSON a mano.

   Given tres features con ids 2, 5 y 9 todas en `in_progress`
   When corro `april status --json`
   Then el mensaje de `blockedReasons` menciona los tres ids (2, 5, 9)
   ordenados ascendentemente

3. Como orquestador, quiero que si solo hay una feature `in_progress`
   (el caso normal) este mensaje no aparezca en absoluto, para no generar
   ruido — esto ya es el comportamiento actual y esta feature no lo toca.

4. Como orquestador, quiero que un receipt `kind:test` ausente para la
   feature `in_progress` me diga el comando exacto de `april verify
   record` con el id real ya puesto, para copiarlo y correrlo sin editar
   nada a mano.

   Given una feature `in_progress` sin ningún receipt `kind:test` en el
   ledger
   When corro `april status --json`
   Then `blockedReasons` contiene una entrada que preserva la substring
   literal `no_test_evidence` y el texto diagnóstico actual como prefijo,
   y agrega `april verify record --feature <id> -- <comando>` con `<id>`
   sustituido por el id real de la feature

5. Como orquestador, quiero el mismo remedio cuando el último receipt
   `kind:test` tiene `exitCode != 0` (se intentó y falló), para saber que
   hay que corregir el problema y volver a correr el mismo comando.

   Given una feature `in_progress` cuyo último receipt `kind:test` tiene
   `exitCode` distinto de 0
   When corro `april status --json`
   Then el mensaje preserva `no_test_evidence` y el diagnóstico de
   `exitCode` actual, y agrega `april verify record --feature <id> --
   <comando>` con el id real

6. Como orquestador, quiero el mismo remedio cuando el receipt está en
   verde pero el `treeHash` quedó desactualizado (el código cambió
   después de la corrida registrada), para saber que hay que refrescar la
   evidencia, no que algo esté roto.

   Given una feature `in_progress` cuyo último receipt `kind:test` tiene
   `exitCode == 0` pero `treeHash` distinto del árbol actual
   When corro `april status --json`
   Then el mensaje preserva `no_test_evidence` y el diagnóstico de
   `treeHash` desactualizado actual, y agrega
   `april verify record --feature <id> -- <comando>` con el id real

7. Como orquestador, quiero que un receipt `kind:review` ausente para la
   feature `in_progress` me diga el comando exacto de `april review
   record` con el id real, incluyendo los tres valores válidos de
   `--verdict`, para no tener que ir a buscar `review.go` para
   recordarlos.

   Given una feature `in_progress` sin ninguna entrada `kind:review` en
   el ledger
   When corro `april status --json`
   Then `blockedReasons` contiene una entrada que preserva la substring
   literal `no_review_verdict` y el texto diagnóstico actual como
   prefijo, y agrega `april review record --feature <id> --verdict
   <valor>` con `<id>` sustituido por el id real y una mención de los
   valores válidos de `<valor>` (APPROVED, APPROVED_WITH_OBJECTION,
   CHANGES_REQUESTED)

8. Como orquestador, quiero el mismo remedio cuando el último veredicto
   registrado es `CHANGES_REQUESTED` (no habilita cierre), para saber que
   hace falta resolver lo pedido y volver a registrar un veredicto que sí
   habilite el cierre.

   Given una feature `in_progress` cuyo último receipt `kind:review`
   tiene un `verdict` distinto de APPROVED/APPROVED_WITH_OBJECTION
   When corro `april status --json`
   Then el mensaje preserva `no_review_verdict` y el diagnóstico actual
   del verdict que no habilita cierre, y agrega
   `april review record --feature <id> --verdict <valor>` con el id real
   y aclara que `<valor>` debe ser APPROVED o APPROVED_WITH_OBJECTION

9. Como orquestador, quiero el mismo remedio cuando el veredicto está en
   verde pero el `treeHash` quedó desactualizado, para saber que hace
   falta refrescar el veredicto, no repetir la revisión completa desde
   cero sin motivo.

   Given una feature `in_progress` cuyo último receipt `kind:review`
   tiene un verdict que habilita cierre pero `treeHash` distinto del
   árbol actual
   When corro `april status --json`
   Then el mensaje preserva `no_review_verdict` y el diagnóstico actual
   de `treeHash` desactualizado, y agrega
   `april review record --feature <id> --verdict <valor>` con el id real

10. Como agent_developer, quiero que un ticket con `Blocked by` no
    interpretable me diga exactamente qué formato espera el parser y qué
    archivo tengo que editar, para no tener que leer `parseBlockedBy` en
    `status.go` para entenderlo.

    Given un ticket cuyo campo `**Blocked by:**` no contiene ni números
    de dos dígitos ni la palabra "none"
    When corro `april status --json`
    Then `blockedReasons` contiene una entrada que preserva el texto
    diagnóstico actual (identifica el ticket por `t.Filename` y la
    feature), y agrega que el formato esperado es "números de ticket de
    dos dígitos separados por coma, o la palabra 'none'" junto con la
    instrucción de editar el campo `**Blocked by:**` de ese mismo archivo
    (`t.Filename`)

11. Como agent_developer, quiero que un ciclo detectado en `Blocked by`
    me diga qué archivo concreto de la cadena editar para romperlo, no
    solo la cadena de ids de ticket, para no tener que ir a abrir cada
    ticket de la cadena a adivinar cuál tocar.

    Given tickets cuyo grafo de `Blocked by` forma un ciclo (ej. 02
    bloqueado por 03, 03 bloqueado por 02)
    When corro `april status --json`
    Then `blockedReasons` contiene una entrada que preserva la substring
    literal `ciclo detectado` y la cadena de tickets actual, y agrega la
    instrucción de editar el campo `**Blocked by:**` de alguno de los
    tickets de esa cadena (nombrando al menos un archivo concreto, ej. el
    primero de la cadena) para quitar o corregir la referencia que cierra
    el ciclo

12. Como desarrollador de April, quiero que el mensaje de
    `no_gwt_coverage` (feature 14) reciba el mismo tratamiento, honrando
    la delegación explícita que le dejó `specs/spec_gwt_mechanical_check/spec.md`,
    para que ningún "reason code" de `blockedReasons` quede a medio
    resolver por una feature que ya sabía que le tocaba a esta.

    Given una spec `sdd:true` sin tickets, sin bloques Given/When/Then ni
    marcador de opt-out, con status distinto de `done`
    When corro `april status --json`
    Then `blockedReasons` contiene una entrada que preserva la substring
    literal `no_gwt_coverage` y el texto diagnóstico actual, y agrega la
    instrucción de agregar al menos un bloque Given/When/Then a esa
    `spec.md`, o el marcador `<!-- gwt: no aplica -->` si ninguna
    historia tiene rama verificable

13. Como humano, quiero que el mensaje de status fuera de
    `rules.valid_status` NO reciba receta — ya identifica la feature, su
    status inválido, y el vocabulario válido está documentado en
    `feature_list.json.rules.valid_status`; corregirlo es una decisión
    editorial (¿a qué status válido se quiso volver?) que ningún comando
    puede adivinar por mí.

14. Como humano, quiero que el mensaje de una feature marcada `blocked`
    NO reciba receta — es autoexplicativo (ya identifica la feature) y
    "blocked" es, por diseño, un estado que requiere intervención humana
    puntual, no un comando mecánico.

15. Como humano, quiero que el mensaje de spec faltante para una feature
    que ya requiere spec (`status` en spec_ready/in_progress/done) NO
    reciba receta de comando `april` — ya nombra la ruta exacta que falta
    (`specs/<name>/spec.md`), y el remedio real es lanzar `spec_writer`
    (un subagente, no un comando de CLI), acción que ya describe
    `CLAUDE.md` en la Fase Spec.

16. Como humano, quiero que el mensaje de un ticket con `Status` fuera de
    pending/in_progress/done NO reciba receta adicional — ya nombra el
    archivo del ticket y los tres valores válidos explícitamente; no hay
    comando `april` dedicado a corregir ese campo (se edita el archivo a
    mano), y el mensaje ya es tan concreto como el de `Blocked by` (que sí
    recibe receta) porque este no necesita explicar un formato adicional.

17. Como humano, quiero que una línea corrupta del ledger (JSON inválido)
    NO reciba receta genérica — el contenido corrupto es arbitrario, no
    hay una única corrección mecánica posible, y el mensaje ya identifica
    archivo y número de línea para que alguien lo abra y lo revise a
    mano.

18. Como reviewer_agent, quiero que los tests existentes de
    `status_test.go` que verifican `blockedReasons` vía
    `strings.Contains`/`anyContains` (incluyendo los que fijan
    `no_test_evidence`, `no_review_verdict`, `ciclo detectado`,
    `no_gwt_coverage`) sigan pasando sin que se les edite ninguna
    aserción, para confirmar que la adición de recetas es aditiva y no
    rompe ningún contrato ya consumido.

19. Como agent_developer, quiero un test de caracterización, escrito
    ANTES de tocar `computeBlockedReasons`/sus helpers, que fije el texto
    exacto actual de los nueve casos cubiertos (in_progress duplicado;
    `no_test_evidence` sin receipt, con `exitCode != 0`, con `treeHash`
    desactualizado; `no_review_verdict` en los mismos tres casos; ticket
    con `Blocked by` no interpretable; ciclo en `Blocked by`), para poder
    demostrar después que el cambio es puramente aditivo.

20. Como agent_developer, quiero que ese mismo test de caracterización,
    reutilizado después del cambio, verifique con
    `strings.HasPrefix(mensajeNuevo, literalCongeladoAntes)` que el
    diagnóstico original sigue intacto carácter por carácter, para no
    depender de una inspección manual "a ojo" de si el resto del mensaje
    cambió.

21. Como humano, quiero que ningún test existente de `status_test.go`
    (`TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons`,
    que fija `blockedReasons: []string{}` para las features `sdd:true` ya
    `done`) se vea afectado por esta feature, porque esas seis/doce
    features no producen ninguno de los nueve mensajes tocados (su
    `blockedReasons` sigue vacío).

22. Como humano, quiero confirmación explícita de que `go build ./...` y
    `go test ./...` siguen en verde tras el cambio, para saber que la
    adición de recetas no rompió la compilación ni ningún test de la
    suite completa (no solo `status_test.go`).

23. Como orquestador, quiero que ningún mensaje nuevo invente un id de
    feature que no exista, ni sustituya `<id>`/`<valor>`/`<comando>` con
    un valor real cuando ese valor no se puede conocer de antemano (ej.
    `<comando>` en `april verify record`, que depende de qué comando
    quiera correr quien lo ejecute, o `<valor>` en `--verdict`, que
    depende del resultado real de la revisión) — esos dos placeholders
    quedan literales, reusando exactamente la misma sintaxis que ya usan
    los mensajes de uso (`usage`) de `verify.go`/`review.go`.

24. Como agent_developer, quiero que el orden en que `computeBlockedReasons`
    agrega las entradas a `reasons` no cambie, para no romper
    implícitamente ningún test que dependiera (aunque hoy ninguno lo
    hace explícitamente) del orden relativo de los mensajes.

25. Como humano, quiero que esta feature no agregue ningún campo nuevo a
    `statusReport`/`doctorReport` ni ningún flag de CLI nuevo — el cambio
    vive enteramente dentro del contenido de los strings que ya se
    agregan a `blockedReasons`, misma disciplina anti-sobre-ingeniería ya
    aplicada por las features 13 y 14.

## Implementation Decisions

**Seam: los mismos cinco puntos de origen dentro de `status.go`, ninguno
nuevo.** MUST modificar únicamente el contenido de los `fmt.Sprintf` ya
existentes en `computeBlockedReasons` (mensaje de `in_progress`
duplicado y de `no_gwt_coverage`), `noTestEvidenceReason`,
`noReviewVerdictReason`, `ticketBlockedByReasons` y `detectBlockedByCycle`.
MUST NOT introducir un mecanismo nuevo (post-procesador de mensajes,
tabla de remedios separada, etc.) — es el seam más alto y el único que ya
tiene, para cada caso, toda la información necesaria (id real, filename,
verdict, treeHash) disponible en el mismo scope donde se construye el
mensaje.

**Preservación del diagnóstico existente como prefijo — regla
transversal a los seis puntos.** MUST preservar, para los nueve casos
cubiertos, el texto que produce el código actual carácter por carácter
como prefijo exacto del mensaje nuevo (incluyendo las substrings de
contrato ya establecidas por specs previas: `no_test_evidence`,
`no_review_verdict`, `ciclo detectado`, `no_gwt_coverage`). MUST agregar
la receta como contenido nuevo después de ese prefijo, introducido con el
separador ` — ` (espacio, guion largo, espacio) — mismo separador en los
nueve casos, para que sea mecánicamente fácil de verificar con
`strings.HasPrefix`. NO MUST usar un separador distinto por caso.

**Mensajes literales actuales — el "antes" exacto que el test de
caracterización MUST fijar.** Verificado leyendo `status.go` completo y
corriendo mentalmente los fixtures ya existentes de
`status_test.go` (reutilizables tal cual para el test nuevo):

- In_progress duplicado (fixture: dos features `in_progress`, ids 2 y 3
  → `inProgressCount = 2`):
  `"hay 2 features en in_progress a la vez (máximo 1 permitido por one_feature_at_a_time)"`
- `no_test_evidence` sin receipt (fixture: feature id 5,
  `verify_record_ledger`):
  `"feature 5 (verify_record_ledger) está in_progress pero no tiene ningún receipt kind:test en .claude/verify-ledger.jsonl (no_test_evidence)"`
- `no_test_evidence` con `exitCode != 0` (mismo fixture, receipt con
  `ExitCode: 1`):
  `"feature 5 (verify_record_ledger) está in_progress pero su último receipt kind:test tiene exitCode 1 != 0 (no_test_evidence)"`
- `no_test_evidence` con `treeHash` desactualizado: mismo patrón con
  `treeHash`/árbol actual interpolados dinámicamente — el test MUST
  comparar el prefijo hasta el primer `(%s)` interpolado con el mismo
  patrón, no un literal con el hash hardcodeado (el hash depende del
  fixture exacto de cada corrida).
- `no_review_verdict` sin receipt (fixture: feature id 6,
  `review_verdict_recorded`):
  `"feature 6 (review_verdict_recorded) está in_progress pero no tiene ningún receipt kind:review en .claude/verify-ledger.jsonl (no_review_verdict)"`
- `no_review_verdict` con verdict que no habilita cierre (mismo fixture,
  `Verdict: "CHANGES_REQUESTED"`):
  `"feature 6 (review_verdict_recorded) está in_progress pero su último receipt kind:review tiene verdict \"CHANGES_REQUESTED\", que no habilita cierre (no_review_verdict)"`
- `no_review_verdict` con `treeHash` desactualizado: mismo caso que el
  análogo de `no_test_evidence`, hash dinámico.
- Ticket con `Blocked by` no interpretable (fixture: ticket
  `01-nucleo.md`, feature `april_status_arbiter`, texto
  `"algo raro sin numero"`):
  `"ticket 01-nucleo.md de la feature april_status_arbiter tiene Blocked by no interpretable (\"algo raro sin numero\"): ni números de ticket de dos dígitos ni \"none\", o referencia un ticket inexistente"`
- Ciclo en `Blocked by` (fixture: tickets `02-frontier.md`/`03-cli.md`
  bloqueados entre sí, feature `april_status_arbiter` — el orden de DFS
  sobre `fstest.MapFS`, que itera directorios en orden alfabético, visita
  primero `02`):
  `"ciclo detectado en Blocked by de tickets de april_status_arbiter: 02 → 03 → 02"`

**Recetas a agregar, caso por caso.** MUST agregar, después del
separador ` — `, el siguiente contenido (id/filename siempre reales,
nunca placeholders, salvo donde se indica lo contrario):

- In_progress duplicado: la lista de ids en `in_progress` (ordenados
  ascendentemente, ej. "ids en in_progress: 2, 3") y el comando literal
  `` `april feature set-status <id> pending` `` — `<id>` MUST quedar
  literal (no hay un único id correcto: es al humano/orquestador a quien
  le toca elegir cuál bajar).
- `no_test_evidence` (los tres casos): `` `april verify record --feature
  <id real> -- <comando>` ``, con `<id real>` sustituido por `f.ID` y
  `<comando>` literal (no se puede conocer de antemano qué comando de
  verificación se quiere correr).
- `no_review_verdict` (los tres casos): `` `april review record
  --feature <id real> --verdict <valor>` ``, con `<id real>` sustituido
  por `f.ID` y `<valor>` literal, mencionando explícitamente los tres
  valores válidos de `--verdict` (APPROVED, APPROVED_WITH_OBJECTION,
  CHANGES_REQUESTED) al menos en el caso "sin receipt", y acotado a
  APPROVED/APPROVED_WITH_OBJECTION en el caso "verdict que no habilita
  cierre" (`CHANGES_REQUESTED` no resuelve el bloqueo, sería engañoso
  sugerirlo ahí).
- Ticket con `Blocked by` no interpretable: el formato esperado en
  prosa ("números de ticket de dos dígitos separados por coma, ej. \"01,
  02\", o la palabra \"none\" si no tiene bloqueadores") y la instrucción
  de editar el campo `**Blocked by:**` del archivo ya nombrado en el
  prefijo (`t.Filename`) — SHOULD repetir el nombre del archivo
  explícitamente en la receta (no solo confiar en que ya apareció antes
  en el mismo mensaje), para que la receta sea autocontenida si se lee
  aislada (ej. copiada a un ticket o a un chat).
- Ciclo en `Blocked by`: la instrucción de editar el campo
  `**Blocked by:**` de alguno de los tickets de la cadena para quitar o
  corregir la referencia que la cierra. MUST nombrar al menos un archivo
  concreto (resolver el primer NN de la cadena detectada a su
  `t.Filename` real, vía el mismo `tickets []ticketInfo` que ya recibe
  `detectBlockedByCycle` — construir el mapeo NN→Filename dentro de esa
  función, no en un archivo/estructura nueva).
- `no_gwt_coverage`: la instrucción de agregar al menos un bloque
  Given/When/Then (líneas que empiecen con "Given"/"When"/"Then") a la
  `spec.md` de la feature (ruta ya nombrada en el prefijo), o el marcador
  `gwtOptOutMarker` (`<!-- gwt: no aplica -->`) si ninguna historia de
  usuario tiene rama de comportamiento verificable.

**Fuera de alcance de receta — decisión explícita por mensaje, no
omisión.** Revisado `status.go` completo, estos son TODOS los mensajes
restantes que produce (directa o indirectamente) `blockedReasons`, y por
qué NO reciben receta:

- Status de feature fuera de `rules.valid_status` (`"feature %d (%s)
  tiene status %q fuera de rules.valid_status"`): autoexplicativo — ya
  identifica la feature y el status inválido; el vocabulario válido está
  en `feature_list.json.rules.valid_status`, y a qué status corregirlo es
  una decisión editorial, no mecánica.
- Feature marcada `blocked` (`"feature %d (%s) está marcada blocked"`):
  autoexplicativo por diseño — `blocked` es, por naturaleza, un estado
  que exige intervención humana puntual (no hay un comando genérico que
  "desbloquee").
- Spec requerida y faltante (`"feature %d (%s) está en status %q pero
  falta %s"`): ya nombra la ruta exacta faltante; el remedio real es
  lanzar `spec_writer` (subagente, no comando `april`), ya documentado en
  `CLAUDE.md`, Fase Spec — no hay comando de CLI que lo resuelva.
- Status de ticket fuera de pending/in_progress/done (`"ticket %s de la
  feature %s tiene Status %q fuera de pending/in_progress/done"`): ya
  nombra el archivo y los tres valores válidos explícitamente en el mismo
  mensaje; se corrige editando el campo `**Status:**` a mano, sin un
  comando `april` dedicado — a diferencia de `Blocked by`, no necesita
  explicar un formato adicional no obvio (los tres valores ya están
  escritos ahí).
- Línea corrupta del ledger (`"línea %d de %s no es JSON válido: %v"`):
  ya identifica archivo y número de línea; el contenido corrupto es
  arbitrario, no hay una única corrección mecánica posible — requiere
  abrir el archivo y decidir a mano.

**Errores de invocación — no son parte de `blockedReasons`.**
`ErrFeatureNotFound` y los errores de `computeStatusFromFS` (JSON
inválido en `feature_list.json`, fallo de lectura) MUST NOT tocarse por
esta feature: son errores Go que aborta la ejecución, no entradas de la
lista `blockedReasons` que esta feature receta — categoría distinta,
fuera de alcance por definición.

## Testing Decisions

Seam principal: `computeStatusFromFS(fsys fs.FS, targetID *int)
(statusReport, error)` — la misma interfaz pública que usa toda la suite
de `status_test.go`, sobre `fstest.MapFS`. Los tests de esta feature MUST
seguir ese mismo patrón: nunca invocar `noTestEvidenceReason`/
`noReviewVerdictReason`/`ticketBlockedByReasons`/`detectBlockedByCycle`
directamente en aislamiento salvo que ya exista precedente de hacerlo
(no lo hay en `status_test.go` — todos los tests existentes pasan por
`computeStatusFromFS`), para no acoplar el test a la firma interna de un
helper que podría reorganizarse.

**Test de caracterización — MUST existir antes del cambio de código.**
Un test nuevo (o una tabla dentro de uno) que, reusando los fixtures ya
existentes en `status_test.go` (`TestDosFeaturesInProgressReportaBlockedReasons`,
`TestSinReceiptParaFeatureInProgressReportaNoTestEvidence`,
`TestReceiptConExitDistintoDeCeroReportaNoTestEvidence`,
`TestReceiptExitosoConArbolDesactualizadoReportaNoTestEvidence`,
`TestSinEntradaReviewParaFeatureInProgressReportaNoReviewVerdict`,
`TestReviewChangesRequestedConHashVigenteReportaNoReviewVerdict`,
`TestReviewApprovedConHashDesactualizadoReportaNoReviewVerdict`,
`TestBlockedByConTextoNoInterpretableReportaBlockedReasons`,
`TestCicloEnBlockedByDeTicketsSeDetectaYNoCuelga` — o fixtures
equivalentes de armado análogo), corre `computeStatusFromFS` y verifica
con **igualdad exacta de string** (no `Contains`) el literal congelado en
"Mensajes literales actuales" de Implementation Decisions, para los siete
casos con hash estático (todos salvo los dos con `treeHash` dinámico,
donde MUST verificar el prefijo hasta el patrón conocido en vez de un
literal completo). MUST escribirse y correrse (confirmando que pasa)
ANTES de modificar `computeBlockedReasons`/sus helpers — no alcanza con
escribirlo después y verificar que ya pasa con el código nuevo.

**Verificación post-cambio — MUST reusar el mismo test, no uno nuevo
paralelo.** Tras el cambio, el mismo test (con sus aserciones de
igualdad exacta reemplazadas por `strings.HasPrefix(mensajeReal,
literalCongelado)`) MUST seguir pasando, y MUST agregar, para cada caso,
una aserción `strings.Contains` sobre el fragmento de receta esperado
(ej. el comando `april verify record --feature 5 -- <comando>` completo,
con el id real 5 ya sustituido). Esto demuestra mecánicamente la
propiedad central de esta feature: se agregó, no se reescribió.

**Casos MUST cubiertos** (nueve, uno por mensaje tocado — ver User
Stories 1-12 para el detalle Given/When/Then de cada uno):

1. In_progress duplicado → prefijo preservado + ids listados + comando
   `set-status ... pending`.
2. `no_test_evidence` sin receipt → prefijo + substring `no_test_evidence`
   preservada + comando `verify record` con id real.
3. `no_test_evidence` con `exitCode != 0` → ídem.
4. `no_test_evidence` con `treeHash` desactualizado → ídem (prefijo hasta
   el patrón conocido).
5. `no_review_verdict` sin receipt → prefijo + substring
   `no_review_verdict` preservada + comando `review record` con id real y
   los tres valores de `--verdict` mencionados.
6. `no_review_verdict` con verdict que no habilita cierre → ídem, pero
   acotado a APPROVED/APPROVED_WITH_OBJECTION.
7. `no_review_verdict` con `treeHash` desactualizado → ídem al caso 4.
8. Ticket con `Blocked by` no interpretable → prefijo + formato esperado
   + archivo a editar (`t.Filename`).
9. Ciclo en `Blocked by` → prefijo + substring `ciclo detectado`
   preservada + archivo concreto de la cadena a editar.

**Caso adicional, no obligatorio por `feature_list.json` pero incluido
por la delegación explícita de la spec previa (ver Solution):**
`no_gwt_coverage` → prefijo + substring `no_gwt_coverage` preservada +
instrucción de agregar GWT o el marcador de opt-out.

**Regresión — MUST correr sin editar** toda la suite existente de
`status_test.go` que verifica `blockedReasons` vía
`strings.Contains`/`anyContains` (en particular los tests que ya fijan
`no_test_evidence`, `no_review_verdict`, `ciclo detectado` y
`no_gwt_coverage` como substrings), y
`TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons` (no se
ve afectado: las features `sdd:true` ya `done` siguen con
`blockedReasons: []string{}`).

**`go build ./...` y `go test ./...` en verde** — MUST, sin excepciones,
antes de reportar la feature como implementada.

Evitar explícitamente: mockear `fs.FS`, comparar el mensaje nuevo con una
recalculación de la misma lógica de construcción del string (tautológico
— el valor esperado MUST venir del literal congelado leído directamente
del código actual antes del cambio, no de una función que reproduce el
mismo `fmt.Sprintf`), o testear los helpers internos en aislamiento sin
pasar por `computeStatusFromFS`.

## Out of Scope

- Los cinco mensajes listados en "Fuera de alcance de receta" de
  Implementation Decisions (status inválido, feature `blocked`, spec
  faltante, Status de ticket inválido, línea corrupta del ledger).
- Cualquier cambio a `nextRecommendedText`/`derivePhase` — esta feature
  toca únicamente el contenido de los strings de `blockedReasons`, no la
  lógica que decide `phase`/`nextRecommended`.
- Cualquier campo nuevo en `statusReport`/`doctorReport` (ej. un array
  estructurado de "remedios" separado del string de diagnóstico) — la
  receta vive dentro del mismo string, mismo formato que ya consumen
  todos los tests y `printStatusText`.
- Cualquier flag de CLI nuevo — no hace falta ninguno para agregar texto
  a un mensaje ya existente.
- Sustituir `<comando>` (en `april verify record`) o `<valor>` (en
  `april review record --verdict`) por un valor real — no se pueden
  conocer de antemano, quedan como placeholders literales.
- Cambiar el formato de salida de `april doctor` — hereda el cambio
  automáticamente porque ya copia `statusReport.BlockedReasons` tal cual
  (mismo mecanismo que documentó la feature 14), sin código propio nuevo
  en `doctor.go`.
- Reescribir o reordenar mensajes de `blockedReasons` que no estén en la
  lista de nueve (más `no_gwt_coverage`) explícitamente cubiertos.

## Further Notes

- Esta spec no contradice ninguna spec previa que tocó
  `computeBlockedReasons` (`april_status_arbiter`, `verify_record_ledger`,
  `review_verdict_recorded`, `tree_hash_respects_gitignore`,
  `spec_gwt_mechanical_check`) — es, en cambio, la que **cumple**
  explícitamente el compromiso que dejó pendiente
  `spec_gwt_mechanical_check` en su sección "Out of Scope" respecto al
  remedio de `no_gwt_coverage`.
- El separador ` — ` (espacio, guion largo, espacio) elegido para todas
  las recetas es el mismo que ya usan varios mensajes existentes de
  `status.go` (ej. el de `treeHash` desactualizado en
  `noTestEvidenceReason`/`noReviewVerdictReason`), así que no introduce
  un estilo tipográfico nuevo al archivo.
- El id real de la feature (`f.ID`) ya está disponible en el scope de los
  cinco puntos de origen — no hace falta ningún parámetro nuevo en
  ninguna firma de función existente para poder sustituirlo.
- Quien implemente debe confirmar, al momento de escribir el código, que
  el orden de iteración de `fstest.MapFS.ReadDir` (usado para construir
  el fixture del ciclo) sigue siendo alfabético — si algo en la stdlib de
  Go cambiara ese comportamiento, el literal congelado del caso "ciclo"
  (`02 → 03 → 02`) tendría que verificarse de nuevo contra el fixture
  real antes de darlo por válido en el test de caracterización.
