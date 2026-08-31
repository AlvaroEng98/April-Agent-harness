# 01: Núcleo de `april status` — fase, selección de feature, blockedReasons básicos y CLI

**What to build:** un comando nuevo, `april status [id] [--json]`, que lee
el disco (`feature_list.json`, `specs/<name>/spec.md`,
`specs/<name>/tickets/*.md`) y devuelve, en JSON (`--json`) o en texto
plano legible (sin la flag), un reporte con cinco campos: `phase`,
`nextRecommended`, `blockedReasons`, `frontier` y `artifactPaths`.

Sin argumento, el comando elige solo la feature activa: la única
`in_progress` si existe, o si no la `pending` de menor `id`; si hay más de
una `in_progress` (estado inconsistente) elige de forma determinística la
de menor `id` entre ellas, pero igual queda reportado en
`blockedReasons`. Si no queda ninguna feature `pending` ni `in_progress`,
no hay target: `phase: "closed"`, `frontier: []` y `nextRecommended`
explica que no queda nada pendiente. Con un `id` explícito, inspecciona
esa feature aunque no sea la activa; si el `id` no existe en
`feature_list.json`, el comando termina con exit code distinto de cero y
un mensaje claro en stderr, sin imprimir JSON.

`phase` se deriva combinando `sdd`, `status` y lo que exista en disco
(spec presente/ausente, tickets presentes/ausentes, `Status` de cada
ticket) según la tabla de siete casos de la spec: `closed` (feature
`done`, siempre, sin importar disco), `grill` (la feature
`bootstrap_project` mientras no esté `done`), `spec` (`sdd:true` sin
`spec.md`), `tickets` (`sdd:true` con spec pero sin ningún archivo en
`specs/<name>/tickets/`), `implementation` (`sdd:true` con tickets y al
menos uno con `Status != done`, o `sdd:false` no-bootstrap con `status`
`pending`/`in_progress`), y `review` (`sdd:true` con tickets, todos en
`Status: done`, pero la feature misma todavía no `done`).

`blockedReasons` es un array de strings calculado siempre sobre **todo**
`feature_list.json` y todo `specs/`, sin importar qué `id` se pidió, y en
este ticket cubre: más de una feature `in_progress`; `status` de alguna
feature fuera de `feature_list.json.rules.valid_status`; feature
`sdd:true` con `status` en `spec_ready`/`in_progress`/`done` sin
`specs/<name>/spec.md`; feature marcada `blocked` (se reporta con su
`id`/`name`, sin impedir calcular `phase` para las demás); y `Status` de
un ticket fuera del vocabulario `pending`/`in_progress`/`done`.
`nextRecommended` es una cadena única, vacía si y solo si `blockedReasons`
no está vacío — nunca hay recomendación de avanzar mientras haya un
problema sin resolver; con `blockedReasons` vacío describe la única
acción legal según `phase` (lanzar `spec_writer`, lanzar `ticket_writer`,
implementar la frontera, lanzar `reviewer_agent`, o "nada — ya está
cerrada"/"nada — no hay pendientes"). `artifactPaths` siempre incluye
`featureList`; si `sdd:true` incluye la ruta esperada de `spec.md` (exista
o no); si hay tickets, incluye el directorio y cada archivo encontrado.

El comando nunca escribe nada: toda la lectura pasa por `fs.ReadFile`/
`fs.ReadDir` sobre un `fs.FS` inyectable (`computeStatusFromFS(fsys
fs.FS, targetID *int)`), con `computeStatus(targetID *int)` como wrapper
delgado sobre `os.DirFS(".")` — mismo patrón que
`planScaffoldFromFS`/`planScaffold` en `scaffold.go`. Exit code `0` si
`blockedReasons` está vacío, `1` si no, con o sin `--json`. `frontier`
puede devolverse vacío en este ticket (su cómputo completo, sobre
`Blocked by`, es el ticket 2) — ningún criterio de este ticket depende de
`Blocked by`.

**Blocked by:** None (can start immediately)

**Status:** done

- [ ] `april status --json` existe (nuevo caso en `main.go`) y devuelve un
      único objeto JSON válido a stdout con los cinco campos.
- [ ] Sin `--json`, `april status` imprime los mismos datos en texto
      plano legible.
- [ ] Feature `sdd:true` sin `specs/<name>/spec.md` ⇒ `phase: "spec"`.
- [ ] Feature `sdd:true` con spec pero sin archivos en
      `specs/<name>/tickets/` ⇒ `phase: "tickets"`.
- [ ] Feature `sdd:true` con tickets y al menos uno con `Status != done`
      ⇒ `phase: "implementation"`.
- [ ] Feature `sdd:true` con todos los tickets en `Status: done` pero la
      feature misma no `done` ⇒ `phase: "review"`.
- [ ] Feature con `status: "done"` ⇒ `phase: "closed"`, sin importar qué
      haya en `specs/`.
- [ ] Feature `bootstrap_project` mientras no esté `done` ⇒
      `phase: "grill"`.
- [ ] Feature `sdd:false` (no bootstrap) con `status` `pending`/
      `in_progress` ⇒ `phase: "implementation"`.
- [ ] Dos features en `in_progress` ⇒ `blockedReasons` no vacío y
      `nextRecommended` vacío.
- [ ] `status` de alguna feature fuera de `rules.valid_status` ⇒
      reportado en `blockedReasons`.
- [ ] Feature `sdd:true` con `status` que requiere spec pero sin
      `specs/<name>/spec.md` ⇒ reportado en `blockedReasons`.
- [ ] Feature marcada `blocked` ⇒ reportada explícitamente en
      `blockedReasons` con su motivo.
- [ ] `Status` de un ticket fuera de `pending`/`in_progress`/`done` ⇒
      reportado en `blockedReasons`.
- [ ] `april status <id> --json` sobre un `id` que no es la feature activa
      inspecciona esa feature sin cambiar el foco actual.
- [ ] `april status <id>` con un `id` inexistente en `feature_list.json`
      termina con exit code distinto de 0 y mensaje claro en stderr, sin
      imprimir JSON.
- [ ] Backlog sin ninguna feature `pending` ni `in_progress` (todo
      `done`/`blocked`) ⇒ `phase: "closed"`, `frontier: []` y
      `nextRecommended` describe explícitamente que no hay nada
      pendiente, sin error.
- [ ] Exit code de `april status --json` es `0` cuando `blockedReasons`
      está vacío y distinto de `0` cuando no lo está.
- [ ] El comando no llama nunca a `os.WriteFile`/`os.Remove`/
      `os.MkdirAll` (hashes del árbol antes/después idénticos).
- [ ] `go build ./...` y `go test ./...` en verde, incluyendo los tests
      unitarios sobre `computeStatusFromFS` con `fstest.MapFS` y los de
      integración sobre `cmdStatus` con `t.TempDir()`.
