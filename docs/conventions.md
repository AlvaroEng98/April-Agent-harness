# Conventions

> Rellenado por la skill `grill-docs` durante la Fase Grill de
> `bootstrap_project`. Sección sin respuesta del humano queda literalmente
> `_pendiente_` — nunca se omite, nunca se inventa una respuesta plausible.
> Este archivo existe para que `agent_developer` no decida por su cuenta
> nada de lo que aquí se responde — gestor de paquetes incluido.

## Estilo

`gofmt` (formato estándar de Go, sin configuración adicional) + `go vet`
como único linter. No se usa `golangci-lint` ni ninguna otra herramienta de
lint externa (decisión confirmada 26/08/2026 — mismo espíritu que "sin
dependencias externas": toolchain mínimo).

## Gestor de paquetes

`go mod` — único gestor posible en Go, no hay alternativa que evaluar.
Política del proyecto: **sin dependencias de terceros, solo stdlib**
(confirmado 26/08/2026). Si alguna vez se necesita una, se agrega con
`go get <módulo>@<version>` y se documenta explícitamente por qué stdlib
no alcanzó — no se agrega en silencio.

## Nombres

| Tipo | Convención | Ejemplo real |
|---|---|---|
| Paquete | minúsculas, sin guiones | `package main` (único paquete del proyecto) |
| Tipo (struct) | camelCase (nada se exporta — no hay API pública, todo vive en `package main`) | `manifestEntry`, `scaffoldPlan` |
| Función | camelCase | `cmdInit`, `scaffoldInit`, `planScaffoldFromFS` |
| Variable | camelCase | `absTarget`, `templateFS` |
| Constante | camelCase, agrupadas por `const (...)` cuando son un conjunto relacionado | `colorCyan`, `colorDim`, `colorReset` |
| Archivo | snake_case por responsabilidad, con `_test.go` gemelo | `scaffold.go` / `scaffold_test.go` |
| Error sentinel | `Err` + PascalCase, aunque no se exporte | `ErrStaleSubjectHash`, `ErrNoTestEvidence` (ejemplos del `ROADMAP.md`, aún no implementados) |

## Estructura de archivo

Un archivo por responsabilidad (nunca varias responsabilidades no
relacionadas en el mismo archivo), con su `_test.go` gemelo en el mismo
directorio, mismo paquete (`package main`) — no se usa `package main_test`
externo. Ejemplo: `scaffold.go` (motor de scaffold) + `scaffold_test.go`
(sus tests), ambos en la raíz del módulo.

## Tests

- Mismo paquete que el código bajo test (`package main`), nunca un paquete
  `_test` externo.
- Nombre: `Test` + descripción del comportamiento. Cuando el test verifica
  un **caso de negocio específico**, el nombre va **en español**
  describiendo ese caso — es el patrón ya establecido y mayoritario en
  `scaffold_test.go` (15 de 19 tests existentes), confirmado como
  convención el 26/08/2026 tras revisar el código real: p. ej.
  `TestArchivoNoTocadoPorUsuarioSeActualiza`,
  `TestLoadManifestAusenteEsAdopcion`,
  `TestManifiestoCorrupto_TratadoComoAdopcionConAviso`. Cuando el test es
  **genérico o estructural** (no describe un escenario de negocio
  puntual), va en inglés: `TestMergeGitignore`, `TestPlanScaffoldIsPure`.
- Los tests verifican comportamiento observable (contenido de archivos
  resultantes, exit codes, estructura del plan) sobre fixtures reales en
  directorios temporales — no se mockea el filesystem, porque la
  interacción con el filesystem es precisamente lo que hay que verificar.

## Manejo de errores

- `error` values de Go estándar — nunca `panic` para errores esperables
  (entrada inválida, archivo faltante, etc.). `panic` solo ante un bug
  interno que de verdad no debería poder ocurrir.
- Sentinel errors (`var ErrX = errors.New("...")`) se definen **junto al
  código que los produce**, no en un `errors.go` centralizado — coherente
  con "un archivo por responsabilidad" (decisión confirmada 26/08/2026).
- Al propagar entre funciones, se envuelve con
  `fmt.Errorf("contexto: %w", err)` para preservar la cadena y permitir
  `errors.Is`/`errors.As` en el llamador.

## Comentarios

Regla general: sin comentarios salvo que el **por qué** no sea obvio desde
el código (constraint oculto, invariante, workaround de un bug concreto).
Nunca comentarios que expliquen el *qué* — el nombre ya lo dice.

- **Aceptable** (real, ya en el `.gitignore` de este repo): el comentario
  que explica por qué `/.claude/manifest.json` está ahí — "si se generara
  aquí por error, el `go:embed` de `scaffold.go` lo empotraría y lo
  propagaría a todos los destinos scaffoldeados". Es una restricción no
  obvia; sin el comentario, alguien podría borrar la línea pensando que es
  ruido. (Nota: el ejemplo original de este bullet era el comentario que
  justificaba `/feature_list.json` anclado con `/` — esa línea completa se
  eliminó en la feature 21, ver hallazgos hermanos abajo; el ejemplo se
  reemplazó porque el original ya no existe en el `.gitignore`.)
- **Rechazado**: `// crea el directorio si no existe` sobre una línea
  `os.MkdirAll(path, 0o755)`. El nombre de la función ya lo dice — el
  comentario no aporta nada que el código no diga.

## Áreas sensibles

Precondición de la feature `review_depth_by_diff_sensitivity`
(`feature_list.json` id 8, `ROADMAP.md` E5): rutas cuyo blast radius exige
revisión más profunda por parte de `reviewer_agent`, confirmadas con el
humano el 26/08/2026:

- `scaffold.go` — el motor que aplica cambios sobre el filesystem del
  usuario; un bug aquí puede borrar o sobrescribir trabajo real.
- `init.sh` — es lo que valida que el entorno es confiable antes de que se
  confíe en él; un bug aquí deja pasar un entorno roto sin avisar.
- `.github/workflows/` — dispara releases automáticas; un bug aquí puede
  publicar una versión rota o filtrar algo indebido en CI.

Cualquier diff que toque una de estas rutas exige el paso adicional de
revisión que defina la feature 8 al implementarse. Fuera de estas tres,
no se exige profundidad extra por defecto.

## Cambios a la lógica de derivación de fase

Origen: candidato C2 de `ROADMAP.md` (comparación contra `gentle-ai`,
28/08/2026), adaptado — no se copia el mecanismo de evaluación en sombra
en runtime de gentle-ai (resuelve un problema de rollout gradual en un
sistema con tráfico en vivo; `april` es un binario CLI que se recompila y
se invoca fresco cada vez, no tiene código viejo y nuevo conviviendo en
producción). Se adopta la disciplina de fondo, en forma de test.

Ya hubo un incidente real de esta clase: la feature 12 corrigió que
`hashTree`/`computeSubjectHash` invalidaran evidencia en silencio ante un
cambio no relacionado (binario regenerado por `go build`), detectado solo
en uso real, no mecánicamente.

Regla: cualquier feature/ticket que modifique `derivePhase`,
`computeBlockedReasons` o `nextRecommendedText` (`status.go`) MUST
incluir, antes del cambio, un test de caracterización que fije el
`phase`/`blockedReasons`/`nextRecommended` actual de las features ya
`done` de este propio repo (o un fixture equivalente que cubra los casos
existentes) — y verificar, después del cambio, que nada de eso se movió
salvo el comportamiento nuevo explícitamente buscado por esa feature.

## Incidentes reales de `blockedReasons`/`nextRecommended` confusos

Origen: candidato C3 de `ROADMAP.md` (comparación contra `gentle-ai`,
28/08/2026). Precedente ya practicado sin nombrarlo: la feature 12 fijó
como test permanente el incidente real (no solo el mecanismo en
abstracto) de un binario regenerado por `go build` invalidando evidencia
en silencio.

Regla: cuando una sesión real tope con un `blockedReasons` o
`nextRecommended` que resultó confuso, ambiguo, o llevó a un callejón sin
salida, la corrección MUST venir acompañada de un test que fije ese
escenario exacto (input → texto exacto esperado) como regresión
permanente — no alcanza con corregir el mecanismo en general sin dejar el
caso real pinneado.

## Presupuesto de tamaño para `.claude/agents/*.md`

Movido a `CLAUDE.md` ("Mecanismos incorporados de April") el
31/08/2026: describe un mecanismo (`.claude/agents/*.md`) que todo
proyecto scaffoldeado hereda, así que debía vivir en un archivo que
`april init` propaga — `docs/conventions.md` de la raíz no viaja con el
scaffold (no está en el `go:embed` de `scaffold.go`). Ver ahí para el
detalle.

## Ratchet de deuda progresiva — patrón reusable, no bespoke

Origen: candidato C12 de `ROADMAP.md` (comparación contra `gentle-ai`,
28/08/2026). La feature 11 (`doctor_debt_ratchet`, `doctor.go`) ya
implementó la forma genérica de un ratchet — no hace falta rediseñar
nada para la próxima métrica de deuda, solo enchufarla:

- **Baseline por métrica nombrada, no por campo fijo.**
  `doctorBaseline.Metrics` (doctor.go:229) es un `map[string]int` —
  agregar una métrica nueva es agregar una clave, no cambiar el esquema.
- **Evaluación pura y genérica.** `evaluateDebtRatchet(name, current,
  baselineFrozen, baseline)` (doctor.go:456) no sabe nada de TODOs
  específicamente: nombre de métrica, valor actual, si hay baseline
  congelado, y el baseline — cualquier métrica de deuda progresiva se
  evalúa llamándola igual.
- **`report.DebtMetrics` es una lista** (`[]doctorDebtMetric`,
  doctor.go:39), no un campo único — ya está pensado para más de una
  métrica a la vez.
- **Congelado solo por flag explícito, nunca side-effect.**
  `runDoctorFreezeBaseline` (doctor.go:467) es la única vía de escritura
  de `.claude/doctor-baseline.json`; se niega a sobreescribir un baseline
  ya congelado — hay que borrarlo a mano para recongelar. Sin baseline
  congelado, la corrida normal de `april doctor` nunca falla por esta
  causa, solo informa el valor actual.
- **Falla solo si crece.** Con baseline congelado, `Exceeded = current >
  baseline` (doctor.go:463) — deuda existente por debajo del baseline
  nunca es motivo de fallo, solo el crecimiento nuevo lo es.

Lo único específico de TODOs en toda la feature es el cálculo del valor
en sí (`len(unlinkedTODOs)`) y las dos líneas donde se arma el
map/slice con esa única entrada — exactamente lo que cambia al sumar
una segunda métrica. Para la próxima métrica de deuda progresiva:
calcular su valor actual, sumarla a `Metrics` en
`runDoctorFreezeBaseline`, y llamar `evaluateDebtRatchet` con su nombre
— sin discutir de nuevo baseline/congelado/umbral de falla.

## `templates/.gitignore` vs `.gitignore` de la raíz — no son el mismo target

Origen: feature 17 (`gitignore_root_tracks_specs`). El `.gitignore` de la
raíz de este repo tenía la línea `specs/` copiada literal de
`templates/.gitignore`. Son dos archivos con target distinto:

- `templates/.gitignore` es lo que `april init` scaffoldea en proyectos
  **nuevos** — ahí `specs/` (y `tests/`) sí es estado de trabajo
  descartable, con comentario explícito en el propio archivo.
- El `.gitignore` de la raíz gobierna **este repo**
  (April-Agent-harness), donde `specs/` es la documentación SDD real
  exigida por `CLAUDE.md` (`require_approved_spec_to_implement`) — nunca
  descartable.

La línea copiada sin verificar a qué árbol aplicaba dejó las 7 `spec.md`
existentes fuera de git desde siempre (`git ls-files specs/` no devolvía
nada), lo que rompía en CI (`git clone` limpio) un test que las lee del
árbol real vía `os.ReadFile`, aunque pasaba en local porque los archivos
seguían en el filesystem de cada desarrollador. Antes de copiar una línea
de un `.gitignore` a otro en este repo, verificar explícitamente que la
regla aplica al mismo contexto en ambos lados — no asumir por similitud
de nombre de carpeta.

### Hallazgo hermano: `/docs/` (feature 18, `gitignore_root_tracks_docs`)

El mismo `.gitignore` de la raíz tenía también una regla `/docs/` que
ignoraba `docs/architecture.md`, `docs/conventions.md`, `docs/specs.md` y
`docs/verification.md` — el mismo error de fondo que `specs/` arriba, pero
sin copy-paste: acá la falla fue un razonamiento que dejó de sostenerse
("`docs/` es estado de trabajo del Grill de `bootstrap_project`, igual que
`/feature_list.json`") sin revalidarlo cuando esos documentos pasaron a
ser documentación vinculante para el equipo (mismo estatus que
`specs/*.md`: referenciados por `CLAUDE.md` y por los propios subagentes
como fuente de verdad, no descartables).

Lección: no alcanza con revisar reglas de exclusión copiadas literal de
otro archivo — una regla escrita a mano, con su propio razonamiento
explícito en el comentario, también puede quedar obsoleta si el contexto
que la justificaba cambia. Antes de dar por válida una regla de
`.gitignore` de este repo solo porque tiene un comentario que la explica,
revalidar si esa explicación sigue siendo cierta hoy, no solo si el
comentario suena razonable.

### Tercer hallazgo hermano: una regla de `templates/.gitignore` puede
### ignorar su propia fuente dentro de `templates/` (feature 20,
### `scaffold_session_handoff_placeholder`)

`templates/.gitignore:9` tiene la línea `/session-handoff.md`, correcta
para el **destino** scaffoldeado: cada proyecto nuevo debe tener su propio
`session-handoff.md` como estado de trabajo, no trackeado por git — mismo
espíritu que la regla equivalente del `.gitignore` de la raíz de este
repo. El matiz que los dos hallazgos anteriores (`specs/`, `docs/`) no
tenían: acá la regla vive **dentro** de `templates/`, y por estar anclada
(`/session-handoff.md` es relativa al directorio donde está el propio
`.gitignore`, o sea `templates/`) también ignora a
`templates/session-handoff.md` — el placeholder que es la **fuente** que
`scaffold.go` embebe vía `all:templates` para copiarlo al destino. Un
`git add templates/` normal no lo agrega; sin forzarlo, el archivo nunca
queda trackeado, no sobrevive a un `git clone` limpio, y `april init`
sobre un checkout fresco deja de generar cualquier `session-handoff.md` en
el destino (detectado por `reviewer_agent` con una prueba de caza negra:
borrar el archivo del disco y reconstruir el binario).

Mitigación en dos partes, no una sola: (1) trackear el archivo a la fuerza
con `git add -f templates/session-handoff.md` — documentado acá para que
nadie lo pierda en un futuro `git clean`/auditoría de `.gitignore`
pensando que es un archivo huérfano; y (2) el test
`TestTemplatesSessionHandoffPlaceholderEstaTrackeadoEnGit`
(`scaffold_test.go`) como guardarraíl permanente: falla si el archivo
existe en disco pero no está en el índice de git, para que la colisión no
vuelva a pasar en silencio con un futuro archivo de plantilla que
comparta nombre con una regla de `templates/.gitignore`. Lección: cuando
un archivo de plantilla vive dentro del mismo árbol que gobierna su propio
`.gitignore` de destino, verificar explícitamente con
`git check-ignore -v` que la regla pensada para el scaffoldeado no
alcanza también a la fuente.

### Cuarto hallazgo hermano: `/feature_list.json` tratado como estado de
### trabajo descartable cuando es el backlog vivo real (feature 21,
### `gitignore_root_tracks_feature_list`)

El `.gitignore` de la raíz tenía, desde el commit `8517803` (31/07/2026),
la línea `/feature_list.json` con un comentario que documentaba una
arquitectura de respaldo ya obsoleta: "el backlog vivo no se versiona, el
registro de lo entregado va a `CHANGELOG.md` vía `./sync-changelog.sh`".
Mismo patrón de fondo que `specs/` y `docs/` arriba — contenido vinculante
(20 features con `acceptance` detallado e historial de `status`) tratado
como descartable. Consecuencia real: `feature_list.json` nunca estuvo
trackeado desde ese commit; un clon fresco de este repo no recibía el
backlog. El humano decidió explícitamente (02/09/2026) trackearlo, mismo
tratamiento que specs/docs.

Matiz propio de este hallazgo (a diferencia de los tres anteriores): acá
la arquitectura de respaldo alternativa (`CHANGELOG.md` +
`sync-changelog.sh`) referenciada por el propio comentario también quedó
obsoleta — `CHANGELOG.md` describe "tres flujos F1/F2/F3" y un
`.claude/agents/orquestador.md` que ya no existen (`CLAUDE.md` actual: "no
hay `.claude/agents/orquestador.md`; el hilo principal actúa como
orquestador directamente"), y `sync-changelog.sh` no parece haberse
corrido en mucho tiempo. Arreglar esa staleness queda explícitamente fuera
de alcance de la feature 21 — `feature_list.json` versionado en git es
ahora la fuente de verdad del backlog, `CHANGELOG.md` queda como resumen
curado aparte, no como único respaldo.

Nota: eliminar esta línea del `.gitignore` de la raíz no afecta
`templates/feature_list.json` (necesario para `go:embed`, ver
`scaffold.go`) — esa exclusión nunca dependió de la línea anclada con `/`
de la raíz, que solo gobierna el propio directorio raíz de este repo, no
`templates/`. Verificado con `git check-ignore -v templates/feature_list.json`
(sin match) antes y después del cambio.
