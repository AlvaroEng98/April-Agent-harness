# __YOUR_PROJECT_NAME__

> __ONE_LINE_DESCRIPTION_OF_YOUR_PROJECT__

Proyecto con arnés de **Spec-Driven Development (SDD)** asistido por agentes de
IA, generado con [`harness init`](https://github.com/AlvaroEng98/April-Agent-harness).

Las features no se implementan a mano de golpe: pasan por spec, aprobación
humana, implementación y review. El hilo principal del agente actúa como
**orquestador** y delega en subagentes.

---

## Empezar

```bash
./init.sh          # verifica el entorno — debe salir todo [OK]
```

Después abre Claude Code en este directorio y pídele la primera tarea.

La feature semilla `bootstrap_project` está `pending`: en la primera sesión el
orquestador te entrevista (objetivo + tech stack), escribe
`progress/project-definition.md`, rellena los placeholders
(`__YOUR_PROJECT_NAME__`, `__ONE_LINE_DESCRIPTION_OF_YOUR_PROJECT__`) y lanza
`planner_agent` para poblar el backlog. **No borres esa feature a mano**: se
cierra cuando apruebas el backlog.

Si prefieres definir el backlog tú, edita `feature_list.json` directamente y
marca `bootstrap_project` como `done`.

## Requisitos

| Requisito | Para qué |
|-----------|----------|
| `bash` | `init.sh`, `recap.sh`, `sync-changelog.sh` |
| `python3` | validación de `feature_list.json` y specs en `init.sh` |
| Claude Code (o agente compatible) | lee `.claude/agents/` y `AGENT.md` |

Añade aquí las dependencias propias de tu stack cuando las definas.

## Comandos

| Comando | Qué hace |
|---------|----------|
| `./init.sh` | Verifica el entorno: archivos base, `feature_list.json`, specs, agentes, build y recap. Se ejecuta al abrir sesión y antes de cerrar una feature. |
| `./recap.sh` | Imprime el estado: última sesión, feature actual, sesión activa. Lo consume `init.sh` y el hook `SessionStart`. |
| `./sync-changelog.sh` | Vuelca las features `done` a `## [Unreleased]` en `CHANGELOG.md`. Idempotente. |
| `./sync-changelog.sh --check` | No escribe; sale con código 1 si falta algo por volcar. |

## Estructura

```
├── .claude/
│   ├── agents/            5 subagentes: orquestador, planner, spec author,
│   │                      developer, reviewer
│   ├── hooks/             SessionStart hook que inyecta el recap
│   └── settings.json      permisos + registro del hook
├── docs/
│   ├── architecture.md    qué significa "hacer un buen trabajo" aquí
│   ├── conventions.md     estilo, nombres, estructura
│   └── verification.md    cómo verificar el trabajo
├── progress/
│   ├── current.md         estado de la sesión en curso
│   └── history.md         bitácora append-only de sesiones cerradas
├── specs/<feature>/       requirements.md + design.md + tasks.md
├── src/                   código de la aplicación
├── tests/                 tests automáticos
├── AGENT.md               mapa de navegación para el agente — empieza aquí
├── CLAUDE.md              rol obligatorio del hilo principal: orquestador
├── CHECKPOINTS.md         criterios C1–C7 para cerrar una feature
├── CHANGELOG.md           registro de lo entregado
├── feature_list.json      manifiesto de features con estado
├── session-handoff.md     plantilla de traspaso entre sesiones
├── init.sh                verificación del entorno
├── recap.sh               recap de estado (fuente única de verdad)
└── sync-changelog.sh      backlog → changelog
```

**Rellena `docs/`.** `architecture.md`, `conventions.md` y `verification.md`
vienen con esqueleto y placeholders: son lo que el agente lee antes de escribir
código. Vacíos, el resultado es genérico.

## Flujo de trabajo

```
pending → [sdd_agent_author] → spec_ready → ⏸ APROBACIÓN HUMANA
       → in_progress → [agent_developer → reviewer_agent] → done
```

- Estados válidos: `pending`, `spec_ready`, `in_progress`, `done`, `blocked`.
- **Una sola feature en `in_progress`** a la vez (`init.sh` lo valida).
- Las features con `"sdd": true` necesitan los tres documentos en
  `specs/<name>/` antes de que exista una línea de código.
- Dos puertas humanas que el agente no salta: aprobar el spec y aprobar el
  review antes de `done`.

Clasificación de complejidad que aplica el orquestador:

| Nivel | Criterio | Quién lo hace |
|-------|----------|---------------|
| SIMPLE | 1-2 archivos, <100 líneas, `acceptance` claros | el orquestador, inline |
| MEDIO | 2-3 archivos, tipos compartidos, descripción clara | `agent_developer` + `reviewer_agent` |
| AMBIGUO | descripción vaga o incompleta | SDD completo: spec → aprobación → dev → review |

Detalle completo en `AGENT.md` y `.claude/agents/orquestador.md`.

## Cierre de sesión

1. `./init.sh` en verde.
2. Feature acabada → `status: "done"` en `feature_list.json` (solo con
   aprobación humana).
3. `./sync-changelog.sh` para reflejarla en `CHANGELOG.md`.
4. Mueve el resumen de `progress/current.md` al inicio de `progress/history.md`.
5. Vacía `progress/current.md` dejando la plantilla.
