## Problem Statement

La feature 13 (`spec_template_gwt_rfc2119`, `done`) ya obliga a la
plantilla de `spec_writer` (`.claude/agents/spec_writer.md`) a agregar un
bloque `Given/When/Then` junto a cada historia de usuario con una rama de
comportamiento verificable. Pero esa obligación vive solo en la
instrucción que lee el subagente al redactar — nadie mecánico la
comprueba después. Si `spec_writer` (o un humano editando `spec.md` a
mano) olvida el bloque, o si un `agent_developer`/`ticket_writer`
retoma una spec vieja, hoy no hay ninguna señal objetiva que lo marque:
`april status`/`april doctor` calculan `phase`/`blockedReasons` leyendo
`feature_list.json` y `specs/<name>/spec.md`/`tickets/*.md`
(`computeStatusFromFS`, `status.go`), pero solo verifican que el archivo
de spec **exista** (`specExistsByFeature`, vía `fileExistsFS`) — nunca
miran su contenido. Una spec vacía de `Given/When/Then` pasa exactamente
igual de "lista para tickets" que una que sí cumple la disciplina de la
feature 13, y `nextRecommended` recomendaría lanzar `ticket_writer` sobre
ella sin que nadie lo note hasta que ya sea tarde (tickets escritos sobre
ramas de comportamiento nunca hechas explícitas).

## Solution

`computeBlockedReasons` (`status.go`) — el mismo mecanismo que ya usa
`april status --json`/`april doctor` para `no_test_evidence`,
`no_review_verdict`, `Blocked by` no interpretable y ciclos de tickets —
gana un chequeo estructural más, barato y determinístico: para cada
feature `sdd:true` cuyo `specs/<name>/spec.md` **existe** pero **todavía
no tiene ningún ticket** (`specs/<name>/tickets/` vacío o inexistente) y
cuyo `status` no es `done`, lee el contenido del archivo y verifica que
contenga al menos un bloque `Given/When/Then` (las tres palabras clave
literales que fijó la feature 13 en la plantilla, en inglés, cada una al
inicio de su propia línea) **o** el marcador explícito de "no aplica"
(ver Implementation Decisions). Si no encuentra ninguna de las dos cosas,
agrega una entrada a `blockedReasons` con la substring literal
`no_gwt_coverage` — el mismo patrón de "reason code" ya usado por
`no_test_evidence`/`no_review_verdict`.

No hay comando nuevo ni flag nuevo: el chequeo vive dentro de
`computeBlockedReasons`, que ya alimenta tanto `april status --json` como
`april doctor` (`computeDoctor` ya copia `statusReport.BlockedReasons` a
su propio reporte) — por eso `april doctor` hereda la señal sin que
`doctor.go` necesite una sola línea de código nueva. `nextRecommendedText`
ya suprime la recomendación de avanzar en cuanto `blockedReasons` no está
vacío, así que "no avanzar en silencio a tickets" queda cubierto por el
mecanismo existente, sin tocar esa función. `phase` no cambia de valor
por este chequeo: sigue derivándose exactamente igual que hoy
(`derivePhase`) — sigue diciendo `"tickets"` como dato informativo de "qué
archivo falta producir", mientras `blockedReasons` es quien dice "no es
seguro avanzar todavía".

El chequeo deja de aplicar en cuanto la feature tiene al menos un archivo
de ticket (ya pasó la puerta) o su `status` es `done` (ya se cerró) — esto
protege automáticamente, sin ninguna excepción ad-hoc, a las seis specs
`sdd:true` ya `done` que preceden a la feature 13
(`april_status_arbiter`, `verify_record_ledger`, `review_verdict_recorded`,
`review_frozen_candidate`, `review_depth_by_diff_sensitivity`,
`tree_hash_respects_gitignore`): las seis ya tienen tickets en disco, así
que ninguna vuelve a evaluarse retroactivamente.

## User Stories

1. Como orquestador, quiero que `april status --json` me avise si la
   spec aprobada de una feature `sdd:true` sin tickets todavía no tiene
   ningún bloque `Given/When/Then`, para no lanzar `ticket_writer` sobre
   una spec que no cumple la disciplina de la feature 13.

   Given una feature `sdd:true` con `specs/<name>/spec.md` existente, sin
   ningún bloque Given/When/Then, sin el marcador de opt-out, sin
   tickets en disco, y con `status` distinto de `done`
   When corro `april status --json`
   Then `blockedReasons` contiene una entrada con la substring literal
   `no_gwt_coverage` que menciona el id/nombre de esa feature

