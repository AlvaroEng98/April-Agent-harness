# 02: Recetas de remedio en los 5 puntos de origen de computeBlockedReasons

**What to build:** cada uno de los 10 mensajes cubiertos de
`blockedReasons` sigue diagnosticando exactamente lo mismo que hoy —el
texto actual se preserva carácter por carácter como prefijo— y además le
dice al orquestador (humano o agente) el comando `april ...` exacto o la
acción de archivo concreta que lo resuelve, separado del diagnóstico por
` — `. Concretamente, después de este ticket:

- El mensaje de más de una feature `in_progress` lista los ids en
  conflicto (ascendente) y agrega el comando literal
  `april feature set-status <id> pending`.
- Los tres mensajes de `no_test_evidence` (sin receipt, `exitCode != 0`,
  `treeHash` desactualizado) agregan
  `april verify record --feature <id real> -- <comando>`, con el id de la
  feature ya sustituido.
- Los tres mensajes de `no_review_verdict` agregan
  `april review record --feature <id real> --verdict <valor>`, con el id
  ya sustituido; el caso "sin receipt" menciona los tres valores válidos
  de `--verdict` (APPROVED, APPROVED_WITH_OBJECTION, CHANGES_REQUESTED),
  el caso "verdict que no habilita cierre" queda acotado a
  APPROVED/APPROVED_WITH_OBJECTION.
- El mensaje de ticket con `Blocked by` no interpretable agrega el formato
  esperado en prosa y repite el nombre del archivo a editar, para que la
  receta sea legible aislada.
- El mensaje de ciclo en `Blocked by` agrega la instrucción de editar el
  campo `**Blocked by:**` de un archivo concreto de la cadena (resuelto a
  su `t.Filename` real), no solo la cadena de ids.
- El mensaje de `no_gwt_coverage` agrega la instrucción de sumar al menos
  un bloque Given/When/Then a la `spec.md` de la feature, o el marcador
  `<!-- gwt: no aplica -->`.

Los cinco mensajes que la spec deja explícitamente fuera de alcance
(status inválido, feature `blocked`, spec faltante, Status de ticket
inválido, línea corrupta del ledger) no cambian en absoluto. El test de
caracterización del ticket 01 se reutiliza tal cual —no se escribe uno
nuevo en paralelo—: sus aserciones de igualdad exacta pasan a
`strings.HasPrefix(mensajeReal, literalCongelado)`, y se agrega, por cada
uno de los 10 casos, una aserción `strings.Contains` sobre el fragmento de
receta esperado. Toda la suite existente de `status_test.go` que ya
verifica `blockedReasons` (incluida la prueba de las features `sdd:true`
ya `done` con `blockedReasons: []string{}`) sigue pasando sin editar
ninguna de sus aserciones.

**Blocked by:** 01 (Test de caracterización de blockedReasons) — este
ticket necesita el literal congelado y el test ya escrito y en verde
contra el código actual antes de poder tocar
`computeBlockedReasons`/sus helpers; sin eso no hay forma mecánica de
demostrar que el cambio fue aditivo.

**Justificación (spec):** Implementation Decisions, secciones "Seam: los
mismos cinco puntos de origen dentro de `status.go`, ninguno nuevo"
(prohíbe mecanismos nuevos, exige modificar solo los `fmt.Sprintf`
existentes en los cinco puntos), "Preservación del diagnóstico existente
como prefijo" (regla del separador ` — ` uniforme), y "Recetas a agregar,
caso por caso" (el contenido exacto de cada receta, incluyendo qué queda
como placeholder literal —`<id>`/`<comando>`/`<valor>`— y qué se sustituye
por el valor real). También "Fuera de alcance de receta — decisión
explícita por mensaje, no omisión" (los cinco mensajes que no se tocan).
Testing Decisions, secciones "Verificación post-cambio — MUST reusar el
mismo test, no uno nuevo paralelo" y "Regresión — MUST correr sin editar".
Out of Scope (ningún campo nuevo en `statusReport`/`doctorReport`, ningún
flag de CLI nuevo, ningún placeholder sustituido con valor inventado).

**Status:** done

- [x] `computeBlockedReasons` (mensaje in_progress duplicado): agrega ids
      en conflicto ordenados ascendentemente y el comando
      `april feature set-status <id> pending` (con `<id>` literal), tras
      el separador ` — `, preservando el diagnóstico actual como prefijo.
- [x] `noTestEvidenceReason` (tres casos: sin receipt, `exitCode != 0`,
      `treeHash` desactualizado): agrega
      `april verify record --feature <id real> -- <comando>` con el id
      real de la feature sustituido, preservando `no_test_evidence` y el
      diagnóstico previo como prefijo.
- [x] `noReviewVerdictReason` (tres casos): agrega
      `april review record --feature <id real> --verdict <valor>` con el
      id real sustituido; el caso "sin receipt" menciona los tres valores
      válidos de `--verdict`, el caso "verdict que no habilita cierre"
      queda acotado a APPROVED/APPROVED_WITH_OBJECTION; preserva
      `no_review_verdict` y el diagnóstico previo como prefijo.
- [x] `ticketBlockedByReasons`: agrega el formato esperado
      ("números de ticket de dos dígitos separados por coma... o la
      palabra 'none'") y repite el nombre del archivo (`t.Filename`) a
      editar, preservando el diagnóstico previo como prefijo.
- [x] `detectBlockedByCycle`: agrega la instrucción de editar
      `**Blocked by:**` de un archivo concreto de la cadena, resolviendo
      el primer NN detectado a su `t.Filename` real dentro de la misma
      función (sin estructura ni archivo nuevo), preservando
      `ciclo detectado` y la cadena de ids como prefijo.
- [x] El mensaje de `no_gwt_coverage` agrega la instrucción de sumar un
      bloque Given/When/Then o el marcador `<!-- gwt: no aplica -->`,
      preservando `no_gwt_coverage` y el diagnóstico previo como prefijo.
- [x] Los cinco mensajes fuera de alcance (status inválido, `blocked`,
      spec faltante, Status de ticket inválido, línea corrupta del
      ledger) quedan exactamente igual, sin ningún cambio.
- [x] El test de caracterización del ticket 01 se reutiliza (mismo test,
      no uno nuevo): sus aserciones de igualdad exacta pasan a
      `strings.HasPrefix`, y se agrega `strings.Contains` por caso sobre
      el fragmento de receta esperado — los 10 casos verificados.
- [x] Toda la suite existente de `status_test.go` que verifica
      `blockedReasons` (incluida
      `TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons`)
      sigue pasando sin editar ninguna de sus aserciones.
- [x] No se agrega ningún campo nuevo a `statusReport`/`doctorReport` ni
      ningún flag de CLI nuevo — el cambio vive solo en el contenido de
      los strings.
- [x] `go build ./...` y `go test ./...` en verde.
