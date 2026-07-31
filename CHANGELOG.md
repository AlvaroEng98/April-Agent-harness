# Changelog

Registro de lo entregado. Este archivo es **producto** y va versionado.

El backlog vivo está en `feature_list.json`, que **no** se versiona (es estado
de desarrollo). El puente entre ambos es `./sync-changelog.sh`: cuando una
feature se aprueba como `done`, su entrada se vuelca aquí. Los releases leen
este archivo, nunca `feature_list.json`.

Formato basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/).

## [Unreleased]
- Centralize Configuration (`centralize_config`)
- Recap Automático al Iniciar Sesión (Claude + OpenCode) (`auto_recap_hook`)
