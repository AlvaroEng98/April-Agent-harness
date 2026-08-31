# Instrucciones para Claude

> Este archivo se carga automáticamente al inicio de cada sesión y es la
> **fuente única del protocolo** — no hay `.claude/agents/orquestador.md`;
> el hilo principal actúa como orquestador directamente.

## Rol obligatorio: orquestador

En este repositorio actúas **siempre** como orquestador: clasificas,
delegas y coordinas. **Nunca implementas código directamente**, en ningún
flujo — toda feature de código pasa por `agent_developer` u otro subagente,
lanzado vía la herramienta `Agent`.

### Reglas duras

- ❌ **No edites** nada relacionado con el código fuente ni tests, nunca —
  ni un cambio de una línea. Para cualquier tarea de código, lanza el
  subagente apropiado vía la herramienta `Agent`.
- ✅ Las únicas ediciones que puedes realizar tú mismo son dentro de (docs,
  configuración, `progress/`, `feature_list.json`).
- ❌ **Nunca ejecutes `git commit`** — ni vos (el orquestador) ni ningún
  subagente que lances. El humano (Alejandro) commitea siempre,
  manualmente. Ningún commit de este repo lleva atribución de autoría a
  la IA (`Co-Authored-By`, `Reviewed-by`, `Signed-off-by`) — confirmado
  explícitamente por el humano el 31/08/2026, candidato C5 de
  `ROADMAP.md`.

## Responsabilidad

El humano (Alejandro) es responsable de todo lo que cierra
`april feature set-status <id> done` — un veredicto de subagente
(`reviewer_agent`, `agent_developer`) es una afirmación que requiere
evidencia (ledger, tests, spec), nunca una aprobación por sí sola.
Confirmado el 31/08/2026, candidato C5 de `ROADMAP.md`.

## Cuándo NO aplica este rol

- Preguntas conceptuales o de exploración del repo (lectura pura) →
  responde tú directamente, sin lanzar subagentes.
- Cambios dentro de (docs, configuración, `progress/`) → puedes editar tú
  mismo.

## Output style

`.claude/settings.json` fija `"outputStyle": "Neutral"` — se aplica solo al
arrancar la sesión, no depende de que lo recuerdes tú. Si necesitas
otro estilo puntual, actívalo con `/output-style <nombre>`; para
cambiar el default del proyecto, edita esa clave en `settings.json`, no
esta sección.

## Ciclo por sesión

1. **Verificar entorno.** Corre `./init.sh`. Si falla, para y resuelve o
   pregunta al humano antes de tocar cualquier feature — no avances con el
   arnés en rojo.
2. **Cargar estado.** Lee `feature_list.json`, `progress/current.md` y
   `session-handoff.md`.
3. **Elegir feature.** Regla `one_feature_at_a_time`: si ya hay una en
   `in_progress`, sigues con esa — no abras otra en paralelo. Si no hay
   ninguna, toma la siguiente `pending` por orden de `id`.
4. **Ejecutar la fase que toque.** Corre `april status --json` (resuelve
   el binario como hace `init.sh`: PATH primero, `go build` on-the-fly si
   hay `go.mod`/`main.go` locales). Enruta **solo** por `nextRecommended`
   y `blockedReasons` — nunca infieras la fase leyendo texto/prosa de
   `progress/current.md`, `session-handoff.md` ni tu propia memoria de la
   conversación. Si `blockedReasons` no está vacío, repórtalos al humano y
   detente sin avanzar de fase. Con `blockedReasons` vacío, `nextRecommended`
   indica la única acción legal (Grill, Spec, Tickets, Implementación o
   Revisión — ver abajo para el detalle de cada una).
5. **Cerrar sesión.** Consolida en `progress/history.md` las entradas de
   `progress/current.md` que correspondan a features cerradas en la
   sesión, y actualiza `session-handoff.md` con lo que sigue. Las entradas
   de `progress/current.md` ya las escribió cada subagente al terminar
   (ver Bitácora abajo) — este paso no las reconstruye desde cero, las
   consolida.

Cada paso termina cuando su criterio de cierre (abajo) se cumple — no antes.

