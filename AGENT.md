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
| `specs/<feature>/`           | `requirements.md` + `design.md` + `tasks.md` (Kiro-style)                   | Antes de implementar cualquier feature con `"sdd": true` (flujo F3) |
| `progress/plan_<feature>.md` | Contrato ligero del flujo F2: archivos + mapa `acceptance → test` + riesgo asumido | Antes de implementar (F2) y antes de revisar (F2) |
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
- **No saltes la fase de spec en F3.** Toda feature con `"sdd": true` y
  `"ambiguity": "vague"` debe pasar por `sdd_agent_author` y obtener aprobación
  humana antes de tocar código.
- **No saltes la puerta de aprobación humana.** Existe en los tres flujos: el
  orquestador para antes de escribir `done`, siempre. En F3 hay dos puertas
  (spec y cierre).
- **No estés de acuerdo por defecto.** Si el planteamiento dispara un gatillo
  G1-G4 (ver §4.1), objetas con evidencia y alternativa antes de ejecutar. Sin
  gatillo, callas y ejecutas.
- **Documenta lo que haces** en `progress/current.md` mientras trabajas, no al final.
- **Deja el repositorio limpio** antes de cerrar la sesión (ver §5).
- **Si no sabes algo, busca en `docs/`** antes de inventarlo.

## 4. Flujos de trabajo

Hay **tres** flujos de construcción. El orquestador clasifica la feature y elige
uno; una vez elegido no se mezcla con otro. Matriz completa en
`.claude/agents/orquestador.md`.

```
[FASE Grill: el orquestador pregunta] ← solo si bootstrap_project no está done
       │
       ▼ progress/project-definition.md
[planner_agent] ← descompone y asigna "ambiguity"
       │
       ▼
feature_list.json poblado
       │
       ▼ el orquestador clasifica → elige flujo
       │
  ┌────┴──────────────────────────────────────────────────────────────┐
  │                                                                   │
F1 Directo   (SIMPLE: 1-2 archivos, acceptance claro)
  pending → in_progress → [orquestador inline] → ⏸ HUMANO → done

F2 Delegado  (MEDIO: 2-3 archivos, claro, SIN SDD)
  pending → in_progress → [agent_developer → reviewer_agent] → ⏸ HUMANO → done
                           └─ escribe progress/plan_<name>.md antes de codear

F3 SDD       (AMBIGUO: vago o no verificable)
  pending → [sdd_agent_author] → spec_ready → ⏸ HUMANO → in_progress
          → [agent_developer → reviewer_agent] → ⏸ HUMANO → done
```

Diferencias que importan:

| | F1 | F2 | F3 |
|---|---|---|---|
| Contrato | el `acceptance` | `progress/plan_<name>.md` | `specs/<name>/` (EARS) |
| Subagentes | ninguno | 2 | 3 |
| Puertas humanas | 1 (cierre) | 1 (cierre) | 2 (spec + cierre) |
| Checkpoints | C1-C3, C6, C7 | + C8, sin C4/C5 | todos C1-C8 |

### 4.1 Puerta de Desafío (atraviesa los tres flujos)

Ningún agente de este repo está de acuerdo por defecto. Antes de ejecutar se
revisan cuatro gatillos:

- **G1 Contradicción** — choca con `docs/`, `progress/project-definition.md`, un
  spec aprobado o una decisión previa.
- **G2 Camino más simple** — existe una solución con menos archivos o piezas que
  cumple el mismo `acceptance`.
- **G3 No verificable** — un criterio de `acceptance` no se puede convertir en
  test concreto.
- **G4 Coste >> valor** — el alcance real es mucho mayor que el enunciado.

Con gatillo se emite una objeción (`⚠️ OBJECIÓN [G<n>]` + `Evidencia:` +
`Alternativa:`). Sin gatillo, silencio: objetar sin motivo es peor que no objetar
porque entrena al usuario a ignorar las objeciones que sí importan. Una objeción
rechazada por el humano se ejecuta igual y queda anotada como **riesgo asumido**,
nunca borrada.

### 4.2 Secuencia detallada

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
1. El orquestador detecta la primera feature no-`done`, la **clasifica** y
   **anuncia el flujo** (F1 / F2 / F3) en una línea. Pasa la Puerta de Desafío
   (§4.1) antes de actuar.
2. **F1** — el orquestador implementa inline (código + test), corre `./init.sh`
   y salta al paso 7.
3. **F2** — el orquestador pone `in_progress` y lanza `agent_developer` **en modo
   F2** (el prompt debe decirlo). Ese agente escribe primero
   `progress/plan_<name>.md`, luego implementa contra el `acceptance`. Sigue en
   el paso 6.
4. **F3** — el orquestador lanza `sdd_agent_author`, que crea
   `specs/<name>/{requirements,design,tasks}.md` —incluida la sección
   `## Desafío` de `design.md`— y marca el status como `spec_ready`.
   **Pausa (puerta 1 de 2).** El humano lee el spec y aprueba o pide cambios.
5. **F3, tras aprobación** — el orquestador pone `in_progress` y lanza
   `agent_developer` **en modo F3**, que ejecuta `tasks.md` una a una
   marcándolas `[x]`.
6. El `reviewer_agent` (con el modo F2/F3 indicado en su prompt) verifica
   trazabilidad, completitud y **sustancia**; devuelve `APPROVED`,
   `APPROVED_WITH_OBJECTION` o `CHANGES_REQUESTED`. Si rechaza, vuelve al
   implementador.
7. **Pausa (puerta de cierre, en los tres flujos).** El orquestador muestra lo
   hecho —y la objeción por delante si hubo `APPROVED_WITH_OBJECTION`— y espera
   aprobación humana.
8. Solo entonces el **orquestador** marca `done` y mueve el resumen a
   `progress/history.md`. Ningún otro agente escribe `done`.

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
