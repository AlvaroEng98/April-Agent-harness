#!/usr/bin/env bash
# init.sh — Verificación e inicialización del entorno
#
# Este script lo ejecuta el agente al COMENZAR una sesión y antes de
# declarar cualquier tarea como `done`. Si falla, la sesión no debe avanzar.
#
# Salida esperada: códigos de salida claros y bloques marcados con [OK]/[FAIL].

set -u
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

ok()    { printf "${GREEN}[OK]${NC}    %s\n" "$1"; }
warn()  { printf "${YELLOW}[WARN]${NC}  %s\n" "$1"; }
fail()  { printf "${RED}[FAIL]${NC}  %s\n" "$1"; }

EXIT_CODE=0

echo "── 1. Verificando archivos base del arnés ──────────────"

# feature_list.json es estado de desarrollo y NO está versionado (ver
# .gitignore), así que en un clone limpio no existe. Se siembra desde el
# template para que el arnés pueda arrancar sobre sí mismo. En un proyecto
# ya scaffoldeado no hay templates/, así que ahí sí es un error real.
if [ ! -f "feature_list.json" ]; then
  if [ -f "templates/feature_list.json" ]; then
    cp templates/feature_list.json feature_list.json
    warn "feature_list.json no existía — sembrado desde templates/"
  else
    fail "Falta feature_list.json y no hay templates/ para sembrarlo."
    fail "Ejecuta 'harness init .' para regenerarlo."
    EXIT_CODE=1
  fi
fi

BASE_FILES=(
  "AGENT.md"
  "feature_list.json"
  "progress/current.md"
  "docs/architecture.md"
  "docs/conventions.md"
  "docs/verification.md"
  "CHECKPOINTS.md"
)
for f in "${BASE_FILES[@]}"; do
  if [ ! -f "$f" ]; then
    fail "Falta archivo base: $f"
    EXIT_CODE=1
  else
    ok "Existe $f"
  fi
done

echo ""
echo "── 2. Validando feature_list.json y specs ─────────────"

python3 - <<'PY'
import json, os, sys
try:
    data = json.load(open("feature_list.json"))
    valid = {"pending", "spec_ready", "in_progress", "done", "blocked"}
    in_progress = [f for f in data["features"] if f["status"] == "in_progress"]
    if len(in_progress) > 1:
        print(f"[FAIL]  Hay {len(in_progress)} features en in_progress (máximo 1)")
        sys.exit(1)
    requires_spec = {"spec_ready", "in_progress", "done"}
    spec_errors = []
    for f in data["features"]:
        if f["status"] not in valid:
            print(f"[FAIL]  Estado inválido en feature {f['id']}: {f['status']}")
            sys.exit(1)
        if f.get("sdd") and f["status"] in requires_spec:
            spec_dir = os.path.join("specs", f["name"])
            for fname in ("requirements.md", "design.md", "tasks.md"):
                if not os.path.isfile(os.path.join(spec_dir, fname)):
                    spec_errors.append(
                        f"feature {f['id']} ({f['name']}) en {f['status']} "
                        f"sin {spec_dir}/{fname}"
                    )
    if spec_errors:
        for e in spec_errors:
            print(f"[FAIL]  {e}")
        sys.exit(1)
    print(f"[OK]    feature_list.json válido ({len(data['features'])} features)")
    print(f"[OK]    Specs presentes para features sdd con estado no-pending")
except SystemExit:
    raise
except Exception as e:
    print(f"[FAIL]  feature_list.json o specs inválidos: {e}")
    sys.exit(1)
PY

if [ $? -ne 0 ]; then EXIT_CODE=1; fi

echo ""
echo "── 3. Verificando agentes ─────────────────────────────"

if [ -d ".claude/agents" ]; then
  agent_count=0
  for agent_file in .claude/agents/*.md; do
    if [ -f "$agent_file" ]; then
      agent_name=$(basename "$agent_file")
      if grep -q "^#" "$agent_file" 2>/dev/null; then
        ok "Agente válido: $agent_name"
        agent_count=$((agent_count + 1))
      else
        fail "Agente sin cabeceras válidas: $agent_name"
        EXIT_CODE=1
      fi
    fi
  done
  if [ $agent_count -eq 0 ]; then
    warn "No se encontraron agentes en .claude/agents/"
  else
    ok "$agent_count agente(s) encontrado(s)"
  fi
else
  warn "Directorio .claude/agents/ no existe (¿proyecto sin harness?)"
fi

echo ""
echo "── 4. Verificando compilación Go ───────────────────────"

if [ -f "go.mod" ]; then
  if command -v go &>/dev/null; then
    if go build -o /dev/null . 2>/dev/null; then
      ok "Código Go compila correctamente"
    else
      fail "Error al compilar código Go"
      EXIT_CODE=1
    fi
  else
    warn "Go no está instalado — saltando verificación de compilación"
  fi
else
  warn "No se encontró go.mod — proyecto no es Go"
fi

echo ""
echo "── 5. Recap — estado del proyecto ──────────────────────"

# Delega en recap.sh (fuente única de verdad). Cada línea no vacía de su
# salida se muestra con ok(). El recap es informativo: si falta o no es
# ejecutable, se advierte pero no se aborta la sección (no es bloqueante).
if [ -x "./recap.sh" ]; then
  while IFS= read -r line; do
    [ -n "$line" ] && ok "$line"
  done <<< "$(./recap.sh 2>/dev/null)"
else
  warn "recap.sh no encontrado o sin permisos de ejecución"
fi

echo ""
echo "── 6. Resumen ──────────────────────────────────────────"

if [ $EXIT_CODE -eq 0 ]; then
  ok "Entorno listo. Puedes empezar a trabajar."
else
  fail "Entorno NO está listo. Resuelve los errores antes de avanzar."
fi

exit $EXIT_CODE
