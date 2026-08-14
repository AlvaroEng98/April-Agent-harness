# Architecture — april-harness

> Define qué significa "hacer un buen trabajo" en este proyecto.

## Principles

1. **Separación de responsabilidades** — Cada módulo hace una cosa bien.
2. **Testabilidad** — Todo código crítico debe ser testeable aisladamente.
3. **Simplicidad** — Preferir soluciones simples sobre soluciones clever.
4. **Consistencia** — Seguir convenciones existentes del proyecto.

## Layer Structure

`april` no tiene capas de detección ni de UI interactiva: `init` siempre
scaffoldea el mismo árbol embebido, sin preguntar nada. Ver
`docs/adr/0001-scaffold-unico-sin-seleccion-interactiva.md` para por qué
(existieron y se descartaron deliberadamente).

```
┌─────────────────────────────────────────────────────────┐
│                    CLI Layer (main.go)                   │
│  - Dispatch de comandos (init, update, version, help)    │
└─────────────────────────────────────────────────────────┘
              │                              │
┌─────────────────────────────┐  ┌─────────────────────────┐
│  Scaffold (scaffold.go)      │  │  Update (update.go)      │
│  - scaffoldInit: compone     │  │  - Resuelve BIN_DIR/     │
│    planScaffold + applyPlan  │  │    VERSION del entorno   │
└─────────────────────────────┘  │  - Delega en install.sh  │
              │                  │    (curl | sh)            │
    ┌─────────┴─────────┐        └─────────────────────────┘
    ▼                   ▼
┌─────────────┐   ┌─────────────┐
│ planScaffold│   │  applyPlan  │
│ (decisión,  │   │ (I/O puro,  │
│  sin I/O de │──▶│  ejecuta el │
│  escritura) │   │  scaffoldPlan)│
└─────────────┘   └─────────────┘
```

`scaffoldInit` no decide ni escribe directamente: `planScaffold` decide (qué
archivos, con qué modo, si hay merge de `.gitignore`, qué limpiar) y devuelve
un `scaffoldPlan` como dato; `applyPlan` solo ejecuta ese plan con llamadas
`os.*`. Ver `docs/adr/` y la feature `scaffold_decision_io_seam` (acceptance
completo en `feature_list.json`) para el porqué de la costura.

## Dependency Rules

- **CLI Layer** (main.go: dispatch) orquesta Scaffold y Update; no contiene
  lógica propia de ninguno de los dos.
- **Scaffold** y **Update** no dependen entre sí.
- **planScaffold** no llama `os.WriteFile`/`os.RemoveAll`/`os.MkdirAll`: solo
  lee (`os.ReadDir`, `templateFS.ReadFile`) para decidir. **applyPlan** no
  toma ninguna decisión de contenido: solo ejecuta las llamadas `os.*` que
  `planScaffold` ya resolvió.
- Ningún módulo del binario depende de `install.sh`: Update lo invoca como
  proceso externo (ver `update.go:cmdUpdate`), no lo importa ni duplica su
  lógica de descarga/checksum.

## Module Map

| Módulo       | Archivo   | Responsabilidad                                                            |
|--------------|-----------|-----------------------------------------------------------------------------|
| CLI          | main.go   | Entry point, dispatch de comandos, banner, usage                            |
| Scaffold     | scaffold.go | `scaffoldInit`: compone `planScaffold` + `applyPlan`; `mergeGitignore`      |
| planScaffold | scaffold.go | Decisión pura: arma el `scaffoldPlan` (archivos, modos, merge, dirs vacíos) a partir del embed.FS y del estado del destino, sin escribir nada |
| applyPlan    | scaffold.go | Ejecutor delgado: aplica un `scaffoldPlan` ya decidido (`os.WriteFile`/`os.MkdirAll`/`os.RemoveAll`) y los mensajes de progreso |
| Update       | update.go | `cmdUpdate` ejecuta lo que decide `buildUpdateCmd` (name/args/env, vía `updateEnv`) y reejecuta `install.sh` |
| Config       | config.go | `version` (inyectada por ldflags en release)                                |

## Data Flow

```
usuario → april init [dir]   → main.go → scaffoldInit → embed.FS → destino/
usuario → april update [ver] → main.go → cmdUpdate → install.sh (subproceso)
```

## Error Handling

- Errores de IO → mostrar en stderr y salir con código != 0
- Errores de usuario → mostrar advertencia y continuar
- Errores de compilación → mostrar detalle y salir