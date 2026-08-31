# 01: `parseSensitiveAreas`/`readSensitiveAreas` — parseo de "Áreas sensibles"

**What to build:** dos funciones puras y verificables por sí solas,
nuevas en `review.go`, que saben leer la sección `## Áreas sensibles` de
`docs/conventions.md` y devolver la lista de rutas ahí declaradas
(`scaffold.go`, `init.sh`, `.github/workflows/` en este repo, pero
cualquier contenido futuro sin recompilar nada).

`parseSensitiveAreas(content string) []string` opera sobre el texto ya
leído: encuentra el encabezado `## Áreas sensibles`, recorta hasta el
siguiente `## ` (o el fin del archivo), y extrae la ruta entre backticks
de cada ítem de lista markdown (`- \`ruta\` — descripción`) en ese rango.
Si la sección no existe, devuelve `[]string{}` — nunca `nil`, nunca error
(la firma no lo permite). Un ítem de lista sin ruta entre backticks se
ignora silenciosamente, sin aportar una entrada vacía.

`readSensitiveAreas(fsys fs.FS) ([]string, error)` es el wrapper de I/O:
lee `docs/conventions.md` del `fs.FS` dado y delega en
`parseSensitiveAreas`. Si el archivo no existe (`fs.ErrNotExist`), no es
un error — devuelve lista vacía. Cualquier otro error de lectura se
propaga envuelto con contexto.

Building-block deliberado: todavía no hay ningún subcomando de `april`
que use estas funciones (eso llega en el ticket 03). Se demuestra y
verifica por sí solo con tests unitarios puros — sin subproceso, sin
`git` — usando literales de `string` para `parseSensitiveAreas` y
`fstest.MapFS` para `readSensitiveAreas`.

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] `parseSensitiveAreas` sobre el contenido literal real de la sección
      de `docs/conventions.md` de este repo devuelve exactamente
      `["scaffold.go", "init.sh", ".github/workflows/"]`, en ese orden.
- [ ] `parseSensitiveAreas` sobre contenido con otras secciones pero sin
      `## Áreas sensibles` devuelve `[]string{}` (no `nil`, sin error).
- [ ] `parseSensitiveAreas` no incluye ítems de una sección posterior al
      siguiente `## ` — se detiene exactamente en ese límite.
- [ ] `parseSensitiveAreas` ignora un ítem de lista sin ruta entre
      backticks (ej. una nota aclaratoria) sin generar una entrada vacía
      o basura.
- [ ] `readSensitiveAreas` sobre un `fstest.MapFS` con
      `docs/conventions.md` sintético conteniendo la sección devuelve la
      lista esperada.
- [ ] `readSensitiveAreas` sobre un `fstest.MapFS` sin esa ruta devuelve
      `err == nil` y lista vacía (no falla por archivo ausente).
- [ ] `go build ./...` y `go test ./...` en verde.
