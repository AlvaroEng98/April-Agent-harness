# 01: Test de caracterización de los 10 mensajes de blockedReasons antes de tocar status.go

**What to build:** un test (o tabla dentro de uno) en `status_test.go`
que, reutilizando los fixtures ya existentes en el archivo
(`TestDosFeaturesInProgressReportaBlockedReasons`,
`TestSinReceiptParaFeatureInProgressReportaNoTestEvidence`,
`TestReceiptConExitDistintoDeCeroReportaNoTestEvidence`,
`TestReceiptExitosoConArbolDesactualizadoReportaNoTestEvidence`,
`TestSinEntradaReviewParaFeatureInProgressReportaNoReviewVerdict`,
`TestReviewChangesRequestedConHashVigenteReportaNoReviewVerdict`,
`TestReviewApprovedConHashDesactualizadoReportaNoReviewVerdict`,
`TestBlockedByConTextoNoInterpretableReportaBlockedReasons`,
`TestCicloEnBlockedByDeTicketsSeDetectaYNoCuelga`, más un fixture análogo
de armado para `no_gwt_coverage` si no existe ya uno reutilizable), corre
`computeStatusFromFS` — nunca los helpers internos en aislamiento — y fija
con **igualdad exacta de string** el texto que produce hoy el código para
los 10 mensajes que la feature 15 va a tocar:

1. In_progress duplicado.
2. `no_test_evidence` sin receipt.
3. `no_test_evidence` con `exitCode != 0`.
4. `no_test_evidence` con `treeHash` desactualizado.
5. `no_review_verdict` sin receipt.
6. `no_review_verdict` con verdict que no habilita cierre.
7. `no_review_verdict` con `treeHash` desactualizado.
8. Ticket con `Blocked by` no interpretable.
9. Ciclo en `Blocked by`.
10. `no_gwt_coverage`.

Para los dos casos con `treeHash` dinámico (4 y 7), el test compara el
prefijo hasta el patrón conocido en vez de un literal completo, porque el
hash depende del fixture exacto de cada corrida. Este test se escribe y se
corre —confirmando que pasa contra el código **actual**, sin ningún
cambio en `computeBlockedReasons` ni sus helpers— antes de que exista
ningún otro cambio de código de la feature 15. Es la red de seguridad que
el ticket 02 reutiliza (no un test nuevo paralelo) para demostrar
mecánicamente que agregar las recetas fue aditivo y no reescribió nada.

**Blocked by:** None (can start immediately)

**Justificación (spec):** Testing Decisions, sección "Test de
caracterización — MUST existir antes del cambio de código" (fija el
requisito de escribirlo y correrlo ANTES de tocar el código, reusando
`computeStatusFromFS` y los fixtures existentes, con igualdad exacta salvo
prefijo en los dos casos de `treeHash` dinámico). Implementation
Decisions, sección "Mensajes literales actuales — el 'antes' exacto que el
test de caracterización MUST fijar" (da el literal congelado de cada uno
de los nueve casos base). Testing Decisions, sección "Casos MUST
cubiertos" (enumera los nueve) y el párrafo siguiente, "Caso adicional...
no_gwt_coverage" (agrega el décimo, por la delegación explícita de
`specs/spec_gwt_mechanical_check/spec.md`).

**Status:** done

- [x] El test nuevo (o tabla) vive en `status_test.go`, pasa exclusivamente
      por `computeStatusFromFS(fsys, targetID)` sobre `fstest.MapFS` — no
      invoca `noTestEvidenceReason`/`noReviewVerdictReason`/
      `ticketBlockedByReasons`/`detectBlockedByCycle` directamente.
- [x] Cubre los 10 casos listados arriba, reutilizando fixtures ya
      existentes en el archivo donde aplique.
- [x] Los 8 casos con hash estático se verifican con igualdad exacta de
      string contra el literal congelado; los 2 casos con `treeHash`
      dinámico (in_progress `no_test_evidence` y `no_review_verdict` con
      árbol desactualizado) se verifican por prefijo hasta el patrón
      conocido, no por literal completo.
- [x] El test se corre contra el código actual (sin tocar
      `computeBlockedReasons` ni sus cinco helpers) y pasa en verde.
- [x] `go test ./...` pasa en verde sin editar ninguna aserción de los
      tests preexistentes de `status_test.go`.
- [x] Este ticket no modifica `computeBlockedReasons`,
      `noTestEvidenceReason`, `noReviewVerdictReason`,
      `ticketBlockedByReasons` ni `detectBlockedByCycle` — solo agrega el
      test de caracterización.
