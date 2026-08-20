# CHECKPOINTS — Evaluación del estado final

> En sistemas multi-agente no se evalúa el camino, se evalúa el destino.
> El orquestador recorre esta lista antes de cerrar una sesión (paso
> "Cerrar sesión" en `.claude/agents/orquestador.md`) y marca cada casilla
> `[x]`/`[ ]`. Si queda una casilla vacía en C1-C5, la sesión no cierra.

## C1 — El arnés está completo

- [ ] Existen `AGENTS.md`, `init.sh`, `feature_list.json`,
      `progress/current.md`.
- [ ] Existen `docs/architecture.md`, `docs/conventions.md` y
      `docs/verification.md`, sin secciones `_pendiente_` donde el humano
      ya respondió.
- [ ] `./init.sh` termina con exit code 0.

## C2 — El estado es coherente

- [ ] Se cumplen las `rules` de `feature_list.json` (`one_feature_at_a_time`,
      `require_approved_spec_to_implement`, `require_tests_to_close`,
      `human_approval_required_to_close`).
- [ ] Toda feature `done` tiene evidencia de tests corridos, visible en el
      reporte de `agent_developer` que quedó en `progress/history.md`.
- [ ] `progress/current.md` describe la feature activa o dice
      explícitamente que no hay ninguna — nada de sesiones anteriores sin
      limpiar.

## C3 — El código respeta la arquitectura

- [ ] El árbol de código solo contiene los módulos previstos en
      `docs/architecture.md`.
- [ ] Las dependencias externas declaradas coinciden con lo permitido en
      `docs/conventions.md`.
- [ ] No hay `TODO` sin feature asociada en `feature_list.json`, ni prints
      de debug sueltos.

## C4 — La verificación es real

- [ ] Cada módulo de código tiene al menos un test, según el mapeo de
      `docs/verification.md`.
- [ ] El comando de verificación documentado en `docs/verification.md`
      corre limpio (0 fallos) y reporta más de 0 tests.

## C5 — La sesión se cerró bien

- [ ] No hay archivos sin trackear fuera de `.gitignore`.
- [ ] `progress/history.md` tiene una entrada nueva para esta sesión.
- [ ] La feature trabajada quedó en el `status` que le corresponde en
      `feature_list.json` — nunca `in_progress` al cerrar.
