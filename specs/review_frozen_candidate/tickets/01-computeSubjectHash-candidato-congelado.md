# 01: `computeSubjectHash` — candidato congelado sobre índice temporal de git

**What to build:** una función pura y verificable por sí sola,
`computeSubjectHash() (string, error)`, nueva en `review.go`, que calcula
el `subject_hash` — el SHA-1 de árbol que devuelve `git write-tree` — del
estado actual del árbol de trabajo (staged + unstaged + untracked no
ignorado), sin tocar nunca el índice real del usuario (`.git/index`).

Usa un índice temporal aislado vía `GIT_INDEX_FILE` (archivo creado con
`os.CreateTemp`, borrado siempre al terminar — éxito o error) para poblar
un snapshot completo del árbol (`git add -A`) y luego excluir de ese
snapshot las mismas dos rutas que `hashTree` ya excluye por el mismo
motivo (auto-invalidación): `.claude/verify-ledger.jsonl` y `progress/`
(`git rm --cached --ignore-unmatch`). El resultado de `git write-tree`
sobre ese índice temporal, recortado de espacio en blanco, es el
`subject_hash`.

Determinístico (mismo contenido de árbol ⇒ mismo hash), sensible a
cualquier cambio fuera de las dos exclusiones, y con falla explícita
(`ErrNotGitRepo`, sentinel nuevo) si el directorio no es un repositorio
git o el binario `git` no está disponible en el `PATH` — sin fallback
silencioso a `hashTree` ni a ningún otro mecanismo.

Building-block deliberado: todavía no hay ningún subcomando de `april`
que lo invoque (eso llega en los tickets 02 y 03). Se demuestra y se
verifica por sí solo con tests unitarios que corren `git` real como
subproceso, usando un helper nuevo `gitRepoTestDir(t *testing.T) string`
en `review_test.go` (primer precedente del repo de un test que depende de
tener `git` instalado en el entorno).

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] `computeSubjectHash()` existe en `review.go` y devuelve el SHA-1 de
      árbol de `git write-tree` sobre un índice temporal, sin efectos
      colaterales sobre `.git/index` del usuario.
- [ ] Dos corridas sucesivas sin cambiar nada en el árbol dan el mismo
      hash (determinismo).
- [ ] Modificar un archivo no excluido entre dos corridas cambia el hash.
- [ ] Modificar/crear `.claude/verify-ledger.jsonl` o cualquier archivo
      bajo `progress/` entre dos corridas **no** cambia el hash.
- [ ] Correr `computeSubjectHash()` fuera de un repositorio git devuelve
      un error que envuelve `ErrNotGitRepo` (`errors.Is`).
- [ ] Correr `computeSubjectHash()` con `PATH` vacío (`git` no
      disponible) devuelve un error que envuelve `ErrNotGitRepo`, sin
      panic.
- [ ] En un repo real, hacer `git add` manual de un archivo (staging del
      usuario) antes de llamar a `computeSubjectHash()` no altera el
      índice real (`.git/index`/`git diff --cached` sigue igual después
      de la llamada).
- [ ] No queda ningún archivo `april-subject-index-*` huérfano en el
      directorio temporal del sistema tras terminar, tanto en el camino
      de éxito como forzando un error.
- [ ] `go build ./...` y `go test ./...` en verde.
