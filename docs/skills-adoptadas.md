# Skills adoptadas — levantamiento contra `GUIA-INTEGRACION-SKILLS.md`

> Fuente: catálogo de 35 skills en `/home/alvaro/Proyectos/skills` (otro
> repo). Este documento registra qué se adoptó acá, qué encaja pero no se
> instaló todavía, y qué se descartó — para no re-evaluar el catálogo entero
> cada sesión.

## Instaladas (`.claude/skills/`)

| Skill | Por qué |
|-------|---------|
| `domain-modeling` | Capa de decisión: `CONTEXT.md` (glosario) + `docs/adr/` (decisiones difíciles de revertir). Cerró el punto 1 del reporte de arquitectura (docs/architecture.md desactualizado) — ver `docs/adr/0001-scaffold-unico-sin-seleccion-interactiva.md`. |
| `codebase-design` | Vocabulario módulo/interfaz/costura/apalancamiento/localidad. Dependencia que `improve-codebase-architecture` ya asumía (`Run the /codebase-design skill...`) pero no estaba instalada — hueco real. |
| `grill-with-docs` | Enganche para retomar en otra sesión una tarjeta elegida del reporte de `improve-codebase-architecture`, formalizando la decisión en `CONTEXT.md`/ADR. |
| `improve-codebase-architecture` | Ya estaba instalada antes de este levantamiento. Escanea hot spots y entrega reporte HTML de oportunidades de profundizar módulos. |
| `grilling` | Ya estaba instalada. Motor de entrevista que usan `grill-with-docs` y otras. |
| `writing-for-agents` | Ya estaba instalada. Referencia para escribir `SKILL.md`/`CLAUDE.md`/`AGENT.md`. |

## Candidatas — encajan, no instaladas todavía

Sin conflicto con lo existente; pendientes de decisión explícita de instalar.

| Skill | Qué hace | Por qué encaja |
|-------|----------|-----------------|
| `diagnosing-bugs` | Disciplina de 6 fases para bugs difíciles: reproducir/minimizar, hipótesis falsables, instrumentar, fix + test de regresión en la costura correcta. | Este repo no tiene equivalente propio; no pisa a `orquestador`/F1-F2-F3 (esos son de entrega de features, no de debugging). |
| `tdd` | Referencia del ciclo rojo-verde: qué hace bueno a un test, dónde van, anti-patrones. | Refuerza a `agent_developer` al escribir tests, sin reemplazarlo. |
| `resolving-merge-conflicts` | Resuelve merge/rebase en curso, hunk por hunk, rastreando intención original; nunca `--abort`. | Standalone, útil independiente del flujo de features. |
| `git-guardrails-claude-code` (misc) | Hook `PreToolUse` que bloquea `push --force`, `reset --hard`, `clean -f`, `branch -D` antes de que el agente los ejecute. | Ya está como regla escrita en `CLAUDE.md` ("Git Safety Protocol"); esto la vuelve mecánica en vez de depender de que el agente "se acuerde". |
| `wizard` | Genera un script bash interactivo para pasos que solo un humano puede hacer (login, dashboards de terceros); nunca lo ejecuta. | Sin dependencia de tracker; útil para pasos manuales de release (tokens, secrets). |

## Descartadas — conflicto real con el flujo existente

Todas asumen un tracker de issues (GitHub/Linear) como fuente de verdad del
backlog. Este repo ya tiene la suya: `feature_list.json` + F1/F2/F3 vía
`orquestador`. Instalarlas correría dos sistemas de backlog en paralelo
(gatillo G1 — contradicción con `.claude/agents/orquestador.md`).

- `triage`
- `wayfinder`
- `to-spec`
- `to-tickets`
- `implement`
- `setup-matt-pocock-skills`
- `code-review` (requiere `docs/agents/issue-tracker.md` de `setup-matt-pocock-skills`)

Redundantes con mecanismo propio ya existente:

- `handoff` / `claude-handoff` — este repo ya tiene su lifecycle de cierre de
  sesión (`progress/current.md` → `progress/history.md`, `AGENT.md` §5).

## No aplica (stack o contexto)

- `setup-ts-deep-modules`, `migrate-to-shoehorn`, `setup-pre-commit`,
  `scaffold-exercises` — TypeScript/Node/`ai-hero-cli` específicos; este repo
  es Go.
- `ask-matt`, `grill-me`, `teach`, `loop-me`, `to-questionnaire`, `wait-what`,
  `writing-fragments`, `writing-shape`, `writing-beats`, `research`,
  `prototype` — productividad genérica, no ligada al flujo de este repo. Se
  evalúan on-demand si aparece una necesidad concreta, no por adopción previa.

## Cómo retomar esto

Antes de volver a evaluar el catálogo completo: revisar solo la sección
"Candidatas" de arriba y decidir cuál instalar. Si el catálogo fuente cambia
(nuevas skills en `/home/alvaro/Proyectos/skills`), re-hacer el levantamiento
solo para lo nuevo, no repetir el análisis de las ya clasificadas acá.
