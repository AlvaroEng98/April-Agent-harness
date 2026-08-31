## Problem Statement

`hashTree`/`isExcludedFromTreeHash` (`verify.go:87-98`) calculan el
`treeHash` que `april status --json` compara contra el último receipt del
ledger para decidir `no_test_evidence`/`no_review_verdict`
(`status.go:noTestEvidenceReason`/`noReviewVerdictReason`, ambos comparan
`last.TreeHash != currentTreeHash`). Ese `hashTree` recorre **todo** el
árbol de trabajo con `fs.WalkDir` y excluye solo tres rutas fijas
(`.git/`, `.claude/verify-ledger.jsonl`, `progress/`) — nada más. `docs/
verification.md` exige correr `go build ./...` dentro del ciclo de
verificación, y como este repo es un único paquete `main` sin
subdirectorios, ese build reescribe el binario `/HarnessInit` en la raíz
— un archivo real, gitignoreado (`.gitignore: /HarnessInit`), nunca
trackeado por git, pero que `hashTree` sí cuenta porque no está en la
lista fija. El resultado: cualquier `go build ./...` corrido **después**
de un `verify record`/`review record` cambia el `treeHash` del árbol,
sin que ni una línea de código haya cambiado, e invalida la evidencia que
ese mismo receipt acababa de registrar — `blockedReasons` vuelve a
mostrar `no_test_evidence`/`no_review_verdict` en la siguiente lectura de
`april status`. Ya pasó en las features 5, 7 y 8 (documentado por
`reviewer_agent` en `progress/history.md`/`progress/current.md`) y se
resolvió a mano cada vez reordenando el orden de los comandos — un
apaño de proceso, no una corrección del código, que además depende de
que quien opera recuerde el orden correcto.

## Solution

`hashTree` sigue recorriendo el árbol completo vía `fs.WalkDir` sobre el
`fs.FS` que recibe (sin exigir un repositorio git, sin cambiar su firma:
`hashTree(fsys fs.FS) (string, error)`), pero antes de decidir si una
ruta cuenta, además de las tres exclusiones fijas ya existentes, consulta
un parser de `.gitignore` implementado en Go puro (stdlib, sin
dependencias nuevas — `docs/architecture.md`, Principio 2): lee
`.gitignore` del mismo `fsys` (si no existe, cero exclusiones extra, cero
error), lo interpreta línea por línea en una lista de patrones, y excluye
cualquier ruta que matchee alguno. Las tres exclusiones fijas
(`.git/`, el ledger, `progress/`) se preservan tal cual, incondicionales
— no dependen de estar también en `.gitignore` (y de hecho, en este
propio repo, `progress/*.md` sí está en `.gitignore`, pero
`progress/current.md`/`progress/history.md` están **trackeados** pese a
matchear ese patrón, exactamente el caso donde "estar en `.gitignore`" y
"deberías excluirlo del hash" no son la misma pregunta).

`computeSubjectHash` (`review.go`) **no cambia su mecanismo**: ya respeta
`.gitignore` de forma nativa y correcta, porque su primer paso
(`git add -A` sobre el índice temporal) es el propio comportamiento de
git — un archivo untracked y gitignoreado (como `/HarnessInit`) nunca
entra al índice temporal, con o sin esta feature. Esto ya lo documentó
`specs/review_frozen_candidate/spec.md` al explicar `git add -A`
("...untracked no ignorado por el `.gitignore` del proyecto"). El
`git rm --cached` explícito sobre el ledger/`progress/` sigue siendo
necesario exactamente por la misma razón que `hashTree` necesita su
propia lista fija: esas dos rutas pueden estar trackeadas (lo están,
`progress/current.md` hoy) pese a matchear `.gitignore`, y `git add -A`
no las destrackea solo porque aparecen en `.gitignore` después. Esta
feature extrae esos dos literales (`verifyLedgerPath`, `"progress"`) a
una única lista compartida entre `isExcludedFromTreeHash` y el argumento
de `git rm --cached`, para que "qué cuenta como exclusión fija" tenga una
sola fuente — y agrega un test de regresión explícito que confirma (no
que corrige, porque no hay nada que corregir) que `computeSubjectHash` ya
se comporta bien frente al caso `HarnessInit`.

