# 01: Test de caracterización de features `done` reales antes de tocar computeBlockedReasons

**What to build:** Un test que corre `computeStatus`/`computeStatusFromFS`
contra el árbol real de este repo (o un fixture `fstest.MapFS` equivalente
que replique fielmente spec/tickets de cada una) para cada feature
`sdd:true` con `status: done` vigente al momento de implementar este
ticket — no un conteo fijo de 12, el conjunto real que exista en
`feature_list.json` en ese momento. El test fija, como literales
hardcodeados (no recalculados por el propio código bajo test), los
valores actuales de `phase`, `blockedReasons` y `nextRecommended` de cada
una de esas features. Debe pasar en verde con el código de hoy, sin
ningún cambio a `status.go`/`doctor.go`. Su propósito es servir de red de
seguridad para el ticket 02: cualquier cambio a `derivePhase`/
`computeBlockedReasons`/`nextRecommendedText` que mueva esos valores sin
querer —incluido el propio cambio del ticket 02— debe hacerlo fallar de
inmediato, y el ticket 02 debe demostrar que este test sigue pasando
exactamente igual después de su cambio.

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] El test cubre, sin excepción, todas las features `sdd:true` con
      `status: done` que existan en `feature_list.json` al momento de
      escribir el test (verificado explícitamente contra el archivo real,
      no asumido de memoria)
- [ ] Para cada una de esas features, el test fija como literal
      hardcodeado (no como recalculación de la lógica bajo test) su
      `phase`, su `blockedReasons` completo y su `nextRecommended` actual
- [ ] El test usa `computeStatus`/`computeStatusFromFS` (la interfaz
      pública) contra `os.DirFS(".")` del árbol real, o un
      `fstest.MapFS` que replique con fidelidad el spec/tickets reales de
      esas features — nunca mockeando ni testeando una función interna en
      aislamiento
- [ ] El test pasa en verde tal cual está el código hoy, sin ningún
      cambio a `status.go`/`doctor.go`
- [ ] `go build ./...` y `go test ./...` en verde
