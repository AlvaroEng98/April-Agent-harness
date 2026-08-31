# 02: `april status` reporta `no_review_verdict` leyendo el ledger

**What to build:** al correr `april status <id> --json` sobre una feature
con `status: "in_progress"`, `blockedReasons` incluye una entrada con la
substring `"no_review_verdict"` en los tres casos en que el cierre no
está habilitado por un veredicto de revisión vigente: no existe ninguna
entrada `kind: "review"` para esa feature en el ledger; la última de esas
entradas tiene un `verdict` que no habilita cierre (`CHANGES_REQUESTED`,
o cualquier valor que no sea `APPROVED`/`APPROVED_WITH_OBJECTION`); o su
`treeHash` no coincide con el árbol actual (el código cambió después de
que se registró el veredicto). Con la última entrada `kind: "review"` en
`APPROVED` o `APPROVED_WITH_OBJECTION` y `treeHash` vigente,
`blockedReasons` NO incluye `no_review_verdict`.

Este chequeo solo se evalúa para la feature `in_progress` consultada —
`pending`, `spec_ready`, `blocked` o `done` nunca lo disparan. Una
entrada `kind: "test"` nunca cuenta como evidencia de revisión (aunque
esté en verde y vigente), y una entrada `kind: "review"` nunca cuenta
como evidencia de tests — la separación por `kind` es estricta en ambos
sentidos. Una secuencia de dos entradas `kind: "review"` para la misma
feature (primero `CHANGES_REQUESTED`, después `APPROVED` con hash
vigente) resuelve el bloqueo, porque se evalúa la última entrada, no la
primera. Una línea corrupta del ledger se sigue reportando en
`blockedReasons` exactamente igual que hoy, sin importar de qué `kind`
era la línea que falló al parsear. `derivePhase`, `frontier` y el resto
de `blockedReasons` no cambian de comportamiento — esta pieza solo agrega
contenido posible al arreglo.

**Blocked by:** 01 (`april review record` — comando completo; esta pieza
necesita un comando funcionando que produzca entradas reales de
`kind: "review"` en el ledger, no solo el tipo de dato)

**Status:** done

- [ ] Feature `in_progress` sin ninguna entrada `kind: "review"` en el
      ledger ⇒ `blockedReasons` incluye una entrada con
      `no_review_verdict` (puede tener entradas `kind: "test"` en verde,
      para aislar que el gap es específicamente de revisión).
- [ ] Feature `in_progress` cuya última entrada `kind: "review"` tiene
      `verdict: "CHANGES_REQUESTED"` y `treeHash` igual al árbol actual ⇒
      `no_review_verdict` presente (el hash vigente no alcanza si el
      verdict no habilita).
- [ ] Feature `in_progress` cuya última entrada `kind: "review"` tiene
      `verdict: "APPROVED"` y `treeHash` igual al árbol actual ⇒
      `no_review_verdict` ausente.
- [ ] Feature `in_progress` cuya última entrada `kind: "review"` tiene
      `verdict: "APPROVED_WITH_OBJECTION"` y `treeHash` igual al árbol
      actual ⇒ `no_review_verdict` ausente.
- [ ] Feature `in_progress` cuya última entrada `kind: "review"` tiene
      `verdict: "APPROVED"` pero `treeHash` de una versión vieja del árbol
      (el árbol cambió después) ⇒ `no_review_verdict` presente.
- [ ] Fixture con dos entradas `kind: "review"` para la misma feature —
      primero `CHANGES_REQUESTED` (hash viejo), después `APPROVED` (hash
      actual) ⇒ `no_review_verdict` ausente, porque se evalúa la última
      entrada, no la primera.
- [ ] Fixture con una entrada `kind: "test"` en verde y vigente, sin
      ninguna entrada `kind: "review"`, para una feature `in_progress` ⇒
      `no_review_verdict` sigue presente (una entrada `kind: "test"`
      nunca cuenta como evidencia de revisión).
- [ ] Fixture con una entrada `kind: "review"` `APPROVED` vigente, sin
      ninguna entrada `kind: "test"`, para una feature `in_progress` ⇒
      `no_test_evidence` sigue presente (una entrada `kind: "review"`
      nunca cuenta como evidencia de test).
- [ ] Ninguna feature `pending`, `spec_ready`, `blocked` o `done` reporta
      `no_review_verdict`, tenga o no entradas `kind: "review"`.
- [ ] Una línea con JSON inválido en el ledger se sigue reportando en
      `blockedReasons` exactamente igual que antes de esta feature (test
      ya existente de la feature 5 sigue en verde, sin modificación).
- [ ] Los 21 tests existentes de `set_status_test.go` siguen en verde sin
      modificación (evidencia de que `set_status.go` no se tocó).
- [ ] `derivePhase`, `frontier` y el resto de `blockedReasons` ya
      existentes no cambian de comportamiento (sin nuevo valor de
      `phase`).
- [ ] `go build ./...` y `go test ./...` en verde.
