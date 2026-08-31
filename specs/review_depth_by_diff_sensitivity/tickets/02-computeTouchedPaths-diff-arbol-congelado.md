# 02: `computeTouchedPaths` — diff de árbol contra el candidato congelado

**What to build:** una función pura de orquestación sobre git,
`computeTouchedPaths(subjectHash string) ([]string, error)`, nueva en
`review.go` junto a `computeSubjectHash`, que calcula qué rutas cambiaron
entre "el estado anterior del repositorio" (el árbol de `HEAD`, o el
árbol vacío de git si todavía no hay ningún commit) y el árbol congelado
que ya representa `subjectHash` (el mismo candidato que produce
`computeSubjectHash`, feature 7).

Reusa `runGit` — mismo seam que `computeSubjectHash`, sin un segundo
mecanismo de subprocess/parseo de diff. Se apoya en dos piezas nuevas:

- `gitEmptyTreeHash`: constante con el SHA-1 fijo y bien conocido del
  árbol vacío de git (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`),
  documentada como valor de git en sí, no calculado por April.
- `baseTreeForDiff() (string, error)`: corre
  `git rev-parse --verify -q HEAD^{tree}`; si falla (repositorio sin
  commits todavía), devuelve `gitEmptyTreeHash` sin propagar el error —
  es el caso normal de "primer commit no existe todavía", no una falla.

`computeTouchedPaths` corre `git diff --name-only <base> <subjectHash>`
(comparación pura de árbol contra árbol, sin tocar índice ni working
tree), parte la salida por líneas, recorta espacio en blanco, descarta
líneas vacías, y filtra las mismas dos rutas que `computeSubjectHash` ya
excluye del árbol congelado (`.claude/verify-ledger.jsonl`, cualquier
ruta bajo `progress/`) — para que un proyecto donde esas rutas sí están
commiteadas en `HEAD` no reporte un falso "cambio" solo por el propio
mecanismo de exclusión del candidato. Devuelve la lista filtrada,
normalizada a `[]string{}` (nunca `nil`) cuando está vacía. Un fallo de
`git diff` que no sea el caso ya cubierto por `baseTreeForDiff` (ej.
corrupción de objetos) se propaga envuelto con contexto, sin fallback
silencioso.

Building-block deliberado: todavía no hay ningún subcomando de `april`
que lo invoque (eso llega en el ticket 03). Se demuestra y verifica por
sí solo con tests que corren `git` real como subproceso, usando un
helper nuevo `gitRepoWithCommitTestDir(t *testing.T, files
map[string]string) string` en `review_test.go` — primer precedente del
repo de un test que necesita un commit real (no solo `git init`), hecho
con autor fijo inline sin tocar la config global del sistema donde corre
el test.

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] `gitEmptyTreeHash` existe como constante con el valor
      `4b825dc642cb6eb9a060e54bf8d69288fbee4904`, documentada como
      constante de git.
- [ ] `baseTreeForDiff()` en un repo con al menos un commit devuelve el
      árbol de `HEAD` (no `gitEmptyTreeHash`).
- [ ] `baseTreeForDiff()` en un repo sin ningún commit todavía devuelve
      `gitEmptyTreeHash` sin error.
- [ ] `gitRepoWithCommitTestDir` existe, crea un repo git real, escribe
      los archivos indicados y hace un commit real (verificable con
      `git log` dentro del test) con autor fijo inline, sin tocar
      `git config --global`.
- [ ] Con un baseline commiteado y un archivo modificado después,
      `computeTouchedPaths(subjectHash)` devuelve exactamente esa ruta
      modificada.
- [ ] Con un baseline commiteado sin ningún cambio posterior,
      `computeTouchedPaths(subjectHash)` devuelve lista vacía.
- [ ] Sin ningún commit previo (`gitRepoTestDir` puro), con archivos
      recién escritos, `computeTouchedPaths(subjectHash)` devuelve todos
      esos archivos (diff contra el árbol vacío), sin error.
- [ ] Con un baseline commiteado sin ledger ni `progress/`, si después
      del commit aparecen `.claude/verify-ledger.jsonl` y
      `progress/current.md` en el árbol de trabajo, ninguna de las dos
      rutas aparece en el resultado de `computeTouchedPaths`.
- [ ] Un archivo nuevo nunca trackeado (sin `git add` manual) sí aparece
      en el resultado.
- [ ] `go build ./...` y `go test ./...` en verde.