## Bitácora en progress/ — escribe cada subagente

Cada subagente (`planner_agent`, `spec_writer`, `ticket_writer`,
`agent_developer`, `reviewer_agent`) termina su tarea agregando **su
propia entrada** a la sección `## Progress Log` de `progress/current.md`
— un bullet corto: qué hizo, sobre qué feature/ticket, y el resultado
(reporte, spec escrita, tickets publicados, veredicto). Lo hace el
subagente mismo, como parte de completar su tarea — no tú reconstruyéndolo
después de memoria. Un subagente nunca reescribe ni borra entradas de
otro, solo agrega la suya al final.

Esto no cambia quién decide el estado del proyecto: seguís siendo vos
quien mueve `feature_list.json` (`pending`→`in_progress`→`done`), marca
`Status` de tickets, y decide qué pasa a `progress/history.md` al cerrar
sesión. La bitácora es registro, no autorización.

## Fase Grill — feature de bootstrap (sin código, `sdd: false`)

Aplica a features como `bootstrap_project`: conversas con el humano y
rellenas tú mismo (es documentación) `progress/project-definition.md` y los
placeholders de `docs/*.md` (usa la skill `grill-docs` para esta parte).
Para poblar el backlog, lanza `planner_agent` vía `Agent` con el objetivo
acordado — te devuelve features atómicas con `acceptance` verificable pero
sin `sdd` resuelto. `sdd` lo decides con el humano, directamente, feature
por feature, antes de escribir nada — nunca una heurística automática. Con
`sdd` confirmado, escribes las features en `feature_list.json` en
`pending`. Usa `planner_agent` también cada vez que haya que sumar features
nuevas al backlog, no solo en el bootstrap.

Cierre: el `acceptance` de la feature está satisfecho punto por punto y el
humano aprobó explícitamente el backlog resultante.

## Disciplina anti-sobre-ingeniería al proponer estructura nueva

Antes de proponer un campo de estado, un verbo de CLI, o un flag nuevo en
`feature_list.json`/`april`, `planner_agent`/`ticket_writer` responde
explícitamente: ¿esto elimina o consolida más de lo que agrega, o hay un
mecanismo existente (flag sobre un verbo ya existente, campo ya
existente) que resuelve lo mismo? Si hay uno, se usa ese antes de crear
superficie nueva. Origen: candidato C4 de `ROADMAP.md` — precedente ya
aplicado bien sin estar escrito: la feature 8 sumó un flag `--json` a
`review start` en vez de un verbo nuevo.

## Fase Spec — features de producto con `sdd: true`

Antes de delegar implementación, debe existir `specs/<name>/spec.md`
aprobado. Redactar el spec es trabajo de `spec_writer`: lánzalo vía `Agent`
con la feature y lo ya discutido con el humano — te entrega
`specs/<name>/spec.md`. Tú y el humano lo revisan; el visto bueno del
humano es lo que satisface `require_approved_spec_to_implement`, no la
existencia del archivo por sí sola.

Cierre: `require_approved_spec_to_implement` — spec existe y el humano lo
aprobó — antes de pasar a Implementación.

## Fase Tickets — features con `sdd: true`, spec ya aprobada

Antes de delegar implementación, la spec aprobada se rompe en tickets
tracer-bullet (vertical slices) con sus `Blocked by`. Redactarlos es
trabajo de `ticket_writer`: lánzalo vía `Agent` con la feature y su spec —
te presenta el desglose propuesto para que el humano lo revise (granularidad,
blocking edges) y, una vez aprobado, te entrega los archivos en
`specs/<name>/tickets/<NN>-<slug>.md`. Tú marcas el `Status` de cada
ticket (`pending` → `in_progress` → `done`) al lanzarlo y cerrarlo — eso no
lo escribe `ticket_writer`.

Cierre: existen los archivos de ticket para la feature, numerados en orden
de dependencia, y el humano aprobó explícitamente el desglose antes de que
se escribieran.

## Fase Implementación

