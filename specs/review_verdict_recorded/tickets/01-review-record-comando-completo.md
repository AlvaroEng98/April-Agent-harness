# 01: `april review record` — comando completo (persistencia del veredicto en el ledger)

**What to build:** `reviewer_agent` (o cualquier humano) puede correr
`april review record --feature <id> --verdict <valor>` y el veredicto
queda anexado como una línea nueva `kind: "review"` en
`.claude/verify-ledger.jsonl`, junto con el hash del árbol actual y un
timestamp — sin correr ningún subproceso, a diferencia de
`april verify record`: el veredicto ya fue decidido por quien invoca el
comando, este solo lo persiste.

Los tres valores del vocabulario exacto que ya usa `reviewer_agent`
(`APPROVED`, `APPROVED_WITH_OBJECTION`, `CHANGES_REQUESTED`) se aceptan y
registran con exit code 0 siempre que la invocación sea válida —
registrar un `CHANGES_REQUESTED` es la función del comando, no un fallo.
Un valor fuera de ese vocabulario (typo, sinónimo, minúsculas) es un
error de invocación explícito en stderr, exit distinto de cero, sin
escribir nada al ledger. Lo mismo aplica si falta `--feature`, falta
`--verdict`, el id no es numérico, o el orden de los flags no es el
esperado.

Dos corridas sucesivas sobre la misma feature (por ejemplo
`CHANGES_REQUESTED` seguido de `APPROVED`, simulando una ronda real de
revisión) producen dos líneas distintas en el ledger, ambas legibles, sin
que la segunda pise a la primera. El comando se descubre sin leer código:
la ayuda del CLI documenta su sintaxis.

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] `april review record --feature <id> --verdict APPROVED` anexa una
      entrada `kind: "review"` al ledger real en disco con `featureId`
      correcto, `verdict: "APPROVED"`, `treeHash` no vacío y `timestamp`
      parseable.
- [ ] `april review record --feature <id> --verdict CHANGES_REQUESTED`
      termina con exit code 0 y la entrada queda registrada en el ledger
      con ese verdict — registrar un rechazo no es un fallo del comando.
- [ ] `april review record --feature <id> --verdict APPROVED_WITH_OBJECTION`
      también termina con exit code 0 y queda registrado igual que los
      otros dos valores.
- [ ] Un `--verdict` fuera del vocabulario exacto (ej. `"aprobado"` o
      `"LGTM"`) no escribe ninguna entrada al ledger y el comando termina
      con exit distinto de cero.
- [ ] Invocar sin `--feature`, sin `--verdict`, con un id no numérico, o
      con los flags en un orden distinto al esperado es un error de
      invocación explícito en stderr, exit ≠ 0, sin tocar el ledger.
- [ ] Dos corridas sucesivas sobre el mismo `featureId` (ej.
      `CHANGES_REQUESTED` y luego `APPROVED`) producen dos líneas en el
      ledger, ambas parseables, en orden, ninguna sobrescribe a la otra.
- [ ] Una entrada `kind: "test"` ya existente en el ledger no se ve
      alterada por una corrida de `review record` sobre la misma feature
      (conviven en el mismo archivo sin interferir).
- [ ] Si el proceso se interrumpe a mitad de la escritura, el ledger
      existente (incluyendo entradas previas `kind: "test"` y
      `kind: "review"`) queda intacto — misma garantía de escritura
      atómica que ya usa `verify record`.
- [ ] `printUsage()` documenta
      `review record --feature <id> --verdict <valor>`.
- [ ] `go build ./...` y `go test ./...` en verde.
