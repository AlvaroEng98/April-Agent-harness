// recap.js — Plugin de OpenCode que inyecta el recap del proyecto como
// contexto de sistema al iniciar cada sesión.
//
// OpenCode no expone un hook "SessionStart" nativo equivalente al de Claude
// Code (verificado contra el código fuente y la doc oficial). El hook más
// cercano con un canal de salida para inyectar contexto en el prompt de
// sistema es "experimental.chat.system.transform", marcado explícitamente
// como experimental por OpenCode. Se usa aquí con deduplicación por sessionID
// para inyectar el recap solo una vez por sesión.
//
// Delega toda la lógica en recap.sh (fuente única de verdad): no reimplementa
// el recap. Cualquier fallo (recap.sh ausente, sin permisos, etc.) se degrada
// en silencio para no romper la sesión de OpenCode.

export const RecapPlugin = async ({ directory, $ }) => {
  // sesiones a las que ya se les inyectó el recap (vive mientras el proceso
  // de OpenCode esté activo).
  const seen = new Set();

  return {
    "experimental.chat.system.transform": async (input, output) => {
      try {
        const sessionID = input && input.sessionID;
        if (sessionID && seen.has(sessionID)) {
          return;
        }
        if (sessionID) {
          seen.add(sessionID);
        }
        const recap = (await $`bash ${directory}/recap.sh`.text()).trim();
        if (recap) {
          output.system.push(recap);
        }
      } catch {
        // degradación en silencio: el recap es informativo, no crítico.
      }
    },
  };
};
