---
name: design-flow
description: Clasifica la entrada de una tarea nueva en uno de tres flujos — directo, planificación o SDD — antes de delegar.
---

## Criterios de decisión

Evalúa la entrada del usuario contra estas ramas, en orden. La primera que aplique gana.

| Rama | Criterio | Acción |
|------|----------|--------|
| **Directo** | Cambio acotado a 1-3 ficheros, lógica clara, sin migraciones ni features implícitas. | Lanza `agent_developer` directo, sin pasar por planificación. |
| **Planificación** | Toca más de 3 ficheros o varios módulos, pero el objetivo y el alcance ya son claros — no hace falta investigar ni preguntar nada para saber qué construir. | Lanza `planner_agent`. |
| **SDD** | La entrada es ambigua: el objetivo no se puede traducir todavía a un `acceptance` verificable, o no está claro cómo abordar la implementación. | Lanza el flujo SDD completo, empezando por `sdd_agent_author`. |

Si dudas entre dos ramas, sube a la más pesada (Directo → Planificación → SDD). Es mejor sobredescribir que subestimar.
