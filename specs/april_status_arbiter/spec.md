## Problem Statement

Hoy nadie objetivo sabe "en qué fase está" una feature. `CLAUDE.md` dice al
orquestador que "ejecute la fase que toque según el estado" — pero eso es
prosa que el orquestador (un agente) interpreta leyendo `feature_list.json`,
`progress/current.md` y su propia memoria de la conversación. Puede
equivocarse, puede narrar que algo está listo sin que lo esté, y nadie lo
contradice porque no existe una fuente de verdad calculada. `init.sh` valida
una parte (estructura de `feature_list.json`, un solo `in_progress`, specs
presentes cuando se requieren) pero solo responde "válido/inválido" en un
heredoc Python embebido en Bash — nunca dice "qué sigue" ni entiende el
grafo de dependencias entre tickets, y no detecta ciclos.

El humano y el orquestador necesitan una sola pregunta con una sola
respuesta objetiva: *dado lo que hay en disco ahora mismo, en qué fase está
esta feature, qué es lo único legal que sigue, y qué me lo está impidiendo.*

## Solution

Un comando nuevo, `april status [id] --json`, que **lee el disco** —
`feature_list.json`, `specs/<name>/spec.md`, `specs/<name>/tickets/*.md`
(sus campos `Status` y `Blocked by`) — y **calcula** `phase`,
`nextRecommended`, `blockedReasons`, `frontier` y `artifactPaths`. Nunca
escribe nada: es de solo lectura, hoy y siempre mientras el modelo esté en
modo advisory (decisión `ROADMAP.md` 26/08/2026, "B llegando por A").
`init.sh` deja de tener el heredoc Python y en su lugar invoca este
comando; si `blockedReasons` no está vacío, `init.sh` falla explícitamente
en vez de solo imprimir un mensaje genérico.

## User Stories

1. Como orquestador, quiero correr `april status --json` sin argumentos y
   que me diga en qué feature debo enfocarme ahora mismo, para no tener que
   inferirlo leyendo `progress/current.md` a ojo.
2. Como orquestador, quiero que `phase` me diga si estoy en `spec`,
   `tickets`, `implementation`, `review`, `closed` o `grill`, para saber
   qué subagente lanzar sin tener que releer `CLAUDE.md` cada vez.
3. Como orquestador, quiero que `nextRecommended` me dé una única acción
   legal, para no tener ambigüedad sobre qué hacer después.
4. Como orquestador, quiero que si `blockedReasons` no está vacío,
   `nextRecommended` quede vacío, para que nunca reciba una recomendación
   de avanzar cuando hay un problema sin resolver.
5. Como orquestador, quiero correr `april status <id> --json` para una
   feature específica que no es la activa, para poder inspeccionar el
   backlog sin cambiar el foco actual.
6. Como orquestador, quiero que pedir el status de un `id` que no existe en
   `feature_list.json` sea un error claro en stderr, no un JSON con campos
   vacíos, para no confundir "no existe" con "fase desconocida".
7. Como humano, quiero que `blockedReasons` me avise si hay dos features en
   `in_progress` a la vez, para poder corregirlo manualmente antes de que
   el agente avance sobre un estado inconsistente.
8. Como humano, quiero que `blockedReasons` me avise si una feature con
   `sdd: true` tiene `status` en `spec_ready`, `in_progress` o `done` pero
   no existe `specs/<name>/spec.md`, para detectar que alguien saltó la
   puerta de aprobación de spec.
9. Como humano, quiero que `blockedReasons` me avise si el `status` de
   alguna feature no es uno de los valores válidos declarados en
   `feature_list.json.rules.valid_status`, para detectar corrupción o
   ediciones manuales erróneas.
10. Como ticket_writer/orquestador, quiero que un ciclo en los `Blocked by`
    de los tickets de una feature (A bloquea a B, B bloquea a A) se detecte
    y se reporte en `blockedReasons` con los tickets involucrados, para
    poder corregir el desglose antes de que nadie intente implementarlo.
11. Como orquestador, quiero que un ciclo en `Blocked by` no cuelgue el
    comando (nada de recursión infinita), para poder confiar en que
    `april status` siempre termina.
12. Como agent_developer, quiero que `frontier` liste exactamente los
    tickets de la feature activa cuyos `Blocked by` están **todos** en
    `done` y cuyo propio `Status` no es `done`, para saber qué tickets
    puedo tomar en paralelo sin tener que leer y cruzar manualmente cada
    archivo de ticket.
