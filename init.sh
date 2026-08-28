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
for seed in feature_list.json; do
  if [ ! -f "$seed" ]; then
    if [ -f "templates/$seed" ]; then
      cp "templates/$seed" "$seed"
      warn "$seed no existía — sembrado desde templates/"
    else
      fail "Falta $seed y no hay templates/ para sembrarlo."
      fail "Ejecuta 'harness init .' para regenerarlo."
      EXIT_CODE=1
    fi
  fi
done

BASE_FILES=(
  "AGENTS.md"
  "feature_list.json"
  "progress/current.md"
  "docs/architecture.md"
  "docs/conventions.md"
  "docs/verification.md"
  "docs/specs.md"
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

# Resuelve el binario april con esta prioridad: (1) april en el PATH — caso
# normal de un proyecto scaffoldeado, que no tiene el código fuente de
# April, solo el binario instalado; (2) si no está en el PATH pero hay
# go.mod y main.go en el directorio actual (este mismo repo dogfoodeando su
# propio arnés, sin binario compilado todavía), se compila on-the-fly a un
# binario temporal. Si ninguna se cumple, falla explícitamente en vez de
# fallar de forma confusa más adelante.
APRIL_BIN=""
APRIL_TMP_BIN=""
if command -v april >/dev/null 2>&1; then
  APRIL_BIN="$(command -v april)"
elif [ -f "go.mod" ] && [ -f "main.go" ]; then
  APRIL_TMP_BIN="$(mktemp)"
  if go build -o "$APRIL_TMP_BIN" .; then
    APRIL_BIN="$APRIL_TMP_BIN"
  else
    fail "No se pudo compilar april on-the-fly (go build) para correr 'status'."
    EXIT_CODE=1
  fi
else
  fail "No se pudo resolver el comando 'status': ni 'april' está en el PATH ni hay go.mod/main.go en el directorio actual."
  EXIT_CODE=1
fi

if [ -n "$APRIL_BIN" ]; then
  "$APRIL_BIN" status --json
  status_exit=$?
  if [ $status_exit -eq 0 ]; then
    ok "april status --json sin blockedReasons"
  else
    fail "april status --json reportó blockedReasons (exit $status_exit)"
    EXIT_CODE=1
  fi
fi

[ -n "$APRIL_TMP_BIN" ] && rm -f "$APRIL_TMP_BIN"

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
echo "── 4. Resumen ──────────────────────────────────────────"

if [ $EXIT_CODE -eq 0 ]; then
  ok "Entorno listo. Puedes empezar a trabajar."
else
  fail "Entorno NO está listo. Resuelve los errores antes de avanzar."
fi

exit $EXIT_CODE
