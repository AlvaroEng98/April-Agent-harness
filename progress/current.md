# Current Session

## Feature in progress
- ID: 4
- Name: rename_cli_to_apil
- Status: in_progress

Feature en curso: 4 — rename_cli_to_apil (modo F2)
Plan: progress/plan_rename_cli_to_apil.md

## Plan
Renombrar la invocación del CLI de `harness` a `apil` en 3 sitios:
printUsage() en main.go, project_name/binary en .goreleaser.yaml, y las
menciones de invocación de comando en README.md (tabla, ejemplos, smoke
test). No se toca el switch de comandos ni go.mod. Ver
progress/plan_rename_cli_to_apil.md.

## Progress Log
- 2026-07-19 — move_recap_to_hooks (feature 3) aprobada por humano y marcada
  done. Entrada en history.md.