13. Como agent_developer, quiero que un ticket ya `done` nunca aparezca en
    `frontier`, para no reabrir trabajo cerrado.
14. Como agent_developer, quiero que un ticket bloqueado por otro que
    todavía no está `done` no aparezca en `frontier`, para no arrancar
    trabajo fuera de orden.
15. Como orquestador, quiero que una feature `sdd: true` sin
    `specs/<name>/spec.md` reporte `phase: "spec"` y `nextRecommended`
    apuntando a lanzar `spec_writer`, para saber que la Fase Spec no ha
    empezado.
16. Como orquestador, quiero que una feature `sdd: true` con spec pero sin
    ningún archivo en `specs/<name>/tickets/` reporte `phase: "tickets"` y
    `nextRecommended` apuntando a lanzar `ticket_writer`, para saber que
    falta el desglose.
17. Como orquestador, quiero que una feature `sdd: true` con spec, con
    tickets, y al menos un ticket con `Status` distinto de `done` reporte
    `phase: "implementation"`, para saber que toca seguir delegando a
    `agent_developer` sobre la frontera.
18. Como orquestador, quiero que una feature `sdd: true` con spec, con
    tickets, y **todos** los tickets en `Status: done` pero la feature
    misma todavía no `done` reporte `phase: "review"`, para saber que
    corresponde lanzar `reviewer_agent` (o ya se lanzó y falta la puerta
    humana), sin tener que contar checkboxes a mano.
19. Como orquestador, quiero que una feature `sdd: false` (sin spec ni
    tickets) reporte `phase: "implementation"` mientras su `status` sea
    `pending` o `in_progress`, para saber que no hay que esperar ninguna
    fase previa.
20. Como orquestador, quiero que la feature de bootstrap
    (`bootstrap_project`) reporte `phase: "grill"` mientras no esté
    `done`, para diferenciarla de una feature de producto `sdd: false`
    normal.
21. Como orquestador, quiero que una feature con `status: "done"` reporte
    siempre `phase: "closed"`, sin importar qué haya en `specs/`, para que
    el cierre sea la señal final e inequívoca.
22. Como humano, quiero que una feature marcada `blocked` en
    `feature_list.json` se refleje en `blockedReasons` con su motivo
    explícito, para no confundirla con una feature que simplemente no ha
    empezado.
23. Como orquestador, quiero correr `april status --json` en un backlog
    donde ninguna feature está `pending` ni `in_progress` (todo `done`),
    para recibir una respuesta clara de "no hay nada que hacer" en vez de
    un error o un JSON vacío.
24. Como desarrollador de April, quiero que `artifactPaths` me diga
    exactamente qué archivo(s) leer o crear según la fase (`spec.md`
    esperado, directorio de tickets, `feature_list.json`), para no tener
    que adivinar rutas.
25. Como desarrollador de April, quiero que `status.go` reutilice el mismo
    patrón que `scaffold.go` (función pura parametrizada por `fs.FS`,
    wrapper delgado sobre el filesystem real), para que el módulo nuevo se
    sienta consistente con el resto del código.
26. Como humano, quiero que correr `april status` (con o sin `--json`, con
    o sin argumento) nunca modifique ningún archivo del repo, para poder
    correrlo tantas veces como quiera sin miedo a efectos secundarios.
27. Como CI/`init.sh`, quiero que el exit code de `april status --json` sea
    `0` cuando `blockedReasons` está vacío y distinto de `0` cuando no lo
    está, para poder usarlo como gate sin tener que parsear el JSON en
    Bash.
28. Como mantenedor de `init.sh`, quiero que el script siga funcionando
    tanto en este repo (que tiene el código fuente de April y puede no
    tener el binario compilado todavía) como en un proyecto scaffoldeado
    (que solo tiene el binario `april` en el PATH, sin código fuente Go),
    para no romper el bootstrapping de ninguno de los dos casos.
29. Como orquestador, quiero que un ticket con un `Blocked by` que no puedo
    interpretar (no dice "None" ni referencia números de ticket
    reconocibles) se reporte explícitamente en `blockedReasons` en vez de
    ignorarse en silencio, para no calcular una `frontier` incorrecta sin
    saberlo.
