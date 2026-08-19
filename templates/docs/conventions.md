# Convenciones de código

> Homogeneidad extrema. La IA predice mejor cuando el repositorio se parece
> a sí mismo en todas partes.
>
> La fase Grill rellena cada sección con las respuestas reales de este
> proyecto. Ninguna sección de abajo se puede dejar sin preguntar — si el
> humano no la responde, queda `_pendiente_`, nunca se omite. El ejemplo en
> Python sustitúyelo por el stack elegido.

## Estilo

- **Lenguaje y versión:** p. ej. Python 3.9+ (sintaxis `list[str]` permitida).
- **Formato:** p. ej. PEP 8, líneas máximo 100 caracteres.
- **Imports:** orden (stdlib primero, luego locales), una línea por módulo.
- **Strings:** comillas a usar y cuándo alternar.
- **Interpolación:** qué mecanismo usar (f-strings) y qué evitar (`.format()`, `%`).

## Nombres

| Tipo                    | Convención        | Ejemplo               |
|-------------------------|-------------------|-----------------------|
| Módulos                 | `snake_case`      | `notes.py`            |
| Clases                  | `PascalCase`      | `Note`                |
| Funciones / variables   | `snake_case`      | `load_notes`          |
| Constantes              | `UPPER_SNAKE`     | `DEFAULT_NOTES_PATH`  |
| Privadas                | prefijo `_`       | `_atomic_write`       |

## Estructura de archivo

Cómo empieza cada archivo fuente (docstring de módulo, orden de imports,
qué va primero). Ejemplo (Python):

```python
"""Una línea describiendo el propósito del módulo."""
from __future__ import annotations

# imports stdlib
import json
import os

# imports locales
from src.notes import Note
```

## Tests

- Convención de un archivo de test por módulo (p. ej. `tests/test_<módulo>.py`).
- Convención de agrupación (una clase o función por unidad lógica).
- Qué se limpia tras cada test y con qué mecanismo (p. ej. `tempfile.TemporaryDirectory()`).
- Convención de nombres de test descriptivos: `test_load_returns_empty_when_file_missing`.

## Manejo de errores

Dónde viven las excepciones del dominio y cómo se nombran. Ejemplo (Python):

```python
class NoteError(Exception):
    """Base para errores del dominio."""

class NoteNotFound(NoteError):
    """Se lanza cuando se busca una nota inexistente."""
```

Qué hace la capa de entrada (CLI/API) al capturar una excepción del dominio:
mensaje al usuario, código de salida, y si propaga o no el stack trace.

## Comentarios

Por defecto **no** se escriben. Solo se permiten cuando explican un *por qué*
no obvio (p. ej. workaround documentado, invariante sutil). Los nombres deben
hacer el resto.
