## Problem Statement

`april review start --feature <id>` (feature 7) ya congela un candidato
(`subject_hash`) y lo imprime, pero no dice nada sobre **qué cambió**. Un
diff que solo toca un comentario en `docs/verification.md` recibe hoy
exactamente la misma señal de entrada que uno que reescribe
`scaffold.go` (el motor que aplica cambios sobre el filesystem del
usuario) o `.github/workflows/` (lo que dispara releases automáticas).
`reviewer_agent` no tiene ninguna pista objetiva, calculada por el
sistema, de que un diff concreto pisa una de esas zonas de alto blast
radius — solo lo nota si lee el diff completo con el mismo cuidado
siempre, sin que nada externo se lo exija ni se lo recuerde. Es la brecha
que el `ROADMAP.md` (E5, extensión del 26/08) ya anticipó: la profundidad
de revisión debería ajustarse a qué tocó el diff, no ser constante.

## Solution

`april review start --feature <id>` gana un flag opcional, `--json`. Sin
él, el comando se comporta exactamente igual que en la feature 7: imprime
en stdout una única línea con el `subject_hash`, sin texto decorativo,
exit 0 — cero cambios de comportamiento para quien ya lo usa así (`HASH=$(april
review start --feature 7)`). Con `--json`, en vez del texto plano imprime
un único objeto JSON con cuatro campos: `subjectHash` (igual que siempre),
`touchedPaths` (la lista de rutas que cambiaron respecto al estado
anterior del repositorio), `sensitiveAreasTouched` (el subconjunto de esas
rutas que coincide con las "Áreas sensibles" de `docs/conventions.md`) y
`extraReviewRequired` (booleano: `true` si `sensitiveAreasTouched` no está
vacío).

El cálculo de qué cambió **reutiliza el candidato congelado que ya
produce `computeSubjectHash`** (feature 7): en vez de construir un segundo
mecanismo de diff en paralelo (walk de filesystem, `git status`, etc.), se
hace un único `git diff --name-only <base> <subject_hash>` — una
comparación de árbol contra árbol, sin tocar el índice ni el working tree,
donde `subject_hash` es literalmente el árbol que `review start` ya
calculó. `<base>` es el árbol de `HEAD` (o el árbol vacío de git si el
repositorio todavía no tiene ningún commit) — "el estado anterior" se
interpreta como el último commit, no el último veredicto de revisión
registrado, para que `review start` siga siendo una consulta pura sin
depender del ledger (mismo principio que ya estableció la feature 7: no
escribe nada, y ahora tampoco *lee* nada del ledger para calcular esto).

