# 03: `hashTree` respeta `.gitignore` de punta a punta

**What to build:** correr `go build ./...` (u otro comando que regenere
un artefacto gitignoreado, ej. `/HarnessInit`) ya no invalida el
`treeHash` registrado por `verify record`/`review record` — en cualquier
orden respecto a esos comandos. Un orquestador o `agent_developer` que
corre un build de verificación después de grabar evidencia, o antes,
sigue viendo `blockedReasons` limpio de `no_test_evidence`/
`no_review_verdict` mientras el código real no haya cambiado.

`hashTree(fsys fs.FS)` (sin cambiar su firma) carga los patrones de
`.gitignore` del propio `fsys` una sola vez, antes de recorrer el árbol,
con `loadGitignorePatterns` (ticket 01). `isExcludedFromTreeHash` gana un
segundo parámetro `patterns []gitignorePattern`: primero consulta,
incondicionales, las tres exclusiones fijas (`.git/`, y las dos de
`fixedTreeExclusions`, ticket 02) y solo si ninguna aplica, consulta
`gitignoreMatches`. Un archivo NO gitignoreado sigue invalidando el hash
exactamente igual que hoy — la corrección no vuelve a `hashTree` ciego a
cambios reales de código.

Se verifica en dos niveles: unitario sobre `fstest.MapFS` (mismo patrón
que los tests ya existentes de `hashTree`) y extremo-a-extremo sobre
disco real, ejercitando el camino completo que le importa al
orquestador — `recordVerify`/`computeStatusFromFS`, no `hashTree` en
aislamiento — en ambos órdenes de comandos.

**Blocked by:** 01 (parser de `.gitignore`), 02 (`fixedTreeExclusions`
compartida)

**Status:** done

- [ ] Los tres tests existentes de `hashTree`
      (`TestHashTreeExcluyeGitProgressYElPropioLedger`,
      `TestHashTreeCambiaSiUnArchivoNoExcluidoCambia`,
      `TestHashTreeEsDeterministicoSinImportarOrden`) siguen en verde sin
      ninguna edición — ninguno define `.gitignore` en su `fstest.MapFS`.
- [ ] Nuevo test `TestHashTreeExcluyeArchivoGitignoreadoAunSinListaFija` —
      `fstest.MapFS` con `.gitignore` conteniendo `"/HarnessInit\n"` y un
      archivo `"HarnessInit"` con contenido A; calcula el hash;
      sobrescribe `"HarnessInit"` con contenido B (simula un rebuild
      real); recalcula; ambos hashes son iguales.
- [ ] Nuevo test
      `TestHashTreeSigueExcluyendoLasTresFijasAunNoEstenEnGitignore` —
      `.gitignore` sintético que no menciona `progress/` ni el ledger
      (ej. solo `"*.pyc\n"`); modificar archivos bajo `progress/` y el
      ledger no cambia el hash.
- [ ] Nuevo test
      `TestHashTreeArchivoNoGitignoreadoSigueCambiandoElHashConGitignorePresente`
      — con un `.gitignore` real presente, modificar un archivo que no
      matchea ningún patrón sigue cambiando el hash.
- [ ] `isExcludedFromTreeHash` referencia `fixedTreeExclusions` (no
      literales sueltos) para sus dos exclusiones fijas no-`.git`.
- [ ] Integración `TestRecordVerifyLuegoRegenerarArtefactoGitignoreadoNoProduceNoTestEvidence`
      — fixture en disco (`feature_list.json` con una feature
      `in_progress`, `sdd:false`; `.gitignore` con `/build-artifact`);
      escribe `build-artifact` (contenido A); corre
      `recordVerify`/`sh -c "exit 0"`; sobrescribe `build-artifact`
      (contenido B, simula un build posterior); corre
      `computeStatus`/`runStatus --json`; `blockedReasons` NO contiene
      `no_test_evidence`.
- [ ] Integración
      `TestRegenerarArtefactoGitignoreadoLuegoRecordVerifyNoProduceNoTestEvidence`
      — mismo fixture, orden invertido: escribe `build-artifact`
      (contenido B) antes de `recordVerify`, corre `recordVerify` una vez,
      `blockedReasons` limpio.
- [ ] Integración de control
      `TestModificarArchivoNoGitignoreadoLuegoRecordVerifySigueDetectandoNoTestEvidence`
      — mismo fixture, corre `recordVerify`, modifica después un archivo
      de código NO gitignoreado del fixture, corre `computeStatus`,
      `blockedReasons` SÍ contiene `no_test_evidence`.
- [ ] `go build ./...`, `go vet ./...` y `go test ./...` en verde.
