# Session History

<!-- Append-only log. Most recent at the top. -->

## 2026-07-19 — Feature: auto_recap_hook (ID: 2)

**Estado:** done

**Resumen:** Recap automático del estado del proyecto al iniciar sesión, con lógica única en `recap.sh` (raíz) invocada por Claude Code (SessionStart hook) y OpenCode (plugin). `init.sh` §5 pasó a delegar en `recap.sh`, eliminando el grep/python inline duplicado. `main.go` embebe `recap.sh` y `.opencode`, y usa guard genérico por herramienta (D4) para copiar solo los artefactos de la herramienta elegida.

**Archivos creados:**
- `recap.sh` — Fuente única de la lógica de recap
- `.claude/settings.json` — Hook SessionStart (matcher startup|resume|clear)
- `.claude/hooks/session_start_recap.sh` — Script delgado que reenvía stdout de recap.sh
- `.opencode/plugins/recap.js` — Plugin OpenCode (experimental.chat.system.transform, dedupe por sessionID)
- `recap_test.go` — Tests de la lógica compartida (3 casos incl. bordes)

**Archivos modificados:**
- `main.go` — embed + guard genérico D4 + mode 0755 para scripts recap
- `init.sh` — §5 delega en recap.sh

**Verificación:** 14/14 tasks. `go build`, `go vet`, `go test ./...`, `./init.sh` verdes. Reviewer APROBADO (0 hallazgos bloqueantes, 2 LOW no bloqueantes).

**Riesgo aceptado:** OpenCode no tiene SessionStart nativo → hook experimental (mitigado con dedupe + try/catch silencioso). Validación manual del plugin contra instalación real de opencode pendiente.

## 2026-07-16 — Feature: centralize_config (ID: 1)

**Estado:** done

**Resumen:** Se creó `config.go` como fuente única de verdad para `version` y `RequiredFiles`. Se eliminaron valores duplicados de `main.go` e `init.sh`. Se implementó `go generate` para crear `required_files.txt` que `init.sh` consume para verificar archivos requeridos.

**Archivos creados/modificados:**
- `config.go` — Fuente única para version y RequiredFiles
- `gen_required.go` — Generador de required_files.txt
- `main.go` — Eliminada declaración de version
- `init.sh` — Lee archivos desde required_files.txt
- `required_files.txt` — Generado vía go generate
- `config_test.go` — Tests para verificar config

**Resultado:** Todas las 9 requirements cubiertas, tests pasan, init.sh verde.
