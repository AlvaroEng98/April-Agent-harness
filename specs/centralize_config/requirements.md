# Requirements — centralize_config

## R1

CUANDO se compila el binario `harness`, el sistema DEBE obtener el valor de `version` desde `config.go` en lugar de tenerlo hardcodeado en `main.go`.

## R2

EL sistema DEBE declarar un slice `RequiredFiles` en `config.go` que contenga la lista de archivos requeridos por `init.sh`.

## R3

EL sistema DEBE exponer `RequiredFiles` como un paquete-level variable exportado en `config.go`, accesible desde cualquier archivo del paquete `main`.

## R4

CUANDO `init.sh` ejecuta la verificación de archivos base, el sistema DEBE consumir la lista de archivos requeridos desde `config.go` (vía `go generate` o `embed`) en lugar de tenerla hardcodeada en el script.

## R5

EL sistema DEBE eliminar la línea `var version = "0.1.0"` de `main.go` después de que `config.go` provea el valor.

## R6

EL sistema DEBE eliminar la lista de archivos hardcodeada del loop `for f in ...` en `init.sh` línea 23, reemplazándola por la fuente generada desde `config.go`.

## R7

SI `config.go` no existe ENTONCES el sistema DEBE compilar sin errores porque `version` y `RequiredFiles` se definen en un archivo válido dentro del paquete `main`.

## R8

EL sistema DEBE mantener la semántica existente: `version` sigue siendo un `string` y `RequiredFiles` sigue siendo un `[]string`, sin cambios de tipo.

## R9

EL sistema DEBE generar un archivo `required_files.txt` (o equivalente) que `init.sh` pueda leer, vía `go generate` o `embed`, de modo que el script bash no contenga la lista duplicada.