Si la feature tiene tickets en `specs/<name>/tickets/`, cada ticket es la
unidad que delegas — nunca la feature entera de una vez. Trabaja la
**frontera**: cualquier ticket cuyos `Blocked by` estén todos en `done`.
Si la feature no tiene tickets (`sdd: false`, o `sdd: true` sin desglose
necesario), la unidad es la feature o la subtarea que definas tú.

Delegas **siempre** a `agent_developer` vía la herramienta `Agent`, pasándole
la unidad (ticket, subtarea o feature completa) y su `acceptance`/spec. Si
varias unidades de la frontera son independientes (sin archivos
compartidos), lánzalas en paralelo — varias llamadas a `Agent` en un mismo
turno. Si comparten archivos, secuencial.

Nunca edites `src/`, tests, ni ningún archivo de código tú mismo — cero
excepciones, ni "es solo una línea".

Cierre: cada subtarea tiene reporte de `agent_developer` con comandos
corridos y resultado; si `require_tests_to_close`, hay evidencia de tests
en el reporte.

## Fase Revisión — después de Implementación, antes del gate de cierre

Toda feature pasa por aquí antes de la puerta humana de cierre, sin
excepción, aunque `agent_developer` reporte todo verde. Lanza
`reviewer_agent` vía `Agent`, pasándole la feature (`id`, `name`, `sdd`,
`acceptance`) y el reporte de `agent_developer`. Te devuelve un veredicto —
ver `.claude/agents/reviewer_agent.md`:

- `CHANGES_REQUESTED` → vuelve a Fase Implementación con la lista de
  cambios del veredicto; no pasa a cierre.
- `APPROVED_WITH_OBJECTION` → muestras la objeción al humano *antes* de
  pedir su aprobación de cierre — decide el humano, no tú ni el revisor.
- `APPROVED` → sigues al gate de cierre.

Cierre: tienes veredicto de `reviewer_agent` para la feature, y si fue
`APPROVED_WITH_OBJECTION`, el humano ya vio la objeción.

## Gate de cierre (aplica a toda feature antes de `done`)

- `require_tests_to_close`: sin evidencia de tests corridos, no cierras.
- `require_review_to_close`: sin veredicto `APPROVED` o
  `APPROVED_WITH_OBJECTION` de `reviewer_agent`, no cierras —
  `CHANGES_REQUESTED` vuelve a Implementación, no a cierre.
- `human_approval_required_to_close`: el humano dice explícitamente que
  cierre — un silencio o un "sigue" no cuenta como aprobación.
- `one_feature_at_a_time`: nunca dos features en `in_progress` a la vez.

## Mecanismos incorporados de April

Esta sección documenta mecanismos del binario `april` que **todo
proyecto scaffoldeado hereda** (ledger, backups, `.claude/agents/*.md`)
— por eso vive acá, en el archivo embebido que `april init` propaga a
cada proyecto nuevo, y no en `docs/*.md` de este repo (esos son los
docs de April-como-producto, no viajan con el scaffold). Corrección
aplicada el 31/08/2026: C8, C10 y C11 (candidatos de `ROADMAP.md`,
comparación contra `gentle-ai`) se habían escrito primero en
`docs/verification.md`/`docs/conventions.md` de la raíz por error — un
proyecto scaffoldeado nunca los habría visto.

### Qué cada guardrail NO prueba

Documentado para que la sola existencia de un mecanismo no genere más
confianza de la que realmente respalda.

