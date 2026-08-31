## Problem Statement

Hoy (feature 6, `review_verdict_recorded`) `april review record --feature <id>
--verdict <valor>` valida el veredicto contra `hashTree` — un hash agregado
calculado con un *walk* de filesystem (`sha256` por archivo, con tres
exclusiones fijas). Es una mejora real sobre la narración en prosa, pero
sigue teniendo un punto ciego: nada impide que el árbol cambie *mientras*
`reviewer_agent` está revisando, entre que empieza a mirar el código y el
momento en que emite su veredicto. Si eso pasa, el veredicto queda
registrado contra un `treeHash` que sí refleja el árbol final, pero
`reviewer_agent` nunca miró ese árbol final — miró una versión anterior. El
ledger no tiene forma de distinguir "revisé exactamente esto" de "revisé
algo parecido a esto". Es el fallo silencioso que el `ROADMAP.md` nombra
explícitamente citando el RDD de gentle-ai: "el revisor revisó otra cosa".

## Solution

`april review start --feature <id>` calcula un **candidato congelado**:
ejecuta `git write-tree` sobre un índice temporal aislado (no el índice real
del usuario) que refleja el contenido actual del árbol de trabajo —
staged, unstaged y untracked no ignorado — con las mismas dos exclusiones
que ya usa `hashTree` para evitar auto-invalidación (`.claude/verify-ledger.jsonl`,
`progress/`). El resultado es un `subject_hash` (el SHA-1 de árbol que
devuelve `git`, un identificador determinístico: mismo contenido de árbol
⇒ mismo hash) impreso en stdout, sin ningún efecto lateral — no escribe
nada al ledger.

`april review record --feature <id> --verdict <valor>` gana un flag nuevo
**opcional**, `--subject-hash <hash>`. Si se pasa: `review record`
recalcula el candidato actual con el mismo mecanismo, lo compara contra el
`hash` recibido, y si no coincide **rechaza la escritura explícitamente**
("stale subject_hash") sin tocar el ledger — a diferencia de
`no_review_verdict` (feature 6), que reporta la obsolescencia *después*,
en una lectura posterior de `april status`, acá el rechazo ocurre en el
momento mismo de intentar registrar el veredicto. Si el `subject_hash`
sigue vigente, se registra normalmente, con el `subject_hash` guardado en
la entrada del ledger junto al `treeHash` de siempre.

Si **no** se pasa `--subject-hash`, `review record` sigue funcionando
exactamente igual que hoy (feature 6): valida solo contra `hashTree`,
cero cambio de comportamiento. Decisión confirmada explícitamente por el
humano (28/08/2026): el candidato congelado es **opt-in** en esta
feature — no se fuerza su uso modificando el flujo de
`.claude/agents/reviewer_agent.md`; eso, si se decide, es una feature
aparte.

`april review start` requiere que el directorio sea un repositorio git
real y que el binario `git` esté disponible en el `PATH` — confirmado
explícitamente por el humano (28/08/2026) como dependencia dura nueva,
sin fallback silencioso a `hashTree`: si no es un repo git (o `git` no
está disponible), falla explícito en stderr, exit distinto de cero.

## User Stories

1. Como `reviewer_agent`, quiero correr `april review start --feature <id>`
   antes de empezar a revisar, para obtener un `subject_hash` que
   identifica exactamente qué árbol voy a mirar.
2. Como `reviewer_agent`, quiero pasar ese mismo `subject_hash` a
   `april review record --feature <id> --verdict <valor> --subject-hash
   <hash>` al terminar, para que mi veredicto quede ligado al árbol exacto
   que revisé, no a "lo que sea que haya en el árbol cuando corrí
   record".
3. Como humano, quiero que si el árbol cambió entre `review start` y
   `review record --subject-hash <hash>` (alguien tocó código mientras se
   revisaba), el comando rechace explícitamente el registro con un mensaje
   que contenga la substring literal `"stale subject_hash"`, y que **no**
   escriba ninguna entrada al ledger — el intento de registrar un
   veredicto stale no debe dejar rastro como si hubiera sido aceptado.
