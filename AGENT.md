# AGENT.md — Mapa de navegación para agentes de IA

> Este archivo es el **punto de entrada** para cualquier agente que trabaje en este
> repositorio. NO es una biblia de reglas: es un **mapa**. Lee solo lo que
> necesites cuando lo necesites (divulgación progresiva).

---

## 1. Antes de empezar (obligatorio)

1. Ejecuta `./init.sh` primero, antes de cualquier otro paso. Si falla, **no
   continúes**: no se puede avanzar con el entorno roto. Resuelve el fallo
   antes de leer nada más o tocar código.

## 2. Mapa del repositorio

| Archivo / carpeta            | Qué contiene                                                                | Cuándo leerlo |
|------------------------------|-----------------------------------------------------------------------------|---------------|
| `feature_list.json`          | Lista de tareas con estado                                                  | Siempre, al empezar |
| `progress/current.md`        | Estado de la sesión actual                                                  | Siempre, al empezar |
| `progress/history.md`        | Bitácora append-only de sesiones anteriores                                 | Si necesitas contexto histórico |
| `specs/<feature>/`           | `requirements.md` + `design.md` + `tasks.md` (Kiro-style)                   | Antes de implementar cualquier feature con `"sdd": true` (flujo F3) |
| `progress/plan_<feature>.md` | Contrato ligero del flujo F2: archivos + mapa `acceptance → test` + riesgo asumido | Antes de implementar (F2) y antes de revisar (F2) |
| `docs/architecture.md`       | Qué significa "hacer un buen trabajo" en este proyecto                      | Antes de implementar |
| `docs/adr/`                  | Decisiones arquitectónicas difíciles de revertir, con su porqué (skill `domain-modeling`) | Antes de deshacer o contradecir una decisión pasada |
| `docs/conventions.md`        | Reglas de estilo, nombres, estructura                                       | Antes de escribir código |
| `docs/specs.md`              | Proceso SDD: EARS notation, los 3 archivos, puerta de aprobación humana     | Antes de redactar o leer un spec |
| `docs/verification.md`       | Cómo verificar que tu trabajo funciona (incluye trazabilidad requirements)  | Antes de declarar una tarea como `done` |
| `CHECKPOINTS.md`             | Criterios objetivos de "estado final correcto"                              | Para auto-evaluarte |
| `.claude/agents/`            | Definiciones de subagentes                                                  | Si orquestas trabajo |

## 3. Reglas duras (no negociables)

- **Una sola feature a la vez.** No mezcles cambios de varias tareas en la misma sesión.
- **No salgas del proyecto actual.** Solo te mueves dentro de sus carpetas y
  subcarpetas.
- **No declares una tarea `done` sin pruebas verdes.** Ejecuta `./init.sh` y
  asegúrate de que el bloque de tests pasa al 100%.
- **No saltes la puerta de aprobación humana.** Existe en los dos flujos: el
  orquestador para antes de escribir `done`, siempre. En F3 hay dos puertas
  (spec y cierre).
- **No estés de acuerdo por defecto.** Siempre que exista una ambigüedad y que
  algo no esté claro cuestiona las decisiones del usuario.
- **Documenta lo que haces** en `progress/current.md` mientras trabajas, no al final.
- **Si no sabes algo, busca en `docs/`**, nunca inventes; si te falta algo,
  pregunta al usuario.
- **Toda decisión difícil de revertir, sorprendente sin contexto y fruto de un
  trade-off real** se registra en `docs/adr/` (skill `domain-modeling`), no
  solo en `docs/architecture.md` — así no vuelve a quedar desactualizado en
  silencio.

## 4. Cómo elegir una tarea

```
1. Abre feature_list.json
2. Filtra por status == "pending"
3. Coge la de menor "id"
4. Cambia su status a "in_progress" y guarda
```

Cómo clasificar esa feature entre F2 y F3, la Puerta de Desafío (gatillos
G1-G4), la FASE Grill y la secuencia de Casos A-G: todo eso vive en
**`.claude/agents/orquestador.md`**, no aquí. Es la fuente única — no lo dupliques.

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
