# Plan de Mejoras — april-harness

> Análisis profundo del proyecto y plan escalonado de mejoras.
> Generado: 2026-07-05

---

## Qué es el proyecto

CLI en Go que scaffolds proyectos para desarrollo asistido por IA con **Spec-Driven Development (SDD)**. Genera estructura de archivos, agentes de IA y flujo de trabajo para llevar features de spec a código.

## Arquitectura actual

```
main.go (CLI) + selector.go (selección interactiva)
    ↓
templateFS embebido → copia archivos a proyecto destino
    ↓
Agentes: orquestador → planner → sdd_author → developer → reviewer
    ↓
Flujo: pending → spec_ready → ⏸ humano → in_progress → done
```

---

## Problemas identificados (por severidad)

### Críticos (bloquean uso real)

1. **`docs/` están vacíos** — Los 4 archivos de docs son solo plantillas con headers. Los agentes dependen de estos para tomar decisiones (`architecture.md`, `conventions.md`, `verification.md`). Sin ellos, los agentes improvisan.

2. **`AGENT.md:15` referencia `AGENTS.md`** — Error de tipeo que confunde al orquestador en el protocolo de arranque.

3. **`feature_list.json` es placeholder** — El proyecto no tiene ningún ejemplo real de features populadas.

4. **No hay tests para el propio harness** — El CLI no tiene validación de que funciona correctamente.

### Altos (degradan calidad)

5. **`init.sh` incompleto** — Solo valida archivos base, no verifica:
   - Que los agentes en `.claude/agents/` estén presentes y sean válidos
   - Que `go.mod`/`go.sum` estén sincronizados
   - Que el binario compile correctamente
   - Saltos de sección (salta de §2 a §5)

6. **Sin integración Git** — No hay:
   - `.gitignore` adecuado (el binario `harness-backend` está tracked)
   - Hooks pre-commit
   - Validación de estado antes de commit

7. **Sin mecanismo de recuperación** — Si el flujo SDD falla a mitad:
   - No hay rollback automático
   - No hay limpieza de estado parcial
   - `session-handoff.md` es solo plantilla sin lógica

### Medios (mejoras de UX)

8. **Solo 4 herramientas AI** — Faltan: Copilot, Windsurf, Aider, Continue, etc.

9. **`README.md` insuficiente** — No incluye:
   - Ejemplos de uso
   - Troubleshooting
   - Screenshots/demo
   - Contributing guide

10. **Sin validación de specs** — El `sdd_agent_author` puede crear specs inválidos sin que nada lo verifique antes de la aprobación humana.

11. **`CHECKPOINTS.md` básico** — Solo 7 checkpoints. Faltan:
    - Validación de performance
    - Security checks
    - Accessibility
    - Documentación de código

### Bajos (deuda técnica)

12. **`go.mod` dice Go 1.25.0** — Versión futura (posible error de tipeo, debería ser 1.21 o 1.22).

13. **Sin CI/CD** — No hay `.gitlab-ci.yml` funcional (existe pero vacío).

14. **Binario en repo** — `harness-backend` está en el directorio raíz sin `.gitignore`.

---

## Plan de Mejora Escalonado

### Fase 1: Fundamentos (1-2 días)

| # | Mejora | Archivos | Esfuerzo |
|---|--------|----------|----------|
| 1 | Poblar `docs/` con contenido real para el propio harness | `docs/*.md` | Medio |
| 2 | Corregir `AGENTS.md` → `AGENT.md` en orquestador | `.claude/agents/orquestador.md:15` | Trivial |
| 3 | Arreglar `go.mod` (versión Go) | `go.mod` | Trivial |
| 4 | Crear `.gitignore` adecuado | `.gitignore` | Trivial |
| 5 | Completar `init.sh` (saltos, validación de agentes) | `init.sh` | Medio |

### Fase 2: Robustez (2-3 días)

| # | Mejora | Archivos | Esfuerzo |
|---|--------|----------|----------|
| 6 | Tests unitarios para CLI (Go) | `*_test.go` | Alto |
| 7 | Validación de specs antes de aprobación | `sdd_agent_author.md`, `init.sh` | Medio |
| 8 | Mecanismo de rollback en flujo SDD | `orquestador.md`, `init.sh` | Alto |
| 9 | Integración Git (hooks, estado) | Nuevo archivo, `.gitignore` | Medio |

### Fase 3: UX (3-5 días)

| # | Mejora | Archivos | Esfuerzo |
|---|--------|----------|----------|
| 10 | Añadir más herramientas AI | `selector.go` | Medio |
| 11 | README completo con ejemplos | `README.md` | Bajo |
| 12 | `CHECKPOINTS.md` extendido | `CHECKPOINTS.md` | Bajo |
| 13 | Template de feature_list con ejemplo real | `feature_list.json` | Bajo |

### Fase 4: Production (5+ días)

| # | Mejora | Archivos | Esfuerzo |
|---|--------|----------|----------|
| 14 | CI/CD funcional | `.gitlab-ci.yml` | Alto |
| 15 | Distribución (releases, Homebrew) | `.goreleaser.yaml` | Alto |
| 16 | Telemetría/opciones de uso | `main.go` | Medio |

---

## Estadísticas del proyecto

| Métrica | Valor |
|---------|-------|
| Archivos Go | 2 (`main.go`, `selector.go`) |
| Agentes definidos | 5 |
| Documentos de proceso | 4 (todos vacíos) |
| Tests | 0 |
| Features de ejemplo | 1 (placeholder) |
| Herramientas AI soportadas | 4 |