4. Como humano, quiero que si el `subject_hash` pasado coincide con el
   candidato recalculado en el momento de `review record`, el veredicto se
   registre normalmente, exactamente como si no se hubiera usado
   `--subject-hash` (mismos campos `kind`, `featureId`, `verdict`,
   `treeHash`, `timestamp`, más el `subject_hash` ya guardado).
5. Como desarrollador de April, quiero que `subject_hash` sea
   determinístico — mismo contenido de árbol de trabajo ⇒ mismo hash, sin
   importar el orden en que git recorra los archivos —, para poder
   comparar candidatos de corridas distintas con confianza, igual que ya
   garantiza `hashTree`.
6. Como desarrollador de April, quiero que modificar cualquier archivo no
   excluido del árbol (código, `feature_list.json`, `docs/`, `specs/`,
   etc.) cambie el `subject_hash`, para que el candidato congelado
   realmente refleje el contenido real bajo revisión.
7. Como `reviewer_agent`, quiero que escribir mi bitácora obligatoria en
   `progress/current.md` al terminar de revisar **no** invalide, en el
   mismo ciclo, el `subject_hash` que acabo de usar para registrar mi
   veredicto — misma necesidad que ya resolvió `hashTree` (feature 5) para
   el mismo problema, ahora aplicada al candidato de git.
8. Como desarrollador de April, quiero que anexar una entrada al ledger
   (`.claude/verify-ledger.jsonl`) no cuente como parte del árbol para el
   cálculo de `subject_hash`, para que el propio acto de registrar un
   veredicto no invalide el candidato contra el que se acaba de validar —
   mismo problema exacto que llevó a excluir el ledger del cálculo de
   `hashTree`.
9. Como desarrollador de April, quiero que el mecanismo de captura del
   árbol (índice temporal vía `GIT_INDEX_FILE`, no el índice real del
   usuario) nunca modifique el índice/staging area real de quien corre el
   comando, para que correr `review start` en medio de una sesión de
   trabajo normal (con cambios staged a medias) sea seguro y no
   interfiera con lo que esa persona esté por commitear.
10. Como humano, quiero que si el directorio no es un repositorio git (o
    `git` no está instalado / no está en el `PATH`), `april review start`
    falle explícito en stderr con un mensaje que deje claro que no es un
    repo git, exit distinto de cero, sin devolver ningún hash ni caer en
    un mecanismo alternativo silencioso.
11. Como humano, quiero la misma falla explícita (sin fallback) cuando se
    invoca `review record --subject-hash <hash>` fuera de un repositorio
    git — no tiene sentido validar un candidato de git sin git.
12. Como desarrollador de April, quiero que `review record` **sin**
    `--subject-hash` no requiera que el directorio sea un repositorio
    git — el flujo existente de la feature 6 (solo `hashTree`, stdlib
    puro) sigue funcionando en cualquier directorio, con o sin git.
13. Como desarrollador de April, quiero que los 11 tests existentes de
    `review_test.go` (feature 6) sigan en verde sin modificación —
    evidencia de que `recordReview(featureID, verdict)` (la función que ya
    usan) no cambia de firma ni de comportamiento.
14. Como humano, quiero que `review start --feature <id>` no escriba nada
    al ledger ni a ningún otro archivo — es una consulta pura sobre el
    estado actual del árbol, no un registro; correrlo varias veces seguidas
    sin cambiar nada en el árbol debe imprimir siempre el mismo hash.
15. Como `reviewer_agent`, quiero que la salida de `review start` en éxito
    sea únicamente el `subject_hash` en una línea de stdout (sin texto
    decorativo alrededor), para poder capturarlo directamente en una
    variable de shell (`HASH=$(april review start --feature 7)`) sin
    parsear nada.
16. Como humano, quiero que `review record --subject-hash <hash>` sin
    valor después del flag (o con basura extra después del valor) sea un
    error de invocación explícito, exit distinto de cero, sin tocar el
    ledger — mismo criterio estricto que ya aplican `--feature`/`--verdict`.
17. Como desarrollador de April, quiero que `--feature <id>` sea
    obligatorio también en `review start`, aunque el cálculo del
    `subject_hash` no dependa del id de la feature — por consistencia con
    el resto de subcomandos de `review`/`verify` (todos piden `--feature`
    para trazabilidad/auditoría), no porque el hash lo necesite.
