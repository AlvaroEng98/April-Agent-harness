#!/usr/bin/env bash
# session_start_recap.sh — Wrapper delgado del hook SessionStart de Claude Code.
#
# Delega toda la lógica en recap.sh (fuente única de verdad) y reenvía su
# stdout. Claude Code inyecta ese stdout como contexto adicional al iniciar
# la sesión (no requiere envolver la salida en JSON para el caso de solo
# cargar contexto).
"$CLAUDE_PROJECT_DIR/recap.sh"
