# Tasks — auto_recap_hook

- [x] T1 — Crear `recap.sh` en la raíz del repo: se auto-localiza
      (`cd "$(dirname "${BASH_SOURCE[0]}")"`) e imprime en stdout, en este
      orden: `Última sesión: ...` (si aplica), `Feature actual: ...` (o
      `Todas las features completadas`), `Sesión activa: ...` (o
      `No hay sesión activa`). Cubre: R1, R2, R3, R4, R5, R6.
- [x] T2 — Dar permisos de ejecución a `recap.sh` (`chmod +x`) y verificar
      manualmente que `./recap.sh` corre standalone desde la raíz del repo
      con los datos reales actuales de `progress/` y `feature_list.json`.
      Cubre: R1, R14.
- [x] T3 — Modificar `main.go` línea 14: agregar `recap.sh` y `.opencode` a
      la directiva `//go:embed`. Cubre: R18.
- [x] T4 — En `cmdInit()` (main.go), reemplazar el guard estrecho
      (`path == ".claude" || path == ".claude/agents"`) por el guard
      genérico por herramienta descrito en `design.md` (D4), usando
      `fs.SkipDir` para directorios de una herramienta no elegida. Cubre:
      R10, R11, R12, R13.
- [x] T5 — Extender el cálculo de `mode` en `cmdInit()` para dar `0755` a
      los archivos `recap.sh` y `session_start_recap.sh` (por nombre base),
      igual que ya ocurre con `init.sh`. Cubre: R14, R8.
- [x] T6 — Crear `.claude/settings.json` con el hook `SessionStart`
      (`matcher: "startup|resume|clear"`, `type: "command"`) que invoque
      `${CLAUDE_PROJECT_DIR}/.claude/hooks/session_start_recap.sh`. Cubre:
      R7.
- [x] T7 — Crear `.claude/hooks/session_start_recap.sh`: script delgado que
      ejecuta `"$CLAUDE_PROJECT_DIR/recap.sh"` y reenvía su stdout sin
      reimplementar lógica. Cubre: R8.
- [x] T8 — Crear `.opencode/plugins/recap.js`: plugin que exporta el hook
      `"experimental.chat.system.transform"`, deduplica por `sessionID` en
      un `Set` de closure, invoca `recap.sh` vía el shell de Bun (`$`)
      provisto en `PluginInput`, y hace `output.system.push(...)` con el
      resultado (dentro de un `try/catch` que degrada en silencio si
      `recap.sh` falla). Cubre: R9.
- [x] T9 — Modificar la sección 5 de `init.sh` (líneas ~118-149) para
      invocar `./recap.sh` y envolver cada línea no vacía de su salida con
      `ok "<línea>"`, eliminando el grep/python inline duplicado. Si
      `recap.sh` no existe o no es ejecutable, usar `warn` en vez de
      abortar la sección. Cubre: R15, R16.
- [x] T10 — Crear `recap_test.go` en la raíz: tests que arman fixtures
      temporales de `progress/history.md`, `feature_list.json` y
      `progress/current.md` en un directorio temporal, ejecutan `recap.sh`
      allí (o pasan la ruta como argumento/CWD) y verifican las 3 líneas de
      salida, incluyendo los casos borde "todas las features done" (R4) y
      "sin sesión activa" (R6). Cubre: R17.
- [x] T11 — Ejecutar `go build .` y confirmar que compila sin errores.
      Cubre: R18.
- [x] T12 — Ejecutar `go test ./...` y confirmar que `recap_test.go` (y el
      resto de la suite) pasa en verde. Cubre: R17.
- [x] T13 — Ejecutar `./init.sh` completo y confirmar que la sección 5
      sigue mostrando `[OK]` para las 3 líneas de recap con el estado real
      del repo, y que el resto de secciones no se ve afectado. Cubre: R15,
      R16.
- [x] T14 — Documentar en `progress/impl_auto_recap_hook.md` el riesgo
      aceptado sobre `experimental.chat.system.transform` de OpenCode (no
      hay `SessionStart` nativo equivalente, hook marcado experimental) y
      los pasos manuales de validación pendientes contra una instalación
      real de `opencode` (no disponible en este entorno de desarrollo).
      Cubre: R9.
