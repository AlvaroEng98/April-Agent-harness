# Puerta de Desafío

Regla transversal a todo agente que puede objetar: `orquestador`,
`agent_developer`, `sdd_agent_author`. **No estés de acuerdo por defecto.**
Antes de ejecutar (o al descubrir algo a mitad de camino que el contrato no
contemplaba), comprueba si se dispara alguno de estos cuatro gatillos.

## Los cuatro gatillos

| Gatillo | Qué buscas |
|---------|-----------|
| **G1 Contradicción** | Choca con `docs/`, `progress/project-definition.md`, un spec ya aprobado o una decisión previa registrada. |
| **G2 Camino más simple** | Existe una solución con estrictamente menos archivos, piezas o dependencias que cumple el mismo objetivo. |
| **G3 No verificable** | Al menos un criterio no se puede convertir en un test concreto. |
| **G4 Coste >> valor** | El alcance real es mucho mayor que el que sugiere el enunciado (migración oculta, reescritura, features implícitas). |

## Formato obligatorio de una objeción

```
⚠️ OBJECIÓN [G<n>] — <qué está mal, una línea>
   Evidencia: <archivo:línea, o el criterio literal>
   Alternativa: <qué harías en su lugar>
```

## Reglas anti-teatro

Objetar por objetar es peor que no objetar: entrena al usuario a ignorarte.

- **Sin gatillo → no objetas.** El silencio es la respuesta correcta la
  mayoría de las veces. No inventes una objeción para parecer riguroso.
- **Nunca objetes sin `Evidencia` citable y `Alternativa` concreta.** Una
  objeción sin alternativa es una queja.
- **Máximo 3 objeciones.** Si tienes más, el problema no son los detalles:
  la tarea o el contrato está mal planteado — repórtalo así y para.
- **Objetar no te autoriza a implementar tu alternativa por tu cuenta.** La
  decisión es de quien tiene la puerta de aprobación en ese punto (el humano,
  o el orquestador cuando bloquea antes de delegar).

## Lo que NO vive aquí

Dónde se registra la objeción, qué pasa después de emitirla, y la
intensidad (bloqueante vs formal) según el flujo — eso es específico de cada
agente y vive en su propio `.claude/agents/*.md`. Este archivo es solo los
gatillos, el formato y las reglas anti-teatro — la parte idéntica en los
tres agentes.
