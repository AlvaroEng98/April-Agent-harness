#!/usr/bin/env bash
# release-notes.sh — Extrae de CHANGELOG.md el cuerpo de una versión.
#
# Lo consume el workflow de release para alimentar release.header de
# goreleaser (via la variable de entorno RELEASE_NOTES). Se lee CHANGELOG.md
# y no feature_list.json porque este último no está versionado: el checkout
# de CI no lo tiene.
#
# Busca la sección "## [<version>]". Si no existe, cae a "## [Unreleased]",
# que es el caso normal cuando se taggea sin haber promovido la sección.
#
# Uso:  ./release-notes.sh v0.3.0

set -u
cd "$(dirname "${BASH_SOURCE[0]}")" || exit 1

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  echo "uso: ./release-notes.sh <version>" >&2
  exit 1
fi

if [ ! -f "CHANGELOG.md" ]; then
  echo "release-notes: no existe CHANGELOG.md" >&2
  exit 1
fi

python3 - "$VERSION" <<'PY'
import re, sys

version = sys.argv[1]
# Aceptar tanto "v0.3.0" como "0.3.0" en el encabezado del changelog.
bare = version[1:] if version.startswith("v") else version

with open("CHANGELOG.md") as fh:
    text = fh.read()

def section(title):
    # Cuerpo entre "## [title]" y el siguiente "## " (o el fin del archivo).
    pattern = rf"^## \[{re.escape(title)}\][^\n]*\n(.*?)(?=^## |\Z)"
    m = re.search(pattern, text, re.MULTILINE | re.DOTALL)
    return m.group(1).strip() if m else None

body = section(version) or section(bare) or section("Unreleased")

if not body:
    # Cubre dos casos que conviene no confundir: sección ausente y sección
    # presente pero sin cuerpo (lo normal al taggear sin correr sync-changelog).
    print(
        f"release-notes: sin notas para {version}. No hay sección '## [{bare}]' "
        "y '## [Unreleased]' está ausente o vacía. Vuelca las features done con "
        "./sync-changelog.sh antes de taggear.",
        file=sys.stderr,
    )
    sys.exit(1)

print(body)
PY
