# AGENT.md — Mapa de navegación para agentes de IA

> Este archivo es el **punto de entrada** para cualquier agente que trabaje en este
> repositorio. NO es una biblia de reglas: es un **mapa**. Lee solo lo que
> necesites cuando lo necesites (divulgación progresiva).

---

## 1. Antes de empezar (obligatorio)

1. Ejecuta `./init.sh` primero, antes de cualquier otro paso. Si falla, **no
   continúes**: no se puede avanzar con el entorno roto. Resuelve el fallo
   antes de leer nada más o tocar código.
2. Lee `progress/current.md` para entender en qué estado quedó la última sesión.
3. Lee `config.json` para el proyecto, la descripción y las reglas del arnés.
4. Lee `feature_list.json`. Toda feature nueva (`"sdd": true`) pasa por
   **Spec Driven Development** — ver `docs/specs.md` y §4 de este archivo.
5. Lee `docs/specs.md` antes de tocar cualquier spec o feature `sdd: true`.

## 2. Mapa del repositorio

| Archivo / carpeta            | Qué contiene                                                                | Cuándo leerlo |
|------------------------------|-----------------------------------------------------------------------------|---------------|
| `config.json`                | Identidad del proyecto (`project`, `description`) y reglas del arnés (`rules`) | Siempre, al empezar |
| `feature_list.json`          | Lista de tareas con estado (`pending` / `spec_ready` / `in_progress` / `done` / `blocked`) | Siempre, al empezar |
| `progress/current.md`        | Estado de la sesión actual                                                  | Siempre, al empezar |
| `progress/history.md`        | Bitácora append-only de sesiones anteriores                                 | Si necesitas contexto histórico |
| `specs/<feature>/`           | `requirements.md` + `design.md` + `tasks.md` (Kiro-style)                   | Antes de implementar cualquier feature con `"sdd": true` (flujo F3) |
| `progress/plan_<feature>.md` | Contrato ligero del flujo F2: archivos + mapa `acceptance → test` + riesgo asumido | Antes de implementar (F2) y antes de revisar (F2) |
| `docs/architecture.md`       | Qué significa "hacer un buen trabajo" en este proyecto                      | Antes de implementar |
| `docs/adr/`                  | Decisiones arquitectónicas difíciles de revertir, con su porqué (skill `domain-modeling`) | Antes de deshacer o contradecir una decisión pasada |
| `docs/skills-adoptadas.md`   | Qué skills externas se instalaron en `.claude/skills/`, cuáles encajan pero no se instalaron, y cuáles se descartaron por qué | Antes de instalar una skill nueva o reevaluar el catálogo |
| `docs/conventions.md`        | Reglas de estilo, nombres, estructura                                       | Antes de escribir código |
| `docs/specs.md`              | Proceso SDD: EARS notation, los 3 archivos, puerta de aprobación humana     | Antes de redactar o leer un spec |
| `docs/verification.md`       | Cómo verificar que tu trabajo funciona (incluye trazabilidad requirements)  | Antes de declarar una tarea como `done` |
| `CHECKPOINTS.md`             | Criterios objetivos de "estado final correcto"                              | Para auto-evaluarte |
| `.claude/agents/`            | Definiciones de subagentes (`orquestador`, `planner_agent`, `sdd_agent_author`, `agent_developer`, `reviewer_agent`) | Si orquestas trabajo |
| `src/`                       | Código de la aplicación                                                     | Para implementar |
| `tests/`                     | Tests automáticos                                                           | Para verificar |

## 3. Reglas duras (no negociables)

- **Una sola feature a la vez.** No mezcles cambios de varias tareas en la misma sesión.
- **No salgas del proyecto actual.** Solo te mueves dentro de sus carpetas y
  subcarpetas. Si necesitas salir, pide permiso al usuario primero.
- **No declares una tarea `done` sin pruebas verdes.** Ejecuta `./init.sh` y
  asegúrate de que el bloque de tests pasa al 100%.
- **No saltes la fase de spec en F3.** Toda feature con `"sdd": true` debe
  pasar por `sdd_agent_author` y obtener aprobación humana antes de tocar
  código.
- **No saltes la puerta de aprobación humana.** Existe en los tres flujos: el
  orquestador para antes de escribir `done`, siempre. En F3 hay dos puertas
  (spec y cierre).
- **No estés de acuerdo por defecto.** Si el planteamiento dispara un gatillo
  G1-G4 (ver §4.1), objetas con evidencia y alternativa antes de ejecutar. Sin
  gatillo, callas y ejecutas.
- **Documenta lo que haces** en `progress/current.md` mientras trabajas, no al final.
- **Deja el repositorio limpio** antes de cerrar la sesión (ver §5).
- **Si no sabes algo, busca en `docs/`** antes de inventarlo.
- **Toda decisión difícil de revertir, sorprendente sin contexto y fruto de un
  trade-off real** se registra en `docs/adr/` (skill `domain-modeling`), no
  solo en `docs/architecture.md` — así no vuelve a quedar desactualizado en
  silencio.

## 4. Flujos de trabajo

Hay **tres** flujos de construcción. El orquestador clasifica la feature y elige
uno; una vez elegido no se mezcla con otro.

| | F1 Directo | F2 Delegado | F3 SDD |
|---|---|---|---|
| Cuándo | SIMPLE: 1-2 archivos, <100 líneas, `acceptance` claro | MEDIO: 2-3 archivos, claro, sin SDD | AMBIGUO: vago o `acceptance` no verificable |
| Ruta | `pending → in_progress → [inline] → ⏸ HUMANO → done` | `pending → in_progress → [agent_developer → reviewer_agent] → ⏸ HUMANO → done` | `pending → [sdd_agent_author] → spec_ready → ⏸ HUMANO → in_progress → [agent_developer → reviewer_agent] → ⏸ HUMANO → done` |
| Contrato | el `acceptance` | `progress/plan_<name>.md` | `specs/<name>/` (EARS) |
| Subagentes | ninguno | 2 | 3 |
| Puertas humanas | 1 (cierre) | 1 (cierre) | 2 (spec + cierre) |
| Checkpoints | C1-C3, C6, C7, C9 | + C8, sin C4/C5 | todos C1-C9 |

Esta tabla es un resumen para orientarte rápido. La **fuente única** del
protocolo completo — Puerta de Desafío (gatillos G1-G4), FASE Grill, secuencia
Casos A-G, y las reglas de "Qué NO haces" — es
**`.claude/agents/orquestador.md`**. No dupliques ese contenido aquí: si algo
de otro archivo lo contradice, gana `orquestador.md`.

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
