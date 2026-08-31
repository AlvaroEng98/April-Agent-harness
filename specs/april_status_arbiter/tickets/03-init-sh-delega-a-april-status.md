# 03: `init.sh` delega la validación a `april status`

**What to build:** al correr `./init.sh`, la sección "2. Validando
feature_list.json y specs" deja de usar el heredoc `python3 - <<'PY'
... PY` embebido y en su lugar resuelve el binario `april` con esta
prioridad: (1) `april` en el PATH — caso normal de un proyecto ya
scaffoldeado, que no tiene el código fuente de April, solo el binario
instalado; (2) si no está en el PATH pero existen `go.mod` y `main.go` en
el directorio actual (el caso de este mismo repo dogfoodeando su propio
arnés, donde puede no haber binario compilado todavía), lo compila
on-the-fly (`go build`) a un binario temporal y usa ese. Si ninguna de las
dos condiciones se cumple, `init.sh` falla explícitamente indicando que no
pudo resolver el comando `status`, en vez de fallar de forma confusa más
adelante.

Con el binario resuelto, `init.sh` corre `april status --json`, imprime su
salida, y traduce el exit code al `EXIT_CODE` que ya maneja el resto del
script — mismo patrón `[OK]`/`[FAIL]` que las demás secciones: exit 0 de
`april status --json` mantiene la sección en verde, exit distinto de 0 la
marca en rojo y falla el script completo, sin necesidad de parsear el JSON
en Bash.

**Blocked by:** 01 (Núcleo de `april status` — fase, selección de
feature, blockedReasons básicos y CLI)

**Status:** done

- [ ] `init.sh` ya no contiene el heredoc `python3 - <<'PY' ... PY` de
      validación.
- [ ] `init.sh` invoca `april status --json` en su lugar, dentro de la
      misma sección "2. Validando feature_list.json y specs".
- [ ] `./init.sh` corrido en este repo (con `go.mod`/`main.go` locales,
      sin binario `april` compilado todavía) resuelve el comando vía
      `go build` on-the-fly y termina en verde sobre un `feature_list.json`
      válido.
- [ ] `./init.sh` corrido en un proyecto scaffoldeado (solo el binario
      `april` en el PATH, sin código fuente Go) también resuelve y corre
      `april status --json` correctamente.
- [ ] Si no se puede resolver ni PATH ni build on-the-fly, `init.sh` falla
      explícitamente señalando que no pudo resolver el comando `status`,
      en vez de un error genérico posterior.
- [ ] El exit code de `april status --json` se traduce correctamente al
      `EXIT_CODE`/`[OK]`/`[FAIL]` de `init.sh` (un `blockedReasons` no
      vacío hace fallar la sección).
- [ ] `TestInitShInvocaAprilStatusSinHeredocPython` (lee el `init.sh` real
      del repo y verifica por contenido: no contiene `<<'PY'`, sí contiene
      `status`) pasa.
- [ ] `go build ./...` y `go test ./...` en verde.
