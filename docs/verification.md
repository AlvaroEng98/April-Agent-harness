# Verification — april-harness

> Cómo verificar que el trabajo funciona.

## Test Strategy

1. **Unit tests** — Funciones puras de detección y utilidades
2. **Integration tests** — CLI completo con argumentos mock
3. **Manual tests** — Ejecutar `./init.sh` y verificar output

## Running Tests

```bash
# Todos los tests
go test ./...

# Tests verbose
go test -v ./...

# Test específico
go test -run TestNombreFuncion ./...

# Con coverage
go test -cover ./...
```

## Coverage

- Mínimo aceptable: 60% para código nuevo
- Funciones críticas: 80%+ (detección, parsing)
- UI: coverage bajo es aceptable (difícil de testear)

## Traceability Map Format

Cada test debe mapearse a una funcionalidad:

```
TestDetectClaude → detector.go:DetectTools()
TestSelectInteractive → selector.go:selectInteractive()
TestMergeGitignore → main.go:mergeGitignore()
```

## Verification Steps

### Al implementar nueva feature:

1. `go build -o /dev/null .` — Compila sin errores
2. `go test ./...` — Todos los tests pasan
3. `./init.sh` — Verificación completa pasa
4. `go vet ./...` — Sin warnings

### Al cambiar docs/:

1. Verificar que agentes referencian el doc correcto
2. Ejecutar `./init.sh` para verificar que archivos existen

### Al cambiar init.sh:

1. Ejecutar `./init.sh` y verificar output esperado
2. Probar con proyecto sin harness (debería warn)
3. Proyecto con harness incompleto (debería fallar)

## Common Issues

| Problema | Causa | Solución |
|----------|-------|----------|
| `go build` falla | go.mod versión incorrecta | Verificar `go 1.22.0` |
| Tests fallan | Import cycle | Revisar estructura de paquetes |
| init.sh falla | Archivo faltante | Crear archivo o agregar a embed