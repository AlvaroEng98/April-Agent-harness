# Architecture

> Rellenado por la skill `grill-docs` durante la Fase Grill de
> `bootstrap_project`. Sección sin respuesta del humano queda literalmente
> `_pendiente_` — nunca se omite, nunca se inventa una respuesta plausible.

## Principios

1. Monolito plano hasta que duela — `package main` sin subcarpetas
   (`internal/`, `cmd/`). Migrar a esa estructura solo si un archivo
   concreto se vuelve inmanejable, no por anticipación (decisión
   confirmada 26/08/2026, comparando contra `internal/` de gentle-ai).
2. Sin dependencias externas — solo stdlib. Cualquier necesidad que
   sugiera una librería de terceros (ej. git nativo) se resuelve invocando
   el binario correspondiente como subproceso.
3. Un archivo por responsabilidad, con su `_test.go` gemelo — nunca un
   archivo con varias responsabilidades no relacionadas.
4. Estado crítico (manifiesto, ledger de receipts) se escribe siempre
   atómico: write-temp-then-rename. Nunca se deja el archivo real a medio
   escribir si el proceso se corta a mitad de camino.
5. El estado del harness se deriva del disco cuando existe un comando que
   pueda calcularlo (`april status`, ver `ROADMAP.md` E1) — nunca se
   infiere leyendo prosa de `progress/*.md` o `CLAUDE.md`.

## Capas / Módulos

| Módulo | Responsabilidad |
|---|---|
| `main.go` | Entry point del CLI: parseo de comando (`init`/`update`/`version`/`help`), banner, delegación a `cmdInit`/`cmdUpdate`. |
| `config.go` | Fuente única de la versión (`version`). |
| `scaffold.go` | Motor de scaffold: manifiesto (`.claude/manifest.json`, sha256 por archivo, patrón last-applied-configuration), `planScaffoldFromFS` (decide create/update/skip/delete por archivo) y `applyPlan` (ejecuta el plan sobre el filesystem destino). |
| `update.go` | Self-update: reejecuta `install.sh` apuntando al directorio del binario en uso. |
| *(futuro, ROADMAP.md E1-E6)* `status.go`, `doctor.go`, `verify.go`, `review.go`, `feature.go` | Un archivo por comando nuevo, mismo patrón que los módulos actuales — sin agrupar en subpaquetes salvo que el principio 1 deje de sostenerse. |

No hay separación por capas (dominio/infra/UI) — la unidad de organización
es "qué comando o subsistema resuelve este archivo", no una arquitectura
en capas.

## Flujo de datos

```
april init [dir]
  └─ cmdInit → scaffoldInit(absTarget)
       └─ planScaffoldFromFS(absTarget, templateFS)   # lee manifest.json existente + templates/ embebido
            └─ scaffoldPlan{ writes[], deletes[] }     # decisión pura, sin efectos
                 └─ applyPlan(plan)                    # único punto que toca el filesystem
                      └─ writeManifest(...)            # atómico: temp + rename
```

Futuro (`ROADMAP.md` E1-E5), mismo principio — cálculo puro separado de
escritura:

```
april status --json
  └─ lee feature_list.json + specs/<name>/{spec.md,tickets/*.md} + ledger
       └─ calcula {phase, nextRecommended, blockedReasons, frontier}   # solo lectura, nunca escribe

april feature set-status <id> <estado>     # E3, autoritativo tras el gate humano
  └─ valida transición contra el grafo de estados
       └─ escribe feature_list.json         # atómico, único punto de escritura válido desde E3
```

## Qué NO hacer

- No crear `internal/`/`cmd/` "por si acaso" — es una decisión tomada
  explícitamente en contra, no un olvido.
- No agregar un módulo de terceros sin que quede documentado por qué stdlib
  no alcanza — la respuesta por defecto es "no".
- No escribir `feature_list.json`, `.claude/manifest.json` o el futuro
  ledger de receipts con un `os.WriteFile` directo — siempre temp+rename.
- No hacer que `april status` (una vez exista) escriba nada — es
  puramente derivado hasta que E3 lo vuelva autoritativo, y ni siquiera
  entonces `status` es el que escribe (`set-status` sí).
- No inferir la fase de una feature leyendo texto libre de `CLAUDE.md` o
  `progress/*.md` una vez que `april status` exista — esa es exactamente
  la falla que el `ROADMAP.md` señala y busca cerrar.
