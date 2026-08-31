# 02: `fixedTreeExclusions` compartida + confirmación de `computeSubjectHash`

**What to build:** en `review.go`, una única fuente de verdad para "cuáles
son las exclusiones fijas del árbol" — hoy duplicadas como literales
sueltos entre `isExcludedFromTreeHash` (`verify.go`) y los argumentos de
`git rm --cached` de `computeSubjectHash` (`review.go`).

Nueva variable de paquete `fixedTreeExclusions = []string{verifyLedgerPath,
"progress"}`, documentada en comentario: son rutas que se excluyen sin
condicionarlo a `.gitignore` — no dependen de estar ahí (en este repo,
`progress/*.md` sí está en `.gitignore` pero `progress/current.md` sigue
trackeado). `.git/` no entra en esta lista (git nunca se trackea a sí
mismo, no lo necesita `computeSubjectHash`).

El `git rm --cached -r --ignore-unmatch --` de `computeSubjectHash` arma
sus argumentos con `append([]string{"rm", "--cached", "-r",
"--ignore-unmatch", "--"}, fixedTreeExclusions...)` en vez de los dos
literales inline. Cero cambio de comportamiento — mismos dos valores,
ahora en un solo lugar; los tests existentes de `computeSubjectHash`
(`review_test.go`) siguen en verde sin editarlos.

Además, un test nuevo que confirma —no corrige, porque no hay nada que
corregir— que `computeSubjectHash` ya respeta `.gitignore` de forma
nativa para archivos untracked, gracias al comportamiento propio de
`git add -A` (documentado en `specs/review_frozen_candidate/spec.md`).

Building-block deliberado, independiente del parser del ticket 01 (no
necesita `gitignorePattern` para nada): solo toca `review.go` y se
demuestra con tests de `computeSubjectHash` ya existentes (sin editar) más
el test de confirmación nuevo. `isExcludedFromTreeHash` todavía no
referencia esta variable — eso lo hace el ticket 03, que sí depende de
esta.

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] `fixedTreeExclusions` existe como variable de paquete en `review.go`
      con el valor `[]string{verifyLedgerPath, "progress"}`, documentada
      en comentario sobre por qué no dependen de `.gitignore`.
- [ ] El comando `git rm --cached` de `computeSubjectHash` arma sus
      argumentos a partir de `fixedTreeExclusions` (vía `append`), no de
      literales inline duplicados.
- [ ] Los tests existentes de `computeSubjectHash` en `review_test.go`
      (`TestComputeSubjectHashDeterministicoMismoArbolMismoHash`,
      `TestComputeSubjectHashCambiaSiElArbolCambia`,
      `TestComputeSubjectHashExcluyeLedgerYProgress`, los de fallo de
      git/PATH, no-mutación del índice real, no-huérfanos) siguen en
      verde sin ninguna edición.
- [ ] Nuevo test `TestComputeSubjectHashYaRespetaGitignoreParaArchivosNoTrackeados`
      — con `gitRepoTestDir`, repo git real, `.gitignore` con
      `/HarnessInit`, escribe `HarnessInit` (contenido A, nunca con
      `git add`), calcula `computeSubjectHash()`, sobrescribe
      `HarnessInit` con contenido B, recalcula, y verifica que ambos
      hashes son iguales.
- [ ] `go build ./...` y `go test ./...` en verde.