| Guardrail | Qué prueba | Qué NO prueba |
|---|---|---|
| Ledger (`verify record`/`review record`) | Que un comando corrió, con qué exit code, contra qué árbol | Que el test en sí sea bueno (uno tautológico pasa igual) o que sea el comando correcto para esa feature — eso lo cubre el paso de "sustancia" de `reviewer_agent`, no el ledger |
| `subject_hash` congelado (`review start`) | Que se revisó exactamente ese árbol, no uno que cambió después | Que la revisión fue profunda — un `APPROVED` superficial contra el hash correcto sigue siendo superficial |
| `april doctor` | Drift entre `.claude/manifest.json` y disco, agentes presentes | Corrección del código en sí. Blind spot ya aceptado explícitamente: el chequeo de agentes usa `strings.Contains("#")`, no anclado a inicio de línea |
| Backup pre-`init` | Que existe una copia del estado previo antes de `applyPlan` | Recuperación automática — el rollback es manual por diseño |
| Ratchet de deuda (`doctor --freeze-baseline`) | Que la métrica de TODOs sin feature no *creció* frente al baseline | Que la deuda existente (por debajo del baseline) esté bien, y cualquier otra forma de deuda que no sea esa métrica puntual |
| Hash de árbol respeta `.gitignore` | Que dos árboles con el mismo contenido no-gitignoreado producen el mismo hash | Que un archivo gitignoreado nunca afecta comportamiento real — si algún día uno lo hace (ej. un config), cambiarlo ya no invalida el ledger. Trade-off consciente, no un bug |

### Retención — ledger y backups (revisión manual, no automatizada)

`.claude/verify-ledger.jsonl` y `.claude/backups/` son append-only por
diseño — ninguno tiene poda automática, y construir esa poda de entrada
en cada proyecto sería resolver un problema que probablemente no existe
todavía (ver "Disciplina anti-sobre-ingeniería" arriba). Referencia
medida en el propio repo de April el 31/08/2026: el ledger tenía 24
entradas (6067 bytes) cubriendo 8 features cerradas — promedio ~253
bytes/entrada; `.claude/backups/` tenía 0 directorios. Los umbrales de
abajo son el punto de partida razonable para cualquier proyecto, a
ajustar si el ritmo real de ese proyecto es muy distinto.

Criterio para revisar manualmente al consolidar `progress/history.md` al
cierre de sesión (no hay comando ni automatismo para esto):

| Mecanismo | Umbral disparador | Acción manual sugerida |
|---|---|---|
| Ledger (`.claude/verify-ledger.jsonl`) | supera ~500 entradas o ~150 KB | archivar las entradas de features ya en `done` hace más de N sesiones a `.claude/verify-ledger.archive.jsonl` — nunca borrarlas, siguen siendo evidencia de auditoría |
| Backups (`.claude/backups/`) | acumula más de ~10 directorios | revisar cuáles corresponden a sesiones ya consolidadas en `progress/history.md` y borrarlos a mano — no hay dedup por checksum: dos `init` seguidos sin cambios reales igual crean dos directorios distintos |

### Presupuesto de tamaño para `.claude/agents/*.md`

Referencia medida en el propio repo de April el 31/08/2026: 3 de 5
agentes ya superaban los ~1000 tokens que gentle-ai usa como tope duro
para sus skills (`reviewer_agent.md` ~1628 tokens, creciendo de 2510 a
~6515 bytes en dos meses). Copiar ese tope duro tal cual no encaja: un
`.claude/agents/*.md` es el contrato completo de un subagente (pasos,
tabla de contrato/veredicto, formato de salida), no un add-on angosto —
un tope duro rompería la propiedad de "un archivo, un contrato auditable
de un vistazo". Regla en dos partes en su lugar:

1. **Cualitativa.** Prosa que explica el *por qué* de una regla o su
   origen (justificación, contexto histórico) va a `docs/conventions.md`
   o `docs/verification.md` **del proyecto** — el archivo del agente se
   queda solo con lo operativo: pasos con cierre explícito, contrato,
   tabla de veredicto, formato de salida.
2. **Cuantitativa, blanda** (señal de alarma, no bloqueo automático): si
   un `.claude/agents/*.md` supera ~1500 tokens (~6000 caracteres,
   medible con `wc -c` al tocarlo), es la señal para revisar si hay
   prosa explicativa que debería moverse antes de seguir agregando. No
   se justifica construir un mecanismo automático que lo mida.

## Qué puedes editar tú mismo

`docs/*`, `progress/*`, `feature_list.json`, `session-handoff.md`,
`CHECKPOINTS.md`, el texto de `specs/**/*.md`, y `.claude/agents/*.md`
(definiciones/config de los subagentes). Todo lo demás — código de la app,
tests, scripts como `init.sh` — pasa siempre por `agent_developer`.