30. Como orquestador, quiero que un ticket con un valor de `Status` fuera
    del vocabulario esperado (`pending`/`in_progress`/`done`) se reporte en
    `blockedReasons`, por el mismo motivo.
31. Como humano, quiero poder pedir `april status` sin `--json` y obtener
    una salida legible en texto plano con los mismos datos, para
    inspeccionar el estado desde la terminal sin tener que leer JSON a
    mano.

## Implementation Decisions

**Módulo nuevo:** `status.go` (+ `status_test.go`), siguiendo el mapeo ya
anticipado en `docs/architecture.md`. Sin dependencias nuevas — solo
`encoding/json`, `io/fs`, `os`, `path/filepath`, `strconv`, `strings`.

**CLI:** nuevo caso `"status"` en el `switch` de `main.go` →
`cmdStatus()`. Args: un argumento posicional opcional (el `id` numérico de
la feature, no su `name` — consistente con el uso `april status <id>` que
ya aparece en el `ROADMAP.md` para las features 5 y 6) y una flag `--json`
en cualquier posición, mismo estilo de parseo simple que ya usa `cmdInit`.
Sin `--json`, la salida es texto plano legible con los mismos campos
(pensado para inspección humana en terminal); con `--json`, un único
objeto JSON a stdout, sin adornos. `printUsage()` se actualiza para listar
`status [id] [--json]`.

**Seam principal — función pura parametrizada por `fs.FS`.** Mismo patrón
que `planScaffoldFromFS`/`planScaffold` en `scaffold.go`:

- `computeStatusFromFS(fsys fs.FS, targetID *int) (statusReport, error)` —
  pura, no toca disco real, recibe todo lo que necesita vía `fsys` (que en
  producción es `os.DirFS(".")` sobre la raíz del repo/proyecto, y en tests
  es un `fstest.MapFS` sintético). `targetID == nil` significa "sin
  argumento, elegí vos la feature activa".
- `computeStatus(targetID *int) (statusReport, error)` — wrapper delgado
  que arma `os.DirFS(".")` y llama a la de arriba. Es el único punto que
  `cmdStatus` invoca.

Es el único seam nuevo que este comando necesita — todo lo demás (parseo
de `feature_list.json`, lectura de specs/tickets, detección de ciclos,
formateo JSON/texto) es código interno de `status.go` que se ejercita a
través de este seam.

**Selección de la feature activa (sin argumento).** Mismo criterio que ya
usa el orquestador humano en `CLAUDE.md` paso "Elegir feature": si existe
exactamente una feature en `in_progress`, esa es el target. Si no hay
ninguna, el target es la feature `pending` de menor `id`. Si hay más de
una en `in_progress` (estado inconsistente, ya cubierto por
`blockedReasons`), el target determinístico es la de menor `id` entre las
`in_progress` — el cálculo nunca falla por esto, pero como
`blockedReasons` queda no-vacío, `nextRecommended` se vacía igual (ver
abajo). Si no hay ninguna feature `pending` ni `in_progress` (backlog
íntegramente `done`/`blocked`), no hay target: `phase: "closed"`,
`frontier: []`, `nextRecommended` describe explícitamente que no queda
nada pendiente.

**Resolución de `id` explícito.** Si se pasa un `id` que no existe en
`feature_list.json`, es un error de invocación (no un `blockedReasons`):
`cmdStatus` termina con exit code distinto de 0 y un mensaje en stderr, sin
imprimir JSON — mismo criterio que `cmdInit` usa hoy para errores de
argumento.

**Derivación de `phase`** (una feature, dado su `sdd`, su `status`, y lo
que exista en disco):

| Condición | `phase` |
|---|---|
| `status == "done"` | `closed` |
| `name == "bootstrap_project"` y `status != "done"` | `grill` |
| `sdd == true`, no existe `specs/<name>/spec.md` | `spec` |
| `sdd == true`, existe `spec.md`, sin archivos en `specs/<name>/tickets/` | `tickets` |
| `sdd == true`, existen tickets, al menos uno con `Status != "done"` | `implementation` |
| `sdd == true`, existen tickets, todos con `Status == "done"`, `status != "done"` | `review` |
| `sdd == false` (y no es `bootstrap_project`), `status` en `pending`/`in_progress` | `implementation` |

