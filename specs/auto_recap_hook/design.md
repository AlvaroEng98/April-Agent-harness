# Design — auto_recap_hook

## Investigación previa (evidencia, no supuestos)

Antes de diseñar se verificó la API real de ambas herramientas contra sus
fuentes primarias (acceso a red disponible en este entorno, verificado
17/07/2026):

- **Claude Code hooks**: `https://docs.claude.com/en/docs/claude-code/hooks.md`
  (referencia oficial de hooks). Confirma:
  - `SessionStart` es un evento válido, soporta `type: "command"`, con
    `matcher` sobre `startup | resume | clear | compact`.
  - El hook recibe JSON por stdin y puede devolver
    `hookSpecificOutput.additionalContext` en JSON por stdout, **o
    simplemente imprimir texto plano en stdout** — la doc dice literalmente:
    "a hook that only loads context can print to stdout directly without
    building JSON". Esto se usa para simplificar el wrapper (ver D2).
  - El comando puede referenciar `${CLAUDE_PROJECT_DIR}` como variable de
    entorno para rutas absolutas dentro del repo.
- **OpenCode plugins**: código fuente real (`anomalyco/opencode`, branch
  `dev`, `packages/plugin/src/index.ts`) y doc oficial
  (`packages/web/src/content/docs/plugins.mdx`). Confirma:
  - Los plugins se cargan automáticamente desde `.opencode/plugins/*.{js,ts}`
    (proyecto) o `~/.config/opencode/plugins/` (global).
  - El objeto `Hooks` **no tiene** un hook `SessionStart` equivalente al de
    Claude Code. Los hooks de sesión disponibles vía `event` son
    `session.created`, `session.idle`, `session.status`, etc., pero el hook
    `event` es *fire-and-forget* (no tiene un `output` para inyectar
    contexto en la conversación).
  - El único hook con un canal de salida diseñado para inyectar texto en el
    prompt de sistema es `"experimental.chat.system.transform"`:
    `(input: { sessionID?: string; model: Model }, output: { system: string[] }) => Promise<void>`.
    Está marcado `experimental` explícitamente por OpenCode — es el
    mecanismo más cercano a "inyectar contexto al iniciar sesión" que existe
    hoy, pero **no es un hook de sesión estricto**: se dispara antes de cada
    llamada al modelo, no una sola vez al crear la sesión.

Este hallazgo (no hay `SessionStart` nativo en OpenCode) es un **riesgo
aceptado y documentado**, no un supuesto sin verificar — ver sección
"Riesgos" más abajo.

## Archivos afectados

| Archivo                                    | Acción      | Descripción                                                                 |
|---------------------------------------------|-------------|------------------------------------------------------------------------------|
| `recap.sh`                                   | **Crear**   | Lógica única de recap, en la raíz del repo (se auto-scaffoldea: es a la vez template y archivo real de este repo). |
| `.claude/settings.json`                      | **Crear**   | Declara el hook `SessionStart` de Claude Code.                              |
| `.claude/hooks/session_start_recap.sh`       | **Crear**   | Wrapper delgado invocado por el hook; delega en `recap.sh`.                 |
| `.opencode/plugins/recap.js`                 | **Crear**   | Plugin OpenCode; delega en `recap.sh` vía el shell de Bun (`$`).            |
| `main.go`                                    | **Modificar** | (a) agrega `recap.sh` y `.opencode` a `//go:embed` (línea 14). (b) generaliza el guard de gating por herramienta en `cmdInit()`. (c) extiende el cálculo de `mode` de archivo para dar `0755` a los nuevos scripts. |
| `init.sh`                                    | **Modificar** | Sección 5 (líneas ~118-149) delega en `recap.sh` en vez de reimplementar grep/python inline. |
| `recap_test.go`                              | **Crear**   | Tests Go que ejecutan `recap.sh` contra fixtures y verifican su stdout.     |

## Decisiones técnicas

### D1 — `recap.sh` como única fuente de verdad, auto-localizable

`recap.sh` vive en la raíz del repo. Al ser invocado desde ubicaciones
distintas (CWD del propio repo por `init.sh`, `$CLAUDE_PROJECT_DIR` por el
hook de Claude, `directory`/`worktree` del `PluginInput` por el plugin de
OpenCode), el script debe resolver su propio directorio al inicio:

```bash
cd "$(dirname "${BASH_SOURCE[0]}")" || exit 1
```