2. Como orquestador, quiero que si la spec sí tiene al menos un bloque
   `Given/When/Then`, el chequeo no reporte nada, para no bloquear
   features que ya cumplen la disciplina.

   Given la misma feature del caso anterior pero con `specs/<name>/spec.md`
   conteniendo al menos un bloque Given/When/Then real
   When corro `april status --json`
   Then `blockedReasons` no contiene ninguna entrada con `no_gwt_coverage`

3. Como `spec_writer` (o un humano editando la spec a mano), quiero poder
   declarar explícitamente que ninguna historia de mi spec tiene rama de
   comportamiento verificable, para no verme obligado a inventar un
   `Given/When/Then` artificial solo para pasar el chequeo mecánico.

   Given una spec sin ningún bloque Given/When/Then pero con el marcador
   de opt-out presente en cualquier parte del archivo
   When corro `april status --json`
   Then `blockedReasons` no contiene ninguna entrada con `no_gwt_coverage`

4. Como orquestador, quiero que el chequeo deje de aplicar en cuanto la
   feature ya tiene al menos un ticket en
   `specs/<name>/tickets/*.md`, para no re-litigar una spec que ya avanzó
   a la fase de implementación.

   Given una feature `sdd:true` cuyo `spec.md` no tiene ningún bloque
   Given/When/Then ni marcador de opt-out, pero con al menos un archivo en
   `specs/<name>/tickets/`
   When corro `april status --json`
   Then `blockedReasons` no contiene ninguna entrada con `no_gwt_coverage`
   para esa feature

5. Como humano, quiero que ninguna de las seis features `sdd:true` ya
   `done` cuyas specs preceden a la feature 13 (sin `Given/When/Then`, sin
   marcador de opt-out) empiece a reportar `no_gwt_coverage` tras esta
   feature, para no generar trabajo retroactivo de reescribir specs ya
   cerradas.

   Given una feature `sdd:true` con `status: done`, con `spec.md` sin
   ningún bloque Given/When/Then ni marcador de opt-out
   When corro `april status --json` (o `<id>` explícito de esa feature)
   Then `blockedReasons` no contiene ninguna entrada con `no_gwt_coverage`
   para esa feature

6. Como orquestador, quiero que `april doctor --json` reporte la misma
   señal `no_gwt_coverage` sin que `doctor.go` necesite código nuevo
   propio, reusando que ya compone `report.BlockedReasons` a partir de
   `computeStatus(nil)`.

   Given una feature `sdd:true` que dispara `no_gwt_coverage` según los
   casos anteriores
   When corro `april doctor --json`
   Then `report.blockedReasons` (el mismo campo que ya expone
   `doctorReport`) contiene la entrada `no_gwt_coverage`, y `report.healthy`
   es `false`

7. Como reviewer_agent, quiero poder confiar en que `blockedReasons` ya
   validó estructuralmente que la spec tiene GWT (o declaró
   explícitamente que no aplica) antes de llegar a mi revisión de la fase
   de Implementación, para no tener que volver a mirar eso a mano en cada
   veredicto.

8. Como humano, quiero que el chequeo sea barato y determinístico — una
   comparación de substrings/prefijos de línea, sin heurística de
   lenguaje natural ni motor de reglas nuevo — para que no dé falsos
   negativos o positivos según cómo esté redactada la prosa de la spec.

9. Como agent_developer, quiero un test de caracterización que fije el
   `phase`/`blockedReasons`/`nextRecommended` de las features `sdd:true`
   ya `done` de este repo **antes** de tocar `computeBlockedReasons`, para
   detectar cualquier regresión no buscada (disciplina de
   `docs/conventions.md`, "Cambios a la lógica de derivación de fase").

10. Como agent_developer, quiero que el marcador de opt-out sea un string
    literal exacto, verificable con una comparación de substring simple,
    no una heurística de NLP sobre frases en español, para que la
    implementación sea trivial y el resultado sea 100% decidible.

11. Como `spec_writer`, quiero poder colocar el marcador de opt-out en
    cualquier parte del archivo (no atado a una sección específica como
    `## User Stories`), para no depender de que la plantilla de
    `.claude/agents/spec_writer.md` (que esta feature no toca) reserve un
    lugar fijo para él.