Las "Áreas sensibles" se leen en tiempo de ejecución de
`docs/conventions.md` (sección `## Áreas sensibles`, ya confirmada con el
humano el 26/08/2026: `scaffold.go`, `init.sh`, `.github/workflows/`) —
no se hardcodea la lista en Go. Si la sección no existe o el archivo no
existe (proyecto sin ese contenido en su Grill), no es un error: se
interpreta como "ninguna área sensible declarada todavía",
`sensitiveAreasTouched` queda vacío y `extraReviewRequired` es siempre
`false`. La verificación de que **este** repositorio sí tiene la sección
(acceptance #1) es una propiedad del contenido de
`docs/conventions.md`, ya satisfecha y confirmada — no algo que el código
de `review start` valide o exija en tiempo de ejecución.

## User Stories

1. Como `reviewer_agent`, quiero pedir `april review start --feature <id>
   --json` antes de revisar, para saber de antemano qué rutas cambiaron
   sin tener que correr `git diff` yo mismo y adivinar el rango correcto.
2. Como `reviewer_agent`, quiero que el reporte me diga explícitamente si
   el diff tocó una de las áreas sensibles de `docs/conventions.md`
   (`scaffold.go`, `init.sh`, `.github/workflows/`), para dedicarle una
   pasada más cuidadosa a esa feature en particular.
3. Como humano, quiero que `extraReviewRequired` sea `true` si y solo si
   al menos una ruta tocada coincide con una de las áreas sensibles
   declaradas, para no tener una señal que dependa de mi lectura manual
   del diff.
4. Como humano, quiero que `extraReviewRequired` sea `false` (sin ruido
   adicional) cuando el diff no toca ninguna área sensible, para no exigir
   el paso extra por defecto en el caso común.
5. Como desarrollador de April, quiero que `touchedPaths` liste rutas
   relativas al repositorio (mismo formato que ya usa `git diff
   --name-only`: `scaffold.go`, `.github/workflows/release.yml`), para
   poder comparar directamente contra las áreas sensibles sin normalizar
   nada primero.
6. Como desarrollador de April, quiero que "el estado anterior" del diff
   sea el árbol de `HEAD`, no el índice ni el working tree del usuario, y
   no el último veredicto registrado en el ledger, para que `review start`
   siga sin depender de nada más que git y el filesystem — mismo alcance
   de dependencias que ya fijó la feature 7.
7. Como humano, quiero que en un repositorio sin ningún commit todavía
   (`HEAD` no resuelve), `touchedPaths` liste todo el contenido del árbol
   congelado (comparado contra el árbol vacío de git), en vez de fallar o
   devolver una lista vacía engañosa.
8. Como desarrollador de April, quiero que `touchedPaths` excluya
   `.claude/verify-ledger.jsonl` y cualquier ruta bajo `progress/` —
   exactamente las mismas dos exclusiones que ya aplica `computeSubjectHash`
   al construir el árbol congelado —, para que un proyecto donde esas
   rutas sí están commiteadas en `HEAD` (por no seguir todavía la
   convención de excluirlas) no reporte un falso "cambio" solo por el
   propio mecanismo de exclusión del candidato.
9. Como desarrollador de April, quiero que las Áreas sensibles se lean de
   `docs/conventions.md` en tiempo de ejecución, no de una lista fija en
   Go, para que si el humano agrega o quita una ruta sensible en ese
   archivo, `review start` refleje el cambio sin recompilar nada.
10. Como humano, quiero que una ruta sensible que termina en `/` (ej.
    `.github/workflows/`) se interprete como prefijo de directorio —
    cualquier archivo dentro cuenta como tocado —, mientras que una que no
    termina en `/` (ej. `scaffold.go`, `init.sh`) se interprete como
    coincidencia exacta de archivo, para que `status.go` (que no es
    `scaffold.go`) no dispare `extraReviewRequired` por error de
    coincidencia parcial de nombre.
11. Como desarrollador de April, quiero que si `docs/conventions.md` no
    existe, o existe pero no tiene la sección `## Áreas sensibles`, el
    cálculo no falle — simplemente no hay ninguna área sensible que
    coincidir, `extraReviewRequired` es `false` siempre en ese caso —,
    para que proyectos que aún no definieron esa sección (o que nunca la
    definen) puedan seguir usando `review start --json` sin error.
12. Como humano, quiero que correr `april review start --feature <id>`
    (sin `--json`) siga imprimiendo únicamente el `subject_hash` en una
    sola línea de stdout, byte por byte igual que en la feature 7, para
    que ningún script o hábito existente (`HASH=$(april review start
    --feature 7)`) se rompa por esta feature.
13. Como desarrollador de April, quiero que en modo sin `--json` no se
    calcule siquiera `touchedPaths`/áreas sensibles (no solo que no se
    impriman) — mismo camino de código que la feature 7, sin pasos
    nuevos —, para que un error nuevo en el cálculo de sensibilidad (ej.
    `docs/conventions.md` con permisos rotos) nunca rompa el uso existente
    que no pidió `--json`.
14. Como `reviewer_agent`/CI, quiero que la salida `--json` sea un único
    objeto JSON parseable con `json.Unmarshal` sin post-procesamiento,
    con los cuatro campos siempre presentes (`touchedPaths` y
    `sensitiveAreasTouched` como arreglo vacío, nunca `null`, cuando no hay
    nada que listar), para poder consumirla desde un script sin manejar un
    caso especial de "campo ausente".
15. Como humano, quiero que `review start --feature <id> --json` en un
    directorio que no es un repositorio git falle exactamente igual que
    hoy (feature 7): error explícito en stderr mencionando que no es un
    repositorio git, exit distinto de cero, sin intentar calcular
    `touchedPaths` ni áreas sensibles.
16. Como desarrollador de April, quiero que `--feature <id> --json` con
    argumentos de más después, o `--json` en cualquier posición que no sea
    inmediatamente después de `<id>`, sea un error de invocación explícito
    —mismo estilo estricto y posicional que ya usan
    `runReviewRecord`/`runReviewStart` (feature 6/7)—, para no adivinar
    qué quiso decir una invocación ambigua.
17. Como humano, quiero que `review start --feature <id>` sin `--json` y
    con argumentos de más siga siendo rechazado con el mismo criterio
    estricto que ya tenía en la feature 7 (`len(args) > 2` hoy pasa a
    "cualquier `rest` que no sea vacío ni exactamente `["--json"]`").
18. Como desarrollador de April, quiero que `extraReviewRequired` no
    cambie el exit code de `review start` — sigue siendo 0 en éxito
    independientemente de si tocó una ruta sensible o no —, porque
    `review start` es una consulta informativa, no un gate que pueda
    fallar el comando; el gate (si algún día existe) es una decisión de
    proceso aparte.
19. Como desarrollador de April, quiero que esta feature no toque
    `status.go`/`computeBlockedReasons` ni el esquema de `ledgerEntry` —
    `touchedPaths`/`sensitiveAreasTouched`/`extraReviewRequired` no se
    persisten en el ledger ni afectan `blockedReasons`; son datos que
    `review start` calcula y reporta en el momento, no evidencia
    registrada.
20. Como desarrollador de April, quiero que `.claude/agents/reviewer_agent.md`
    no se modifique en esta feature —igual que la feature 7 dejó el
    mecanismo de `subject_hash` opt-in sin forzar su adopción—, para que
    decidir *cómo* `reviewer_agent` reacciona a `extraReviewRequired: true`
    (qué pasos adicionales exige en concreto) quede como una decisión de
    proceso separada, no mezclada con esta pieza de infraestructura.
21. Como CI/humano, quiero que `go build ./...` y `go test ./...` sigan en
    verde, incluyendo los tests nuevos que dependen de `git` real
    (ya establecido como dependencia de entorno desde la feature 7) y que
    ahora además necesitan al menos un commit real dentro del repositorio
    de prueba (no solo `git init`).
22. Como desarrollador de April, quiero que el parseo de
    `## Áreas sensibles` sea tolerante a formato: reconozca cualquier
    línea de lista markdown (`- \`ruta\` — descripción`) bajo ese
    encabezado hasta el siguiente `## ` o el fin de archivo, para que
    agregar o reordenar descripciones en `docs/conventions.md` no rompa el
    parseo mientras la ruta siga entre backticks al inicio del ítem.
23. Como humano, quiero que dos rutas sensibles que aparecen ambas en
    `touchedPaths` (ej. un diff que toca `scaffold.go` y también
    `.github/workflows/release.yml`) queden ambas listadas en
    `sensitiveAreasTouched`, no solo la primera encontrada, para tener el
    detalle completo de qué disparó el requisito.
24. Como desarrollador de April, quiero que `computeTouchedPaths` sea una
    función pura de orquestación sobre `runGit` (mismo seam que ya usa
    `computeSubjectHash`), sin un segundo mecanismo de subprocess/parseo de
    diff, para que ambas funciones compartan exactamente la misma noción
    de "cómo se habla con git" en este archivo.

## Implementation Decisions

**No se crea un archivo nuevo.** Todo se agrega a `review.go` (+
`review_test.go`) — misma responsabilidad ("comando `review`") que ya
cubre `record`/`start`.

**`gitEmptyTreeHash` — constante nueva.** El SHA-1 fijo y bien conocido
del árbol vacío de git (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`), usado
como base de diff cuando el repositorio todavía no tiene ningún commit
(`HEAD` no resuelve). No es un valor calculado por April — es una
constante de git en sí (el árbol vacío siempre tiene ese hash, sin
importar el repositorio), documentada como tal en el código.

**`baseTreeForDiff() (string, error)`** — nueva, junto a `computeSubjectHash`.
Corre `git rev-parse --verify -q HEAD^{tree}` vía `runGit(nil, ...)`. Si
falla (caso esperado: repositorio sin commits todavía — `computeSubjectHash`
ya validó antes que SÍ es un repositorio git válido, así que la única
causa realista de que esta llamada falle es un `HEAD` sin resolver), la
función devuelve `gitEmptyTreeHash` sin propagar el error — es el caso
normal de "primer commit todavía no existe" (US7), no una falla. Cualquier
otro tipo de fallo de git en este punto sería indistinguible de ese caso
por diseño (mismo criterio simple que ya aceptó la feature 7 para
`ErrNotGitRepo`: un solo camino de "no hay base todavía", sin
diferenciar subcasos que el llamador trataría igual).

**`computeTouchedPaths(subjectHash string) ([]string, error)`** — nueva,
junto a `computeSubjectHash`, reusando `runGit` (mismo seam, sin segundo
mecanismo de subprocess, US24):

1. `base, err := baseTreeForDiff()`.
2. `stdout, _, err := runGit(nil, "diff", "--name-only", base, subjectHash)`
   — comparación de árbol contra árbol (dos SHA-1 de `git write-tree`/
   `rev-parse ...^{tree}`), sin necesidad de `GIT_INDEX_FILE` ni de tocar
   el working tree: es una operación pura sobre objetos ya existentes en
   `.git/objects`.
3. Parte `stdout` por líneas, recorta espacio en blanco, descarta líneas
   vacías.
4. Filtra las mismas dos rutas que `computeSubjectHash` ya excluye del
   árbol congelado (`verifyLedgerPath`, cualquier ruta con prefijo
   `"progress/"`) — si esas rutas están commiteadas en `HEAD` en un
   proyecto que todavía no adoptó la convención de excluirlas del propio
   VCS, no deben aparecer como "tocadas" solo porque el candidato
   congelado nunca las incluye (US8).
5. Devuelve la lista filtrada (posiblemente vacía, nunca `nil` — se
   normaliza a `[]string{}` antes de serializar).

Si el `git diff` del paso 2 falla por una razón que no sea la ya cubierta
en `baseTreeForDiff` (ej. corrupción de objetos), el error se propaga tal
cual envuelto con contexto — no hay fallback silencioso.

**Sección "Áreas sensibles" — parseo desde `docs/conventions.md`.**

```go
var sensitiveAreasHeadingRe = regexp.MustCompile(`(?m)^## Áreas sensibles\s*$`)
var nextHeadingRe = regexp.MustCompile(`(?m)^## `)
var sensitiveAreaItemRe = regexp.MustCompile("(?m)^- `([^`]+)`")

func parseSensitiveAreas(content string) []string {
    loc := sensitiveAreasHeadingRe.FindStringIndex(content)
    if loc == nil {
        return []string{}
    }
    rest := content[loc[1]:]
    if end := nextHeadingRe.FindStringIndex(rest); end != nil {
        rest = rest[:end[0]]
    }
    matches := sensitiveAreaItemRe.FindAllStringSubmatch(rest, -1)
    areas := make([]string, 0, len(matches))
    for _, m := range matches {
        areas = append(areas, m[1])
    }
    return areas
}

func readSensitiveAreas(fsys fs.FS) ([]string, error) {
    data, err := fs.ReadFile(fsys, "docs/conventions.md")
    if err != nil {
        if errors.Is(err, fs.ErrNotExist) {
            return []string{}, nil
        }
        return nil, fmt.Errorf("leyendo docs/conventions.md: %w", err)
    }
    return parseSensitiveAreas(string(data)), nil
}
```

Mismo patrón de dos capas que ya usa `status.go`
(`parseTicketStatus`/`parseBlockedBy` puros sobre `string` + `readTickets`
que hace la lectura de `fs.FS`) — `parseSensitiveAreas` es la función pura
testeable con literales de string, `readSensitiveAreas` es el wrapper de
I/O testeable con `fstest.MapFS`. Ausencia del archivo o de la sección no
es error (US11): devuelve lista vacía. Solo un error de lectura genuino
(no `fs.ErrNotExist`) se propaga.

**`matchSensitiveAreas(touched, sensitive []string) []string`** — nueva,
pura:

```go
func matchSensitiveAreas(touched, sensitive []string) []string {
    matched := []string{}
    for _, t := range touched {
        for _, s := range sensitive {
            if strings.HasSuffix(s, "/") {
                if strings.HasPrefix(t, s) {
                    matched = append(matched, t)
                    break
                }
            } else if t == s {
                matched = append(matched, t)
                break
            }
        }
    }
    return matched
}
```

Área sensible terminada en `/` (ej. `.github/workflows/`) es prefijo de
directorio; cualquier otra (ej. `scaffold.go`, `init.sh`) exige
coincidencia exacta de ruta completa — no substring ni prefijo parcial
(US10).

**`reviewStartReport` — struct nueva, solo para el modo `--json`.**

```go
type reviewStartReport struct {
    SubjectHash           string   `json:"subjectHash"`
    TouchedPaths          []string `json:"touchedPaths"`
    SensitiveAreasTouched []string `json:"sensitiveAreasTouched"`
    ExtraReviewRequired   bool     `json:"extraReviewRequired"`
}
```

Nombres de campo en camelCase, consistentes con el resto del JSON que ya
expone April (`statusReport`: `nextRecommended`, `blockedReasons`,
`artifactPaths`; `ledgerEntry`: `featureId`, `treeHash`, `subjectHash`) —
deliberadamente distinto del `"extra_review_required"` en snake_case que
sugería la descripción original de la feature a modo de ejemplo; el
nombre exacto no estaba cerrado, y snake_case rompería la única
convención de nombres JSON que ya tiene el proyecto.

**Extensión de `runReviewStart(args []string) int`.** Los primeros dos
argumentos (`--feature <id>`) se parsean exactamente igual que en la
feature 7 — mismo chequeo de flag/valor/numérico, cero cambios ahí. Lo
que sigue (`args[2:]`) se interpreta así:

- Vacío → comportamiento **idéntico** a la feature 7: llama a
  `computeSubjectHash()`, en éxito imprime solo el hash en stdout, en
  error lo imprime en stderr; nunca llama a `computeTouchedPaths` ni a
  `readSensitiveAreas` (US13). Exit 0/1 sin cambios.
- Exactamente `["--json"]` → llama a `computeSubjectHash()` (falla igual
  que siempre si no es un repo git, US15); en éxito, llama además a
  `computeTouchedPaths(subjectHash)` y `readSensitiveAreas(os.DirFS("."))`,
  arma `reviewStartReport` con `ExtraReviewRequired: len(matched) > 0`, y
  la imprime con `json.MarshalIndent(report, "", "  ")` (mismo formato que
  `runStatus` para `--json`) en una sola línea de `fmt.Println`. Cualquier
  error en cualquiera de los tres pasos se imprime en stderr y devuelve
  exit 1, sin imprimir nada en stdout.
- Cualquier otra cosa (`--json` mal escrito, argumentos de más, `--json`
  seguido de algo) → error de invocación explícito, exit 1, sin llamar a
  ninguna función de cálculo (US16/US17).

**`cmdReview()`/`main.go` no cambian de estructura** — `review start`
sigue siendo el mismo subcomando; solo gana el flag opcional. `printUsage()`
se actualiza:

```
review start --feature <id> [--json]   Ejecuta git write-tree, imprime subject_hash; con --json, agrega touchedPaths/sensitiveAreasTouched/extraReviewRequired
```

**No se toca `status.go`/`computeBlockedReasons`/`ledgerEntry`.** Los tres
campos nuevos existen solo en la salida de `review start`; no se
persisten en `.claude/verify-ledger.jsonl` ni se leen de vuelta en
ningún cálculo de `blockedReasons` (US19).

**`.claude/agents/reviewer_agent.md` no se toca en esta feature** —
mismo criterio que la feature 7: el dato queda disponible, pero decidir
cómo `reviewer_agent` reacciona ante `extraReviewRequired: true` (qué
pasos concretos exige) es una decisión de proceso aparte (US20).

## Testing Decisions

**Seam principal, en orden de preferencia: pure functions sobre `fs.FS`/
`string` antes que subprocess real, y reuso del seam de git ya existente
(`runGit`) antes que uno nuevo.** Solo dos funciones necesitan git real
(`baseTreeForDiff`, `computeTouchedPaths`, indirectamente ejercitadas
también vía `runReviewStart`); el resto (`parseSensitiveAreas`,
`readSensitiveAreas`, `matchSensitiveAreas`) es puro y se testea sin
ningún subproceso, mismo precedente que `parseBlockedBy`/`readTickets`.

**Unitario y puro — `parseSensitiveAreas`:**

- `TestParseSensitiveAreasExtraeRutasDeLaSeccion` — contenido literal con
  la sección real de `docs/conventions.md` (las tres rutas
  `scaffold.go`, `init.sh`, `.github/workflows/`), verifica que
  `parseSensitiveAreas` devuelve exactamente esas tres, en ese orden.
- `TestParseSensitiveAreasVacioSiSeccionAusente` — contenido con otras
  secciones pero sin `## Áreas sensibles`, devuelve `[]string{}` (no
  `nil`, no error — la firma no devuelve error).
- `TestParseSensitiveAreasSeDetieneEnElSiguienteEncabezado` — sección
  "Áreas sensibles" seguida de otra sección con más ítems de lista;
  verifica que los ítems de la sección siguiente NO se cuelan en el
  resultado.
- `TestParseSensitiveAreasIgnoraTextoSinBackticks` — un ítem de lista sin
  ruta entre backticks (ej. una nota aclaratoria) no aporta una entrada
  vacía o basura al resultado.

**Unitario y puro, sobre `fstest.MapFS` — `readSensitiveAreas`:**

- `TestReadSensitiveAreasLeeDocsConventions` — `fstest.MapFS` con
  `docs/conventions.md` sintético conteniendo la sección, verifica el
  resultado.
- `TestReadSensitiveAreasArchivoAusenteNoEsError` — `fstest.MapFS` sin esa
  ruta, verifica `err == nil` y lista vacía.

**Unitario y puro — `matchSensitiveAreas`:**

- `TestMatchSensitiveAreasPrefijoDeDirectorio` — `touched =
  [".github/workflows/release.yml"]`, `sensitive = [".github/workflows/"]`,
  verifica que coincide.
- `TestMatchSensitiveAreasCoincidenciaExactaDeArchivo` — `touched =
  ["scaffold.go"]` vs `sensitive = ["scaffold.go"]` coincide;
  `touched = ["scaffold_test.go"]` vs `sensitive = ["scaffold.go"]`
  **no** coincide (US10: sin match parcial de nombre de archivo).
- `TestMatchSensitiveAreasDevuelveTodasLasCoincidencias` — dos rutas
  tocadas, ambas sensibles, verifica que ambas aparecen (US23), no solo
  la primera.
- `TestMatchSensitiveAreasVacioSinCoincidencias` — ninguna ruta tocada
  coincide, devuelve `[]string{}`.

**Integración con subproceso git real — extensión de `gitRepoTestDir`.**
Nuevo helper en `review_test.go`, `gitRepoWithCommitTestDir(t *testing.T,
files map[string]string) string`: llama a `gitRepoTestDir(t)`, escribe los
archivos indicados, y hace un commit real con autor fijo inline (`git -c
user.email=test@april.dev -c user.name=test commit -q -m "baseline"`, sin
tocar la config global del sistema donde corre el test) — primer
precedente del repo de un test que necesita un commit real, no solo
`git init` (US21).

- `TestComputeTouchedPathsDetectaArchivoModificado` — baseline con
  `a.txt`, modifica `a.txt` después del commit, calcula `subjectHash` y
  `computeTouchedPaths(subjectHash)`, verifica que la lista contiene
  exactamente `"a.txt"`.
- `TestComputeTouchedPathsVacioSinCambios` — baseline commiteado, calcula
  `subjectHash` inmediatamente sin tocar nada, verifica lista vacía.
- `TestComputeTouchedPathsSinCommitsPrevios` — `gitRepoTestDir` (sin
  commit), escribe dos archivos, verifica que `computeTouchedPaths`
  devuelve ambos (diff contra el árbol vacío, US7), sin error.
- `TestComputeTouchedPathsExcluyeLedgerYProgress` — baseline commiteado
  sin ledger ni `progress/`; después del commit, crea
  `.claude/verify-ledger.jsonl` y `progress/current.md` (quedan como
  untracked/nuevos respecto a `HEAD`); verifica que ninguna de las dos
  rutas aparece en `touchedPaths`, aunque sí estén en el árbol de trabajo
  (US8) — regresión directa del mismo problema que `hashTree`/
  `computeSubjectHash` ya resolvieron para otros cálculos.
- `TestComputeTouchedPathsArchivoNuevoNoTrackeadoCuenta` — baseline
  commiteado, agrega un archivo nuevo (nunca trackeado, sin `git add`
  manual), verifica que aparece en `touchedPaths` (consistente con que
  `subject_hash` ya lo incluye vía `git add -A` sobre el índice temporal
  de `computeSubjectHash`).

**Integración, sobre `runReviewStart` extendido:**

- `TestRunReviewStartSinJsonMantieneSalidaDeSoloHash` — reuso exacto del
  fixture/aserciones de `TestRunReviewStartImprimeSubjectHashEnStdout`
  (feature 7) sobre un repo con commit y un cambio real tocando
  `scaffold.go`; verifica que, SIN `--json`, la salida sigue siendo una
  sola línea con el hash, sin ningún rastro de `touchedPaths` ni
  `extraReviewRequired` — prueba directa de US12/US13.
- `TestRunReviewStartJsonReportaTouchedPathsYExtraReviewRequiredTrue` —
  fixture con `docs/conventions.md` conteniendo la sección real de
  "Áreas sensibles" (las tres rutas), commit baseline, modifica
  `scaffold.go` después; corre `runReviewStart(["--feature", "8",
  "--json"])`, parsea el stdout con `json.Unmarshal` a
  `reviewStartReport`, verifica `ExtraReviewRequired == true`,
  `TouchedPaths` contiene `"scaffold.go"`, `SensitiveAreasTouched`
  contiene `"scaffold.go"`, `SubjectHash` no vacío.
- `TestRunReviewStartJsonExtraReviewRequiredFalseSiNoTocaAreaSensible` —
  mismo fixture de `docs/conventions.md`, pero el cambio después del
  commit toca un archivo no sensible (ej. `otra_cosa.go`); verifica
  `ExtraReviewRequired == false` y `SensitiveAreasTouched` vacío,
  mientras `TouchedPaths` sí contiene `"otra_cosa.go"`.
- `TestRunReviewStartJsonSinSeccionDeAreasSensiblesSiempreFalse` —
  `docs/conventions.md` sin la sección (o archivo ausente), modifica
  `scaffold.go`, verifica `ExtraReviewRequired == false` sin error (US11).
- `TestRunReviewStartJsonEsJsonValidoConCamposNuncaNulos` — verifica
  explícitamente, sobre un caso sin cambios, que `touchedPaths` y
  `sensitiveAreasTouched` deserializan como arreglo vacío (`[]`), nunca
  como `null` (US14).
- `TestRunReviewStartJsonFueraDeRepositorioGitFallaExplicito` — sin `git
  init`, corre con `--feature <id> --json`, verifica exit≠0 y stderr
  mencionando explícitamente que no es un repositorio git (US15), sin
  necesidad de llegar a `computeTouchedPaths`.
- `TestRunReviewStartArgumentosDeMasConJsonEsErrorDeInvocacion` — `["--feature",
  "8", "--json", "extra"]`, exit≠0.
- `TestRunReviewStartFlagDesconocidoTrasFeatureEsErrorDeInvocacion` —
  `["--feature", "8", "--jason"]` (typo), exit≠0.

**Regresión — tests existentes de la feature 7 no se tocan.** Todos los
tests de `review_test.go` ya escritos para `runReviewStart`/
`computeSubjectHash` (feature 7) siguen en verde sin modificación —
evidencia de que extender el parseo de `args[2:]` no cambió el contrato
de los primeros dos argumentos.

## Out of Scope

- Enforzar `extraReviewRequired` como un gate automático (ej. que
  `set-status <id> done` o `computeBlockedReasons` lo consulten y bloqueen
  el cierre) — esta feature solo calcula y reporta; convertirlo en un gate
  obligatorio es una decisión de proceso aparte, no decidida acá.
- Modificar `.claude/agents/reviewer_agent.md` para que consuma
  `--json` y ajuste su propia profundidad de revisión en la práctica —
  el dato queda disponible, opt-in, mismo criterio que la feature 7 dejó
  para `subject_hash`.
- Persistir `touchedPaths`/`sensitiveAreasTouched`/`extraReviewRequired`
  en el ledger (`ledgerEntry`) — no se extiende ese esquema en esta
  feature; son datos calculados al vuelo por `review start`, no evidencia
  registrada.
- Extender `templates/docs/conventions.md` (el template que `april init`
  scaffoldea a proyectos nuevos) para incluir por defecto una sección
  `## Áreas sensibles` vacía/`_pendiente_` — el `docs/conventions.md` de
  este propio repo ya tiene la sección rellena (precondición ya
  satisfecha sin reabrir la feature 1); si en el futuro se decide que
  todo proyecto scaffoldeado por April deba preguntar esto en su propio
  Grill, es una extensión del template/skill `grill-docs` aparte, no
  parte de esta spec.
- Detección de renombres (`git diff -M`/similarity detection) en
  `touchedPaths` — se usa `git diff --name-only` sin flags de detección
  de rename; un archivo renombrado aparece como una ruta borrada y una
  agregada, tratado igual que cualquier otro cambio de rutas.
- Cualquier noción de "paquete" más allá de rutas de archivo — el
  proyecto es un único paquete Go (`package main`, `docs/architecture.md`
  principio 1: monolito plano), así que "paquetes tocados" del
  `acceptance` original se resuelve como "rutas de archivo tocadas"; no
  hay granularidad de paquete que reportar.
- Cambiar el exit code de `review start` según `extraReviewRequired` —
  sigue siendo 0 en éxito sin importar el valor de ese campo (US18).
- Cualquier salida en modo sin `--json` más allá de la línea del
  `subject_hash` (ej. una nota en stderr) — se evaluó y se descartó a
  favor de mantener el modo plano bit-a-bit idéntico a la feature 7;
  toda la información nueva vive exclusivamente detrás de `--json`.

## Further Notes

- Esta es la primera vez que April parsea contenido de `docs/*.md` en
  tiempo de ejecución (hasta ahora, `docs/conventions.md`/`architecture.md`
  eran documentos leídos por humanos y agentes vía `Read`, nunca por el
  propio binario). Se documenta acá porque no había precedente exacto:
  el patrón elegido (regex sobre encabezados/ítems de lista markdown)
  reusa exactamente el mismo estilo que ya usa `status.go` para parsear
  campos de tickets (`**Status:**`, `**Blocked by:**`), no uno nuevo.
- `touchedPaths` y `subject_hash` comparten el mismo árbol congelado por
  construcción: cualquier archivo que `computeSubjectHash` habría
  excluido nunca puede aparecer como "tocado", y cualquier archivo que sí
  entra al candidato (incluido untracked no ignorado) sí puede aparecerlo
  — es deliberado que ambos mecanismos vean exactamente el mismo árbol,
  para que no haya una noción de "lo que se congeló" distinta de "lo que
  se reportó como cambiado".
- La comparación contra `HEAD` (en vez de contra el último veredicto
  registrado en el ledger) es una decisión explícita de esta síntesis, no
  dictada por la descripción original de la feature: mantiene `review
  start` sin ninguna dependencia del ledger (paridad con el diseño de la
  feature 7, donde `review start` "no escribe nada al ledger... es una
  consulta pura"). Si en el futuro se decide que "el estado anterior"
  debería ser el último `subject_hash` con el que se registró un
  veredicto (para ver "qué cambió desde la última revisión", no "qué
  cambió desde el último commit"), es una variante distinta con su propia
  spec — ambas nociones son legítimas y no se excluyen mutuamente.
