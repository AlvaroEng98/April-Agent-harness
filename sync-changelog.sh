#!/usr/bin/env bash
# sync-changelog.sh — Puente entre el backlog (dev) y el changelog (producto).
#
# Lee las features con status "done" de feature_list.json y agrega las que
# falten a la sección "## [Unreleased]" de CHANGELOG.md.
#
# Por qué existe: feature_list.json NO se versiona (es estado de desarrollo,
# ver .gitignore). CHANGELOG.md sí. Los releases leen CHANGELOG.md porque es
# lo único que el checkout de CI tiene disponible.
#
# Es idempotente: cada entrada se marca con el `name` de la feature entre
# backticks al final de la línea, y las que ya están no se duplican.
#
# Uso:  ./sync-changelog.sh          aplica los cambios
#       ./sync-changelog.sh --check  no escribe; exit 1 si falta algo por volcar

set -u
cd "$(dirname "${BASH_SOURCE[0]}")" || exit 1

CHECK_ONLY=0
if [ "${1:-}" = "--check" ]; then
  CHECK_ONLY=1
fi

if [ ! -f "feature_list.json" ]; then
  echo "sync-changelog: no existe feature_list.json" >&2
  exit 1
fi
if [ ! -f "CHANGELOG.md" ]; then
  echo "sync-changelog: no existe CHANGELOG.md" >&2
  exit 1
fi

python3 - "$CHECK_ONLY" <<'PY'
import json, re, sys

check_only = sys.argv[1] == "1"

with open("feature_list.json") as fh:
    features = json.load(fh)["features"]

done = [f for f in features if f.get("status") == "done"]

with open("CHANGELOG.md") as fh:
    changelog = fh.read()

# Una feature ya está volcada si su marcador `name` aparece en el changelog.
pending = [f for f in done if f"`{f['name']}`" not in changelog]

if not pending:
    print(f"sync-changelog: nada que volcar ({len(done)} done ya en CHANGELOG.md)")
    sys.exit(0)

if check_only:
    for f in pending:
        print(f"sync-changelog: falta volcar '{f['name']}' ({f['title']})")
    sys.exit(1)

# Insertar bajo "## [Unreleased]", justo después de la cabecera de sección.
marker = "## [Unreleased]"
if marker not in changelog:
    print(f"sync-changelog: CHANGELOG.md no tiene '{marker}'", file=sys.stderr)
    sys.exit(1)

lines = [f"- {f['title']} (`{f['name']}`)" for f in pending]
block = "\n" + "\n".join(lines) + "\n"

idx = changelog.index(marker) + len(marker)
# Consumir el salto de línea que sigue al marcador, si lo hay, para no
# introducir una línea en blanco extra en cada corrida.
rest = changelog[idx:]
rest = re.sub(r"^\n+", "\n", rest)
updated = changelog[:idx] + block + rest.lstrip("\n")

with open("CHANGELOG.md", "w") as fh:
    fh.write(updated)

for line in lines:
    print(f"sync-changelog: agregado {line}")
PY