Los dos mecanismos siguen siendo dos implementaciones distintas — una
parseando `.gitignore` en Go puro sobre un `fs.FS` abstracto (`hashTree`,
compatible con `fstest.MapFS`, sin requerir git), la otra delegando en el
propio `git add -A` sobre un repo real (`computeSubjectHash`, ya
correcta) — no se unifican en una sola función compartida: forzar eso
obligaría a `hashTree` a depender de subprocesos de `git` (rompiendo los
tests puros existentes sobre `fstest.MapFS` y la garantía de "no
requiere ser un repo git" que ya fijó la feature 5) o a `computeSubjectHash`
a dejar de usar la semántica real de git a favor de una reimplementación
más débil. Quedan consistentes en **efecto** (ambos respetan
`.gitignore` más las tres exclusiones fijas), no en **una sola
implementación**.

## User Stories

1. Como orquestador, quiero que correr `go build ./...` **después** de un
   `verify record` no cambie el `treeHash` registrado, para que
   `april status --json` no reporte `no_test_evidence` sin que el código
   haya cambiado.
2. Como orquestador, quiero lo mismo para `review record` y
   `no_review_verdict` — el mismo problema, el mismo mecanismo
   (`hashTree`), la misma corrección.
3. Como orquestador, quiero que el orden **inverso** (correr
   `go build ./...` antes de `verify record`/`review record`) también
   quede limpio — no quiero depender de que quien opera recuerde un orden
   "seguro" en vez de que el mecanismo sea correcto en cualquier orden.
4. Como `agent_developer`, quiero poder correr `go build ./...` las veces
   que necesite durante una sesión de implementación, incluso después de
   ya haber registrado evidencia, sin que cada build invalide el receipt
   que acabo de grabar.
5. Como `reviewer_agent`, quiero la misma garantía para mis veredictos
   registrados — correr un build de verificación después de emitir
   `APPROVED` no debe convertir mi veredicto en obsoleto.
6. Como desarrollador de April, quiero que un archivo gitignoreado
   cualquiera (no solo `/HarnessInit`: `*.exe`, `__pycache__/`,
   `.DS_Store`, cualquier patrón real de `.gitignore`) quede excluido del
   `treeHash`, no solo el caso puntual del binario de build — la
   corrección debe generalizar al patrón, no parchear un archivo
   específico.
7. Como desarrollador de April, quiero que modificar un archivo NO
   gitignoreado (cualquier `.go`, `feature_list.json`, `docs/`, `specs/`)
   siga cambiando el `treeHash` exactamente igual que hoy — la corrección
   no debe volver a `hashTree` ciego a cambios reales de código.
8. Como desarrollador de April, quiero que las tres exclusiones fijas
   (`.git/`, el ledger, `progress/`) sigan excluidas **aunque dejen de
   estar en `.gitignore`** el día de mañana (alguien edita el archivo por
   error, o un proyecto scaffoldeado con `april init` usa un
   `templates/.gitignore` más angosto que no las cubre) — no quiero que
   la corrección de esta feature vuelva más frágil una garantía que ya
   existía.
9. Como desarrollador de April, quiero que `hashTree` siga sin requerir
   que el directorio sea un repositorio git ni que el binario `git` esté
   disponible — la lectura de `.gitignore` es un archivo de texto plano,
   no una invocación a git.
10. Como desarrollador de April, quiero que los tres tests existentes de
    `hashTree` en `verify_test.go`
    (`TestHashTreeExcluyeGitProgressYElPropioLedger`,
    `TestHashTreeCambiaSiUnArchivoNoExcluidoCambia`,
    `TestHashTreeEsDeterministicoSinImportarOrden`) sigan en verde sin
    editarlos — ninguno de los tres define un `.gitignore` en su
    `fstest.MapFS`, así que la ausencia de ese archivo debe comportarse
    exactamente igual que hoy (cero patrones extra, cero cambio de
    resultado).
11. Como desarrollador de April, quiero que los tests existentes de
    `computeSubjectHash` en `review_test.go` sigan en verde sin editarlos
    — esta feature no cambia su mecanismo (`git add -A` +
    `git rm --cached`), solo confirma con un test nuevo que ya se
    comporta bien.
12. Como desarrollador de April, quiero al menos un test que reproduzca
    el caso real que originó esta feature: generar un archivo (ej.
    `HarnessInit`), calcular el hash, "regenerarlo" con contenido
    distinto (simulando un rebuild real), recalcular el hash, y verificar
    que ambos hashes coinciden.
13. Como desarrollador de April, quiero un test de extremo a extremo que
    corra `recordVerify` (o el flujo equivalente) y luego regenere el
    artefacto gitignoreado, y verifique con `computeStatusFromFS`/
    `computeBlockedReasons` (no solo con `hashTree` en aislamiento) que
    `no_test_evidence` no reaparece — la garantía que de verdad le
    importa al orquestador es sobre `april status`, no sobre `hashTree`
    como función aislada.
14. Como desarrollador de April, quiero el mismo test en el orden inverso
    (regenerar el artefacto, luego `recordVerify`) para cubrir
    explícitamente el acceptance de la feature que pide ambos órdenes.
15. Como desarrollador de April, quiero que un patrón de `.gitignore`
    anclado a la raíz (`/HarnessInit`, con `/` inicial) NO excluya un
    archivo del mismo nombre en un subdirectorio (ej.
    `sub/HarnessInit`) — replicar la semántica real de anclaje de git, no
    una coincidencia de substring/basename ciega.
16. Como desarrollador de April, quiero que un patrón sin ancla (ej.
    `*.pyc`, `.DS_Store`) excluya coincidencias a cualquier profundidad
    del árbol (`sub/dir/x.pyc`, no solo en la raíz) — misma semántica que
    git aplica a patrones de un solo componente.
17. Como desarrollador de April, quiero que un patrón terminado en `/`
    (ej. `.vscode/`, `specs/`) excluya todo el contenido de ese
    directorio, no solo una entrada exacta con ese nombre.
18. Como desarrollador de April, quiero que patrones intermedios con `/`
    en medio (ej. `progress/*.md`) se interpreten anclados a la raíz
    (regla real de git: cualquier `/` que no sea el final ancla el
    patrón), no como un patrón de un solo componente aplicable a
    cualquier profundidad.
19. Como desarrollador de April, quiero que líneas vacías y comentarios
    (`# ...`) de `.gitignore` se ignoren al parsear, sin producir un
    patrón espurio.
20. Como desarrollador de April, quiero que negación (`!patrón`), `**`, y
    clases de caracteres (`[abc]`) — ninguno presente en el `.gitignore`
    real de este repo hoy — se documenten explícitamente como no
    soportados en vez de fallar silenciosamente con un resultado
    incorrecto sin avisar (ver Out of Scope/Further Notes).
21. Como humano, quiero que la lista de exclusiones fijas
    (`verifyLedgerPath`, `"progress"`) se comparta entre
    `isExcludedFromTreeHash` y los argumentos de `git rm --cached` de
    `computeSubjectHash`, para que "cuáles son las exclusiones fijas" sea
    una sola fuente de verdad en el código, no dos listas de literales
    mantenidas por separado.
22. Como humano, quiero que esta corrección no reintroduzca la opción
    "exclusiones configurables" que ya se descartó explícitamente al
    diseñar `hashTree` (`specs/verify_record_ledger/spec.md`, Out of
    Scope) — `.gitignore` es un artefacto del propio proyecto, versionado
    y ya usado para exactamente esta clase de propósito, no un flag ni un
    archivo de configuración nuevo de `april`.
23. Como humano, quiero que la feature deje explícito, en la propia spec,
    que no contradice `specs/review_frozen_candidate/spec.md` (que ya
    explicó por qué el ledger/`progress/` no pueden depender solo de
    `.gitignore`) — esta feature agrega `.gitignore` como fuente
    **adicional**, no reemplaza el motivo original de esas dos
    exclusiones fijas.
24. Como desarrollador de April que corre `april` sobre un proyecto
    scaffoldeado (no este repo), quiero que la lectura de `.gitignore` de
    `hashTree` funcione igual — lee el `.gitignore` que exista en la raíz
    del proyecto target, sea el que sea, sin asumir el contenido
    específico del `.gitignore` de este repo.
25. Como desarrollador de April, quiero que `parseGitignore`/
    `gitignoreMatches` sean funciones puras sobre `string`/`[]string`
    (sin `fs.FS`, sin I/O), testeables con literales de texto, siguiendo
    el mismo patrón pure-function + wrapper-de-I/O que ya usa
    `parseSensitiveAreas`/`readSensitiveAreas` en `review.go`
    (`status.go`/`review.go`, patrón ya establecido en el repo).
26. Como desarrollador de April, quiero que `go vet ./...`/`gofmt -l .`
    sigan limpios (o al menos no peor que el estado ya conocido de
    `config.go`/`config_test.go`, señalado aparte por `reviewer_agent` en
    la feature 8) tras este cambio.
27. Como humano, quiero que `go build ./...` y `go test ./...` sigan en
    verde después de esta feature — acceptance explícito.

## Implementation Decisions

**Sin archivos nuevos.** El parser de `.gitignore` vive en `verify.go`
(junto a `hashTree`/`isExcludedFromTreeHash`, su único consumidor); el
ajuste de `computeSubjectHash` (extracción de la lista compartida) vive
en `review.go`. Mismo criterio de "un archivo por responsabilidad" que ya
fija `docs/architecture.md` — no se crea `gitignore.go` aparte porque no
hay una segunda responsabilidad detrás, solo una función más del cálculo
de `hashTree`.

**`hashTree(fsys fs.FS) (string, error)` no cambia de firma.** Antes de
recorrer el árbol, carga los patrones de `.gitignore` una sola vez:

```go
func hashTree(fsys fs.FS) (string, error) {
	patterns, err := loadGitignorePatterns(fsys)
	if err != nil {
		return "", fmt.Errorf("leyendo .gitignore: %w", err)
	}
	// ... fs.WalkDir igual que hoy, pero:
	if isExcludedFromTreeHash(p, patterns) {
		return nil
	}
	// ...
}
```

**`isExcludedFromTreeHash` gana un segundo parámetro.** Ningún test
existente la llama directamente (los tres tests de `hashTree` en
`verify_test.go` solo ejercitan `hashTree`, nunca `isExcludedFromTreeHash`
en aislamiento — confirmado leyendo `verify_test.go`), así que cambiar su
firma no rompe nada que ya esté congelado:

```go
func isExcludedFromTreeHash(rel string, patterns []gitignorePattern) bool {
	if rel == verifyLedgerPath {
		return true
	}
	if rel == ".git" || strings.HasPrefix(rel, ".git/") {
		return true
	}
	if rel == "progress" || strings.HasPrefix(rel, "progress/") {
		return true
	}
	return gitignoreMatches(rel, patterns)
}
```

Las tres exclusiones fijas van **primero** y son incondicionales — el
matching de `.gitignore` es un chequeo adicional al final, nunca un
reemplazo (US8/US23).

**`loadGitignorePatterns(fsys fs.FS) ([]gitignorePattern, error)`** — el
wrapper de I/O: lee `.gitignore` de la raíz de `fsys` con
`fs.ReadFile`; si no existe (`fs.ErrNotExist`), devuelve `nil, nil` (cero
patrones, cero error — mismo criterio que `readSensitiveAreas` en
`review.go` para `docs/conventions.md` ausente). Cualquier otro error de
lectura se propaga envuelto. Solo lee el `.gitignore` de la raíz — no hay
`.gitignore` anidados en este repo ni se soportan en esta feature (US24,
Out of Scope).

**`parseGitignore(content string) []gitignorePattern`** — función pura,
testeable con literales de texto (sin `fs.FS`), mismo patrón de dos capas
que `parseSensitiveAreas`/`readSensitiveAreas` (US25). Por línea: recorta
`\r` final, ignora líneas vacías y las que empiezan con `#`; ignora
líneas que empiezan con `!` (negación, no soportada — US20, deja constancia
en un comentario de por qué). Si termina en `/`, marca `dirOnly` y recorta
esa barra. Si empieza con `/`, o si (después de recortar esa barra
inicial) todavía contiene `/`, el patrón queda `anchored` (regla real de
git: cualquier `/` que no sea el último ancla el patrón a la raíz del
`.gitignore` — que en este repo siempre es la raíz del árbol, no hay
`.gitignore` anidados). El resto, sin la barra inicial, es el `glob`
listo para `path.Match`.

Sketch de la estructura y el matching (diseño, no un prototipo corrido —
a validar/ajustar en implementación):

```go
type gitignorePattern struct {
	anchored bool   // "/" al inicio o en medio: se compara desde la raíz
	dirOnly  bool   // terminaba en "/": excluye también todo lo que cuelgue debajo
	glob     string // patrón sin "/" inicial ni final, listo para path.Match
}

func gitignoreMatches(rel string, patterns []gitignorePattern) bool {
	for _, p := range patterns {
		if gitignorePatternMatches(rel, p) {
			return true
		}
	}
	return false
}

func gitignorePatternMatches(rel string, p gitignorePattern) bool {
	if p.anchored {
		if ok, _ := path.Match(p.glob, rel); ok {
			return true
		}
		return p.dirOnly && strings.HasPrefix(rel, p.glob+"/")
	}
	// No anclado: p.glob es un único componente (nunca contiene "/", por
	// construcción de parseGitignore) — matchea el nombre de CUALQUIER
	// segmento de rel (archivo final o directorio intermedio), igual que
	// git trata un patrón sin "/" como "**/patrón" implícito.
	segments := strings.Split(rel, "/")
	for i, seg := range segments {
		ok, _ := path.Match(p.glob, seg)
		if !ok {
			continue
		}
		isLastSegment := i == len(segments)-1
		if !p.dirOnly || !isLastSegment {
			return true
		}
	}
	return false
}
```

`rel` que le llega a `isExcludedFromTreeHash`/`gitignoreMatches` es
siempre una ruta de **archivo** (`hashTree` ya filtra directorios con
`d.IsDir()` antes de llamar a `isExcludedFromTreeHash`), así que un
segmento intermedio que matchea un patrón `dirOnly` siempre es, por
construcción, un directorio real en la ruta — no hace falta consultar el
filesystem de nuevo para confirmarlo.

**`review.go` — lista compartida de exclusiones fijas, sin cambiar el
mecanismo.**

```go
// fixedTreeExclusions son las rutas que tanto isExcludedFromTreeHash
// (hashTree) como computeSubjectHash excluyen sin condicionarlo a
// .gitignore — no dependen de estar ahí (pueden estarlo o no; en este
// repo progress/*.md sí está en .gitignore pero progress/current.md
// sigue trackeado, ver Problem Statement). .git/ no entra en esta lista:
// computeSubjectHash nunca necesita excluirlo a mano porque git no se
// trackea a sí mismo (ver specs/review_frozen_candidate/spec.md).
var fixedTreeExclusions = []string{verifyLedgerPath, "progress"}
```

`isExcludedFromTreeHash` referencia `fixedTreeExclusions[0]`/`[1]` (o,
más legible, dos constantes derivadas) en vez de los literales sueltos
que tiene hoy. El `git rm --cached -r --ignore-unmatch --` de
`computeSubjectHash` arma sus argumentos con
`append([]string{"rm", "--cached", "-r", "--ignore-unmatch", "--"},
fixedTreeExclusions...)` en vez de los dos literales inline. Cero cambio
de comportamiento — mismos dos valores, ahora en un solo lugar.

**Ningún otro cambio en `computeSubjectHash`.** `git add -A` ya respeta
`.gitignore` de forma nativa para archivos untracked (comportamiento de
git, no de April) — no hay una corrección de comportamiento que aplicar
ahí, solo el refactor de la lista compartida de arriba y un test de
regresión nuevo que lo confirma explícitamente (ver Testing Decisions).

**`computeTouchedPaths` (review.go:497) no se toca.** Tiene su propio
chequeo fijo inline (`path == verifyLedgerPath ||
strings.HasPrefix(path, "progress/")`) para una responsabilidad distinta
(qué rutas reporta el diff de sensibilidad de la feature 8) — fuera del
alcance del `acceptance` de esta feature, que habla exclusivamente de
`hashTree`/`isExcludedFromTreeHash` y `computeSubjectHash`. Se señala
explícitamente en Out of Scope para que no se confunda con un tercer
mecanismo que también necesitaría el fix.

**No hay CLI nueva ni cambio de `ledgerEntry`/esquema del ledger** — esta
feature es puramente interna a `hashTree`/`isExcludedFromTreeHash`/
`computeSubjectHash`; `main.go`, `status.go` y el formato del ledger no
cambian.

## Testing Decisions

**Unitario, puro sobre `string`/`[]string` — parser de `.gitignore`.**
Mismo espíritu que `TestParseSensitiveAreas*` en `review_test.go`
(literales de texto, sin `fs.FS`):

- `TestParseGitignoreReconocePatronesBasicos` — sobre un literal con una
  línea de cada clase real del `.gitignore` de este repo (`/HarnessInit`,
  `*.exe`, `.vscode/`, `progress/*.md`, `harness-backend`, una línea en
  blanco, un comentario `# ...`), verifica campo por campo
  (`anchored`/`dirOnly`/`glob`) de cada patrón resultante contra lo
  esperado a mano — valores conocidos, no recalculados por la misma
  lógica que el código bajo test.
- `TestParseGitignoreIgnoraNegacionSinFallar` — una línea `!importante.txt`
  no produce ningún patrón (ni error, ni un patrón incorrecto).
- `TestGitignoreMatchesPatronAncladoSoloEnRaiz` — patrón `/HarnessInit`
  matchea `"HarnessInit"` pero no `"sub/HarnessInit"` (US15).
- `TestGitignoreMatchesPatronSinAnclaCualquierProfundidad` — `*.pyc`
  matchea tanto `"x.pyc"` como `"sub/dir/x.pyc"` (US16).
- `TestGitignoreMatchesDirOnlyExcluyeContenidoCompleto` — `.vscode/`
  matchea `"sub/.vscode/settings.json"` (archivo dentro del directorio,
  no solo el nombre exacto del directorio) (US17).
- `TestGitignoreMatchesPatronConSlashInternoQuedaAnclado` —
  `progress/*.md` matchea `"progress/current.md"` pero NO
  `"otro/progress/current.md"` (US18).

**Unitario, sobre `hashTree` con `fstest.MapFS` — regresión y caso
nuevo.** Mismo patrón que los tres tests ya existentes:

- Los tres tests existentes
  (`TestHashTreeExcluyeGitProgressYElPropioLedger`,
  `TestHashTreeCambiaSiUnArchivoNoExcluidoCambia`,
  `TestHashTreeEsDeterministicoSinImportarOrden`) **no se editan** — su
  `fstest.MapFS` nunca define `.gitignore`, así que `loadGitignorePatterns`
  devuelve `nil` y el comportamiento es idéntico al de hoy (US10).
- `TestHashTreeExcluyeArchivoGitignoreadoAunSinListaFija` (el test que
  reproduce el bug real, US6/US12) — `fstest.MapFS` con un `.gitignore`
  con contenido `"/HarnessInit\n"`, un archivo `"HarnessInit"` con
  contenido A; calcula el hash; sobrescribe `"HarnessInit"` con contenido
  B (mismo archivo, distinto contenido — simula un rebuild real);
  recalcula el hash; verifica que ambos son iguales.
- `TestHashTreeSigueExcluyendoLasTresFijasAunNoEstenEnGitignore` —
  `.gitignore` sintético que **no** menciona `progress/` ni el ledger
  (ej. solo `"*.pyc\n"`); modifica archivos bajo `progress/` y el ledger;
  verifica que el hash no cambia igual — la exclusión fija no depende de
  `.gitignore` (US8/US23).
- `TestHashTreeArchivoNoGitignoreadoSigueCambiandoElHashConGitignorePresente`
  — con un `.gitignore` real presente en el `fstest.MapFS`, modificar un
  archivo que no matchea ningún patrón (ej. `status.go` sintético) sigue
  cambiando el hash (US7) — la variante de
  `TestHashTreeCambiaSiUnArchivoNoExcluidoCambia` pero con `.gitignore`
  en juego, para confirmar que no se volvió permisivo de más.

**Integración, extremo a extremo — `recordVerify`/`computeStatusFromFS`
sobre disco real (`chdirTemp` + `t.TempDir()`, mismo patrón que
`status_test.go`/`verify_test.go`).** Cubre US1/US2/US3/US13/US14 y el
acceptance explícito de ambos órdenes:

- `TestRecordVerifyLuegoRegenerarArtefactoGitignoreadoNoProduceNoTestEvidence`
  — fixture mínimo en disco (`feature_list.json` con una feature
  `in_progress`, `sdd:false`; `.gitignore` con `/build-artifact`); escribe
  `build-artifact` con contenido A; corre `recordVerify`/`sh -c "exit 0"`;
  sobrescribe `build-artifact` con contenido B (simula `go build ./...`
  después del record); corre `computeStatus`/`runStatus --json`; verifica
  que `blockedReasons` NO contiene la substring `no_test_evidence`.
- `TestRegenerarArtefactoGitignoreadoLuegoRecordVerifyNoProduceNoTestEvidence`
  — mismo fixture, orden invertido: escribe `build-artifact` (contenido
  B) antes de `recordVerify`, corre `recordVerify` una sola vez sobre ese
  estado, verifica `blockedReasons` limpio — cubre el orden "seguro" que
  ya funcionaba, para no perder cobertura de que sigue funcionando.
- `TestModificarArchivoNoGitignoreadoLuegoRecordVerifySigueDetectandoNoTestEvidence`
  — mismo fixture, corre `recordVerify`, modifica un archivo de código NO
  gitignoreado (ej. un `.go` del fixture) después, corre `computeStatus`,
  verifica que `blockedReasons` SÍ contiene `no_test_evidence` — la
  corrección no debe volver ciego el mecanismo a cambios reales (US7,
  reafirmado a nivel de `status.go`, no solo de `hashTree` aislado).

**Unitario, sobre `computeSubjectHash` — confirmación, no corrección.**
Con `gitRepoTestDir` (helper ya existente en `review_test.go`):

- `TestComputeSubjectHashYaRespetaGitignoreParaArchivosNoTrackeados` —
  repo git real, `.gitignore` con `/HarnessInit`, escribe `HarnessInit`
  (contenido A, nunca con `git add`), calcula `computeSubjectHash()`,
  sobrescribe `HarnessInit` con contenido B, recalcula, verifica que
  ambos hashes son iguales — documenta explícitamente (US11) que
  `computeSubjectHash` ya se comporta correctamente sin cambios de
  código, solo por el comportamiento nativo de `git add -A`.
- Los tests existentes de `computeSubjectHash` en `review_test.go`
  (`TestComputeSubjectHashDeterministicoMismoArbolMismoHash`,
  `TestComputeSubjectHashCambiaSiElArbolCambia`,
  `TestComputeSubjectHashExcluyeLedgerYProgress`, los de fallo de git/PATH,
  no-mutación del índice real, no-huérfanos) **no se editan**.

**Precedente reusado.** `fstest.MapFS` para lo puro (mismo patrón que
`verify_test.go`/`status_test.go`), `chdirTemp`+`t.TempDir()` para lo que
necesita disco real (mismo patrón que
`TestRecordVerifyComandoExitosoRegistraExitCeroYTreeHash` en
`verify_test.go` y `gitRepoTestDir` en `review_test.go`) — ningún
mecanismo de test nuevo, solo aplicado a este caso.

## Out of Scope

- Semántica completa de `.gitignore`: negación (`!patrón`), `**`
  (doble-asterisco), clases de caracteres (`[abc]`), `.gitignore`
  anidados en subdirectorios, `.git/info/exclude`, `core.excludesFile`
  global. Ninguno aparece en el `.gitignore` real de este repo hoy; si se
  necesitan en el futuro, extender `parseGitignore`/`gitignoreMatches` es
  trabajo de una feature aparte (ver Further Notes).
- Tocar `computeTouchedPaths` (`review.go:497`) — su exclusión fija
  inline es una responsabilidad distinta (diff de sensibilidad de la
  feature 8), fuera del `acceptance` de esta feature.
- Unificar `isExcludedFromTreeHash`/`computeSubjectHash` en una sola
  función/mecanismo compartido — decisión explícita en contra (ver
  Solution): son dos implementaciones distintas por necesidad
  (`fs.FS` abstracto vs. `git` real), consistentes en efecto, no en
  código.
- Reintroducir "exclusiones configurables" (flag o archivo de
  configuración de `april` para ampliar qué se excluye del hash) — sigue
  descartado, igual que en `specs/verify_record_ledger/spec.md`;
  `.gitignore` no es una exclusión configurable por invocación, es un
  artefacto ya versionado del proyecto con un propósito ya establecido.
- Cambiar `templates/.gitignore` (el que `april init` scaffoldea a
  proyectos consumidores) — esta feature cambia el comportamiento de
  `hashTree`/`computeSubjectHash` (leen el `.gitignore` que exista en el
  proyecto donde corren), no el contenido que se scaffoldea.
- Rendimiento de leer/parsear `.gitignore` en cada llamada a `hashTree` —
  se lee una vez por llamada (mismo costo que ya paga
  `readSensitiveAreas` por invocación de `review start --json`), sin
  cachear entre invocaciones del proceso `april` — no hay indicio de que
  sea un problema real a este tamaño de repo.
- Cualquier cambio a `ledgerEntry`, al esquema del ledger, o a la CLI de
  `verify`/`review` — no hace falta ninguno para esta corrección.

## Further Notes

- Esta spec no contradice `specs/verify_record_ledger/spec.md` (feature
  5): esa spec descartó explícitamente "exclusiones configurables"
  (opción C) y "reemplazar el walk por `git ls-files`" (opción B) a favor
  de "walk completo con exclusiones fijas" (opción A). Esta feature
  mantiene el walk completo y las tres exclusiones fijas (siguen ahí,
  incondicionales) y agrega `.gitignore` como una fuente **adicional** de
  exclusión, en Go puro, sin requerir git — no es la opción B (no se
  reemplaza el walk por `git ls-files`) ni la opción C (no es un flag ni
  una lista configurable por quien invoca `april`; es el `.gitignore` que
  el propio proyecto ya versiona). Se documenta el apartamiento porque
  `CLAUDE.md`/el protocolo de `spec_writer` lo exige, no porque haya una
  contradicción real de fondo.
- Tampoco contradice `specs/review_frozen_candidate/spec.md` (feature 7),
  que ya explicó por qué el ledger/`progress/` no pueden depender
  **solo** de `.gitignore` (esas dos rutas pueden estar trackeadas pese a
  matchear un patrón, y de hecho lo están en este repo). Esta feature
  no cambia esa conclusión — la reafirma con un test explícito
  (`TestHashTreeSigueExcluyendoLasTresFijasAunNoEstenEnGitignore`) y
  extrae los dos literales a una lista compartida en vez de duplicarlos.
- El hallazgo más importante de esta spec, para quien la lea más
  adelante: el bug real (auto-invalidación por `go build ./...`) vivía
  **solo** en `hashTree`/`treeHash` (lo único que `status.go` compara
  para `no_test_evidence`/`no_review_verdict`). `computeSubjectHash`/
  `subjectHash` nunca tuvo este problema — `git add -A` ya ignora
  archivos untracked que matchean `.gitignore`, de forma nativa, desde
  que existe la feature 7. La descripción original de la feature (escrita
  por `planner_agent` a partir del hallazgo de `reviewer_agent`) agrupaba
  ambos mecanismos como si tuvieran el mismo defecto; esta spec corrige
  esa premisa explícitamente para que `ticket_writer`/`agent_developer`
  no dediquen esfuerzo a "arreglar" un mecanismo que ya funciona.
- Caso límite documentado, no un bug: si en el futuro alguien agrega al
  `.gitignore` un patrón que matchea un archivo **ya trackeado** en git
  (como pasó con `progress/current.md`), `hashTree` empezará a excluirlo
  del `treeHash` (porque solo mira el filesystem + `.gitignore`, no el
  estado de tracking de git) aunque siga siendo un archivo real y
  versionado. Es el comportamiento esperado dado el diseño ("respetar
  `.gitignore`"), no una regresión — pero vale la pena que quien lo lea
  después sepa que "está en `.gitignore`" y "está excluido del hash" son,
  a partir de esta feature, la misma pregunta para todo excepto las tres
  exclusiones fijas.
- Esta feature es puramente correctiva/interna — no cambia ningún
  comportamiento observable de `april status`/`verify record`/
  `review record` salvo dejar de reportar falsos `no_test_evidence`/
  `no_review_verdict` causados por artefactos gitignoreados. No hay
  superficie de API/CLI nueva que documentar en `printUsage()`.
