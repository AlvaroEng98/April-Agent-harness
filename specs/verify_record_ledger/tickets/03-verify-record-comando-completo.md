# 03: `april verify record` — comando completo (subproceso real + hash + append + CLI)

**What to build:** el tracer-bullet completo y usable de esta feature.
Un usuario (agente o CI) corre
`april verify record --feature <id> -- <comando>`; `<comando>` se ejecuta
como subproceso real vía `exec.Command`, argv a argv, sin pasar por una
shell (sin pipes/redirects/`&&` implícitos — quien necesite eso lo pide
explícito con `-- sh -c "..."`). Mientras corre, su stdout/stderr se ven
en vivo en la terminal de quien invocó el comando, y a la vez quedan
capturados por separado (dos buffers) para la entrada del ledger.

Se distinguen dos clases de resultado: si el comando ni siquiera llegó a
arrancar (binario inexistente, permiso denegado), es un error de
invocación — no se escribe ninguna entrada al ledger; si el comando
arrancó y terminó con exit code distinto de cero, es una corrida válida
que sí se registra normalmente con ese exit code. Tras la corrida (en el
caso de que haya arrancado), se calcula `hashTree` sobre el árbol actual,
se arma la `ledgerEntry` (integrando los tipos y el mecanismo de append
atómico del ticket 2) y se anexa al ledger. El exit code del propio
proceso `april verify record` refleja el del comando corrido, para poder
encadenarlo en scripts (`april verify record ... || exit 1`).

El parseo de CLI exige `--feature <id>` (obligatorio, numérico) seguido de
`--` (obligatorio) y al menos un token de comando después — la ausencia de
cualquiera de los tres es un error explícito en stderr, exit distinto de
cero, sin tocar el ledger. Nuevo caso `"verify"` en el `switch` de
`main.go` (subcomando `"record"`; cualquier otro subcomando es error
explícito), y `printUsage()` documenta el comando nuevo.

**Blocked by:** 01 (`hashTree`), 02 (esquema y append atómico del ledger)

**Status:** done

- [ ] `april verify record --feature <id> -- <comando>` ejecuta
      `<comando>` directo vía `exec.Command` (sin `sh -c` implícito).
- [ ] Un comando exitoso (ej. `sh -c "exit 0"`) queda registrado en el
      ledger real en disco con `exitCode == 0`, `featureId` correcto y
      `treeHash` no vacío.
- [ ] Un comando fallido (ej. `sh -c "exit 3"`) queda registrado con
      `exitCode == 3` real, no un booleano genérico de "falló".
- [ ] `stdout` y `stderr` del comando quedan capturados por separado en la
      entrada del ledger, con el contenido exacto emitido por el comando.
- [ ] Un comando cuyo binario no existe/no arranca devuelve error de
      invocación y NO escribe ninguna entrada al ledger (el archivo no se
      crea, o si ya existía, queda con el mismo contenido antes/después).
- [ ] Dos corridas sucesivas sobre el mismo `featureId` producen dos
      líneas en el ledger, ambas parseables, ninguna sobrescribe a la
      otra.
- [ ] Invocar sin `--feature` es error de invocación explícito en stderr,
      exit distinto de cero, sin tocar el ledger.
- [ ] Invocar sin el separador `--` es error de invocación explícito, sin
      tocar el ledger.
- [ ] Invocar con `--` pero sin ningún comando después es error de
      invocación explícito, sin tocar el ledger.
- [ ] El exit code del propio proceso `april verify record` es el mismo
      que el del comando corrido (comando exitoso → exit 0 del proceso;
      comando con exit 3 → exit 3 del proceso).
- [ ] `printUsage()` documenta `verify record --feature <id> -- <comando>`.
- [ ] `go build ./...` y `go test ./...` en verde.
