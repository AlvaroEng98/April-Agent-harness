# Arquitectura — Qué significa "hacer un buen trabajo"

> Este documento define el estándar de calidad. Los agentes revisores
> evalúan código contra este archivo. Si no está aquí, no es un requisito.

## Principios

1. Que cada función ejecute una sola cosa.
    - Reducir un gran problema en pequeñas partes, aplicando el principio divide y vencerás.

2. Aplicar siempre la metodología de trabajo definida en las fases iniciales, separando todo por módulos, cada uno con sus apartados correspondientes.
    *(La fase Grill rellena aquí un ejemplo concreto del stack y patrón de este proyecto.)*

3. Siempre tener una sección para verificar las variables de entorno. Nunca asignar valores por defecto a una variable de entorno: si falta, el sistema debe fallar explícitamente al arrancar, no seguir con un valor supuesto.

4. Revisar siempre que no se inserten datos comprometedores directamente en el código; siempre cargarlos desde el fichero correspondiente (`.env`, secrets manager, etc.).