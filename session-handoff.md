# Session Handoff

## Current Objective

- Goal: segunda comparación contra `gentle-ai` (31/08/2026) — dos
  análisis independientes exploraron `/home/avalor/Proyectos/gentle-ai`
  completo contra el estado actual de April (post features 1-12),
  buscando metodología NO cubierta por la primera comparación
  (`ROADMAP.md` E0-E6). Resultado: 13 candidatos (C1-C13), agregados a
  `ROADMAP.md` para atacarlos uno a uno con el humano.
- Current status: **backlog completo: features 1-14 y 16-18 `done`**;
  solo la **feature 15** (`blocked_reasons_remedy_commands`, `sdd: true`)
  sigue `pending`, sin bloqueos. La segunda comparación contra
  `gentle-ai` (C1-C13) está cerrada. En una **continuación de la misma
  sesión**, el humano pegó un log real de CI de GitHub Actions mostrando
  un test roto; investigar la causa raíz encontró y corrigió dos bugs
  reales de `.gitignore` (features 17 y 18, detalle abajo).
- Branch / commit: `main`. El grueso del trabajo de C1-C13 + features
  13/14/16 **ya está commiteado** (`cfd4dd0`, por el humano). Sin
  commitear todavía: `.gitignore` (features 17+18),
  `.claude/verify-ledger.jsonl` (evidencia de las features 14/16/17/18),
  `progress/current.md`/`progress/history.md` (esta consolidación), y
  — importante — **`docs/*.md` y `specs/**/*.md` aparecen como `??`
  (untracked) por primera vez**: el fix de las features 17/18 los sacó
  del `.gitignore`, pero nadie corrió `git add` todavía (el orquestador
  nunca lo hace). El humano debe correr `git add specs/ docs/
  .gitignore .claude/verify-ledger.jsonl progress/ && git commit` para
  cerrar esto.

## Completed This Session (31/08/2026)

- [x] Segunda comparación contra `gentle-ai`: dos análisis independientes
      (agentes sin contexto compartido) → 13 hallazgos de metodología,
      comparados y sintetizados. `ROADMAP.md` reescrito: primera
      comparación comprimida a resumen + puntero a `progress/history.md`,
      candidatos C1-C13 agregados con evidencia, relevancia y adaptación
      propuesta cada uno.
- [x] **C1** — Given/When/Then + RFC 2119 en specs, verificado
      mecánicamente. Partido en 2 features vía `planner_agent`:
      **feature 13** `spec_template_gwt_rfc2119` (`sdd: false`, pending)
      y **feature 14** `spec_gwt_mechanical_check` (`sdd: true`, pending,
      bloqueada por la 13).
- [x] **C2** — evaluación en sombra antes de cambiar
      `derivePhase`/`computeBlockedReasons`/`nextRecommendedText`.
      Corregido: no aplica el shadow-flag de runtime de gentle-ai (April
      es CLI que se recompila, no tiene tráfico en vivo). Adoptado como
      disciplina de **test de caracterización**, documentada en
      `docs/conventions.md` y agregada como criterio de acceptance de la
      feature 14.
- [x] **C3** — `blockedReasons` sin comando de remedio ejecutable.
      Partido en convención (`docs/conventions.md`, incidentes reales
      futuros) + **feature 15** `blocked_reasons_remedy_commands`
      (`sdd: true`, pending) — evidencia línea por línea en `status.go`
      de los 5 mensajes que hoy diagnostican sin recetar.