18. Como desarrollador de April, quiero que el vocabulario de `--verdict`
    aceptado por `review record --subject-hash <hash>` sea exactamente el
    mismo (`APPROVED`, `APPROVED_WITH_OBJECTION`, `CHANGES_REQUESTED`) que
    ya valida el camino sin `--subject-hash`, sin una segunda lista de
    valores válidos definida en otro lugar.
19. Como humano, quiero que un `--verdict` fuera de vocabulario con
    `--subject-hash` presente sea rechazado igual que sin él — un valor mal
    escrito no se vuelve válido por venir acompañado de un candidato
    congelado.
20. Como desarrollador de April, quiero que la entrada del ledger que
    resulta de un registro con `--subject-hash` válido tenga tanto
    `treeHash` (el de `hashTree`, sin cambios) como `subjectHash` (el nuevo
    campo), para no perder ninguna de las dos señales — son mecanismos de
    validación distintos y complementarios, no uno reemplaza al otro.
21. Como desarrollador de April, quiero que las entradas del ledger que
    **no** usan `--subject-hash` (todo lo escrito hasta ahora, y todo lo
    que se siga escribiendo sin el flag) serialicen exactamente igual que
    antes — el campo `subjectHash` nuevo debe ser invisible (`omitempty`)
    para no romper `TestLedgerEntrySerializaAlEsquemaExactoDeLaSpec`, que ya
    fija el esquema exacto de las entradas `kind:"test"`.
22. Como humano, quiero que `printUsage()` documente el subcomando nuevo
    (`review start --feature <id>`) y el flag nuevo de `review record`
    (`--subject-hash <hash>`), para descubrirlos sin leer el código.
23. Como desarrollador de April, quiero que `review.go` siga concentrando
    toda la familia de subcomandos `review` (`record`, ahora también
    `start`), en vez de crear un archivo nuevo — mismo criterio de "un
    módulo por responsabilidad" que ya agrupa todo `verify` en `verify.go`,
    donde la responsabilidad es el comando de primer nivel, no cada
    subcomando por separado.
24. Como humano, quiero que `april status`/`computeBlockedReasons`
    (`no_review_verdict`, feature 6) siga evaluando exactamente lo mismo
    que hoy — el `treeHash` de `hashTree` contra el árbol actual — sin
    depender de si una entrada tiene o no `subjectHash`; esta feature no
    toca `status.go`.
25. Como desarrollador de April, quiero que ni `recordReview` (el camino
    sin `--subject-hash`) ni el chequeo de `computeBlockedReasons` exijan
    nunca que el directorio sea un repositorio git — la dependencia dura de
    `git` queda acotada exactamente a `review start` y al camino de
    `review record` que sí recibe `--subject-hash`.
26. Como humano, quiero poder correr `review record --feature <id>
    --verdict <valor>` (sin `--subject-hash`) en un directorio que no es un
    repositorio git, exactamente como podía hacerlo antes de esta feature —
    ninguna regresión de portabilidad para quien no usa el mecanismo nuevo.
27. Como desarrollador de April, quiero que el mecanismo de captura del
    árbol se implemente sobre un índice temporal (`GIT_INDEX_FILE`
    apuntando a un archivo creado con `os.CreateTemp`, nunca el índice
    real `.git/index`), para que ejecutar `review start`/`review record
    --subject-hash` en paralelo con trabajo normal de git (staging manual,
    otro proceso de git corriendo) no produzca condiciones de carrera
    sobre el índice real.
28. Como desarrollador de April, quiero que el archivo de índice temporal
    se borre siempre al terminar (éxito o error), para no acumular
    archivos temporales huérfanos en el sistema tras correr `review
    start`/`review record --subject-hash` muchas veces.
29. Como desarrollador de April, quiero que, si `git` no está en el
    `PATH`, la falla se distinga claramente de "no es un repositorio git"
    solo en el mensaje (ambos casos caen bajo el mismo sentinel
    `ErrNotGitRepo`, pero el texto de error indica cuál de los dos pasó),
    para no tener que adivinar la causa desde el mensaje genérico.
30. Como CI/humano, quiero que `go build ./...` y `go test ./...` sigan en
    verde después de esta feature, incluyendo los tests nuevos que
    invocan `git` real como subproceso — primer precedente del repo de
    depender de `git` instalado también en el entorno de test, no solo en
    producción.

