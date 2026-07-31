# Architecture — april-harness

> Define qué significa "hacer un buen trabajo" en este proyecto.

## Principles

1. **Separación de responsabilidades** — Cada módulo hace una cosa bien.
2. **Testabilidad** — Todo código crítico debe ser testeable aisladamente.
3. **Simplicidad** — Preferir soluciones simples sobre soluciones clever.
4. **Consistencia** — Seguir convenciones existentes del proyecto.

## Layer Structure

```
┌─────────────────────────────────────────────────────────┐
│                    CLI Layer (main.go)                   │
│  - Parseo de argumentos                                 │
│  - Dispatch de comandos                                 │
└─────────────────────────────────────────────────────────┘
                           │
┌─────────────────────────────────────────────────────────┐
│                  Detection Layer (detector.go)           │
│  - Detección de herramientas AI instaladas              │
│  - Lógica pura, sin dependencias de UI                  │
└─────────────────────────────────────────────────────────┘
                           │
┌─────────────────────────────────────────────────────────┐
│                    UI Layer (selector.go)                │
│  - Selección interactiva de herramientas                │
│  - Renderizado de terminal                              │
└─────────────────────────────────────────────────────────┘
                           │
┌─────────────────────────────────────────────────────────┐
│                  Template Layer (embed.FS)               │
│  - Archivos embebidos del harness                       │
│  - Copia y transformación de templates                  │
└─────────────────────────────────────────────────────────┘
```

## Dependency Rules

- **UI Layer** puede dependeer de Detection y Template.
- **Detection Layer** NO puede depender de UI.
- **Template Layer** NO puede depender de UI ni Detection.
- **CLI Layer** orquesta las demás capas.

## Module Map

| Módulo        | Archivo         | Responsabilidad                           |
|---------------|-----------------|-------------------------------------------|
| CLI           | main.go         | Entry point, parseo de args, dispatch     |
| Detection     | detector.go     | Detectar herramientas AI instaladas       |
| UI            | selector.go     | Interacción interactiva con usuario       |
| Template      | embed.FS        | Copiar archivos del harness al destino    |
| Config        | config.go       | Versión, rutas, archivos requeridos       |

## Data Flow

```
usuario → main.go → detector.go → selector.go → embed.FS → destino/
```

## Error Handling

- Errores de IO → mostrar en stderr y salir con código != 0
- Errores de usuario → mostrar advertencia y continuar
- Errores de compilación → mostrar detalle y salir