12. Como orquestador, quiero que el chequeo detecte el bloque
    `Given/When/Then` sin importar en qué sección del documento aparezca
    (no exclusivamente bajo `## User Stories`), para que sea robusto a
    reordenamientos menores del documento y no dependa de parsear
    encabezados Markdown.

13. Como orquestador, quiero que el chequeo sea sensible a mayúsculas
    sobre las palabras clave literales `Given`/`When`/`Then` (en inglés,
    tal como las fijó la feature 13 en la plantilla real de
    `spec_writer.md`), para no confundir palabras comunes en español
    (`cuando`, `entonces`) con el marcador de GWT.

14. Como desarrollador de April, quiero que la propia
    `specs/spec_gwt_mechanical_check/spec.md` (esta spec) cumpla el
    chequeo que describe — que tenga al menos un bloque Given/When/Then
    real —, para no quedar bloqueada por su propia regla el día que se
    implemente.

15. Como orquestador, quiero que `nextRecommendedText` quede vacío (sin
    recomendar lanzar `ticket_writer`) mientras `no_gwt_coverage` esté
    presente, igual que ya pasa hoy con cualquier otro `blockedReasons` no
    vacío — sin necesidad de tocar `nextRecommendedText`.

16. Como humano, quiero que `phase` no cambie de valor por este chequeo
    — sigue siendo el dato informativo de "qué artefacto falta producir",
    separado de "si es seguro avanzar", que ya vive en `blockedReasons`.

17. Como agent_developer, quiero que los tests existentes de
    `status_test.go`/`doctor_test.go` que ya usan
    `anyContains`/`strings.Contains` sobre `blockedReasons` sigan pasando
    sin que se les edite ninguna aserción.

18. Como humano, quiero que esta feature no agregue ningún flag de CLI
    nuevo ni cambie la forma de invocar `april status`/`april doctor` — el
    chequeo es automático, siempre activo, sin opt-in por flag (misma
    disciplina anti-sobre-ingeniería de `CLAUDE.md`).

19. Como orquestador, quiero que si una spec tiene el marcador de opt-out
    **y además** bloques `Given/When/Then` reales (inconsistencia
    editorial), el chequeo no falle igual — prefiero un chequeo simple que
    nunca reporte ausencia en ese caso, a una heurística que intente
    arbitrar la contradicción.

20. Como agent_developer, quiero que el chequeo funcione igual sobre
    `fstest.MapFS` (tests puros) y sobre el disco real (`os.DirFS(".")`),
    igual que el resto de `computeStatusFromFS`, para poder testearlo sin
    tocar el filesystem real.

21. Como humano, quiero evidencia explícita (test) de que ni
    `april status --json` ni `april doctor --json` escriben, borran o
    modifican ningún archivo al correr este chequeo — mismo criterio
    read-only ya establecido por la feature 9 (`doctor_readonly_check`).

    Given un directorio con `feature_list.json`, specs y manifest válidos
    When corro `april status --json` y `april doctor --json` varias veces
    Then el hash del árbol de archivos antes y después es idéntico

22. Como desarrollador de April, quiero que la feature 15
    (`blocked_reasons_remedy_commands`, todavía `pending`) pueda extender
    más adelante el mensaje de `no_gwt_coverage` con un comando de remedio
    concreto, sin que el diseño de esta feature se lo impida — el mensaje
    de esta feature no fija un formato tan rígido (más allá de conservar
    la substring `no_gwt_coverage`) que rompa esa extensión futura.

23. Como humano, quiero que la spec documente explícitamente qué función
    pública se testea (no colaboradores internos mockeados), para que
    `reviewer_agent` pueda verificar el seam sin ambigüedad.

24. Como desarrollador de April, quiero que agregar esta métrica al
    cálculo global de `blockedReasons` no rompa la regla de "una sola
    feature `in_progress` a la vez" ni ninguno de los otros chequeos ya
    existentes en `computeBlockedReasons` — el chequeo nuevo se agrega
    como una entrada más de la lista, sin alterar el orden ni el
    contenido de las demás.

