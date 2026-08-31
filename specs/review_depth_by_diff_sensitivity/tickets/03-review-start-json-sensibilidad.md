# 03: `april review start --feature <id> --json` — reporte de sensibilidad del diff

**What to build:** `reviewer_agent` (o cualquier humano) corre
`april review start --feature <id> --json` antes de revisar y recibe en
stdout un único objeto JSON con cuatro campos: `subjectHash` (igual que
siempre), `touchedPaths` (rutas que cambiaron respecto al último commit,
o respecto al árbol vacío si no hay ninguno todavía), `sensitiveAreasTouched`
(el subconjunto de esas rutas que coincide con las "Áreas sensibles"
declaradas en `docs/conventions.md`) y `extraReviewRequired` (`true` si y
solo si `sensitiveAreasTouched` no está vacío). Así, un diff que toca
`scaffold.go` o `.github/workflows/` queda marcado para una pasada de
revisión más cuidadosa sin que nadie tenga que leer el diff completo para
notarlo.

`review start --feature <id>` **sin** `--json` sigue comportándose
exactamente igual que en la feature 7 — una sola línea con el
`subject_hash` en stdout, sin calcular siquiera `touchedPaths` ni las
áreas sensibles (no solo "sin imprimirlos"): mismo camino de código, cero
riesgo de que un error nuevo en el cálculo de sensibilidad rompa el uso
existente (`HASH=$(april review start --feature 7)`).

Cualquier invocación de `args[2:]` que no sea ni vacía ni exactamente
`["--json"]` (typo del flag, argumentos de más, `--json` en otra
posición) es un error de invocación explícito, exit≠0, sin llamar a
ninguna función de cálculo. Fuera de un repositorio git, `--json` falla
igual que hoy (feature 7): error explícito en stderr, sin intentar
calcular nada de lo nuevo. `extraReviewRequired` nunca cambia el exit
code — `review start` sigue siendo una consulta informativa, no un gate.
`printUsage()` se actualiza con la nueva forma del comando.

Esto cierra el ensamblaje: la nueva función pura `matchSensitiveAreas`
(cruza `touchedPaths` contra las áreas sensibles — coincidencia exacta de
archivo, o de prefijo de directorio si el área termina en `/`, sin
substring parcial) y la struct `reviewStartReport` (con los cuatro campos
en camelCase) se agregan acá, junto con el cableado real de
`runReviewStart` que las conecta con `computeSubjectHash` (feature 7),
`computeTouchedPaths` (ticket 02) y `readSensitiveAreas` (ticket 01).

**Blocked by:** 01 (`parseSensitiveAreas`/`readSensitiveAreas`), 02
(`computeTouchedPaths`)

**Status:** done

- [ ] `matchSensitiveAreas` con un área sensible terminada en `/` (ej.
      `.github/workflows/`) hace match de cualquier ruta tocada dentro de
      ese directorio (prefijo).
- [ ] `matchSensitiveAreas` con un área sensible sin `/` (ej.
      `scaffold.go`) exige coincidencia exacta de ruta completa —
      `scaffold_test.go` no hace match contra `scaffold.go`.
- [ ] `matchSensitiveAreas` devuelve **todas** las rutas tocadas que
      coinciden (no solo la primera) cuando hay más de una.
- [ ] `matchSensitiveAreas` devuelve `[]string{}` (no `nil`) cuando
      ninguna ruta tocada coincide.
- [ ] `review start --feature <id>` (sin `--json`), en un repo con commit
      y un cambio real tocando `scaffold.go`, imprime únicamente la línea
      del hash — mismo fixture/aserciones que
      `TestRunReviewStartImprimeSubjectHashEnStdout` (feature 7) — sin
      ningún rastro de `touchedPaths` ni `extraReviewRequired` en stdout.
- [ ] `review start --feature <id> --json`, con `docs/conventions.md`
      conteniendo la sección real de "Áreas sensibles" y un cambio que
      toca `scaffold.go` después del commit baseline, imprime un JSON
      parseable con `json.Unmarshal` donde `extraReviewRequired == true`,
      `touchedPaths` contiene `"scaffold.go"`, `sensitiveAreasTouched`
      contiene `"scaffold.go"`, y `subjectHash` no vacío.
- [ ] Mismo fixture de `docs/conventions.md`, pero el cambio toca un
      archivo no sensible (ej. `otra_cosa.go`): `extraReviewRequired ==
      false`, `sensitiveAreasTouched` vacío, `touchedPaths` sí contiene
      `"otra_cosa.go"`.
- [ ] Con `docs/conventions.md` sin la sección "Áreas sensibles" (o
      ausente) y un cambio tocando `scaffold.go`, `extraReviewRequired ==
      false` sin error.
- [ ] En un caso sin cambios, la salida `--json` deserializa
      `touchedPaths` y `sensitiveAreasTouched` como arreglo vacío (`[]`),
      nunca como `null`.
- [ ] Fuera de un repositorio git, `--feature <id> --json` termina con
      exit≠0 y stderr menciona explícitamente que no es un repositorio
      git, sin llegar a calcular `touchedPaths`.
- [ ] `["--feature", "<id>", "--json", "extra"]` es error de invocación
      explícito, exit≠0.
- [ ] `["--feature", "<id>", "--jason"]` (typo) es error de invocación
      explícito, exit≠0.
- [ ] Los tests ya existentes de `review_test.go` para
      `runReviewStart`/`computeSubjectHash` (feature 7) siguen en verde
      sin ninguna modificación.
- [ ] `printUsage()` documenta
      `review start --feature <id> [--json]` y su efecto.
- [ ] `go build ./...` y `go test ./...` en verde.
