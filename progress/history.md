# Session History

<!-- Append-only log. Most recent at the top. -->

## 2026-08-25 — Feature 2 `scaffold_manifest_sync` (done)

`/improve-codebase-architecture` detectó que `april init` sobreescribía
siempre `feature_list.json`/`progress/*.md` en un segundo `init` (solo
`.claude/agents/` se limpiaba con `RemoveAll`, todo lo demás se pisaba sin
condición). Se diseñó vía plan mode un manifiesto `.claude/manifest.json`
(sha256 por archivo, patrón "last-applied-configuration" tipo `kubectl
apply`) que distingue archivo por archivo: nuevo → crea; no tocado por el
usuario → actualiza con la plantilla nueva; tocado y plantilla sin cambios →
deja intacto en silencio; tocado y plantilla también cambió (conflicto
real) → deja intacto y avisa; obsoleto en la plantilla nueva → borra solo
si el usuario no lo tocó. Manifiesto ausente o corrupto → modo adopción (no
toca nada existente, solo adopta hashes de línea base).

`classifyExistingEntries`/`isExistingHarness`/`agentDirToClean` eliminados
por completo — la limpieza de agentes ya no es un caso especial, es una
instancia más de la regla general. `feature_list.json`/`progress/*.md`
quedan protegidos sin ningún `if relPath == "..."` hardcodeado: en cuanto
divergen del hash registrado, la regla general los protege sola.

`agent_developer` implementó en 6 pasos incrementales (`go build`/`go test`
verde en cada uno). `reviewer_agent` primera pasada: `APPROVED_WITH_OBJECTION`
(el criterio de "sobrevive sin caso especial" solo tenía test para
`feature_list.json` en raíz, no para `progress/*.md` en subdirectorio).
Corregido con `TestProgressHistoryMdProtegidoEnConflictoRealSinCasoEspecial`.
Segunda pasada: `APPROVED` sin objeciones. Humano aprobó cierre.

Archivos tocados: `scaffold.go`, `scaffold_test.go`, `.gitignore` (raíz,
excluye `/.claude/manifest.json` para no propagar el manifiesto del propio
repo vía dogfooding).