25. Como humano, quiero que ni `set_status.go`
    (`april feature set-status`) ni ningún otro comando de escritura
    consulten este chequeo — `april status`/`april doctor` siguen siendo
    puramente advisory (modo confirmado en `ROADMAP.md`, "B llegando por
    A"); la decisión de avanzar de fase la sigue tomando el humano/
    orquestador leyendo `blockedReasons`, no un gate automático de
    escritura.

## Implementation Decisions

**Seam: `computeBlockedReasons` (`status.go`), no `doctor.go`.** MUST
extender `computeBlockedReasons` (y el bucle de precómputo por feature
`sdd:true` que ya hace `computeStatusFromFS` para `specExistsByFeature`/
`ticketsByFeature`) para que, cuando el spec de una feature exista, lea
su contenido una sola vez y derive si satisface el requisito de GWT.
`doctor.go` NO MUST cambiar ninguna línea: `computeDoctor` ya asigna
`report.BlockedReasons = statusReport.BlockedReasons` a partir de
`computeStatus(nil)`, y `computeBlockedReasons` ya opera sobre **todas**
las features de `feature_list.json` (no solo la feature activa/target) —
exactamente el mismo alcance que ya tienen `no_test_evidence`/
`no_review_verdict`/los chequeos de `Blocked by`. Este es el seam más
alto disponible y el único que evita duplicar lectura de disco o lógica
entre los dos comandos (disciplina anti-sobre-ingeniería de `CLAUDE.md`:
ya existe un mecanismo — `computeBlockedReasons` compartido — que resuelve
"ambos comandos" sin construir nada nuevo en `doctor.go`).

**Condición de aplicación del chequeo.** MUST aplicar el chequeo
únicamente cuando, para una feature dada: `f.SDD` es verdadero, su
`specs/<name>/spec.md` existe, **no** tiene ningún archivo de ticket
todavía (`len(ticketsByFeature[f.Name]) == 0`), y su `status` no es
`"done"`. Fuera de esas condiciones (spec inexistente, ya hay tickets, o
la feature ya está cerrada) el chequeo MUST NOT producir ninguna entrada
de `blockedReasons` — es deliberadamente una puerta puntual en la
transición spec→tickets, no una invariante permanente sobre el archivo.
Esta condición es la que protege automáticamente, sin lista de
excepciones a mano, a las seis specs `sdd:true` ya `done` que preceden a
la feature 13 (todas ya tienen tickets en disco).

**Detección del bloque Given/When/Then — barata, sobre texto plano.**
MUST considerar que una spec "tiene GWT" si su contenido completo (el
archivo `spec.md` entero, sin restringir a una sección) contiene, cada
una en el inicio de su propia línea (ignorando espacios en blanco
iniciales), al menos una línea que empiece literalmente con `Given`, al
menos una que empiece con `When`, y al menos una que empiece con `Then`
— las tres palabras clave en inglés, sensibles a mayúsculas/minúsculas,
tal como las fijó la feature 13 en la plantilla real de
`.claude/agents/spec_writer.md` (que usa únicamente el formato en inglés,
no `Dado/Cuando/Entonces`, pese a que el `acceptance` original de la
feature 13 permitía cualquiera de los dos). NO MUST exigir que las tres
líneas sean consecutivas ni que aparezcan en un orden relativo
específico (Given antes que When antes que Then) — exigir adyacencia o
un parser de bloques agregaría complejidad sin beneficio claro dado que
"Given"/"When"/"Then" en mayúscula al inicio de línea son, en la práctica,
exclusivos de este formato dentro de una spec mayormente en español.

**Marcador de opt-out — string literal, no NLP.** MUST definir un
marcador exacto, verificable con una comparación de substring simple,
que una spec puede incluir en cualquier parte del archivo para declarar
que ninguna de sus historias de usuario tiene rama de comportamiento
verificable: el comentario Markdown/HTML `<!-- gwt: no aplica -->`
(invisible al renderizar, no colisiona con prosa real, no depende de
redacción en español que pueda variar). Su sola presencia MUST bastar
para que el chequeo no reporte `no_gwt_coverage` para esa feature,
**incluso si** la spec además contiene bloques `Given/When/Then` reales
(no se exige exclusividad mutua: el marcador es una declaración
explícita que basta por sí sola, no una corrección de una condición que
el chequeo intente arbitrar).

**Mensaje de `blockedReasons`.** El mensaje nuevo MUST contener la
substring literal `no_gwt_coverage` (mismo patrón de "reason code" que
`no_test_evidence`/`no_review_verdict`) y MUST identificar la feature por
id y nombre, y la ruta de su `spec.md`, siguiendo el mismo estilo de
`fmt.Sprintf` que ya usan `noTestEvidenceReason`/`noReviewVerdictReason`.
SHOULD dejar espacio para que la feature 15
(`blocked_reasons_remedy_commands`, pendiente) le agregue después un
comando de remedio concreto (ej. señalar qué archivo editar) sin romper
la substring de contrato — no MUST implementarlo ahora, es responsabilidad
de esa feature.

**`nextRecommendedText`/`derivePhase` no cambian.** MUST NOT modificar
ninguna de las dos funciones: `nextRecommendedText` ya devuelve `""`
cuando `blockedReasons` no está vacío (cubre "no avanzar en silencio" sin
tocar código), y `derivePhase` sigue derivando `phase` exactamente igual
que hoy — `"tickets"` sigue siendo el valor correcto una vez que el spec
existe y no hay tickets, sea o no sea válido su GWT; la validez de avanzar
vive enteramente en `blockedReasons`, no en `phase`.

**Test de caracterización obligatorio antes del cambio.** MUST escribir,
antes de tocar `computeBlockedReasons`, un test que corra
`computeStatus`/`computeStatusFromFS` sobre las features `sdd:true` ya
`done` de este propio repo (vía `os.DirFS(".")` contra el árbol real, o
un fixture `fstest.MapFS` equivalente que replique su spec/tickets) y
fije su `phase`/`blockedReasons`/`nextRecommended` actual. El número
exacto de features `done` en el momento de escribir el `acceptance`
original de esta feature era 12; al momento de implementarla puede ser
mayor (13, 16 y otras ya cerraron después) — el test MUST cubrir el
conjunto real de features `done` vigente al momento de implementar, no
un conteo fijo de 12. Dado el diseño de la condición de aplicación de
arriba, el resultado esperado tras el cambio es que **ninguna** de esas
features `done` cambie su `blockedReasons` ni una coma (todas ya tienen
tickets en disco) — el test MUST afirmar igualdad exacta antes/después
para ese subconjunto, no solo "nada inesperado cambió".

## Testing Decisions

Seam principal: `computeStatusFromFS(fsys fs.FS, targetID *int) (statusReport, error)`
— interfaz pública ya usada por toda la suite de `status_test.go`
(`TestFeatureConSpecYSinTicketsEsFaseTickets` y similares), sobre
`fstest.MapFS`. Los tests de esta feature MUST seguir ese mismo patrón:
construir un `feature_list.json` + `specs/<name>/spec.md` (+ opcionalmente
`tickets/`) mínimos en un `fstest.MapFS`, llamar a `computeStatusFromFS`,
y verificar `report.BlockedReasons` con el helper ya existente
`anyContains` (`status_test.go`) — nunca inspeccionando una función
interna en aislamiento ni mockeando el filesystem.

Casos MUST cubiertos (nombres orientativos, en español por ser casos de
negocio puntuales, mismo criterio que el resto de `status_test.go`):

- Spec sin GWT, sin marcador, sin tickets, `status` distinto de `done` →
  `anyContains(report.BlockedReasons, "no_gwt_coverage")` verdadero (US1).
- Spec con al menos un bloque Given/When/Then → falso (US2).
- Spec sin GWT pero con el marcador `<!-- gwt: no aplica -->` → falso
  (US3) — cubre explícitamente el `acceptance` "al menos un test cubre
  ese caso".
- Spec sin GWT ni marcador, pero con al menos un archivo en
  `specs/<name>/tickets/` → falso (US4).
- Spec sin GWT ni marcador, `status: done` → falso (US5).
- Spec con GWT y además el marcador (redundancia, no contradicción
  arbitrada) → falso (US19).
- El test de caracterización de las features `done` reales del repo
  (arriba, Implementation Decisions) — MUST ejecutarse contra el árbol
  real del repo o un fixture equivalente, antes y después del cambio.

Sobre `doctor.go`: MUST agregar (o confirmar con uno ya existente si
aplica) un test en `doctor_test.go` que dispare `no_gwt_coverage` vía
fixture y verifique que `computeDoctor().BlockedReasons` (y
`report.Healthy == false`) lo reflejan **sin que `doctor.go` tenga código
propio para ello** — el test documenta la herencia, no una
responsabilidad nueva de ese archivo.

Sobre read-only: MUST reusar el patrón exacto de
`TestDoctorNoEscribeArchivos` (`doctor_test.go`) —
`hashTreeSnapshot`/`snapshotsEqual` antes/después de correr
`runStatus`/`runDoctor` varias veces sobre un directorio temporal real
con una feature que dispara `no_gwt_coverage` — mismo criterio que la
feature 9.

Evitar explícitamente: mockear `fs.FS`, testear
`specDeclaresGWTOrOptOut`-o-como-se-llame en aislamiento sin pasar por
`computeStatusFromFS` (acoplaría el test al nombre interno de la función
en vez de al comportamiento observable), o comparar `blockedReasons` con
una recalculación de la misma lógica (tautológico) — el valor esperado en
cada test MUST ser un literal fijado a mano (presencia/ausencia de la
substring), no una repetición del cálculo del código bajo test.

## Out of Scope

- Soportar `Dado/Cuando/Entonces` (la variante en español que el
  `acceptance` de la feature 13 permitía pero que la plantilla real de
  `spec_writer.md` no usa hoy) — si algún día la plantilla cambia para
  aceptar ambas formas, extender la detección es una feature aparte.
- Cualquier cambio a `.claude/agents/spec_writer.md` para enseñarle a
  escribir el marcador `<!-- gwt: no aplica -->` — esta feature define el
  contrato que lee `april status`/`april doctor`, no vuelve a tocar la
  plantilla (ya cerrada por la feature 13). Enseñarle al agente a usarlo
  proactivamente queda para una feature de documentación aparte si el
  humano lo pide.
- Validar la calidad o correctitud del contenido dentro de cada bloque
  Given/When/Then (que el `Given` describa un estado real, que el `Then`
  sea observable, etc.) — el chequeo es puramente estructural (presencia
  de las tres palabras clave o del marcador), no semántico.
- Parsear Markdown de verdad (AST, secciones, encabezados) para acotar la
  búsqueda a `## User Stories` — se busca sobre el archivo completo (ver
  Implementation Decisions).
- Cualquier flag nuevo de CLI (`--json` ya existe en ambos comandos y
  sigue siendo la única forma de pedir salida estructurada).
- Cambiar `set_status.go`/`april feature set-status` para que consulte
  este chequeo — `status`/`doctor` siguen siendo puramente advisory.
- Retroactivamente marcar o re-evaluar features `sdd:true` ya `done` —
  quedan protegidas por construcción (ver Implementation Decisions), no
  por una lista de exclusión explícita por nombre.
- El comando de remedio concreto para el mensaje `no_gwt_coverage`
  (ej. "editá tal archivo") — queda para la feature 15
  (`blocked_reasons_remedy_commands`) cuando se implemente, esta feature
  solo deja la substring de contrato estable para que esa feature la
  extienda.

## Further Notes

- Esta spec no contradice `specs/april_status_arbiter/spec.md` (feature
  2, que definió `computeBlockedReasons`/`derivePhase`/`nextRecommendedText`)
  ni ninguna de las specs que ya extendieron `computeBlockedReasons`
  (`verify_record_ledger`, `review_verdict_recorded`,
  `tree_hash_respects_gitignore`): sigue el mismo patrón exacto que esas
  tres — un chequeo más agregado a la lista de `reasons`, con su propio
  "reason code" de substring literal, sin tocar la forma de
  `statusReport` ni agregar campos nuevos al JSON de salida.
- La condición "sin tickets todavía y `status` distinto de `done`" no
  solo protege el pasado (las seis specs pre-feature-13) sino que acota el
  chequeo exactamente al momento que describe la feature: "antes de
  considerar la spec lista para pasar a la fase de tickets". Una vez que
  el desglose de tickets ya existe, el chequeo deliberadamente deja de
  opinar — no es una invariante de "toda spec aprobada debe tener GWT para
  siempre", es una puerta puntual en una transición de fase.
- Dato de contexto para quien implemente: al momento de escribir esta
  spec, las seis specs `sdd:true` `done` que preceden a la feature 13 ya
  tienen tickets en disco (`april_status_arbiter`: 3,
  `verify_record_ledger`: 4, `review_verdict_recorded`: 2,
  `review_frozen_candidate`: 3, `review_depth_by_diff_sensitivity`: 3,
  `tree_hash_respects_gitignore`: 3) — verificado listando
  `specs/*/tickets/*.md` contra el repo real. Ninguna tiene cero tickets,
  así que la condición de aplicación las excluye a todas sin necesitar
  una lista de nombres a mano.
- Esta misma spec (`specs/spec_gwt_mechanical_check/spec.md`) debe
  cumplir, una vez escrita, el chequeo que describe — ya lo hace: las
  historias 1, 2, 3, 4, 5, 6 y 21 arriba incluyen bloques
  `Given/When/Then` reales, así que no hace falta el marcador de opt-out
  en este archivo.
