# Design — centralize_config

## Archivos afectados

| Archivo            | Acción       | Descripción                                              |
|--------------------|--------------|----------------------------------------------------------|
| `config.go`        | **Crear**    | Fuente única de verdad para `version` y `RequiredFiles`. |
| `main.go`          | **Modificar**| Eliminar `var version = "0.1.0"` (línea 17).            |
| `init.sh`          | **Modificar**| Reemplazar lista hardcodeada por archivo generado.       |

## Decisiones técnicas

### D1: `config.go` como paquete `main`

`config.go` será un archivo dentro del paquete `main` con:

```go
package main

var version = "0.1.0"

var RequiredFiles = []string{
    "AGENT.md",
    "feature_list.json",
    "progress/current.md",
    "docs/architecture.md",
    "docs/conventions.md",
    "docs/verification.md",
    "CHECKPOINTS.md",
}
```

Esto mantiene `version` como variable de paquete accedible desde `main.go` y `selector.go` (que ya usan `version` en `printBanner()`).

### D2: Generación de archivo para `init.sh`

**Opción elegida:** Usar `go generate` con un小小的 generador que escribe `required_files.txt` desde `RequiredFiles`.

Se añadirá un archivo `gen_required.go` (build tag `generate`) con una `//go:generate` directive y una función `main()` que:
1. Importa o redeclara `RequiredFiles` (o lo lee de `config.go` vía análisis estático simple).
2. Escribe cada archivo del slice en `required_files.txt`, uno por línea.

En `config.go` se añadirá la directiva:
```go
//go:generate go run gen_required.go
```

**Alternativa descartada — `embed` de `required_files.txt`:** Requeriría que el archivo exista antes de compilar, creando un orden de dependencia circular (el archivo se genera desde el código, pero el código necesita el archivo embebido). `go generate` es el patrón estándar de Go para esto.

### D3: Consumo en `init.sh`

`init.sh` cambiará su loop de verificación (línea 23-30) de:

```bash
for f in AGENT.md feature_list.json ...; do
```

a:

```bash
while IFS= read -r f; do
    # verificar $f
done < required_files.txt
```

### D4: No mover `version` a const

Se mantiene como `var` porque el patrón existente lo usa como variable (posible inyección futura vía `-ldflags`). Cambiarlo a `const` sería un cambio de contrato que esta feature no debe introducir.

## Puntos de contacto con architecture.md y conventions.md

- `architecture.md` está vacío (placeholder) → sin restricciones documentadas.
- `conventions.md` está vacío (placeholder) → sin convenciones existentes que conflicten.
- Se sigue el patrón de archivo único por responsabilidad: `config.go` para configuración centralizada.
