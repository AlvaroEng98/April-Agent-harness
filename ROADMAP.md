# ROADMAP — April con gentle-ai como frontera

> Plan de evolución del arnés, derivado de comparar April contra
> [gentle-ai](https://github.com/Gentleman-Programming/gentle-ai). No es un
> plan de réplica: gentle-ai marca el destino, el camino es el de este
> repositorio — **se adopta metodología, nunca se copia implementación ni
> escala**.
>
> Primera comparación: 25-26/08/2026. Segunda comparación: 28/08/2026.

---

## Tesis (vigente para ambas rondas)

Un solo cambio de fondo; todo lo demás es consecuencia:

> **Mover la autoridad del estado de la prosa al binario.**

Frase del README de gentle-ai que resume la brecha:

> *"Trust what the system can derive, not agent narration."*

---

## Primera comparación (25-26/08/2026) — CERRADA

Resultado: 6 etapas (E0-E6), todas implementadas como features 1-12 de
`feature_list.json`, todas en `done`. Detalle completo (decisiones de
diseño, objeciones de revisión, lecciones de proceso) en
`progress/history.md` — no se repite acá.

Mecanismos entregados: `april status --json` (árbitro de fase),
`CLAUDE.md` enrutando solo por `nextRecommended`/`blockedReasons`,
`april feature set-status` (única vía de escritura, grafo de transiciones
validado), ledger append-only de evidencia (`verify record` + veredictos
de review), candidato congelado por `subject_hash` con profundidad de
revisión según sensibilidad del diff, `april doctor` read-only con ratchet
de deuda progresiva, backup automático antes de `applyPlan`, hash de árbol
respetando `.gitignore`.

**Decisión de fondo tomada:** *"B, llegando por A"* — `april status`
empezó advisory (E1/E2) y, tras validarse en uso real, `feature_list.json`
pasó a escribirse exclusivamente vía `april feature set-status` (E3). No
se reabre esta decisión.

---

## Fuera de alcance, deliberadamente (vigente para ambas rondas)

Filtro ya aplicado a los candidatos de la segunda ronda — si algo de
gentle-ai cae en esta lista, se descarta sin más análisis:

- Soporte multi-runtime (16 agentes)
- Backends de persistencia intercambiables (`engram | openspec | hybrid`)
- Grafo de fases dinámico — las 5 fases fijas de April son suficientes
- Contratos negociados con transiciones opacas y `consent envelopes`
- Firma minisign de releases y canales stable/beta
- TUI
- Gobernanza multi-contribuidor (flujo issue-first, labels de PR, CI
  multi-OS, `renovate`/`goreleaser`, triage de backlog por "waves") — April
  es de un solo operador, no tiene contribuidores externos
- Ruteo graduado "direct inline / delegated / SDD" por conteo de
  archivos — contradice la regla dura ya vigente ("nunca implementas
  código directamente, ningún flujo, ni una línea")
- Registro completo de guard-population (fingerprints AST/SHA256) —
  proporcional a un codebase de 85k líneas con superficie de seguridad
  Windows/RAR, sin equivalente en la escala de April
- Harness E2E con modelo mockeado — resuelve un problema que April no
  tiene (el binario `april` nunca llama a un modelo)

---

## Segunda comparación (28/08/2026) — candidatos para atacar 1 a 1

Metodología de esta ronda: dos análisis independientes (agentes sin
contexto compartido, mismo encargo) exploraron `/home/avalor/Proyectos/gentle-ai`
completo contra el estado actual de April (post features 1-12), buscando
mecanismos/convenciones NO cubiertos por la primera ronda. Se comparan
resultados abajo — el ítem C1 es el único donde ambos análisis
convergieron de forma independiente por caminos distintos (señal más
fuerte).

**Cómo usar esta lista:** se analiza un candidato a la vez con el humano.
Al aceptar uno, se decide `sdd: true/false` igual que cualquier feature
nueva y se lanza `planner_agent`/`spec_writer` según corresponda. Al
descartar uno, se anota el motivo acá mismo (no se borra — deja registro
de qué ya se evaluó y por qué no se hizo).

Estado de cada candidato: **pendiente de análisis**, salvo que se indique
lo contrario.

---

### C1 — Escenarios Given/When/Then + RFC 2119 en specs, verificados mecánicamente (consenso de ambos análisis)

**Estado: ACEPTADO (31/08/2026).** Partido en dos features vía
`planner_agent` — `feature_list.json` id 13 (`spec_template_gwt_rfc2119`,
`sdd: false`) e id 14 (`spec_gwt_mechanical_check`, `sdd: true`, bloqueada
por la 13). Backlog aprobado por el humano, en `pending`.

- **Mecanismo en gentle-ai:** `openspec/config.yaml` (`rules.specs`) exige
  formato Given/When/Then por escenario y palabras clave RFC 2119
  (MUST/SHALL/SHOULD/MAY) en las decisiones de diseño.
- **Problema que resuelve:** cierra la brecha entre una historia de
  usuario en prosa y el test concreto que la cubre — hoy `reviewer_agent`
  tiene que reconstruir esa traza a mano (paso 2 de `reviewer_agent.md`).
- **Relevancia para April:** alta. La plantilla de `spec_writer` tiene
  "Historias de usuario" en formato "Como/quiero/para" pero sin escenario
  de aceptación estructurado, y "Implementation Decisions" en prosa libre
  sin distinguir obligatorio de preferido.
- **Cómo se adaptaría:** dos partes que se refuerzan —
  (a) extender la plantilla de `spec_writer` con un bloque Given/When/Then
  por historia de usuario y MUST/SHOULD/MAY en Implementation Decisions;
  (b) extender `april doctor`/`april status` con un chequeo estructural
  barato: ¿`specs/<name>/spec.md` contiene al menos un bloque
  Given/When/Then?, ¿cada ticket declara `Blocked by`? Sin motor de reglas
  nuevo.

---

### C2 — Evaluación en sombra antes de cambiar la lógica de derivación de fase

**Estado: ACEPTADO CON CORRECCIÓN (31/08/2026).** No se copia el
mecanismo de shadow-flag en runtime de gentle-ai — resuelve un problema
de rollout gradual en un sistema con tráfico en vivo; `april` es un
binario CLI sin ese problema (se recompila y se invoca fresco cada vez).
Se adopta la disciplina de fondo como **test de caracterización**, no
como feature aparte: documentada en `docs/conventions.md` ("Cambios a la
lógica de derivación de fase") y ya incorporada como criterio de
acceptance de la feature 14 (`spec_gwt_mechanical_check`), que es el
primer cambio real a `derivePhase`/`computeBlockedReasons` desde que
existen.

- **Mecanismo en gentle-ai:** `docs/architecture/rdd-shadow-evaluation.md`
  — la lógica nueva corre en paralelo a la vieja, solo observa (nunca
  bloquea/altera nada), loguea divergencia a stderr, y el apagado tiene
  una prueba dedicada que exige salida byte-idéntica sin el código de
  sombra.
- **Problema que resuelve:** migrar la lógica que decide un gate/estado
  sin arriesgar el camino en vivo, con evidencia medible de divergencia
  antes del corte.
- **Relevancia para April:** alta. Cuando la lógica de `april status`
  (nuevas reglas de fase, nuevos `blockedReasons`) cambie, el riesgo es
  el mismo que resolvió gentle-ai acá.
- **Cómo se adaptaría:** una bandera de entorno (ej. `APRIL_SHADOW=1`)
  que, al correr `april status`, calcula también la regla propuesta y
  loguea a stderr si difiere de la vigente, más un test que pruebe que
  con la bandera apagada el output es idéntico al build sin el cambio.
  Aplica la skill `migration` ya disponible en el harness.

---

### C3 — `blockedReasons` confusos como corpus de regresión permanente

**Estado: ACEPTADO, partido en dos (31/08/2026).** Convención en
`docs/conventions.md` ("Incidentes reales de blockedReasons/nextRecommended
confusos") para incidentes futuros, más una feature concreta y ya
specificable hoy: `feature_list.json` id 15
(`blocked_reasons_remedy_commands`, `sdd: true`) — cada mensaje de
`blockedReasons` gana el comando `april ...` exacto que lo resuelve, sin
tocar las substrings de contrato ya establecidas.

- **Mecanismo en gentle-ai:** `bench/README.md` clasifica cada bloqueo
  real de forma mecánica, y cada incidente reportado se fija para
  siempre como test (`skills/gentle-ai-bench/SKILL.md`: *"cuando cambia
  una semántica ratificada, grep el corpus por lo que fija el
  comportamiento VIEJO"*).
- **Problema que resuelve:** evita que un mensaje de bloqueo se vuelva
  "callejón sin salida" silenciosamente al evolucionar el código.
- **Relevancia para April:** alta, directamente alineada con el contrato
  de `blockedReasons`/`nextRecommended` — es justo lo que puede
  degradarse sin que nadie lo note.
- **Cómo se adaptaría:** no un harness separado — cada vez que una
  sesión real tope con un `blockedReasons` confuso o un callejón sin
  salida, agregar un test de tabla en el propio test suite de `april`
  que fije ese escenario exacto (input → texto exacto esperado, con un
  comando `april ...` ejecutable dentro).

---

### C4 — Pregunta anti-sobre-ingeniería antes de sumar estado/verbo/flag nuevo

**Estado: ACEPTADO (31/08/2026).** Sin feature — convención agregada
directamente a `CLAUDE.md` (sección "Disciplina anti-sobre-ingeniería al
proponer estructura nueva"), aplicable a `planner_agent`/`ticket_writer`
antes de proponer campo/verbo/flag nuevo.

- **Mecanismo en gentle-ai:**
  `skills/systemic-issue-triage/SKILL.md`: *"¿Agrega un estado, un verbo,
  un flag de config, un gate, o una representación paralela de una
  verdad ya existente? Si sí, rediseña — el fix correcto usualmente
  ELIMINA o RELAJA algo."*
- **Problema que resuelve:** evita que el proyecto acumule
  estados/flags/verbos hasta volverse difícil de mantener — motivó 7
  waves de simplificación en gentle-ai.
- **Relevancia para April:** alta como disciplina preventiva. El árbitro,
  el ledger y el grafo de transiciones de `feature_list.json` son
  justo el tipo de superficie que crece por goteo si nadie hace esta
  pregunta antes.
- **Cómo se adaptaría:** una línea en `docs/conventions.md`/`CLAUDE.md`:
  antes de que `planner_agent`/`ticket_writer` propongan un campo de
  estado, verbo de CLI o flag nuevo, el orquestador pregunta
  explícitamente "¿esto elimina o consolida más de lo que agrega?".
  Costo: cero código.

---

### C5 — Postura explícita de responsabilidad humana / no-atribución a la IA

**Estado: ACEPTADO (31/08/2026).** Sin feature — reglas agregadas
directamente a `CLAUDE.md` ("Reglas duras" y nueva sección
"Responsabilidad").

**Corrección sobre la evidencia inicial:** el análisis original afirmó
que *"el flujo estándar de commits de este harness agrega
`Co-Authored-By: Claude` por defecto"*, presentado como tensión sin
resolver. Se verificó contra los 50 commits reales del historial
(`git log --all`) y **ninguno lleva ese trailer** — la afirmación era
incorrecta, no había ninguna tensión de facto, solo el riesgo de que
apareciera si el orquestador llegara a commitear alguna vez (nunca lo
había hecho hasta ahora porque el humano commitea siempre a mano). El
humano confirmó explícitamente: el orquestador y cualquier subagente
**nunca** ejecutan `git commit`, y ningún commit lleva atribución a la
IA. Se agregó también la línea de responsabilidad humana explícita sobre
el cierre de features.

- **Mecanismo en gentle-ai:** `AI_POLICY.md` — el humano es siempre
  responsable completo; la IA nunca recibe `Co-Authored-By`/
  `Reviewed-by`/`Signed-off-by` ni aprobación; lista explícita de
  comportamientos inaceptables (inventar APIs/evidencia/resultados de
  test, enmascarar síntomas).
- **Adaptación real aplicada:** no se portó el documento — dos reglas
  cortas en `CLAUDE.md` (nunca commitear + responsabilidad humana
  explícita sobre el cierre), más el mismo límite duplicado en
  `.claude/agents/agent_developer.md` por ser el subagente con `Bash`
  más probable de tocar git.

---

### C6 — `verify-report.md` por feature, archivado junto a la spec

**Estado: ACEPTADO, adaptado (31/08/2026).** Sin feature — evidencia real
mostró que `reviewer_agent.md` ya produce, en su "Formato de salida", la
misma matriz de cumplimiento que gentle-ai archiva (trazabilidad,
completitud, sustancia) — solo se descartaba al cerrar sesión, comprimida
a una frase en `progress/history.md`. Se agregó `Write` a sus `tools` y
un paso que persiste ese bloque, sin resumir, en
`specs/<name>/verify-report.md` (solo `sdd: true`). El ledger sigue
siendo la única fuente de gate.

- **Mecanismo en gentle-ai:**
  `openspec/changes/archive/<fecha>-<nombre>/{proposal,design,tasks,verify-report}.md`
  — cada change cerrado deja un reporte estructurado (completitud,
  matriz de cumplimiento de spec, hallazgos por severidad, veredicto)
  archivado permanentemente junto a la spec.
- **Problema que resuelve:** trazabilidad humana legible por feature,
  distinta del ledger append-only (que es evidencia cruda, no lectura
  cómoda).
- **Relevancia para April:** el veredicto de `reviewer_agent` hoy vive en
  `.claude/verify-ledger.jsonl` (fuente de verdad) y en
  `progress/history.md` (consolidado, no por feature). No hay artefacto
  por feature que junte spec + tickets + veredicto para auditoría futura.
- **Cómo se adaptaría:** al cerrar una feature, `reviewer_agent` (o el
  orquestador) escribe un resumen corto derivado del ledger — nunca
  compitiendo con él como fuente de verdad — en
  `specs/<name>/verify-report.md`.

---

### C7 — Límite de rollback explícito en el reporte de `agent_developer`

**Estado: ACEPTADO (31/08/2026).** Sin feature — bullet agregado al paso
4 de `.claude/agents/agent_developer.md`: el reporte ahora exige el
límite de rollback (qué archivos/funciones/comportamiento exacto
revertiría la unidad), relevante porque varios tickets de una misma
feature suelen tocar el mismo archivo (ej. feature 8).

- **Mecanismo en gentle-ai:** `skills/work-unit-commits/SKILL.md` — cada
  unidad de trabajo debe declarar "Rollback boundary names the exact
  files/behavior removable without unrelated work", incluso si el
  trabajo no está commiteado todavía.
- **Problema que resuelve:** quien aprueba el cierre sabe, sin releer el
  diff completo, qué habría que deshacer si la unidad se revierte.
- **Relevancia para April:** el reporte de `agent_developer` hoy pide
  archivos tocados + comandos + cobertura de acceptance, pero no el
  límite de reversión.
- **Cómo se adaptaría:** agregar un bullet al formato de reporte de
  `agent_developer`: "límite de rollback — qué archivos/comportamiento
  exacto habría que revertir para deshacer esta unidad sin tocar nada
  más". Cambio de una línea en el agente.

---

### C8 — "Qué NO prueba este mecanismo", documentado junto a cada guardrail

**Estado: ACEPTADO (31/08/2026).** Sin feature — nueva sección en
`CLAUDE.md` ("Mecanismos incorporados de April" → "Qué cada guardrail NO
prueba"), tabla con los seis guardrails reales de April (ledger,
subject_hash, doctor, backup, ratchet, hash respetando .gitignore) y su
límite concreto cada uno. **Corrección aplicada el 31/08/2026:** se
había escrito primero en `docs/verification.md` de la raíz por error —
ese archivo no propaga a proyectos scaffoldeados (no está en el
`go:embed` de `scaffold.go`), y este contenido describe mecanismos que
todo proyecto hereda. Movido a `CLAUDE.md`, que sí se embebe.

- **Mecanismo en gentle-ai:** patrón repetido en tres subsistemas
  distintos — `docs/architecture/guard-population.md:47-49` ("Proof
  boundary"), `scripts/deadcode-ratchet.sh:14-18` ("What this does NOT
  catch, stated plainly so nobody trusts it further than it goes"),
  `docs/rollback.md:69-72` ("What rollback does NOT cover").
- **Problema que resuelve:** evita que la sola existencia de un
  guardrail (ledger, ratchet, backup) genere más confianza de la que
  realmente respalda.
- **Relevancia para April:** alta y barata. April ya tiene varios
  mecanismos "confiar en el disco, no en narración" (ledger, `april
  doctor`, backup pre-`init`) pero ninguno documenta sus límites
  explícitamente.
- **Cómo se adaptaría:** en `docs/verification.md`/`docs/conventions.md`,
  junto a cada mecanismo, un párrafo corto "qué esto NO prueba/cubre".
  Cero código.

---

### C9 — Golden files para lo que `april init` escribe — DESCARTADO (31/08/2026)

- **Mecanismo en gentle-ai:** `testdata/golden/*.golden` fija byte a
  byte el output esperado de cada generador de scaffold.
- **Problema que resuelve:** un cambio silencioso en la lógica de
  generación de scaffold se detecta como diff exacto, no solo por
  revisión manual.
- **Decisión: descartado, confirmado por el humano.** Evidencia
  verificada en `scaffold.go`/`scaffold_test.go`:
  - `TestCmdInitScaffoldsEmptyDir` (scaffold_test.go:277-294) ya compara
    byte a byte lo que escribe `scaffoldInit` contra el archivo fuente en
    vivo (`os.ReadFile(c.srcPath)`) — cubre que `go:embed` copia
    correctamente lo que hay hoy en el repo, que es lo único que un
    golden file adicional protegería a nivel de mecanismo.
  - Los archivos que `april init` embebe directo en la raíz (`CLAUDE.md`,
    `AGENTS.md`, `.claude/agents/*.md`, vía la directiva `//go:embed` de
    `scaffold.go:33`) aparecen en 23 de ~50 commits del historial del
    repo — un golden file ahí se rompería casi cada sesión por evolución
    intencional del proceso, no por regresión.
  - Lo único razonablemente estable es `templates/docs/*.md` (los
    placeholders `_pendiente_` de Fase Grill — 8 commits en total,
    ninguno desde el 25/08), pero ese contenido trivial ya está cubierto
    por la misma comparación byte a byte que usa
    `TestCmdInitScaffoldsEmptyDir`.
  - Conclusión: el golden file protegería lo que menos vale la pena
    proteger (contenido vivo, cambia por diseño) y sería redundante donde
    sí sería barato aplicarlo (el placeholder estable, ya cubierto).

---

### C10 — Política de retención/poda para lo que crece sin límite — RESUELTO (31/08/2026)

- **Mecanismo en gentle-ai:** `docs/rollback.md:24-31` (retención de 5
  backups, pin, dedup por checksum); `docs/trigger-rules.md:91-95`
  ("Review authority accumulates without bound... hasta ahora un store
  degradado no tenía salida" — motivó `review store-reset`).
- **Problema que resuelve:** gentle-ai resolvió esto reactivamente,
  después de que el problema ya doliera.
- **Decisión: sin feature ni tooling nuevo, confirmado por el humano.**
  Medido contra el estado real del repo: `.claude/verify-ledger.jsonl`
  tenía 24 entradas (6067 bytes, ~253 bytes/entrada) cubriendo las
  features 5-12; `.claude/backups/` tenía 0 directorios. Construir poda
  automática ahora sería resolver un problema que hoy no existe
  (disciplina anti-sobre-ingeniería, C4). Se agregó en su lugar una
  sección a `CLAUDE.md` ("Mecanismos incorporados de April" →
  "Retención — ledger y backups") con umbrales concretos disparadores
  (ledger: ~500 entradas/~150 KB; backups: ~10 directorios) para
  revisión manual al cerrar sesión — no automatizada. **Corrección
  aplicada el 31/08/2026:** se había escrito primero en
  `docs/verification.md` de la raíz por error — ese archivo no propaga a
  proyectos scaffoldeados y este contenido describe un mecanismo que
  todo proyecto hereda. Movido a `CLAUDE.md`, que sí se embebe.

---

### C11 — Presupuesto de tamaño para `.claude/agents/*.md` — RESUELTO (31/08/2026)

- **Mecanismo en gentle-ai:** `docs/skill-style-guide.md:27-31` —
  objetivo 180-450 tokens de cuerpo, máximo duro 1000, orden de
  secciones fijo, regla explícita de mover prosa/historia a
  `references/`.
- **Problema que resuelve:** evita que las instrucciones de un agente se
  inflen con el tiempo hasta volverse caras de cargar y difíciles de
  seguir.
- **Decisión: convención en dos partes, confirmado por el humano.**
  Medido contra el repo real: 3 de 5 agentes (`reviewer_agent.md` ~1628
  tokens, `ticket_writer.md` ~1309, `spec_writer.md` ~1093) ya superan
  los ~1000 tokens que gentle-ai usa como tope duro — la premisa inicial
  de "razonablemente ajustados hoy" no se sostuvo con datos.
  `reviewer_agent.md` además muestra crecimiento real medido (2510 →
  ~6515 bytes en dos meses). Se descartó copiar el tope duro de gentle-ai
  tal cual (sus skills son add-ons angostos; un agente de April es un
  contrato completo — pasos, tabla de veredicto, formato de salida — y
  un tope duro forzaría fragmentar esa propiedad). Se agregó en su lugar
  a `CLAUDE.md` ("Mecanismos incorporados de April" → "Presupuesto de
  tamaño para `.claude/agents/*.md`"): regla cualitativa (prosa de "por
  qué" va a `docs/`, ya practicada sin nombrarla en C6/C8) + tope
  numérico blando de ~1500 tokens como señal de revisión, no bloqueo
  automático. **Corrección aplicada el 31/08/2026:** se había escrito
  primero en `docs/conventions.md` de la raíz por error — ese archivo no
  propaga a proyectos scaffoldeados y este contenido describe un
  mecanismo (`.claude/agents/*.md`) que todo proyecto hereda. Movido a
  `CLAUDE.md`, que sí se embebe.

---

### C12 — Ratchet documentado como patrón reusable, no bespoke — RESUELTO (31/08/2026)

- **Mecanismo en gentle-ai:** tres ratchets independientes con la misma
  forma (`.deadcode-baseline.txt`, `.guard-population-baseline.txt`,
  `.refusal-ratchet-baseline.txt`): congela baseline → falla solo si
  crece → comando de regeneración documentado en el propio script.
- **Problema que resuelve:** evita rediseñar desde cero cada vez que
  aparece una nueva métrica de deuda.
- **Decisión: sin cambio de código, confirmado por el humano.** Revisado
  `doctor.go` a fondo: la premisa de "bespoke" no se sostuvo del todo —
  `doctorBaseline.Metrics` ya es `map[string]int` (doctor.go:229),
  `report.DebtMetrics` ya es una lista (doctor.go:39), y
  `evaluateDebtRatchet` (doctor.go:456) ya es una función pura genérica
  sin nada de TODOs hardcodeado. Lo único específico de TODOs es el
  cálculo del valor y dos líneas donde se arma el map/slice con una sola
  entrada. Se documentó el patrón en `docs/conventions.md` ("Ratchet de
  deuda progresiva — patrón reusable, no bespoke") citando esas piezas
  reusables, para que la próxima métrica se enchufe sin discutir diseño
  de nuevo — sin generalizar código que ya no lo necesitaba.

---

### C13 — Disciplina de commits por unidad de trabajo entregable — DESCARTADO (31/08/2026)

- **Mecanismo en gentle-ai:** `skills/work-unit-commits/SKILL.md` — un
  commit = un comportamiento/fix/migración/doc entregable, nunca por
  tipo de archivo; tests y docs viajan con el código que explican; el
  mensaje explica el resultado, no la lista de archivos.
- **Problema que resuelve:** mantiene revisable y reversible el
  historial sin depender de PRs grandes con squash al final.
- **Decisión: descartado, confirmado por el humano.** La premisa del
  candidato ("aplica a cómo `agent_developer` debería commitear al
  implementar un ticket") dejó de ser cierta dentro de esta misma
  sesión: C5 estableció como regla dura que ningún agente —
  `agent_developer` incluido — ejecuta jamás `git commit`; solo el
  humano commitea, manualmente, cuando decide. No hay ningún agente cuyo
  comportamiento de commit `CLAUDE.md` pueda gobernar, así que no hay
  dónde aplicar la convención propuesta. Cómo commitea el humano es su
  prerrogativa, fuera del alcance del protocolo del orquestador.

---

## Descartado en la segunda ronda (evaluado, no recomendado)

- **Freeze-expansion policy** para migraciones grandes de formato
  (`rdd-freeze-expansion-policy.md`) — diseñada para coordinar
  contribuciones aditivas de terceros durante una migración de 7 waves;
  April tiene un solo operador. Idea a guardar sin implementar: si algún
  día `feature_list.json` o el formato del ledger requieren un cambio de
  formato incompatible, congelar explícitamente los parches aditivos al
  formato viejo durante la migración.
- **Lenguaje de "invariante responsable"** en el veredicto de
  `reviewer_agent` (`AI_POLICY.md:46`) — el paso 5 de `reviewer_agent.md`
  ya cubre esto en sustancia (complejidad no pedida, mocks de la propia
  implementación); solo afinado de redacción, no mecanismo nuevo. Baja
  prioridad — aplicar solo si se está tocando ese archivo de todos
  modos.