## Implementation Decisions

**No se crea un archivo nuevo.** `review.go` (+ `review_test.go`) se
extiende con el subcomando `start` — misma responsabilidad de "comando
`review`" que ya cubre `record`, mismo criterio que agrupa todo `verify`
en `verify.go` (docs/architecture.md, principio 3).

**Nuevo campo en `ledgerEntry` (en `verify.go`).**

```go
type ledgerEntry struct {
    Kind        string   `json:"kind"`
    FeatureID   int      `json:"featureId"`
    Command     []string `json:"command"`
    ExitCode    int      `json:"exitCode"`
    TreeHash    string   `json:"treeHash"`
    Timestamp   string   `json:"timestamp"`
    Stdout      string   `json:"stdout"`
    Stderr      string   `json:"stderr"`
    Verdict     string   `json:"verdict,omitempty"`
    SubjectHash string   `json:"subjectHash,omitempty"` // nuevo — solo kind:"review" con --subject-hash
}
```

`SubjectHash` lleva `omitempty` por el mismo motivo que `Verdict`: ninguna
entrada existente (ni `kind:"test"`, ni `kind:"review"` sin
`--subject-hash`) lo usa, así que la serialización congelada que ya fija
`TestLedgerEntrySerializaAlEsquemaExactoDeLaSpec` no cambia ni un byte
para esos casos.

**Dos sentinels nuevos, junto al código que los produce (`review.go`)**,
siguiendo `docs/conventions.md`:

```go
var ErrNotGitRepo = errors.New("no es un repositorio git")
var ErrStaleSubjectHash = errors.New("stale subject_hash")
```

`ErrNotGitRepo` cubre tanto "el binario `git` no está disponible en
`PATH`" como "el directorio no es un repositorio git" — un único sentinel,
con el mensaje envolvente (`fmt.Errorf("%w: ...", ErrNotGitRepo, detalle)`)
distinguiendo cuál de los dos pasó en cada caso concreto, sin necesidad de
dos sentinels separados para algo que el llamador trata igual (falla,
sin fallback). `ErrStaleSubjectHash` es exactamente el sentinel que
`docs/conventions.md` ya anticipaba como ejemplo ("aún no implementado")
en su tabla de nombres.

**`computeSubjectHash() (string, error)`** — la función central, en
`review.go`. A diferencia de `hashTree(fsys fs.FS)`, no es inyectable
sobre un `fs.FS` sintético: necesita un repositorio git real y subprocesos
reales, mismo tipo de seam que `recordVerify` (que también corre
`exec.Command` sobre el estado real del proceso). Pasos, todos vía
`exec.Command("git", ...)` con `cmd.Env` extendido a partir de
`os.Environ()` agregando `GIT_INDEX_FILE=<tmp>`:

1. Verifica que `git` esté disponible y que el cwd sea un repositorio:
   `git rev-parse --is-inside-work-tree`. Si el comando ni arranca
   (binario ausente) o termina con exit≠0 (no es un repo), devuelve
   `ErrNotGitRepo` envuelto con el detalle (stderr capturado o el error de
   arranque).
2. Crea un archivo temporal para el índice (`os.CreateTemp("",
   "april-subject-index-*")`), cierra y borra el archivo inmediatamente
   (le interesa solo el *path*, no el contenido — git crea el índice desde
   cero en ese path la primera vez que se usa). `defer os.Remove(path)`
   para garantizar limpieza incluso si un paso posterior falla (US28).
3. `git add -A` con `GIT_INDEX_FILE` apuntando al índice temporal — puebla
   ese índice con todo el contenido actual del árbol de trabajo (staged +
   unstaged + untracked no ignorado por el `.gitignore` del proyecto),
   sin tocar el índice real (`.git/index`) del usuario (US9/US27).
4. `git rm --cached -r --ignore-unmatch -- .claude/verify-ledger.jsonl
   progress` con el mismo `GIT_INDEX_FILE` — remueve del índice temporal
   las mismas dos rutas que `hashTree` ya excluye por el mismo motivo
   (auto-invalidación del ledger/la bitácora, US7/US8). `--ignore-unmatch`
   evita error si esas rutas no existen todavía (ledger nunca escrito,
   `progress/` vacío). No hace falta excluir `.git/` explícitamente: git
   nunca se trackea a sí mismo.
