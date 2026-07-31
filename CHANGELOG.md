# Changelog

Registro de lo entregado. Este archivo es **producto** y va versionado.

El backlog vivo está en `feature_list.json`, que **no** se versiona (es estado
de desarrollo). El puente entre ambos es `./sync-changelog.sh`: cuando una
feature se aprueba como `done`, su entrada se vuelca aquí. Los releases leen
este archivo, nunca `feature_list.json`.

Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/).

## [Unreleased]

## [0.3.1] - 2026-07-31
Release de infraestructura y documentación: no cambia el comportamiento del
scaffolding. Estas entradas no vienen de `feature_list.json` porque no son
features del backlog.

### Añadido
- `install.sh`: instalador vía `curl … | sh`. Detecta OS/arquitectura, resuelve
  la última release, verifica el checksum SHA-256 e instala **a nivel de
  usuario** en `~/.local/bin` (nunca usa `sudo`). Configurable con `VERSION` y
  `BIN_DIR`.
- `templates/README.md`: README del proyecto scaffoldeado, embebido vía
  `go:embed`.

### Cambiado
- README: la instalación recomendada pasa a ser el instalador; la descarga
  manual y la compilación desde fuente quedan como opciones 2 y 3, ya sin
  `sudo`.

### Corregido
- Workflow de release: las notas se capturan en una variable antes de abrir el
  here-doc de `$GITHUB_ENV`. Antes, si `release-notes.sh` fallaba, el
  delimitador de cierre no se escribía y el error real quedaba tapado por
  `Invalid value. Matching delimiter not found`.
- `release-notes.sh`: el mensaje de error distingue una sección ausente de una
  presente pero vacía, y apunta a `./sync-changelog.sh`.

## [0.3.0] - 2026-07-31
Primera release publicada desde este repositorio. Continúa la numeración de
la lineage anterior (`v0.2.10`), cuyo historial no está en este `main`.

- Centralize Configuration (`centralize_config`)
- Recap Automático al Iniciar Sesión (Claude + OpenCode) (`auto_recap_hook`)