Así, sin importar desde dónde se invoque (`./recap.sh`, `bash
/ruta/absoluta/recap.sh`), las rutas relativas internas
(`progress/history.md`, `feature_list.json`, `progress/current.md`) siempre
resuelven contra la raíz del proyecto.

Formato de salida (texto plano, sin color, una línea por hecho, líneas
ausentes si no aplican):

```
Última sesión: <última entrada de progress/history.md>
Feature actual: <title> (<status>)
Sesión activa: <name> (<status>)
```

o en los casos borde:

```
Feature actual: Todas las features completadas
No hay sesión activa
```

Esto reutiliza exactamente la misma semántica que ya tenía `init.sh` §5
(grep `^## `, parseo Python de `feature_list.json`, grep de
`progress/current.md`), sólo que ahora vive en un solo lugar.

### D2 — Claude Code: hook delgado + passthrough de stdout, sin JSON

`.claude/hooks/session_start_recap.sh`:

```bash
#!/bin/bash
"$CLAUDE_PROJECT_DIR/recap.sh"
```

La documentación oficial confirma que para `SessionStart`, si sólo se
necesita `additionalContext`, imprimir texto plano en stdout basta — Claude
Code lo añade como contexto igual que si viniera en
`hookSpecificOutput.additionalContext`. Se descarta envolver la salida en
JSON con `jq`/`python3` (ver alternativa descartada A1) porque añade una
dependencia y complejidad no necesaria (principio de simplicidad).

`.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume|clear",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/session_start_recap.sh"
          }
        ]
      }
    ]
  }
}
```

`matcher` se restringe a `startup|resume|clear` (no incluye `compact`)
porque el recap es información de "inicio de sesión", y volver a
inyectarlo tras cada compactación automática no aporta valor nuevo y podría
mostrar datos ya vistos por el agente en la misma sesión.

### D3 — OpenCode: plugin vía `experimental.chat.system.transform` con dedupe por sesión

`.opencode/plugins/recap.js` (JS plano, sin imports de tipos, para evitar
depender de resolución de paquetes npm en tiempo de carga):

- Recibe `{ directory, $ }` del `PluginInput`.
- Mantiene un `Set` de `sessionID` ya procesados en el closure del módulo
  (vive mientras el proceso de OpenCode esté vivo).
- En `"experimental.chat.system.transform"`, si `input.sessionID` no está en
  el set: lo agrega, ejecuta `` await $`bash ${directory}/recap.sh`.text() ``
  y hace `output.system.push(texto)` si el resultado no es vacío.
- Envuelve la llamada en `try/catch` para que un fallo de `recap.sh` (no
  encontrado, sin permisos, etc.) no rompa la sesión de OpenCode — se
  degrada en silencio.

Esto satisface "inyectar contexto al iniciar sesión" de la forma más
cercana posible dado que no existe un hook `SessionStart` real en OpenCode
(ver Riesgos).

### D4 — `main.go`: generalizar el gating por herramienta

Hoy el `fs.WalkDir` de `cmdInit()` sólo gatea dos rutas exactas
(`".claude"` y `".claude/agents"`) detrás de `want[".claude"]`. Cualquier
otro archivo nuevo bajo `.claude/` (como `settings.json` o `hooks/*.sh`) o
bajo `.opencode/` caería en la rama genérica "resto de archivos y
directorios" y se copiaría **sin condición**, violando el acceptance
criterion 4. Se reemplaza el guard estrecho por uno genérico, insertado
después de la rama de transformación de agentes y antes de la rama
genérica:

```go
for _, toolDir := range []string{".claude", ".opencode"} {
    if path == toolDir || strings.HasPrefix(path, toolDir+"/") {
        if !want[toolDir] {
            if d.IsDir() {
                return fs.SkipDir
            }
            return nil
        }
        break
    }
}
```

`fs.SkipDir` evita además recorrer innecesariamente el subárbol de una
herramienta no elegida. Esto subsume y reemplaza la rama estrecha existente
(`path == ".claude" || path == ".claude/agents"`), que ya no es necesaria.

### D5 — Permisos de ejecución para los nuevos scripts

Se extiende la condición existente en `cmdInit()`:

```go
mode := fs.FileMode(0644)
if d.Name() == "init.sh" {
    mode = 0755
}
```

a también cubrir `"recap.sh"` y `"session_start_recap.sh"` (ambos
identificados por nombre base, igual que `init.sh`).

### D6 — `init.sh` §5 delega, no reimplementa