5. `git write-tree` con el mismo `GIT_INDEX_FILE` — imprime en stdout el
   SHA-1 del árbol resultante. Se recorta espacio en blanco/salto de línea
   y ese es el `subject_hash`.

Por qué no alcanza con `.gitignore` solo: el `.gitignore` de este propio
repo excluye `.claude/manifest.json`, `progress/*.md`, `/feature_list.json`,
`/docs/`, `specs/`, pero **no** `.claude/verify-ledger.jsonl`; y el
`templates/.gitignore` que `april init` scaffoldea a proyectos consumidores
solo excluye `specs/`, `tests/`, `session-handoff.md` — ni el ledger ni
`progress/`. Confiar solo en `.gitignore` reintroduciría, en cualquier
proyecto real, el mismo bug de auto-invalidación que la feature 5 ya
resolvió explícitamente para `hashTree`. Por eso las exclusiones se aplican
a mano sobre el índice temporal, con las mismas dos rutas literales que ya
usa `isExcludedFromTreeHash` (sin reusar esa función directamente —
opera sobre rutas de `fs.FS`, no sobre argumentos de `git rm --cached` —
pero sí los mismos dos literales, para que el criterio de "qué cuenta como
cambio en el árbol" sea consistente entre `hashTree` y `subject_hash`).

**`runReviewStart(args []string) int`** — parsea `--feature <id>` (mismo
estilo estricto que el resto de subcomandos: falta el flag, falta el
valor, o `id` no numérico son errores de invocación explícitos, exit≠0,
sin llamar a `computeSubjectHash`). Con parseo válido, llama a
`computeSubjectHash()`; en error, lo imprime en stderr y devuelve exit 1;
en éxito, imprime **solo** el `subject_hash` en una línea de stdout (sin
texto decorativo, US15) y devuelve exit 0. El `id` de `--feature` no
participa del cálculo — se pide únicamente por consistencia con el resto
de la familia `review`/`verify` (US17), no se usa dentro de la función.

**Extensión de `runReviewRecord(args []string) int`.** Los primeros cuatro
argumentos (`--feature <id> --verdict <valor>`) se siguen parseando
exactamente igual que en la feature 6 — cero cambios ahí, es lo que
garantiza que los 11 tests existentes sigan en verde sin tocar una línea
(US13). Lo que sigue después de esos cuatro argumentos (`args[4:]`) se
interpreta así:

- Vacío → comportamiento idéntico a hoy: llama a `recordReview(featureID,
  verdict)` sin cambios.
- Exactamente `["--subject-hash", "<hash>"]` → llama a la función nueva
  `recordReviewWithSubjectHash(featureID, verdict, hash)`.
- Cualquier otra cosa (flag desconocido, `--subject-hash` sin valor,
  argumentos de más después del valor) → error de invocación explícito,
  exit≠0, sin llamar a ninguna de las dos funciones de registro (US16).

**`recordReviewWithSubjectHash(featureID int, verdict, subjectHash string)
(entry ledgerEntry, err error)`** — nueva, en `review.go`. Valida el
vocabulario de `verdict` con el mismo chequeo que ya usa `recordReview`
(se extrae a un helper compartido `isValidVerdict(v string) bool` para no
duplicar el `switch` de tres valores — pequeño refactor de `recordReview`
para reusarlo, sin cambiar su firma ni su comportamiento observable,
US18). Con `verdict` válido, llama a `computeSubjectHash()` para obtener
el candidato **actual**; si difiere del `subjectHash` recibido, devuelve
un error que envuelve `ErrStaleSubjectHash` con ambos hashes en el mensaje
(contiene la substring literal `"stale subject_hash"`, US3), sin tocar el
ledger. Si coincide, calcula también `hashTree(os.DirFS("."))` (igual que
`recordReview`, para no perder esa señal — US20), arma la `ledgerEntry`
con `Kind: "review"`, `TreeHash`, `SubjectHash: subjectHash`, `Verdict`,
`Timestamp`, y hace `appendToLedger(entry)` (reusada tal cual).

Si `computeSubjectHash()` falla dentro de esta función (no es un repo git,
`git` no disponible), el error se propaga tal cual — `review record
--subject-hash <hash>` en un directorio sin git falla explícito, sin
fallback (US11); `review record` **sin** `--subject-hash` nunca llama a
`computeSubjectHash()`, así que sigue funcionando en cualquier directorio
(US12/US26).

**`cmdReview()`** — el `switch` sobre `os.Args[2]` gana el caso `"start"` →
`os.Exit(runReviewStart(os.Args[3:]))`, junto al `"record"` existente.
Cualquier otro valor sigue siendo error explícito.

**`main.go` / `printUsage()`.** Dos líneas nuevas:

```
review start --feature <id>                                   Ejecuta git write-tree, imprime subject_hash (candidato congelado)
review record --feature <id> --verdict <valor> [--subject-hash <hash>]  Registra el veredicto; con --subject-hash, rechaza si el árbol cambió
```

**`status.go` no cambia.** `no_review_verdict` sigue evaluando `TreeHash`
contra `hashTree(fsys)` exactamente como en la feature 6, sin mirar
`SubjectHash` — el rechazo por candidato stale ocurre en `review record`
al momento de escribir, no en una lectura posterior de `april status`
(US24). `derivePhase`/`frontier`/`nextRecommendedText` tampoco cambian.

**`.claude/agents/reviewer_agent.md` no se toca en esta feature** —
decisión explícita del humano (28/08/2026): el mecanismo queda
disponible pero opt-in; forzar su adopción en el flujo de
`reviewer_agent` es una decisión de proceso aparte, con su propia feature
si se decide.

## Testing Decisions

**Primer precedente: subproceso `git` real dentro de tests.** Se agrega un
helper `gitRepoTestDir(t *testing.T) string` en `review_test.go` —
`chdirTemp(t)` (ya existente) + `exec.Command("git", "init", "-q",
".").Run()` (no hace falta `git config user.name/user.email`: `git
write-tree` no necesita autor, a diferencia de un commit). Documentado
explícitamente como nueva asunción de entorno (US30), en el mismo espíritu
que `verify_test.go` ya documentó para `sh -c`.

**Unitario, sobre `computeSubjectHash`:**

- `TestComputeSubjectHashDeterministicoMismoArbolMismoHash` — fixture con
  varios archivos en un repo git real, corre `computeSubjectHash()` dos
  veces sin cambiar nada entre medio, verifica que da el mismo hash.
- `TestComputeSubjectHashCambiaSiElArbolCambia` — corre una vez, modifica
  un archivo no excluido, corre de nuevo, verifica que el hash cambió.
- `TestComputeSubjectHashExcluyeLedgerYProgress` — regresión directa
  del problema que la feature 5 ya resolvió para `hashTree`: computa el
  hash, escribe/modifica `.claude/verify-ledger.jsonl` y
  `progress/current.md`, vuelve a computar, verifica que el hash **no**
  cambió — mismo espíritu que `TestEscribirEnProgressNoInvalidaReceiptVigente`
  de la feature 5, ahora sobre el mecanismo de git.
- `TestComputeSubjectHashFallaSiNoEsRepositorioGit` — `chdirTemp` sin `git
  init`, verifica `errors.Is(err, ErrNotGitRepo)`.
- `TestComputeSubjectHashFallaSiGitNoEstaEnPath` — `gitRepoTestDir` +
  `t.Setenv("PATH", "")`, verifica `errors.Is(err, ErrNotGitRepo)` sin
  panic (US29).
- `TestComputeSubjectHashNoMutaElIndiceReal` — en un repo real, hace
  `git add` de un archivo sobre el índice real (staging normal del
  usuario), corre `computeSubjectHash()`, y verifica con `git diff
  --cached` (o comparando el contenido de `.git/index` antes/después) que
  el índice real no cambió (US9/US27).
- `TestComputeSubjectHashNoDejaArchivoTemporalHuerfano` — corre la
  función y verifica que no queda ningún archivo `april-subject-index-*`
  en el directorio temporal del sistema tras terminar, tanto en éxito como
  forzando un error (US28).

**Integración, sobre `runReviewStart`:**

- `TestRunReviewStartImprimeSubjectHashEnStdout` — `gitRepoTestDir`, corre
  `runReviewStart(["--feature", "7"])`, captura stdout, verifica exit 0 y
  una sola línea no vacía (formato hex de SHA-1, 40 caracteres) sin texto
  decorativo (US14/US15).
- `TestRunReviewStartNoEscribeNadaAlLedger` — mismo caso, verifica que
  `.claude/verify-ledger.jsonl` no se creó (o su hash antes/después es
  idéntico si ya existía) — confirma que es una consulta pura.
- `TestRunReviewStartFaltaFeatureEsErrorDeInvocacion` — sin `--feature`,
  exit≠0, sin invocar `computeSubjectHash` (verificable indirectamente:
  funciona igual con y sin `git init` previo, porque nunca llega a
  necesitar git).
- `TestRunReviewStartFueraDeRepositorioGitFallaExplicito` — `chdirTemp` sin
  `git init`, corre con `--feature` válido, verifica exit≠0 y que stderr
  menciona explícitamente que no es un repositorio git (US10).

**Unitario/integración, sobre `recordReviewWithSubjectHash` /
`runReviewRecord` extendido:**

- `TestRecordReviewWithSubjectHashVigenteAdmiteNormalmente` —
  `gitRepoTestDir`, obtiene el candidato actual vía `computeSubjectHash()`,
  llama a `recordReviewWithSubjectHash(7, verdictApproved, hash)`,
  verifica `err == nil` y que la entrada anexada al ledger tiene
  `SubjectHash == hash`, `TreeHash` no vacío, `Verdict == APPROVED` (US4).
- `TestRecordReviewWithSubjectHashStaleEsRechazado` — obtiene el
  candidato, modifica un archivo no excluido (invalidando ese candidato),
  llama a `recordReviewWithSubjectHash` con el hash viejo, verifica
  `errors.Is(err, ErrStaleSubjectHash)`, que el mensaje contiene la
  substring literal `"stale subject_hash"`, y que el ledger **no** se creó
  (o su contenido antes/después es idéntico) — literal de US3.
- `TestRecordReviewWithSubjectHashVerdictFueraDeVocabularioNoEscribeLedger`
  — verdict inválido con un `subjectHash` cualquiera, verifica error antes
  de siquiera intentar `computeSubjectHash` (o al menos sin tocar el
  ledger) — US19.
- `TestRunReviewRecordConSubjectHashVigenteAdmiteNormalmente` — integración
  vía `runReviewRecord(["--feature", "7", "--verdict", "APPROVED",
  "--subject-hash", hash])`, exit 0.
- `TestRunReviewRecordConSubjectHashStaleRechazaConExitDistintoDeCero` —
  mismo camino con hash desactualizado, exit≠0, ledger sin tocar.
- `TestRunReviewRecordSubjectHashSinValorEsErrorDeInvocacion` — `--subject-hash`
  como último argumento sin valor, exit≠0.
- `TestRunReviewRecordArgumentosDeMasTrasSubjectHashEsErrorDeInvocacion` —
  algo después del valor de `--subject-hash`, exit≠0.
- `TestRunReviewRecordFueraDeRepositorioGitConSubjectHashFallaExplicito` —
  `chdirTemp` sin `git init`, corre con `--subject-hash` presente, verifica
  exit≠0 y mención explícita de que no es un repositorio git (US11).

**Regresión — el camino sin `--subject-hash` no cambia.** No se modifica
ningún test existente de `review_test.go` (los 11 de la feature 6): deben
seguir en verde sin edición, como evidencia de que `recordReview` no
cambió de firma ni de comportamiento (US13). Se agrega, además:

- `TestRunReviewRecordSinSubjectHashFuncionaFueraDeUnRepositorioGit` —
  `chdirTemp` **sin** `git init`, corre `runReviewRecord(["--feature",
  "7", "--verdict", "APPROVED"])` (sin el flag nuevo), verifica exit 0 —
  prueba directa de que esta feature no agrega una dependencia de git al
  camino existente (US12/US26).

**Esquema — reuso del test ya congelado.** No se agrega un test nuevo para
verificar que `subjectHash` no aparece en entradas sin él:
`TestLedgerEntrySerializaAlEsquemaExactoDeLaSpec` (feature 5/6) ya cuenta
las claves exactas de una entrada sin `Verdict`/`SubjectHash` seteados; si
sigue en verde tras el campo nuevo (`omitempty`), ya cubre el caso (US21).

## Out of Scope

- Modificar `.claude/agents/reviewer_agent.md` (o cualquier otro documento
  de flujo) para exigir el uso de `review start`/`--subject-hash` — el
  mecanismo queda opt-in en esta feature, decisión explícita del humano
  (28/08/2026). Adoptarlo como flujo obligatorio es una feature futura, no
  decidida acá.
- Tocar `status.go`/`computeBlockedReasons`/`no_review_verdict` — sigue
  evaluando `treeHash`/`hashTree` exactamente como en la feature 6, sin
  ninguna referencia a `subject_hash`.
- Fallback automático a `hashTree` cuando el directorio no es un
  repositorio git — decisión explícita del humano: falla dura, sin
  degradar silenciosamente a un mecanismo más débil.
- Profundidad de revisión ajustada por sensibilidad del diff (qué
  paquetes/rutas tocó el cambio) — es la feature 8
  (`review_depth_by_diff_sensitivity`), que se apoya en el `subject_hash`
  de esta feature pero no lo construye acá.
- Un flag `--json` o cualquier salida estructurada para `review start` —
  la salida es una sola línea con el hash en texto plano; nada en el
  `acceptance` pide un formato estructurado.
- Registrar en el ledger el propio acto de correr `review start` (un
  `kind` nuevo tipo `"review_start"`) — `review start` es una consulta
  pura, sin efectos en el ledger; solo `review record` escribe.
- Validar que `--feature <id>` corresponda a una feature real de
  `feature_list.json` — mismo criterio permisivo que ya usan `verify
  record`/`review record` (features 5/6).
- Rotación, compactación o locking del ledger más allá de
  `writeFileAtomic` — misma deuda reconocida ya por las features 5/6, no
  crece acá.
- Soportar rutas de trabajo (`cwd`) distintas de la raíz del repositorio
  git — igual que el resto del código (`os.DirFS(".")`), se asume que
  `april` corre desde la raíz.

## Further Notes

- `subject_hash` (SHA-1 de árbol de git) y `treeHash` (SHA-256 agregado de
  `hashTree`) son deliberadamente dos mecanismos distintos que coexisten,
  no uno que reemplaza al otro — mismo espíritu de "coexistencia
  deliberada, no descuido" que ya documentó la feature 6 para
  `reviewVerdict`/ledger. `treeHash` sigue siendo la señal que lee `april
  status` para `no_review_verdict` (post-hoc, cualquier momento del ciclo
  de vida); `subject_hash` es una validación adicional, más estricta,
  que ocurre **en el momento de escribir** el veredicto, solo cuando quien
  invoca `review record` decide usarla.
- Esta es la primera vez que `april` depende de `git` como binario externo
  desde código de producción (antes, `exec.Command` solo corría el comando
  arbitrario que el usuario le pasaba a `verify record`, nunca un binario
  fijo elegido por el propio código) — y, por extensión, la primera vez
  que un test del repo también depende de tener `git` instalado en el
  entorno donde corre `go test`. Es una asunción razonable (este mismo
  repo es un checkout de git), pero queda documentada acá porque no había
  precedente exacto antes de esta feature.
- El uso de un índice temporal vía `GIT_INDEX_FILE` (en vez de operar
  directo sobre `.git/index`) es la pieza que hace posible correr `review
  start`/`review record --subject-hash` en cualquier momento, incluso con
  cambios a medio stagear, sin interferir con el trabajo normal de quien
  esté usando git en paralelo — es, en esencia, la misma técnica que usa
  `git stash create` internamente, aplicada aquí de forma explícita para
  tener control total sobre las exclusiones (`.claude/verify-ledger.jsonl`,
  `progress/`), que un simple `git stash create` no permitiría filtrar.
- Si en el futuro se decide que el `subject_hash` reemplace por completo a
  `treeHash` para veredictos de revisión (unificación, no solo
  coexistencia), o que `reviewer_agent` lo use siempre, son decisiones de
  producto aparte, con su propia spec — exactamente el mismo estatus que
  la feature 6 ya dejó anotado para la posible unificación de
  `reviewVerdict`/ledger.
