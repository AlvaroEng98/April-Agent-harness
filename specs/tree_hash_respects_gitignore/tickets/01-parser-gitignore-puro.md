# 01: Parser de `.gitignore` en Go puro

**What to build:** funciones puras y verificables por sí solas, nuevas en
`verify.go`, que saben interpretar el texto de un `.gitignore` y decidir
si una ruta relativa lo matchea, sin ninguna dependencia de `hashTree`
todavía.

`parseGitignore(content string) []gitignorePattern` recorre el contenido
línea por línea: recorta `\r` final, ignora líneas vacías y comentarios
(`#`), ignora líneas de negación (`!patrón` — no soportada, se deja
constancia en comentario de por qué). Si una línea termina en `/`, marca
`dirOnly` y recorta esa barra. Si empieza con `/`, o si (después de
recortar esa barra inicial) todavía contiene `/`, el patrón queda
`anchored` (regla real de git: cualquier `/` que no sea el último ancla
el patrón a la raíz). El resto, sin la barra inicial, queda como `glob`
listo para `path.Match`.

`gitignoreMatches(rel string, patterns []gitignorePattern) bool` recorre
los patrones y delega en `gitignorePatternMatches` (helper interno): si el
patrón es `anchored`, compara `rel` completo contra el `glob` (más el caso
`dirOnly` con prefijo `glob+"/"`); si no es `anchored`, compara cada
segmento de `rel` por separado (semántica de un componente sin `/`
matchea a cualquier profundidad, como el `**/patrón` implícito de git),
respetando `dirOnly` solo cuando el segmento matcheado no es el último.

`loadGitignorePatterns(fsys fs.FS) ([]gitignorePattern, error)` es el
wrapper de I/O — mismo patrón dos-capas que
`parseSensitiveAreas`/`readSensitiveAreas` (`review.go`): lee `.gitignore`
de la raíz de `fsys` con `fs.ReadFile` y delega en `parseGitignore`. Si el
archivo no existe (`fs.ErrNotExist`), no es un error — devuelve `nil, nil`
(cero patrones extra). Cualquier otro error de lectura se propaga
envuelto.

Building-block deliberado: todavía no hay ningún llamador real
(`hashTree` se conecta en el ticket 03). Se demuestra y verifica por sí
solo con tests unitarios puros sobre literales de `string` para
`parseGitignore`/`gitignoreMatches`, y `fstest.MapFS` para
`loadGitignorePatterns` — sin walk de árbol, sin `git`.

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] `parseGitignore` sobre un literal con una línea de cada clase real
      del `.gitignore` de este repo (`/HarnessInit`, `*.exe`, `.vscode/`,
      `progress/*.md`, `harness-backend`, una línea en blanco, un
      comentario `# ...`) produce, campo por campo
      (`anchored`/`dirOnly`/`glob`), exactamente los patrones esperados a
      mano.
- [ ] `parseGitignore` sobre una línea `!importante.txt` no produce ningún
      patrón (ni error, ni un patrón incorrecto).
- [ ] `gitignoreMatches` con patrón `/HarnessInit` matchea `"HarnessInit"`
      pero no `"sub/HarnessInit"` (anclaje a raíz).
- [ ] `gitignoreMatches` con patrón `*.pyc` matchea tanto `"x.pyc"` como
      `"sub/dir/x.pyc"` (sin ancla, cualquier profundidad).
- [ ] `gitignoreMatches` con patrón `.vscode/` matchea
      `"sub/.vscode/settings.json"` (contenido completo del directorio,
      no solo el nombre exacto).
- [ ] `gitignoreMatches` con patrón `progress/*.md` matchea
      `"progress/current.md"` pero NO `"otro/progress/current.md"`
      (`/` intermedio ancla el patrón a la raíz).
- [ ] `loadGitignorePatterns` sobre un `fstest.MapFS` sin `.gitignore`
      devuelve `nil, nil` (cero patrones, cero error).
- [ ] `loadGitignorePatterns` sobre un `fstest.MapFS` con `.gitignore`
      sintético devuelve los mismos patrones que produciría `parseGitignore`
      sobre ese mismo contenido.
- [ ] `go build ./...` y `go test ./...` en verde.
