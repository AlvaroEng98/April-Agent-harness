# 02: Esquema del ledger y append atómico

**What to build:** el tipo `ledgerEntry` (nuevo en `verify.go`) con los
campos `kind`, `featureId`, `command` ([]string), `exitCode`, `treeHash`,
`timestamp`, `stdout`, `stderr` — serializable exactamente al formato de
línea JSON de la spec (una línea por entrada, sin pretty-print). El campo
`kind` queda ya en el esquema como string libre (esta feature solo escribe
`"test"`), reservado para que la feature 6 agregue `"review"` al mismo
archivo sin migrar el formato.

Además, una función pura de append: dado el contenido actual del ledger
(bytes, posiblemente vacío si el archivo no existe todavía) y una
`ledgerEntry` nueva, devuelve el contenido completo resultante con la
línea nueva anexada al final, sin alterar ni reordenar ninguna línea
previa. El wrapper de I/O de esta pieza lee el archivo real con
`os.ReadFile` (archivo inexistente se trata como contenido vacío, no como
error — mismo criterio de "adopción" que `loadManifest` en `scaffold.go`)
y escribe el resultado completo con `writeFileAtomic`, reusada tal cual de
`set_status.go` (ya es genérica, no atada a `feature_list.json`) — sin
duplicar el patrón temp-then-rename.

Esta pieza es un building-block deliberado: todavía no hay ningún comando
de `april` que la dispare ni un subproceso real corriendo detrás (eso
llega en el ticket 3). Se demuestra y se verifica por sí sola con tests
unitarios sobre contenido sintético del ledger.

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] `ledgerEntry` existe en `verify.go` con los ocho campos del esquema
      y serializa/deserializa al JSON exacto de la spec (una línea, sin
      pretty-print).
- [ ] La función pura de append, llamada dos veces sucesivas sobre el
      mismo contenido inicial, produce un resultado con ambas líneas, en
      orden, con la primera intacta (nunca se pisa).
- [ ] La función de append sobre contenido inicial vacío (o archivo
      inexistente) produce un resultado con una sola línea válida.
- [ ] El wrapper de I/O trata un ledger inexistente en disco como
      contenido vacío, no como error.
- [ ] El wrapper de I/O escribe usando `writeFileAtomic` (reusada de
      `set_status.go`, sin una segunda implementación del patrón
      temp-then-rename).
- [ ] `go build ./...` y `go test ./...` en verde.