- [x] **C4** — pregunta anti-sobre-ingeniería antes de sumar
      estado/verbo/flag nuevo. Sin feature — convención agregada a
      `CLAUDE.md` (nueva sección "Disciplina anti-sobre-ingeniería al
      proponer estructura nueva").
- [x] **C5** — responsabilidad humana / no-atribución IA. Sin feature —
      **corrección factual importante**: se verificó que ningún commit
      de los 50 del historial real lleva `Co-Authored-By` (la afirmación
      original de `ROADMAP.md`/del análisis decía lo contrario, sin
      verificar). El humano confirmó: el orquestador y cualquier
      subagente **nunca** ejecutan `git commit` — reglas nuevas en
      `CLAUDE.md` ("Reglas duras" + nueva sección "Responsabilidad") y en
      `.claude/agents/agent_developer.md`.
- [x] **C6** — `verify-report.md` por feature. Sin feature —
      `reviewer_agent.md` ya producía la matriz de cumplimiento
      (trazabilidad/completitud/sustancia) pero se descartaba al cerrar
      sesión. Se agregó `Write` a sus `tools` y un paso que persiste ese
      bloque en `specs/<name>/verify-report.md` (solo `sdd: true`).
- [x] **C7** — límite de rollback en el reporte de `agent_developer`.
      Sin feature — bullet agregado al paso 4 de
      `.claude/agents/agent_developer.md`.
- [x] **C8** — "qué NO prueba cada guardrail". Sin feature — tabla con
      los 6 guardrails reales de April (ledger, subject_hash, doctor,
      backup, ratchet, hash respetando `.gitignore`) y su límite
      concreto cada uno. **Reubicado a `CLAUDE.md` el 31/08/2026** (ver
      corrección de desvío más abajo) — no quedó en `docs/verification.md`.
- [x] **C9** (golden files para lo que `april init` escribe) —
      **descartado, confirmado por el humano (31/08/2026)**. Revisado a
      profundidad contra el código real: `TestCmdInitScaffoldsEmptyDir`
      (scaffold_test.go:277-294) ya compara byte a byte contra el
      archivo fuente en vivo, cubriendo lo único que un golden file
      protegería a nivel de mecanismo (`go:embed`). Los archivos
      embebidos en la raíz (`CLAUDE.md`, `AGENTS.md`,
      `.claude/agents/*.md`) aparecen en 23 de ~50 commits del historial
      — un golden ahí se rompería casi cada sesión por evolución
      intencional, no por regresión. Lo único estable
      (`templates/docs/*.md`, placeholders `_pendiente_`) ya está
      cubierto por la misma comparación existente. Ver `ROADMAP.md` para
      el detalle completo.
- [x] **C10** (política de retención/poda para ledger y backups) —
      **resuelto sin feature, confirmado por el humano (31/08/2026)**.
      Medido contra el repo real: ledger con 24 entradas/6067 bytes
      (features 5-12), 0 directorios en `.claude/backups/` — volumen
      demasiado bajo para justificar tooling de poda automática. Sección
      "Retención — ledger y backups" con umbrales concretos (ledger:
      ~500 entradas/~150 KB; backups: ~10 directorios) para revisión
      manual al cerrar sesión. **Reubicada a `CLAUDE.md`** (ver
      corrección de desvío más abajo) — no quedó en `docs/verification.md`.
- [x] **C11** (presupuesto de tamaño para `.claude/agents/*.md`) —
      **resuelto, confirmado por el humano (31/08/2026)**. Medido: 3 de
      5 agentes ya superan los ~1000 tokens que gentle-ai usa como tope
      duro (`reviewer_agent.md` ~1628, con crecimiento real de 2510 a
      ~6515 bytes en dos meses) — la premisa de "ajustados hoy" del
      `ROADMAP.md` no se sostuvo. Se descartó el tope duro de gentle-ai
      (no encaja con agentes de April, que son contratos completos, no
      skills angostas). Regla en dos partes: cualitativa (prosa de "por
      qué" va a `docs/`) + tope numérico blando de ~1500 tokens como
      señal de revisión. **Reubicada a `CLAUDE.md`** (ver corrección de
      desvío más abajo) — no quedó en `docs/conventions.md`.
- [x] **C12** (ratchet como patrón reusable) — **resuelto sin cambio de
      código, confirmado por el humano (31/08/2026)**. Revisado
      `doctor.go`: la premisa de "bespoke" no se sostuvo — `Metrics` ya
      es `map[string]int`, `DebtMetrics` ya es lista,
      `evaluateDebtRatchet` ya es genérico. Se documentó el patrón en
      `docs/conventions.md` citando esas piezas, sin tocar código.
- [x] **Corrección de desvío (planteada por el humano, 31/08/2026):**
      C8, C10 y C11 se habían escrito primero en `docs/verification.md`
      y `docs/conventions.md` de la **raíz** — que es la documentación de
      April-como-producto, fuera del `go:embed` de `scaffold.go`. Ese
      contenido describe mecanismos (`ledger`, `backups`,
      `.claude/agents/*.md`) que **todo proyecto scaffoldeado hereda**,
      así que nunca habría llegado a un proyecto nuevo. Movidos los tres
      a una sección nueva en `CLAUDE.md` ("Mecanismos incorporados de
      April"), que sí se embebe y propaga. C2 y C12 se dejaron donde
      estaban (`docs/conventions.md`) porque son guía para quien
      desarrolla el **código Go** de April, algo que un proyecto que solo
      usa el binario compilado nunca toca — ahí no había desvío.
- [x] **Hallazgo nuevo, resuelto y cerrado (31/08/2026):** verificando el
      `go:embed` se encontró que `.claude/verify-ledger.jsonl`
      (trackeado en git, cae dentro del patrón `.claude` del `go:embed`
      de `scaffold.go:33`) no tenía guard de exclusión — el único guard
      existente (`scaffold.go:288-291`) es específico para
      `.claude/manifest.json`. **Feature 16**
      (`scaffold_excludes_verify_ledger`, `sdd: false`, confirmado por el
      humano) creada vía `planner_agent`, implementada por
      `agent_developer` (guard nuevo en `scaffold.go:303-305` reusando
      la constante `verifyLedgerPath` de `verify.go:26`; test
      `TestVerifyLedgerEmbebidoNuncaSePropaga` en `scaffold_test.go`,
      análogo a `TestManifestJsonEmbebidoNuncaSePropaga`), revisada por
      `reviewer_agent` (**APPROVED**, sin objeciones — verificó guard,
      test y `go build`/`go test ./...` en verde de forma independiente)
      y cerrada `done` con aprobación explícita del humano. Ciclo
      completo Implementación→Revisión→cierre en esta sesión.
- [x] **Gap de proceso encontrado y corregido (31/08/2026):** al cerrar
      la feature 14 (primera vez en la sesión que se sigue el gate de
      cierre al pie de la letra hasta el final), se notó que las
      features 13 y **16** se habían marcado `done` sin correr `april
      verify record`/`april review record` — el ledger real
      (`.claude/verify-ledger.jsonl`) no tenía evidencia de ninguna de
      las dos, pese a que `require_tests_to_close`/`require_review_to_close`
      son reglas de `feature_list.json`. Para la **feature 13** no
      aplica (edición directa sin `agent_developer`/`reviewer_agent`,
      mismo patrón sin ledger que la feature 3 — precedente ya
      establecido antes de que existiera el mecanismo del ledger). Para
      la **feature 16** sí era un gap real (pasó por el flujo completo
      `agent_developer`→`reviewer_agent` APPROVED) — el humano aprobó
      backfillearlo: se corrió `april verify record --feature 16 --
      go test ./...` y `april review record --feature 16 --verdict
      APPROVED` con el binario reconstruido en `/tmp/april-fresh`
      (verde, sin cambios de código). No rompía nada mecánicamente hoy
      (`no_test_evidence`/`no_review_verdict` se eximen una vez
      `status: done`), pero dejaba el rastro de auditoría incompleto.
      **Lección de proceso adoptada para lo que sigue:** después de que
      `reviewer_agent` da su veredicto, correr `april verify record`/
      `april review record` (binario reconstruido, no el de PATH si
      puede estar desactualizado) **antes** de mover `feature_list.json`
      a `done` — no asumir que el veredicto narrado alcanza sin quedar
      también en el ledger.
- [x] **C13** (disciplina de commits por unidad de trabajo entregable) —
      **descartado, confirmado por el humano (31/08/2026)**. La premisa
      ("guía a `agent_developer` para commitear por ticket") quedó
      contradicha por C5, resuelto antes en esta misma sesión: ningún
      agente commitea jamás, solo el humano — no hay dónde aplicar la
      convención propuesta. Cierra la segunda comparación contra
      `gentle-ai`: los 13 candidatos (C1-C13) quedan resueltos.
- [x] **Feature 14** (`spec_gwt_mechanical_check`) — **ejecutada
      completa y cerrada `done` (31/08/2026)**. Fase Spec: `spec_writer`
      escribió `specs/spec_gwt_mechanical_check/spec.md` (25 historias);
      decisión clave aprobada por el humano al pasar a Tickets: el
      chequeo vive enteramente en `computeBlockedReasons` (`status.go`),
      `doctor.go` hereda la señal gratis sin código propio. Fase
      Tickets: `ticket_writer` propuso 2 tickets (test de
      caracterización sin bloqueadores → chequeo `no_gwt_coverage`
      bloqueado por el primero), aprobados tal cual por el humano antes
      de publicar los archivos. Fase Implementación: `agent_developer`
      implementó cada ticket por separado — el ticket 01 agregó
      `TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons`
      (fixture `fstest.MapFS` aislado por feature, decisión de diseño
      para evitar contaminación cruzada de `blockedReasons` global); el
      ticket 02 agregó el chequeo real en `status.go`
      (`gwtOptOutMarker`, `specSatisfiesGWT`) más 8 tests nuevos, y tocó
      el **fixture** (no la aserción) de un test preexistente para
      eliminar un confound no relacionado. Fase Revisión: `reviewer_agent`
      revisó la feature completa (no ticket por ticket, mismo patrón que
      features anteriores de varios tickets como la 5) — **APPROVED**,
      con verificación activa del punto de escrutinio del fixture
      tocado (lo reprodujo revirtiéndolo, confirmó que el fallo sin el
      cambio era real, no maquillado). `verify-report.md` archivado en
      `specs/spec_gwt_mechanical_check/`. 217 tests en verde.
- [x] **Feature 13** (`spec_template_gwt_rfc2119`) — **ejecutada y
      cerrada `done` (31/08/2026)**. Edición directa del orquestador
      (precedente: feature 3), sin `agent_developer`/`reviewer_agent` —
      confirmado explícitamente por el humano para esta feature. Diff
      acotado a `.claude/agents/spec_writer.md`: bloque Given/When/Then
      agregado junto al formato Como/quiero/para en `## User Stories`
      (con la aclaración de que no se fuerza en historias sin rama
      verificable), e instrucción RFC 2119 (MUST/SHOULD/MAY, con
      definición de qué rompe el acceptance) agregada en
      `## Implementation Decisions`. Ningún `specs/*/spec.md` existente
      tocado. Humano revisó el diff real (`git diff`) y aprobó el
      cierre.
- [x] **Continuación de sesión — hallazgo por log de CI:** el humano
      pegó un log real de GitHub Actions mostrando
      `TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons`
      (feature 14) fallando en CI con `no such file or directory` sobre
      `specs/<name>/spec.md` para las 6 features `done`. Investigada la
      causa raíz: dos bugs reales en el `.gitignore` de la **raíz** de
      este repo (no `templates/.gitignore`, que está bien tal cual).
- [x] **Feature 17** (`gitignore_root_tracks_specs`) — **ejecutada y
      cerrada `done` (31/08/2026)**. Causa: la línea `specs/` del
      `.gitignore` de la raíz era copia literal de `templates/.gitignore`
      (correcta ahí — proyecto scaffoldeado nuevo, `specs/` es estado de
      trabajo descartable), incorrecta en este propio repo, donde
      `specs/*.md` es la documentación SDD real que `CLAUDE.md` exige.
      `git ls-files specs/` confirmó que ninguna de las 7 specs
      existentes estuvo nunca en git. Fix: se sacó la línea de la raíz,
      verificado con `git check-ignore -v` contra las 7 specs reales
      (ninguna matchea ya). Nota nueva en `docs/conventions.md`.
      `reviewer_agent`: **APPROVED**. Ledger registrado
      (`april verify record`/`april review record --feature 17`).
- [x] **Feature 18** (`gitignore_root_tracks_docs`) — **ejecutada y
      cerrada `done` (31/08/2026)**. Hallazgo hermano de la 17,
      encontrado por `reviewer_agent` al revisar esa feature: `/docs/`
      también estaba en el `.gitignore` de la raíz — pero a diferencia
      de `specs/`, esta regla se agregó **a propósito** (commit
      `3c24d6b`, 24/08/2026) bajo el razonamiento "`docs/` es estado de
      trabajo... igual que `/feature_list.json`". El humano determinó
      que ese razonamiento no se sostiene: `docs/conventions.md`/
      `docs/verification.md` son documentación vinculante citada
      repetidamente esta sesión, no estado operativo descartable.
      `git log --diff-filter=D -- "docs/*.md"` confirmó que esos
      archivos estuvieron trackeados hasta el 24/08, cuando se sacaron
      del tracking junto con la regla. Fix: mismo tratamiento que la 17,
      nota "hermana" en `docs/conventions.md`. `reviewer_agent`:
      **APPROVED**. Ledger registrado (`--feature 18`).

Ver `ROADMAP.md` (sección "Segunda comparación") para el detalle completo
de cada candidato de C1-C13, y `progress/history.md` (secciones
"2026-08-31" y "2026-08-31 (continuación)") para la bitácora completa ya
consolidada — `progress/current.md` se reseteó al cerrar esta sesión,
queda limpio para la feature 15.

## Verification Evidence

Features 16 y 14 son el código nuevo de esta sesión (todo lo demás
resuelto fue documentación/config de agentes, o edición directa de
`.claude/agents/spec_writer.md` en la feature 13). Feature 15 sigue en
`pending`, sin implementación.

| Check | Resultado |
|---|---|
| `./init.sh` (antes de empezar) | verde, `blockedReasons: []` |
| `git status --short` (antes de empezar) | limpio, backlog 1-12 commiteado |
| `go build ./...` (feature 16, corrido por `agent_developer` y de nuevo por `reviewer_agent`) | verde |
| `go test ./...` (feature 16, ídem) | verde — `ok github.com/AlvaroEng98/HarnessInit` |
| `go test ./... -run 'TestVerifyLedgerEmbebidoNuncaSePropaga\|TestManifestJsonEmbebidoNuncaSePropaga' -v` | ambos `PASS` |
| `go test ./... -run 'TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons\|NoGwtCoverage\|HeredaNoGwtCoverage' -v` (feature 14, ticket 01+02) | 15 subtests `PASS` |
| `go test ./... -count=1` (feature 14, corrido por `agent_developer` y de nuevo por `reviewer_agent`) | verde — 217 tests, 0 FAIL |
| `april verify record --feature 14 -- go test ./...` / `april review record --feature 14 --verdict APPROVED` | registrado en `.claude/verify-ledger.jsonl`, `treeHash` compartido |
| `april status --json 14` (tras registrar) | `blockedReasons: []` |
| Backfill: `april verify record --feature 16 -- ...` / `april review record --feature 16 --verdict APPROVED` | registrado (ver "Decisions Made", gap de proceso) |
| `git check-ignore -v` contra las 7 specs reales (feature 17, corrido por `agent_developer` y de nuevo por `reviewer_agent` sobre las 7, no solo 1) | ninguna matchea |
| `git check-ignore -v` contra `docs/{architecture,conventions,specs,verification}.md` (feature 18, ídem) | ninguno matchea |
| `git diff --stat -- templates/.gitignore` (features 17 y 18) | vacío — sin cambios, confirmado que el template sigue correcto |
| `go build ./...`/`go test ./...` (features 17 y 18) | verde ambas veces |
| `april verify record`/`april review record --feature 17` y `--feature 18` | registrados en el ledger, `blockedReasons: []` para ambas |

## Files Changed (esta sesión)

- `ROADMAP.md` — reescrito completo: primera comparación comprimida,
  candidatos C1-C13 agregados, C1-C12 marcados con estado y decisión
  (incluye las notas de corrección de C8/C10/C11).
- `feature_list.json` — features 13, 14, 15 agregadas en `pending`;
  feature 16 (`scaffold_excludes_verify_ledger`) agregada, ejecutada y
  cerrada `done` en esta misma sesión.
- `scaffold.go` — guard nuevo para `.claude/verify-ledger.jsonl`
  (feature 16), junto al guard existente de `.claude/manifest.json`.
- `scaffold_test.go` — test nuevo `TestVerifyLedgerEmbebidoNuncaSePropaga`
  (feature 16).
- `.claude/agents/spec_writer.md` — bloque Given/When/Then en
  `## User Stories` e instrucción RFC 2119 en
  `## Implementation Decisions` (feature 13).
- `specs/spec_gwt_mechanical_check/spec.md` — spec nueva (feature 14, 25
  historias), `verify-report.md` archivado por `reviewer_agent`.
- `specs/spec_gwt_mechanical_check/tickets/01-...md` y `02-...md` —
  tickets publicados y cerrados `done` (feature 14).
- `status.go` — `gwtOptOutMarker`, `specSatisfiesGWT`, precómputo
  `specSatisfiesGWTByFeature`, chequeo `no_gwt_coverage` en
  `computeBlockedReasons` (feature 14, ticket 02).
- `status_test.go` — `TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons`
  (ticket 01); 6 tests de negocio + 1 read-only para `no_gwt_coverage`,
  y ajuste puntual del fixture (no la aserción) de
  `TestFeatureConSpecYSinTicketsEsFaseTickets` (ticket 02).
- `doctor_test.go` — `TestDoctorHeredaNoGwtCoverageSinCodigoPropio`
  (feature 14, ticket 02).
- `.claude/verify-ledger.jsonl` — evidencia real registrada para las
  features 14 y 16 (backfill de la 16, ver "Decisions Made").
- `CLAUDE.md` — nueva sección "Disciplina anti-sobre-ingeniería al
  proponer estructura nueva" (C4); "Reglas duras" (nunca `git commit`) y
  nueva sección "Responsabilidad" (C5); nueva sección "Mecanismos
  incorporados de April" con tres subsecciones (C8, C10, C11 —
  reubicadas acá desde `docs/*.md` de la raíz por el desvío corregido
  esta sesión).
- `docs/conventions.md` — nuevas secciones "Cambios a la lógica de
  derivación de fase" (C2), "Incidentes reales de
  blockedReasons/nextRecommended confusos" (C3), "Ratchet de deuda
  progresiva — patrón reusable, no bespoke" (C12); sección de C11
  agregada y luego removida (reemplazada por puntero a `CLAUDE.md`).
- `docs/verification.md` — secciones de C8 y C10 agregadas y luego
  removidas (reemplazadas por puntero a `CLAUDE.md`).
- `.claude/agents/reviewer_agent.md` — `Write` agregado a `tools`, paso
  nuevo que persiste `specs/<name>/verify-report.md` (C6).
- `.claude/agents/agent_developer.md` — bullet de límite de rollback en
  el reporte (C7); bullet de "nunca `git commit`" (C5).
- `progress/history.md` — consolidada la sección "2026-08-31" (C1-C13,
  features 13/14/16, lección de proceso del ledger) y "2026-08-31
  (continuación)" (features 17/18). `progress/current.md` reseteado,
  listo para la feature 15.
- `.gitignore` — eliminada la línea `specs/` (feature 17) y el bloque
  `/docs/` (feature 18) de la raíz de este repo. `templates/.gitignore`
  sin tocar en ninguna de las dos.
- `docs/conventions.md` — dos secciones nuevas: "`templates/.gitignore`
  vs `.gitignore` de la raíz — no son el mismo target" (feature 17) y su
  "Hallazgo hermano: `/docs/`" (feature 18).

## Decisions Made

- **Nunca commitear.** El orquestador y cualquier subagente lanzado
  nunca ejecutan `git commit` — el humano commitea siempre, manualmente.
  Ningún commit lleva atribución de autoría a la IA. Confirmado
  31/08/2026 (C5). Corrige una afirmación previa incorrecta sobre este
  mismo tema (ver arriba).
- **Test de caracterización, no shadow-flag de runtime**, para cambios a
  `derivePhase`/`computeBlockedReasons`/`nextRecommendedText` — el
  mecanismo de gentle-ai no traduce 1:1 a un binario CLI sin tráfico en
  vivo (C2).
- **`verify-report.md` no compite con el ledger** como fuente de verdad
  — es lectura humana archivada, el ledger sigue siendo lo único que
  consulta `require_review_to_close` (C6).
- Cada candidato aceptado que no requiere código nuevo se resuelve como
  edición directa de docs/config (`CLAUDE.md`, `docs/*.md`,
  `.claude/agents/*.md`) — sin abrir feature ni pasar por
  `agent_developer`, consistente con lo que ya permite `CLAUDE.md`
  ("Qué puedes editar tú mismo").
- **Dónde documentar cada convención nueva depende de si describe un
  mecanismo que todo proyecto scaffoldeado hereda o uno específico del
  desarrollo del propio April.** Regla nueva, confirmada por el humano
  el 31/08/2026 tras detectar el desvío de C8/C10/C11: si el contenido
  describe un mecanismo del binario `april` (ledger, backups,
  `.claude/agents/*.md`, cualquier cosa que un proyecto scaffoldeado
  reciba y pueda tocar por su cuenta), va a `CLAUDE.md` — el único
  archivo de convenciones que el `go:embed` de `scaffold.go` propaga. Si
  describe cómo desarrollar el **código Go** de April en sí (lógica de
  `status.go`, `doctor.go`, tests) — algo que un proyecto que solo usa el
  binario compilado nunca toca — va a `docs/conventions.md`/
  `docs/verification.md` de la raíz, que no propagan y no necesitan
  hacerlo.
- **Registrar en el ledger real antes de marcar `done`, no solo confiar
  en el veredicto narrado.** Tras el veredicto de `reviewer_agent`,
  correr `april verify record --feature <id> -- <comando>` y `april
  review record --feature <id> --verdict <valor>` (con un binario
  reconstruido — `go build -o /tmp/april-fresh .` — no el de PATH, que
  puede estar desactualizado) **antes** de mover `feature_list.json` a
  `done`. Confirmado el 31/08/2026 tras encontrar que las features 13 y
  16 se habían cerrado sin esto (13 correctamente exenta por ser edición
  directa sin agente; 16 backfilleada por ser un gap real). Ver bullet
  de "Gap de proceso" en "Completed This Session".
- **`templates/.gitignore` y el `.gitignore` de la raíz tienen targets
  distintos — nunca copiar una línea de uno a otro sin revalidar el
  contexto.** Confirmado el 31/08/2026 (continuación) tras encontrar dos
  bugs reales: `specs/` (copy-paste accidental, feature 17) y `/docs/`
  (razonamiento que dejó de sostenerse, feature 18) excluían del
  `.gitignore` de la raíz documentación vinculante real de este repo.
  Documentado en `docs/conventions.md`.

## Blockers / Risks

- Ninguno técnico — backlog completo (1-14, 16-18) `done`, ninguna
  feature `in_progress`.
- Segunda comparación contra `gentle-ai` **cerrada** (C1-C13 resueltos).
- Solo queda **feature 15** (`blocked_reasons_remedy_commands`) en
  `pending`, `sdd: true`, independiente — nunca estuvo bloqueada.
  Requiere `spec_writer` antes de tickets/implementación.
- **Acción pendiente del humano, no de ningún agente:** correr `git add
  specs/ docs/ .gitignore .claude/verify-ledger.jsonl progress/ && git
  commit` — las specs y la documentación de este repo aparecen `??`
  (untracked) por primera vez tras el fix de las features 17/18, y el
  ledger/progress de esta sesión tampoco está commiteado todavía. Sin
  esto, CI sigue fallando exactamente como mostró el log que motivó
  estas dos features.

## Next Session Startup

1. Leer `AGENTS.md`/`CLAUDE.md` (ya incluye las reglas nuevas de esta
   sesión: nunca commitear, disciplina anti-sobre-ingeniería,
   responsabilidad humana, "Mecanismos incorporados de April").
2. Verificar si el humano ya corrió `git add specs/ docs/ .gitignore
   .claude/verify-ledger.jsonl progress/ && git commit` (ver "Blockers /
   Risks") — si no, recordárselo antes de asumir que CI está en verde.
3. Correr `./init.sh` — debe estar en verde.
4. Correr `april status --json` — debe reportar `phase: closed` (la
   única feature no cerrada, 15, sigue en `pending`, no `in_progress`).
5. Preguntar al humano: ¿arrancar la feature 15 (Fase Spec), o hay algo
   más para revisar antes?

## Recommended Next Step

- Confirmar que el commit de `specs/`+`docs/` ya se hizo (ver Blockers)
  — sin eso, cualquier CI que corra sigue fallando igual que el log que
  disparó las features 17/18.
- Único trabajo de backlog restante: feature 15
  (`blocked_reasons_remedy_commands`) — lanzar `spec_writer` para
  arrancar su Fase Spec, mismo patrón ya usado con la feature 14 en esta
  sesión (spec_writer → aprobación humana → ticket_writer con desglose
  propuesto → aprobación humana → agent_developer por ticket →
  reviewer_agent sobre la feature completa → `april verify
  record`/`april review record` → aprobación humana de cierre).
- Si se abren convenciones nuevas más adelante, ubicarlas según la regla
  de "Decisions Made" (mecanismo heredado por todo proyecto →
  `CLAUDE.md`; desarrollo del código Go de April → `docs/`).
- Recordar el nuevo paso del gate de cierre (ver "Decisions Made"):
  registrar en el ledger real (`april verify record`/`april review
  record`) antes de mover cualquier feature a `done`, no solo confiar en
  el veredicto narrado de `reviewer_agent`.
