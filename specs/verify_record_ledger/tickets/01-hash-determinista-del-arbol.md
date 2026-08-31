# 01: `hashTree` — hash determinístico del árbol de trabajo con exclusiones fijas

**What to build:** una función pura `hashTree(fsys fs.FS) (string, error)`,
nueva en `verify.go`, que lleva a producción el mismo algoritmo que hoy
solo vive como test helper (`hashDirTree` en `status_test.go`): recorre
todo el árbol bajo `fsys`, calcula `sha256` del contenido de cada archivo,
arma pares `ruta-relativa:hash`, los ordena por ruta (para que el
resultado no dependa del orden en que el filesystem devuelva las
entradas) y calcula el `sha256` del agregado completo.

Excluye deliberadamente solo tres cosas — nada más:

- cualquier ruta bajo `.git/` (prefijo);
- exactamente `.claude/verify-ledger.jsonl` (el propio ledger que esta
  feature va a escribir);
- cualquier ruta bajo `progress/` (prefijo).

Todo lo demás cuenta para el hash: `feature_list.json`, `docs/`, `specs/`
y todo el código fuente. No hay lista configurable ni flag para ampliar
las exclusiones.

Esta pieza es un building-block deliberado: todavía no hay ningún comando
de `april` que la invoque (eso llega en el ticket 3). Se demuestra y se
verifica por sí sola con tests unitarios puros sobre `fstest.MapFS`.

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] `hashTree(fsys fs.FS) (string, error)` existe en `verify.go` y
      calcula el hash agregado según el algoritmo descrito (sha256 por
      archivo, pares ordenados por ruta, sha256 del agregado).
- [ ] Modificar solo archivos bajo `.git/`, el archivo exacto
      `.claude/verify-ledger.jsonl`, o archivos bajo `progress/` no cambia
      el hash resultante.
- [ ] Modificar un archivo fuera de esas tres exclusiones (ej. un `.go`
      sintético en el fixture) sí cambia el hash.
- [ ] Dos `fstest.MapFS` con el mismo contenido, construidos insertando
      las mismas claves en distinto orden, producen el mismo hash.
- [ ] `go build ./...` y `go test ./...` en verde.
