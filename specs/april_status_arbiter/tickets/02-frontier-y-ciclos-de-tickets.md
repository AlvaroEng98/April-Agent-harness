# 02: Frontier y grafo de dependencias de tickets — `Blocked by`, detección de ciclos

**What to build:** para la feature activa en `phase: "implementation"`,
`april status [id] --json` (y su salida en texto plano) devuelve en
`frontier` exactamente los tickets tomables en paralelo ahora mismo: los
que tienen **todos** sus bloqueadores ya resueltos a `Status: done`, y
cuyo propio `Status` es distinto de `done`. Un ticket ya `done` nunca
aparece en `frontier`; un ticket bloqueado por otro que todavía no está
`done` tampoco. `frontier` sigue vacío en cualquier fase que no sea
`implementation` con tickets existentes.

Para calcularlo, el comando interpreta el campo `Blocked by` de cada
ticket con esta convención: busca dentro del texto todos los números de
dos dígitos (`\d{2}`) y los toma como el `NN` de los tickets que bloquean
a este; si el texto (sin distinguir mayúsculas/minúsculas) contiene "none"
y ningún número de dos dígitos, el ticket no tiene bloqueadores.
Cualquier otro caso — texto sin número y sin "none", o números que no
corresponden a ningún archivo de ticket existente — se reporta en
`blockedReasons` en vez de romper el cálculo o asumir "sin bloqueadores"
en silencio.

Si el grafo de `Blocked by` de los tickets de una feature tiene un ciclo
(A bloquea a B, B bloquea a A, directo o a través de una cadena más
larga), se detecta con DFS + pila de recursión — acotado por construcción
al número de archivos de ticket leídos, nunca recursión sin límite — y se
reporta en `blockedReasons` con la feature y la cadena de tickets que
forma el ciclo (ej. `"ciclo detectado en Blocked by de tickets de
april_status_arbiter: 02 → 03 → 02"`). El comando siempre termina, nunca
cuelga, incluso frente a un ciclo.

**Blocked by:** 01 (Núcleo de `april status` — fase, selección de
feature, blockedReasons básicos y CLI)

**Status:** done

- [ ] `frontier` lista exactamente los tickets con todos sus `Blocked by`
      en `Status: done` y su propio `Status` distinto de `done`.
- [ ] Un ticket ya `done` nunca aparece en `frontier`.
- [ ] Un ticket bloqueado por otro que no está `done` no aparece en
      `frontier`.
- [ ] Un ciclo en `Blocked by` (fixture con A→B→A, dos o tres tickets) se
      detecta, se reporta en `blockedReasons` con la cadena de tickets
      involucrados, y el comando termina (no cuelga) — verificado con
      timeout de test explícito o conteo de nodos visitados.
- [ ] Un `Blocked by` con texto no interpretable (sin número de dos
      dígitos y sin "none", o número que no corresponde a ningún ticket
      existente) se reporta en `blockedReasons`.
- [ ] `frontier` sigue vacío en cualquier fase que no sea
      `implementation` con tickets existentes (spec, tickets sin
      desglose, grill, closed, `sdd:false` sin tickets).
- [ ] `go build ./...` y `go test ./...` en verde, incluyendo los tests
      unitarios de ciclo, frontier y `Blocked by` no interpretable sobre
      `computeStatusFromFS`.
