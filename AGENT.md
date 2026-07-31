# AGENT.md — Mapa de navegación para agentes de IA

> Este archivo es el **punto de entrada** para cualquier agente que trabaje en este
> repositorio. NO es una biblia de reglas: es un **mapa**. Lee solo lo que
> necesites cuando lo necesites (divulgación progresiva).

---

## 1. Antes de empezar (obligatorio)

1. Ejecuta `./init.sh` y verifica que termina sin errores. Si falla, **para**
   y resuelve el entorno antes de tocar código.
2. Lee `progress/current.md` para entender en qué estado quedó la última sesión.
3. Lee `feature_list.json`. Toda feature nueva (`"sdd": true`) pasa por
   **Spec Driven Development** — ver `docs/specs.md` y §4 de este archivo.
4. Lee `docs/specs.md` antes de tocar cualquier spec o feature `sdd: true`.

## 2. Mapa del repositorio

| Archivo / carpeta            | Qué contiene                                                                | Cuándo leerlo |
|------------------------------|-----------------------------------------------------------------------------|---------------|
| `feature_list.json`          | Lista de tareas con estado (`pending` / `spec_ready` / `in_progress` / `done` / `blocked`) | Siempre, al empezar |
| `progress/current.md`        | Estado de la sesión actual                                                  | Siempre, al empezar |
| `progress/history.md`        | Bitácora append-only de sesiones anteriores                                 | Si necesitas contexto histórico |
| `specs/<feature>/`           | `requirements.md` + `design.md` + `tasks.md` (Kiro-style)                   | Antes de implementar cualquier feature con `"sdd": true` |
| `docs/architecture.md`       | Qué significa "hacer un buen trabajo" en este proyecto                      | Antes de implementar |
| `docs/conventions.md`        | Reglas de estilo, nombres, estructura                                       | Antes de escribir código |
| `docs/specs.md`              | Proceso SDD: EARS notation, los 3 archivos, puerta de aprobación humana     | Antes de redactar o leer un spec |
| `docs/verification.md`       | Cómo verificar que tu trabajo funciona (incluye trazabilidad requirements)  | Antes de declarar una tarea como `done` |
| `CHECKPOINTS.md`             | Criterios objetivos de "estado final correcto"                              | Para auto-evaluarte |
| `.claude/agents/`            | Definiciones de subagentes (`orquestador`, `planner_agent`, `sdd_agent_author`, `agent_developer`, `reviewer_agent`) | Si orquestas trabajo |
| `src/`                       | Código de la aplicación                                                     | Para implementar |
| `tests/`                     | Tests automáticos                                                           | Para verificar |

## 3. Reglas duras (no negociables)

- **Una sola feature a la vez.** No mezcles cambios de varias tareas en la misma sesión.
- **No salgas del proyecto actual, solo desplasarte dentro de las carpeta y subcarpetas del directorio actual. Solicitar permiso del usuario en caso de que sea necesario moverse.
- **No declares una tarea `done` sin pruebas verdes.** Ejecuta `./init.sh` y
  asegúrate de que el bloque de tests pasa al 100%.
- **No saltes la fase de spec.** Toda feature con `"sdd": true` debe pasar
  por `sdd_agent_author` y obtener aprobación humana antes de tocar código.
- **No saltes la puerta de aprobación humana.** El orquestador detiene el flujo
  en `spec_ready` y espera.
- **Documenta lo que haces** en `progress/current.md` mientras trabajas, no al final.
- **Deja el repositorio limpio** antes de cerrar la sesión (ver §5).
- **Si no sabes algo, busca en `docs/`** antes de inventarlo.

## 4. Flujo de trabajo (SDD)

```
[FASE Grill: el orquestador pregunta] ← solo si bootstrap_project no está done
       │
       ▼ progress/project-definition.md
[planner_agent] ← descompone
       │
       ▼
feature_list.json poblado
       │
       ▼
pending → [sdd_agent_author] → spec_ready → ⏸ HUMANO → in_progress → [agent_developer → reviewer_agent] → done
```

0. **Planificación — condicional, no en cada sesión.** Se dispara por **estado
   explícito del backlog**, no por heurística: el template embarca una feature
   semilla `bootstrap_project` (`id 1`, `pending`) y el orquestador solo corre
   la planificación si esa feature no está `done`, o si el usuario la pide. En
   cualquier otro caso salta directo al paso 1. Un backlog agotado **no**
   dispara planificación: el orquestador reporta "backlog vacío" y para.

   Cuando toca: el **orquestador** conduce la FASE Grill en el hilo principal
   (2 preguntas: objetivo y tech stack) y escribe
   `progress/project-definition.md`; después lanza **`planner_agent`**, que lee
   ese archivo y descompone en features. El `planner_agent` no habla con el
   usuario — es un subagente y no tiene canal con él.
1. El orquestador detecta la primera feature `pending` con `"sdd": true`.
2. El orquestador lanza `sdd_agent_author`, que crea
   `specs/<name>/{requirements,design,tasks}.md` y marca el status como
   `spec_ready`.
3. **Pausa.** El humano lee el spec en `specs/<name>/` y aprueba (o pide cambios).
4. Una vez aprobado, el orquestador cambia el status a `in_progress` y lanza `agent_developer`.
5. El agent_developer ejecuta `tasks.md` una a una, marcándolas `[x]`.
6. El reviewer_agent verifica trazabilidad `R<n>` ↔ test y tasks completas;
   aprueba o rechaza.
7. Si aprueba, el agent_developer marca `done` y mueve el resumen a
   `progress/history.md`.

## 5. Cierre de sesión (lifecycle)

Antes de terminar:

1. Ejecuta `./init.sh` — todo verde.
2. Si la tarea está acabada: marca `status: "done"` en `feature_list.json`.
3. Mueve el resumen de `progress/current.md` al final de `progress/history.md`.
4. Vacía `progress/current.md` dejando solo la plantilla.
5. No dejes archivos temporales, ni `print()` de debug, ni TODOs sin contexto.

## 6. Si te bloqueas

- Relee la sección relevante de `docs/`.
- Si la herramienta no hace lo que esperas, **no inventes un workaround**:
  documenta el bloqueo en `progress/current.md` y para la sesión.