Se reemplaza el cuerpo actual de la sección 5 (grep + bloque `python3`
inline) por una invocación a `./recap.sh`, formateando cada línea no vacía
de su salida con la función `ok()` ya existente, preservando el
comportamiento visual actual. Si `recap.sh` no existe o falla, la sección
no debe hacer fallar el script completo (requirement no-deseado R16):

```bash
if [ -x "./recap.sh" ]; then
  while IFS= read -r line; do
    [ -n "$line" ] && ok "$line"
  done <<< "$(./recap.sh 2>/dev/null)"
else
  warn "recap.sh no encontrado o sin permisos de ejecución"
fi
```

### D7 — `RequiredFiles` (config.go) no se modifica

Se evaluó agregar `recap.sh`, `.claude/settings.json` o
`.opencode/plugins/recap.js` a `RequiredFiles` (config.go). Se decide
**no hacerlo**: `RequiredFiles` valida archivos que deben existir siempre en
*cualquier* proyecto scaffoldeado (agnósticos de herramienta) — pero
`.claude/settings.json` y `.opencode/plugins/recap.js` son condicionales a
la herramienta elegida, y `RequiredFiles` no tiene hoy un concepto de
"requerido condicionalmente". Forzar esa distinción ahí sería un cambio de
alcance mayor no pedido por esta feature. `recap.sh` en cambio sí es
agnóstico y siempre se copia (R14), pero no está en el `acceptance` de la
feature agregarlo a `RequiredFiles`, así que no se añade — no se inventa un
requirement no solicitado.

## Alternativas descartadas

- **A1 — Construir `hookSpecificOutput.additionalContext` con `jq`/`python3`
  en el hook de Claude.** Descartada: la doc oficial confirma que stdout
  plano ya es suficiente para este caso de uso, y añadir una dependencia de
  `jq` (no usada hoy en el proyecto) para escapar JSON contradice el
  principio de simplicidad sin aportar funcionalidad extra.
- **A2 — Usar el hook `event` de OpenCode sobre `session.created` en vez de
  `experimental.chat.system.transform`.** Descartada: `event` es
  fire-and-forget, sin `output` para inyectar contexto en la conversación;
  no existe una vía documentada para que un handler de `event` modifique el
  prompt de sistema. `chat.system.transform` es el único hook con un canal
  de salida (`output.system`) diseñado para este propósito.
- **A3 — Reescribir la lógica de recap en Go como subcomando `harness
  recap` en vez de bash.** Descartada: el binario `harness` es una
  herramienta de desarrollo (no se distribuye dentro del proyecto
  scaffoldeado), así que un subcomando Go no estaría disponible en el
  proyecto destino sin cambios de arquitectura mayores (habría que
  embeber/distribuir el binario). Bash mantiene paridad con el patrón
  existente (`init.sh` ya usa `python3` inline) con footprint mínimo.
- **A4 — Mantener el guard estrecho de `.claude`/`.claude/agents` y agregar
  casos especiales uno por uno por cada archivo nuevo.** Descartada: no
  escala si se agregan más archivos condicionales por herramienta en el
  futuro; un guard genérico por prefijo de directorio de herramienta es más
  simple y correcto.

## Riesgos a validar (declarados explícitamente, no supuestos)

- **Riesgo 1 (alto):** OpenCode no tiene un hook `SessionStart` nativo
  equivalente al de Claude Code (confirmado en el código fuente y la doc
  oficial a fecha 17/07/2026). Se usa `experimental.chat.system.transform`
  como la aproximación más cercana, con dedupe por `sessionID`. Es un hook
  marcado explícitamente `experimental` por OpenCode — su firma o
  disponibilidad puede cambiar en versiones futuras sin aviso de
  compatibilidad. Mitigación: `try/catch` para degradar sin romper la
  sesión; validar manualmente contra una instalación real de `opencode`
  quede como tarea pendiente (ver `tasks.md`).
- **Riesgo 2 (medio):** El comportamiento exacto de carga automática de
  `.opencode/plugins/*.js` (orden de carga, si corre en cada mensaje o una
  vez) se documenta en `plugins.mdx` pero no se pudo ejecutar `opencode`
  real en este entorno para confirmarlo empíricamente.
- **Riesgo 3 (bajo):** El matcher `startup|resume|clear` de
  `.claude/settings.json` asume sintaxis de lista separada por `|`,
  soportada según la doc oficial (`hooks.md`, sección "Matcher patterns").
