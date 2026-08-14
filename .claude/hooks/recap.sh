#!/usr/bin/env bash
# recap.sh — Fuente única de verdad del recap de estado del proyecto.
#
# Imprime en stdout, en este orden (una línea por hecho, sin color):
#   1. "Última sesión: <última entrada de progress/history.md>"  (solo si aplica)
#   2. "Feature actual: <title> (<status>)"  ó
#      "Feature actual: Todas las features completadas"
#   3. "Sesión activa: <name> (<status>)"  ó  "No hay sesión activa"
#
# Este script es el único punto de verdad del recap: lo consumen tanto
# init.sh (§5) como el SessionStart hook de Claude Code. No debe duplicarse
# su lógica en ningún otro lugar.
#
# Se auto-localiza al inicio para que las rutas relativas internas resuelvan
# siempre contra la raíz del proyecto, sin importar desde dónde se invoque.
# Vive en .claude/hooks/, así que la raíz es $CLAUDE_PROJECT_DIR si está
# seteada (caso hook de Claude Code) o, si no, dos niveles arriba de este
# script (.claude/hooks/ → raíz).

if [ -n "${CLAUDE_PROJECT_DIR:-}" ]; then
  cd "$CLAUDE_PROJECT_DIR" || exit 1
else
  cd "$(dirname "${BASH_SOURCE[0]}")/../.." || exit 1
fi

# 1. Última sesión — primera entrada '## ' de progress/history.md
if [ -f "progress/history.md" ]; then
  last_entry=$(grep -m1 '^## ' progress/history.md | sed 's/^## //')
  if [ -n "$last_entry" ]; then
    echo "Última sesión: $last_entry"
  fi
fi

# 2. Feature actual — primera feature no-done de feature_list.json
if [ -f "feature_list.json" ]; then
  current_feature=$(python3 -c "
import json
try:
    data = json.load(open('feature_list.json'))
    for f in data['features']:
        if f['status'] != 'done':
            print(f'{f[\"title\"]} ({f[\"status\"]})')
            break
    else:
        print('Todas las features completadas')
except: print('')
" 2>/dev/null)
  if [ -n "$current_feature" ]; then
    echo "Feature actual: $current_feature"
  fi
fi

# 3. Sesión activa — línea '- Name:' de progress/current.md
if [ -f "progress/current.md" ]; then
  current_name=$(grep -m1 '^- Name:' progress/current.md 2>/dev/null | sed 's/^- Name: *//')
  current_status=$(grep -m1 '^- Status:' progress/current.md 2>/dev/null | sed 's/^- Status: *//')
  if [ -n "$current_name" ]; then
    echo "Sesión activa: $current_name ($current_status)"
  else
    echo "No hay sesión activa"
  fi
else
  echo "No hay sesión activa"
fi
