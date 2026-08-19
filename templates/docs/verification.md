# Verificación — Cómo demostrar que el trabajo funciona

> Regla de oro: **el agente no dice "funciona", lo demuestra**.
> Toda feature termina con evidencia tangible, no con afirmaciones.

## Niveles de verificación

### Nivel 1 — Tests unitarios (obligatorio)

Toda función pública tiene al menos un test en que:

1. Cubre el camino feliz.
2. Cubre varios caminos de error si la función puede fallar.

### Nivel 2 — Test de integración del CLI (obligatorio para features de UI)

Las features que añaden comandos al CLI se verifican ejecutando el CLI real
contra un archivo temporal.

### Nivel 3 — Smoke test manual (opcional pero recomendado)

Antes de cerrar la sesión, ejecuta un flujo end-to-end con un archivo
temporal en `/tmp` y pega la salida real en `progress/current.md`. Ejemplo:

```bash
tmpfile=$(mktemp)
echo "..." > "$tmpfile"
./mi-cli comando "$tmpfile"   # pega aquí la salida real obtenida
```

## Cobertura

Mínimo exigido por C9 (`CHECKPOINTS.md`): 60% para código nuevo, 80%+ para
funciones críticas. Ajusta el número si tu proyecto exige otro umbral —
`CHECKPOINTS.md` y `reviewer_agent.md` citan el valor de esta sección.

## Anti-patrones (no hacer)

- ❌ "He añadido el comando, debería funcionar." → falta test ejecutable.
- ❌ Test que solo verifica que la función no lanza excepción. → tiene que
  comprobar el resultado concreto.
- ❌ Marcar la feature como `done` sin pasar `./init.sh`.

## Verificación final antes de cerrar

```bash
./init.sh           # debe terminar con [OK] Entorno listo
```

Si `./init.sh` está rojo, **no** marques nada como `done`. Anota el bloqueo
en `progress/current.md` con estado `blocked` en `feature_list.json`.