La feature de bootstrap se identifica por convención de nombre
(`bootstrap_project`) porque es la única prevista por `CLAUDE.md`/
`AGENTS.md` para la Fase Grill — no hay ninguna otra señal en disco que la
distinga de una feature `sdd: false` cualquiera. Si el backlog no tiene
ninguna feature con ese nombre, `grill` simplemente no se reporta nunca (no
es un error).

**Límite reconocido de esta feature:** no hay ninguna señal en disco hoy
que distinga objetivamente "review en curso" de "implementation en curso"
para una feature `sdd: false`, ni para una `sdd: true` cuyos tickets no
están todos `done` — el ledger de evidencia de tests y el veredicto
registrado (features 5 y 6 del `ROADMAP.md`) son los que van a dar esa
señal. Hasta entonces, esos casos se quedan en `implementation` y avanzan
directo a `closed` cuando el humano marca `done` a mano.

**`nextRecommended`.** Cadena única, vacía (`""`) si y solo si
`blockedReasons` no está vacío — la regla explícita de gentle-ai
("no se avanza") se implementa así: nunca hay una recomendación de avanzar
mientras haya un `blockedReason` sin resolver. Con `blockedReasons` vacío,
el texto describe la única acción legal según `phase` (lanzar
`spec_writer`, lanzar `ticket_writer`, implementar la `frontier`, lanzar
`reviewer_agent`, o "nada — la feature ya está cerrada" / "nada — no hay
features pendientes").

**`blockedReasons`.** Array de strings, siempre calculado sobre **todo**
`feature_list.json` y todo lo que exista en `specs/`, sin importar qué
`id` se haya pedido — es el mismo alcance "global" que hoy tiene la
validación de `init.sh`, y absorbe exactamente esas cuatro comprobaciones
más una nueva:

1. Más de una feature en `in_progress`.
2. `status` de alguna feature fuera de `feature_list.json.rules.valid_status`.
3. Feature `sdd: true` con `status` en `spec_ready`/`in_progress`/`done`
   sin `specs/<name>/spec.md`.
4. Feature marcada `blocked` (se reporta igual, con su `id`/`name`, aunque
   no impida calcular `phase` para las demás).
5. **Nuevo:** ciclo en el grafo de `Blocked by` de los tickets de cualquier
   feature — detectado con DFS + pila de recursión (nunca recursión sin
   límite: el grafo tiene como máximo tantos nodos como archivos de
   ticket). El mensaje incluye la feature y la cadena de tickets que forma
   el ciclo (ej. `"ciclo detectado en Blocked by de tickets de
   april_status_arbiter: 02 → 03 → 02"`).
6. **Nuevo:** un ticket cuyo `Blocked by` no se puede interpretar (ver
   convención de parseo abajo), o cuyo `Status` no es
   `pending`/`in_progress`/`done`.

**Convención de parseo de `Blocked by` en tickets.** El campo es texto
libre en la plantilla de `ticket_writer` ("números/títulos... o 'None'"),
pero `ticket_writer` ya numera los archivos `<NN>-<slug>.md` "en orden de
dependencia" — por lo tanto el parser busca, dentro del texto de
`Blocked by`, todos los números de dos dígitos (`\d{2}`) y los interpreta
como el `NN` de los tickets que bloquean a este. Si el texto (sin
distinguir mayúsculas/minúsculas) contiene "none" y ningún número de dos
dígitos, el ticket no tiene bloqueadores. Cualquier otro caso (texto sin
número y sin "none", o números que no corresponden a ningún archivo de
ticket existente) cae en el punto 6 de `blockedReasons` de arriba en vez de
romper el cálculo o asumir "sin bloqueadores" en silencio. Esta convención
no cambia la plantilla de `ticket_writer` — ya era compatible, solo se fija
cómo se interpreta.

**`frontier`.** Array de identificadores de ticket (su `NN-slug`) de la
feature target: los que tienen **todos** sus `Blocked by` (ya resueltos a
`NN`) en `Status: done`, y su propio `Status` distinto de `done`. Vacío en
cualquier fase que no sea `implementation` con tickets existentes (spec,
tickets sin desglose, grill, closed, `sdd: false` sin tickets).

**`artifactPaths`.** Objeto con las rutas relevantes según fase: siempre
`"featureList": "feature_list.json"`; si `sdd: true`,
`"spec": "specs/<name>/spec.md"` (la ruta esperada, exista o no, para que
quien reciba el JSON sepa dónde crearla); si hay tickets,
`"ticketsDir": "specs/<name>/tickets/"` y `"tickets": [...]` con cada
archivo encontrado.

**Exit code y no-escritura.** `cmdStatus` termina con exit code `0` si
`blockedReasons` está vacío, `1` si no lo está (ya sea con o sin `--json`)
— así `init.sh` lo puede usar como gate sin parsear el JSON. El comando no
llama nunca a `os.WriteFile`/`os.Remove`/`os.MkdirAll`: toda la lectura
pasa por `fs.ReadFile`/`fs.ReadDir` sobre el `fs.FS` recibido.

**`init.sh`.** Se borra el bloque `python3 - <<'PY' ... PY` completo (líneas
62-98 actuales) y la sección "2. Validando feature_list.json y specs" pasa
a resolver el binario `april` con esta prioridad: (1) `april` en el PATH
— el caso normal en un proyecto scaffoldeado, que no tiene el código
fuente de April, solo el binario instalado; (2) si no está en el PATH pero
existen `go.mod` y `main.go` en el directorio actual (el caso de este
mismo repo dogfoodeando su propio arnés, donde puede no haber un binario
compilado todavía), lo compila on-the-fly (`go build`) a un binario
temporal y usa ese. Si ninguna de las dos condiciones se cumple, `init.sh`
falla explícitamente indicando que no pudo resolver el comando `status`.
Corre `april status --json`, imprime su salida, y traduce el exit code al
`EXIT_CODE` que ya maneja el resto del script (mismo patrón `[OK]`/`[FAIL]`
que las demás secciones).

## Testing Decisions

**Unitario, sobre el seam puro (`computeStatusFromFS`).** Mismo precedente
que `TestPlanScaffoldIsPure` y el resto de `scaffold_test.go`:
`fstest.MapFS` sintético con `feature_list.json`, `specs/<name>/spec.md` y
`specs/<name>/tickets/*.md` fabricados a mano, sin tocar disco real. Casos
a cubrir uno por uno (nombres en español, patrón mayoritario ya establecido
en `scaffold_test.go`):

- `TestFeatureSddSinSpecEsFaseSpec`
- `TestFeatureConSpecYSinTicketsEsFaseTickets`
- `TestFeatureConTicketsPendientesEsFaseImplementation`
- `TestFeatureConTodosLosTicketsDoneEsFaseReview`
- `TestFeatureDoneEsFaseClosedSinImportarDisco`
- `TestBootstrapProjectPendienteEsFaseGrill`
- `TestFeatureSddFalseSinBootstrapEsFaseImplementation`
- `TestDosFeaturesInProgressReportaBlockedReasons`
- `TestStatusInvalidoEnFeatureListReportaBlockedReasons`
- `TestSpecRequeridaYFaltanteReportaBlockedReasons`
- `TestFeatureBlockedSeReportaEnBlockedReasons`
- `TestCicloEnBlockedByDeTicketsSeDetectaYNoCuelga` (arma A→B→A con dos o
  tres tickets, corre con un timeout de test explícito o cuenta de nodos
  visitados para probar que termina)
- `TestFrontierListaSoloTicketsConBloqueadoresEnDone`
- `TestFrontierExcluyeTicketsYaDone`
- `TestBlockedByConTextoNoInterpretableReportaBlockedReasons`
- `TestStatusDeTicketFueraDeVocabularioReportaBlockedReasons`
- `TestNextRecommendedVacioCuandoHayBlockedReasons`
- `TestSinFeaturePendienteNiInProgressReportaClosedSinTarget`
- `TestSeleccionaFeatureIdExplicitoAunqueNoSeaLaActiva`
- `TestIdInexistenteEsErrorDeInvocacion` (verifica que no crashea con pánico
  y que el error identifica el `id`)

**Integración, sobre `cmdStatus` con disco real.** `t.TempDir()` +
`captureStdout` (ya existe en `scaffold_test.go`, se reutiliza tal cual):

- `TestCmdStatusJsonEsValidoYTieneLosCincoCampos` — corre contra un fixture
  real en disco, decodifica el JSON de salida y verifica presencia de
  `phase`, `nextRecommended`, `blockedReasons`, `frontier`,
  `artifactPaths`.
- `TestCmdStatusExitCodeReflejaBlockedReasons` — un fixture limpio sale 0,
  uno con dos `in_progress` sale distinto de 0.
- `TestCmdStatusNoEscribeNingunArchivo` — hashea (`sha256`, mismo helper
  que ya usa `scaffold.go`) el árbol completo del fixture antes y después
  de correr `computeStatus`/`cmdStatus`, y verifica que los hashes son
  idénticos. Es la prueba directa del criterio de aceptación
  correspondiente.

**Regresión sobre `init.sh` en sí (no un `go test` de comportamiento, sino
un guardarraíl barato).** `TestInitShInvocaAprilStatusSinHeredocPython` lee
el `init.sh` real del repo (mismo patrón que otros tests leen archivos
reales del repo, ej. el `embed.FS` en `scaffold_test.go`) y verifica por
contenido: no contiene `<<'PY'`, sí contiene `status`. No reemplaza la
revisión humana del diff de `init.sh` (área sensible, ver
`docs/conventions.md`) — la complementa como red de seguridad barata contra
que alguien reintroduzca el heredoc más adelante sin darse cuenta.

**Precedente para "no reventar con ciclos".** No hay precedente de
detección de ciclos en este código todavía (`scaffold.go` no tiene grafos);
se usa el algoritmo estándar DFS+pila-de-recursión con un set de "visitado
en esta rama", acotado por construcción al número de archivos de ticket
leídos — no hay estructura de datos externa ni límite artificial de
profundidad que mantener.

## Out of Scope

- Escribir `feature_list.json` — modo advisory, decisión explícita del
  `ROADMAP.md` (26/08/2026). Ninguna forma de autocorrección o "arreglo
  automático" de `blockedReasons`.
- `april feature set-status` (feature 4) — la única vía de escritura
  futura, bloqueada además por confirmación humana explícita de que este
  modelo de fases resultó confiable en uso real.
- El ledger de evidencia de tests (`april verify record`, feature 5) y el
  registro de veredicto de `reviewer_agent` (feature 6) — hasta que
  existan, esta feature no puede distinguir objetivamente "implementation"
  de "review" para features sin tickets, ni confirmar que una feature
  `done` de verdad tuvo tests/revisión (ver límite reconocido arriba).
- `subject_hash`/candidato congelado (feature 7), profundidad de revisión
  por sensibilidad de diff (feature 8), `april doctor` (feature 9), backup
  de `april init` (feature 10), ratchet de deuda (feature 11) — todas
  siguientes en el `ROADMAP.md`, ninguna depende de que esta spec las
  resuelva.
- Cambiar la plantilla de ticket de `ticket_writer.md` — la convención de
  parseo de `Blocked by` (números de dos dígitos) ya es compatible con la
  plantilla existente ("números/títulos... o 'None'"); no hace falta
  tocar ese archivo para que este comando funcione.
- Cualquier UI más allá de texto plano/JSON por stdout — nada de colores
  elaborados, tablas ANSI, ni modo interactivo.
- Soportar un segundo criterio de selección de "feature activa" (por
  ejemplo, por `name` en vez de `id`) — se decide `id` únicamente, igual
  que las features 5 y 6 del `ROADMAP.md` ya asumen.

## Further Notes

- `docs/architecture.md` ya anticipaba `status.go` como módulo futuro con
  este mismo nombre y responsabilidad — esta spec no introduce ningún
  módulo fuera de lo ya previsto ahí.
- El texto de `init.sh` tiene hoy una referencia residual a
  `'harness init .'` (línea 34) que debería decir `april init .` — es un
  bug preexistente, no introducido por esta spec, y queda fuera de su
  `acceptance`; se señala acá para que quien implemente lo corrija de paso
  si toca esa misma sección del script (barato, mismo archivo, mismo
  commit) pero no es un criterio de cierre de esta feature.
- La feature 8 (`review_depth_by_diff_sensitivity`) ya identificó
  `init.sh` como área sensible en `docs/conventions.md`; esta feature toca
  justamente esa área, así que la revisión de `reviewer_agent` debería ser
  más profunda ahí (aunque el mecanismo formal de "profundidad ajustada"
  todavía no existe — es la feature 8, no esta).
- Si en el futuro se agregan más valores a `phase` (por ejemplo, algo entre
  `review` y `closed` una vez exista el ledger de veredictos de la feature
  6), este documento debería tratarse como ADR de la versión actual del
  modelo de fases — quien lo extienda debe señalar explícitamente en su
  propia spec que se desvía de esta tabla, no reemplazarla en silencio.
