# 04: `april status` reporta `no_test_evidence` leyendo el ledger

**What to build:** al correr `april status <id> --json` sobre una feature
con `status: "in_progress"`, `blockedReasons` incluye una entrada que
contiene la substring `"no_test_evidence"` (junto con detalle legible de
cuál de los tres casos aplicó) cuando: no existe ningún receipt
`kind: "test"` para esa feature en `.claude/verify-ledger.jsonl`; el
último receipt de ese tipo tiene `exitCode != 0`; o su `treeHash` no
coincide con el hash del árbol actual (el código cambió después de la
corrida registrada). Con un receipt `kind: "test"` en verde
(`exitCode == 0`) y vigente (mismo `treeHash` que el árbol actual),
`blockedReasons` NO incluye `no_test_evidence` para esa feature.

Este chequeo solo se evalúa para la feature con `status: "in_progress"` —
features `pending`, `spec_ready`, `blocked` o `done` nunca lo disparan,
aunque no tengan ningún receipt. Una entrada `kind: "review"` (reservada
para la feature 6) nunca cuenta como evidencia de test, aunque esté en
verde. Una línea corrupta del ledger (JSON inválido, por ejemplo por
edición manual accidental) se reporta como una razón adicional en
`blockedReasons`, identificando la línea, sin que el resto del cálculo de
`blockedReasons`/`phase`/`frontier` se rompa ni el comando falle con un
error opaco. `derivePhase`/`frontier`/`nextRecommended` no cambian de
comportamiento — esta pieza solo agrega contenido posible a
`blockedReasons`.

Internamente, esto se apoya en una nueva función `readLedger(fsys fs.FS)`
en `status.go` (línea por línea, líneas vacías ignoradas, líneas
inválidas van a una lista de "líneas corruptas" en vez de abortar la
lectura completa — archivo inexistente no es error, entries vacío) y en
invocar `hashTree(fsys)` (del ticket 3) una vez por corrida de
`computeStatusFromFS` para comparar contra el `treeHash` de cada receipt.

**Blocked by:** 03 (`april verify record` — comando completo; esta pieza
necesita un comando funcionando que produzca entradas reales en el
ledger, no los tramos intermedios 01/02 que todavía no emiten un ledger
de verdad)

**Status:** done

- [ ] Feature `in_progress` sin ningún receipt `kind: "test"` en el
      ledger ⇒ `blockedReasons` incluye una entrada con `no_test_evidence`.
- [ ] Feature `in_progress` cuyo último receipt `kind: "test"` tiene
      `exitCode != 0` ⇒ `no_test_evidence` presente.
- [ ] Feature `in_progress` cuyo último receipt tiene `exitCode == 0` y
      `treeHash` igual al hash del árbol actual ⇒ `no_test_evidence`
      ausente.
- [ ] Feature `in_progress` cuyo último receipt tiene `exitCode == 0` pero
      `treeHash` desactualizado (el árbol cambió después) ⇒
      `no_test_evidence` presente.
- [ ] Agregar contenido a `progress/current.md` (o cualquier ruta bajo
      `progress/`) después de un receipt vigente NO invalida ese receipt
      (regresión directa del problema de auto-invalidación detectado
      durante la fase de spec).
- [ ] Ninguna feature `pending`, `spec_ready`, `blocked` o `done` reporta
      `no_test_evidence`, tengan o no receipt.
- [ ] Una línea con JSON inválido en el ledger se reporta explícitamente
      en `blockedReasons` (identificando la línea), sin romper el resto
      del cálculo ni hacer fallar el comando con un error opaco.
- [ ] Una entrada `kind: "review"` para la misma feature no cuenta como
      evidencia de test — no evita que aparezca `no_test_evidence` si no
      hay ninguna entrada `kind: "test"` en verde y vigente.
- [ ] `derivePhase`, `frontier` y el resto de `blockedReasons` ya
      existentes no cambian de comportamiento (sin nuevo valor de
      `phase`).
- [ ] `go build ./...` y `go test ./...` en verde.
