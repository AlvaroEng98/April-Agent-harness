# Conventions — april-harness

> Reglas de estilo, nombres y estructura del código Go.

## Naming

- **Paquetes**: nombres cortos, en minúsculas, sin guiones (`detector`, `selector`)
- **Funciones**: CamelCase para exportadas, camelCase para privadas
- **Variables**: camelCase para locales, PascalCase para globales exportadas
- **Constantes**: PascalCase para exportadas, camelCase para privadas
- **Archivos**: snake_case.go para archivos nuevos

## File Structure

```
april-harness/
├── main.go           # Entry point, CLI dispatch
├── detector.go       # Detección de herramientas AI
├── selector.go       # UI interactiva
├── config.go         # Configuración centralizada (futuro)
├── go.mod            # Dependencias
├── go.sum            # Lock de dependencias
└── *_test.go         # Tests al lado del código que testean
```

## Error Handling

- Usar `fmt.Fprintf(os.Stderr, ...)` para errores
- Retornar `error` en funciones que pueden fallar
- No panic! usar `os.Exit(1)` solo en `main()`
- Wrapping: `fmt.Errorf("contexto: %w", err)`

## Comments

- Comentarios en español para documentación de usuario
- Comentarios en inglés para docstrings de código
- No comentar código obvio — el código debe ser autoexplicativo

## Testing

- Archivos `*_test.go` al lado del código que testean
- Funciones `TestNombreFuncion` (PascalCase)
- Usar `t.Helper()` en funciones de setup
- Tab tests para casos múltiples

## Git

- Commits en inglés: `feat:`, `fix:`, `docs:`, `refactor:`
- Un concepto por commit
- No commitear binarios ni archivos temporales

## Go Style

- `gofmt` es obligatorio
- Imports agrupados: stdlib, externos, internos
- No imports sin usar
- Error always checked (no `_` para errors)