# Tasks — centralize_config

- [x] T1 — Crear `config.go` con `var version = "0.1.0"` y `var RequiredFiles = []string{...}` conteniendo los 7 archivos actuales de `init.sh`. Cubre: R1, R2, R3, R7, R8.
- [x] T2 — Eliminar `var version = "0.1.0"` de `main.go:17`. Cubre: R5.
- [x] T3 — Crear `gen_required.go` con función `main()` que lee `RequiredFiles` y escribe `required_files.txt` (uno por línea). Cubre: R9.
- [x] T4 — Añadir `//go:generate go run gen_required.go` en `config.go`. Cubre: R9.
- [x] T5 — Ejecutar `go generate ./...` para crear `required_files.txt` inicial. Cubre: R4.
- [x] T6 — Modificar `init.sh` para leer la lista de archivos desde `required_files.txt` en lugar del loop hardcodeado (líneas 23-30). Cubre: R4, R6.
- [x] T7 — Ejecutar `go build .` y verificar que compila sin errores. Cubre: R7.
- [x] T8 — Ejecutar `./init.sh` y verificar que la verificación de archivos pasa correctamente. Cubre: R4.
- [x] T9 — Verificar que `go generate ./...` regenera `required_files.txt` correctamente después de un cambio en `RequiredFiles`. Cubre: R9.
