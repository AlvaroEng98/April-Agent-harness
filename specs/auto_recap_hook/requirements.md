# Requirements — auto_recap_hook

> Fuente: `feature_list.json`, feature ID 2, `acceptance` (7 criterios). Cada
> criterio original está cubierto por uno o más `R<n>` de abajo — ver tabla de
> cobertura al final.

## R1

EL sistema DEBE proveer un único script `recap.sh` en la raíz del proyecto
que contenga toda la lógica de recap (última entrada de
`progress/history.md`, feature no-`done` de `feature_list.json`, sesión
activa de `progress/current.md`), de modo que sea el único punto de verdad
reutilizable por cualquier mecanismo que necesite este contexto.

## R2

CUANDO se ejecuta `recap.sh` y `progress/history.md` existe y contiene al
menos una línea que empieza con `## `, el sistema DEBE imprimir en stdout
una línea `Última sesión: <texto de esa entrada sin el prefijo "## ">`.

## R3

CUANDO se ejecuta `recap.sh` y existe en `feature_list.json` al menos una
feature con `status` distinto de `done`, el sistema DEBE imprimir en stdout
una línea `Feature actual: <title> (<status>)` correspondiente a la feature
no-`done` de menor `id`.

## R4

SI todas las features en `feature_list.json` tienen `status` igual a
`done` ENTONCES `recap.sh` DEBE imprimir en stdout la línea
`Feature actual: Todas las features completadas`.

## R5

CUANDO se ejecuta `recap.sh` y `progress/current.md` tiene una línea
`- Name:` con un valor no vacío, el sistema DEBE imprimir en stdout una
línea `Sesión activa: <name> (<status>)`.

## R6

SI `progress/current.md` no tiene sesión activa (línea `- Name:` ausente o
con valor vacío) ENTONCES `recap.sh` DEBE imprimir en stdout la línea
`No hay sesión activa`.

## R7

EL sistema DEBE incluir en el template embebido el archivo
`.claude/settings.json` con un hook `SessionStart` (`type: "command"`) que
invoque un script de recap y cuya salida se use como contexto adicional
inyectado al iniciar una sesión de Claude Code.

## R8

EL sistema DEBE incluir en el template embebido el script
`.claude/hooks/session_start_recap.sh`, invocado por el hook `SessionStart`
de `.claude/settings.json`, que ejecute `recap.sh` y reenvíe su salida sin
reimplementar la lógica de recap.

## R9

EL sistema DEBE incluir en el template embebido el archivo
`.opencode/plugins/recap.js`, un plugin nativo de OpenCode que use el hook
`experimental.chat.system.transform` para inyectar el resultado de
`recap.sh` como contexto adicional del sistema en la primera interacción de
cada sesión, sin reimplementar la lógica de recap.

## R10

CUANDO el usuario ejecuta `harness init` seleccionando Claude Code, el
sistema DEBE copiar `.claude/settings.json` y
`.claude/hooks/session_start_recap.sh` al proyecto destino.

## R11

SI el usuario ejecuta `harness init` sin seleccionar Claude Code ENTONCES
el sistema NO DEBE copiar `.claude/settings.json` ni
`.claude/hooks/session_start_recap.sh` al proyecto destino.

## R12

CUANDO el usuario ejecuta `harness init` seleccionando OpenCode, el sistema
DEBE copiar `.opencode/plugins/recap.js` al proyecto destino.

## R13

SI el usuario ejecuta `harness init` sin seleccionar OpenCode ENTONCES el
sistema NO DEBE copiar `.opencode/plugins/recap.js` al proyecto destino.

## R14

CUANDO el usuario ejecuta `harness init` con cualquier combinación de
herramientas, el sistema DEBE copiar `recap.sh` a la raíz del proyecto
destino con permisos de ejecución (modo `0755`), independientemente de las
herramientas elegidas.

## R15

EL sistema DEBE modificar la sección 5 de `init.sh` para invocar
`recap.sh` en lugar de reimplementar el grep de `progress/history.md`, el
parseo Python de `feature_list.json` y el grep de `progress/current.md`,
preservando el mismo formato de salida (`[OK]` por línea) que la versión
actual.

## R16

SI `recap.sh` falla o no existe ENTONCES `init.sh` DEBE continuar su
ejecución sin convertir la sección 5 en un `[FAIL]` bloqueante, porque el
recap es informativo y no crítico para el resto de la verificación de
entorno.

## R17

EL sistema DEBE incluir tests automatizados en Go (`recap_test.go`) que
ejecuten `recap.sh` contra fixtures temporales de `progress/history.md`,
`feature_list.json` y `progress/current.md`, y verifiquen las tres líneas
de salida (última sesión, feature actual, sesión activa), incluyendo los
casos borde de R4 y R6.

## R18

EL sistema DEBE agregar `recap.sh` y el directorio `.opencode` a la
directiva `//go:embed` de `main.go` para que ambos se incluyan en el
binario compilado de `harness`.

---

## Cobertura (acceptance → R<n>)

| # | Acceptance original (feature_list.json) | Cubierto por |
|---|---|---|
| 1 | Script/función de recap reutilizable, sin duplicar, invocado por ambos mecanismos | R1, R8, R9, R15 |
| 2 | Template incluye `.claude/settings.json` con hook `SessionStart` | R7, R10, R11 |
| 3 | Template incluye plugin OpenCode con mecanismo nativo | R9, R12, R13 |
| 4 | `main.go` embebe y copia respetando la selección de herramientas | R10, R11, R12, R13, R14, R18 |
| 5 | El recap muestra las 3 piezas de contexto | R2, R3, R4, R5, R6 |
| 6 | `init.sh` sigue funcionando igual, sin eliminar ni duplicar la sección 5 | R15, R16 |
| 7 | Tests cubren la lógica de recap compartida | R17 |
