# 03: `april review record --subject-hash` — rechazo de veredicto stale

**What to build:** `reviewer_agent` corre
`april review record --feature <id> --verdict <valor> --subject-hash <hash>`
y, si el árbol de trabajo cambió desde que se calculó ese `hash` (alguien
tocó código entre que `reviewer_agent` empezó a revisar y el momento de
registrar el veredicto), el comando **rechaza el registro** con un mensaje
que contiene la substring literal `"stale subject_hash"`, sin tocar el
ledger — a diferencia del bloqueo post-hoc de `no_review_verdict`
(feature 6), acá el rechazo ocurre en el momento mismo de intentar
registrar. Si el candidato sigue vigente, el veredicto se registra
normalmente, con el `subject_hash` guardado en la entrada del ledger junto
al `treeHash` de siempre (ambas señales coexisten, ninguna reemplaza a la
otra).

El vocabulario de `--verdict` aceptado (`APPROVED`,
`APPROVED_WITH_OBJECTION`, `CHANGES_REQUESTED`) es exactamente el mismo
que ya valida el camino sin `--subject-hash`, sin una segunda lista
definida en otro lugar; un valor fuera de vocabulario se rechaza igual que
hoy, con o sin `--subject-hash` presente.

El camino **sin** `--subject-hash` (`april review record --feature <id>
--verdict <valor>`, feature 6) sigue funcionando exactamente igual que
hoy — cero cambio de comportamiento, incluyendo que sigue funcionando en
cualquier directorio, con o sin git: los 11 tests existentes de
`review_test.go` pasan sin edición. Cualquier invocación con
`--subject-hash` mal formada (sin valor, o con argumentos de más después
del valor) es un error de invocación explícito, exit≠0, sin tocar el
ledger. Fuera de un repositorio git, `--subject-hash` presente hace fallar
el comando explícito (mismo criterio que `review start`, sin fallback).
`printUsage()` documenta el flag nuevo.

**Blocked by:** 01 (`computeSubjectHash`)

**Status:** done

- [ ] `review record --feature <id> --verdict <valor> --subject-hash
      <hash>` con un `hash` que coincide con el candidato recalculado en
      el momento anexa una entrada al ledger con `SubjectHash == hash`,
      `TreeHash` no vacío, y el `Verdict` correcto.
- [ ] La misma invocación con un `hash` desactualizado (el árbol cambió
      desde que se calculó) rechaza el registro, el error contiene la
      substring literal `"stale subject_hash"`, y el ledger **no** se
      creó/modificó.
- [ ] Un `--verdict` fuera de vocabulario con `--subject-hash` presente se
      rechaza igual que sin él, sin tocar el ledger.
- [ ] `--subject-hash` sin valor después (último argumento) es un error
      de invocación explícito, exit≠0, sin tocar el ledger.
- [ ] Argumentos de más después del valor de `--subject-hash` es un error
      de invocación explícito, exit≠0, sin tocar el ledger.
- [ ] Fuera de un repositorio git, con `--subject-hash` presente, el
      comando falla explícito (exit≠0, stderr menciona que no es un
      repositorio git) sin fallback.
- [ ] `review record --feature <id> --verdict <valor>` **sin**
      `--subject-hash` sigue funcionando fuera de un repositorio git,
      exactamente como antes de esta feature.
- [ ] Los 11 tests existentes de `review_test.go` (feature 6) siguen en
      verde sin ninguna edición.
- [ ] `TestLedgerEntrySerializaAlEsquemaExactoDeLaSpec` sigue en verde sin
      cambios tras agregar el campo `SubjectHash` (`omitempty`) a
      `ledgerEntry`.
- [ ] `printUsage()` documenta
      `review record --feature <id> --verdict <valor> [--subject-hash <hash>]`.
- [ ] `go build ./...` y `go test ./...` en verde.
