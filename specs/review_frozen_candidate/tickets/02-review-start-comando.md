# 02: `april review start` — consulta pura del candidato congelado

**What to build:** `reviewer_agent` (o cualquier humano) corre
`april review start --feature <id>` y recibe en stdout, en una sola línea
sin texto decorativo alrededor, el `subject_hash` del árbol de trabajo tal
como está en ese momento — capturable directamente en una variable de
shell (`HASH=$(april review start --feature 7)`) sin parsear nada.

Es una consulta pura: no escribe nada al ledger ni a ningún otro archivo.
Correrlo varias veces seguidas sin cambiar nada en el árbol imprime
siempre el mismo hash. `--feature <id>` es obligatorio (mismo estilo
estricto de parseo que el resto de subcomandos de `review`/`verify`: falta
el flag, falta el valor, o el id no es numérico son errores de invocación
explícitos, exit≠0) aunque el id no participe del cálculo del hash — se
pide únicamente por consistencia y trazabilidad con el resto de la
familia de comandos.

Si el directorio no es un repositorio git (o `git` no está disponible),
el comando falla explícito en stderr con un mensaje que deja claro cuál
de los dos casos ocurrió, exit distinto de cero, sin devolver ningún hash
ni caer en un mecanismo alternativo silencioso. El comando se descubre
sin leer código: `printUsage()` documenta su sintaxis.

**Blocked by:** 01 (`computeSubjectHash`)

**Status:** done

- [ ] `april review start --feature <id>` en un repositorio git imprime
      exit 0 y una sola línea no vacía en stdout con el `subject_hash`
      (formato hex de SHA-1, 40 caracteres), sin texto decorativo
      alrededor.
- [ ] Correr el comando dos veces seguidas sin cambiar el árbol imprime
      el mismo hash ambas veces.
- [ ] El comando no crea ni modifica `.claude/verify-ledger.jsonl` (ni
      ningún otro archivo) — verificable comparando su estado antes y
      después de la corrida.
- [ ] Invocar sin `--feature` (o sin su valor, o con un id no numérico)
      es un error de invocación explícito, exit≠0, sin necesidad de un
      repositorio git para fallar (falla antes de llegar a
      `computeSubjectHash`).
- [ ] Fuera de un repositorio git, con `--feature` válido, el comando
      termina con exit≠0 y stderr menciona explícitamente que no es un
      repositorio git.
- [ ] `cmdReview()` reconoce `"start"` como subcomando válido junto a
      `"record"`.
- [ ] `printUsage()` documenta `review start --feature <id>`.
- [ ] `go build ./...` y `go test ./...` en verde.
