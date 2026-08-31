# 02: computeBlockedReasons detecta ausencia de Given/When/Then (o marcador de opt-out); april doctor hereda la señal

**What to build:** `april status --json` reporta una entrada con la
substring literal `no_gwt_coverage` (identificando id, nombre de la
feature y la ruta de su `spec.md`) para cualquier feature `sdd:true` cuyo
`specs/<name>/spec.md` exista, no contenga ningún bloque Given/When/Then
(al menos una línea, en cualquier parte del archivo, que empiece —
ignorando espacios iniciales— literalmente con `Given`, al menos una con
`When` y al menos una con `Then`, sensible a mayúsculas/minúsculas, sin
exigir adyacencia ni orden entre ellas) ni el marcador de opt-out
`<!-- gwt: no aplica -->` en ninguna parte del archivo, que todavía no
tenga ningún archivo en `specs/<name>/tickets/`, y cuyo `status` no sea
`done`. El chequeo deja de reportar en cuanto se cumple cualquiera de
estas condiciones: la spec sí tiene un bloque Given/When/Then real; la
spec tiene el marcador de opt-out (incluso si además tiene bloques
Given/When/Then reales — la redundancia no se arbitra, la sola presencia
del marcador basta); la feature ya tiene al menos un archivo de ticket en
disco; o el `status` de la feature es `done`. La misma señal aparece
igual en `april doctor --json` (`report.blockedReasons` contiene
`no_gwt_coverage` y `report.healthy` es `false`) sin que `doctor.go`
reciba ninguna línea de código nueva — se verifica que la hereda de
`computeStatus(nil)` tal como ya hace hoy `computeDoctor`. Ni
`april status` ni `april doctor` escriben, borran ni modifican ningún
archivo al correr el chequeo (hash del árbol de archivos idéntico
antes/después de correrlos varias veces). Tras este cambio, el test de
caracterización del ticket 01 sigue pasando exactamente igual — ninguna
feature `done` cambia su `phase`/`blockedReasons`/`nextRecommended`.

**Blocked by:** 01 (Test de caracterización de features `done` reales) —
la spec exige explícitamente escribir y correr ese test antes de tocar
`computeBlockedReasons`.

**Status:** done

- [ ] Spec sin GWT, sin marcador de opt-out, sin tickets en disco,
      `status` distinto de `done` → `blockedReasons` contiene una entrada
      con la substring `no_gwt_coverage` que menciona id y nombre de la
      feature
- [ ] Spec con al menos un bloque Given/When/Then real (líneas que
      empiezan con `Given`/`When`/`Then`) → `blockedReasons` no contiene
      ninguna entrada con `no_gwt_coverage` para esa feature
- [ ] Spec sin GWT pero con el marcador `<!-- gwt: no aplica -->` en
      cualquier parte del archivo → `blockedReasons` no contiene
      `no_gwt_coverage` para esa feature
- [ ] Spec sin GWT ni marcador, pero con al menos un archivo en
      `specs/<name>/tickets/` → `blockedReasons` no contiene
      `no_gwt_coverage` para esa feature
- [ ] Spec sin GWT ni marcador, `status: done` → `blockedReasons` no
      contiene `no_gwt_coverage` para esa feature
- [ ] Spec con GWT real Y el marcador de opt-out simultáneamente →
      `blockedReasons` no contiene `no_gwt_coverage` para esa feature
      (no se arbitra la redundancia)
- [ ] Un test en `doctor_test.go` dispara `no_gwt_coverage` vía fixture y
      verifica que `computeDoctor().BlockedReasons` (y
      `report.Healthy == false`) lo reflejan, documentando que
      `doctor.go` no tiene código propio nuevo para esto
- [ ] Un test read-only (mismo patrón que `TestDoctorNoEscribeArchivos`,
      feature 9) corre `runStatus`/`runDoctor` varias veces sobre un
      directorio temporal real con una feature que dispara
      `no_gwt_coverage`, y confirma que el hash del árbol de archivos
      antes/después es idéntico
- [ ] Todos los tests nuevos usan la interfaz pública
      `computeStatusFromFS`/`fstest.MapFS` y el helper `anyContains` ya
      existente — ninguno mockea `fs.FS` ni testea una función interna en
      aislamiento, ninguno recalcula la lógica bajo test para obtener el
      valor esperado
- [ ] Los tests existentes de `status_test.go`/`doctor_test.go` siguen
      pasando sin editar ninguna de sus aserciones
- [ ] El test de caracterización del ticket 01 sigue pasando, sin
      cambios en su resultado, tras este cambio
- [ ] No se agrega ningún flag de CLI nuevo ni se toca
      `nextRecommendedText`/`derivePhase`/`set_status.go`
- [ ] `go build ./...` y `go test ./...` en verde
