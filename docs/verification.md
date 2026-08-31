# Verification

> Rellenado por la skill `grill-docs` durante la Fase Grill de
> `bootstrap_project`. Sección sin respuesta del humano queda literalmente
> `_pendiente_` — nunca se omite, nunca se inventa una respuesta plausible.

## Niveles de verificación

- **Unitario (obligatorio).** `go test ./...` — cada módulo (`scaffold.go`,
  `update.go`, y los que agregue el `ROADMAP.md`) tiene su `_test.go`
  gemelo cubriendo camino feliz y camino de error.
- **Integración.** No hay suite automática de integración separada hoy —
  los tests unitarios de `scaffold_test.go` ya ejercitan el flujo completo
  contra un filesystem real en directorios temporales (no mocks), lo que
  cumple ese rol. El smoke test manual de abajo es el nivel de integración
  real hasta que exista una suite dedicada.
- **Smoke test manual (opcional, recomendado antes de un release).**
  Confirma que el binario compilado scaffoldea correctamente sobre un
  directorio real, algo que los tests unitarios (que llaman funciones
  Go directamente) no cubren.

Sin umbral de cobertura numérico — se mantiene el criterio cualitativo ya
fijado en `CHECKPOINTS.md` (C4: cada módulo tiene al menos un test, camino
feliz y de error). Un porcentaje arbitrario no agrega señal real que
`CHECKPOINTS.md` no pida ya (decisión confirmada 26/08/2026).

## Comando

```bash
go build ./...
go vet ./...
go test ./... -v
```

Formato (no falla el build, pero se corrige antes de cerrar si señala algo):

```bash
gofmt -l .
```

Smoke test manual:

```bash
go build -o april . && ./april init /tmp/prueba-harness
```

## Anti-patrones

- Afirmar "tests en verde" sin haber corrido el comando de verdad — el
  reporte de `agent_developer`/`reviewer_agent` siempre incluye el output
  real, no una narración de que "debería pasar".
- Mockear el filesystem cuando el comportamiento a verificar ES la
  interacción con el filesystem — `scaffold_test.go` ya usa directorios
  temporales reales, seguir ese patrón en los módulos nuevos.
- Editar un test existente que falla para que pase, en vez de pararse y
  reportar por qué falla — la regla dura de este proyecto (ver
  `CLAUDE.md`/`.claude/agents/reviewer_agent.md`) es nunca tocar un test
  que ya fallaba para maquillar el resultado.

## Verificación final antes de cerrar

Corre `./init.sh` — debe salir con exit code 0 antes de cerrar cualquier
feature.

## Qué cada guardrail NO prueba

Movido a `CLAUDE.md` ("Mecanismos incorporados de April") el 31/08/2026:
ese contenido describe mecanismos del binario `april` que todo proyecto
scaffoldeado hereda, así que debía vivir en un archivo que `april init`
propaga — `docs/verification.md` de la raíz no viaja con el scaffold
(no está en el `go:embed` de `scaffold.go`). Ver ahí para el detalle.

## Retención — ledger y backups

Movido a `CLAUDE.md` ("Mecanismos incorporados de April") el 31/08/2026,
mismo motivo que arriba.
