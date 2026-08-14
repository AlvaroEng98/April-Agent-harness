#!/bin/sh
# Instalador de harness (April Agent Harness).
#
# Uso:
#   curl -fsSL https://raw.githubusercontent.com/AlvaroEng98/April-Agent-harness/main/install.sh | sh
#
# Variables de entorno opcionales:
#   VERSION   versión a instalar (por defecto: la última release). Con o sin "v".
#   BIN_DIR   directorio destino. Por defecto ~/.local/bin. La instalación es
#             siempre a nivel de usuario: el script nunca usa sudo.
#
# POSIX sh a propósito: el script se ejecuta con `sh`, no con bash.
set -eu

REPO="AlvaroEng98/April-Agent-harness"
BINARY="apil"

err() {
  echo "error: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || err "falta '$1' en el PATH"
}

need curl
need tar
need uname

# --- Plataforma -------------------------------------------------------------
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux|darwin) ;;
  *) err "sistema no soportado: $OS (solo linux y darwin)" ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) err "arquitectura no soportada: $ARCH (solo amd64 y arm64)" ;;
esac

# --- Versión ----------------------------------------------------------------
# Sin VERSION explícita se resuelve la última release por la API de GitHub.
# Se parsea con sed para no depender de jq.
if [ -z "${VERSION:-}" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1)
  [ -n "$VERSION" ] || err "no se pudo resolver la última versión; reintenta con VERSION=x.y.z"
fi
# Los assets de goreleaser llevan la versión sin "v"; el tag sí lo lleva.
VERSION=${VERSION#v}
TAG="v${VERSION}"

ASSET="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

# --- Destino ----------------------------------------------------------------
# Instalación a nivel de usuario: nunca se usa sudo ni se escribe en rutas del
# sistema. Si el destino no es escribible, se aborta y se pide otro BIN_DIR.
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$BIN_DIR" 2>/dev/null || err "no se pudo crear ${BIN_DIR}; usa BIN_DIR=<ruta escribible>"
[ -w "$BIN_DIR" ] || err "sin permiso de escritura en ${BIN_DIR}; usa BIN_DIR=<ruta escribible>"

# --- Descarga y verificación ------------------------------------------------
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT INT TERM

echo "Descargando ${BINARY} ${TAG} (${OS}/${ARCH})..."
curl -fsSL -o "${TMP}/${ASSET}" "${BASE_URL}/${ASSET}" \
  || err "no se pudo descargar ${BASE_URL}/${ASSET}"

if curl -fsSL -o "${TMP}/checksums.txt" "${BASE_URL}/checksums.txt" 2>/dev/null; then
  if command -v sha256sum >/dev/null 2>&1; then
    SUM=$(sha256sum "${TMP}/${ASSET}" | cut -d' ' -f1)
  elif command -v shasum >/dev/null 2>&1; then
    SUM=$(shasum -a 256 "${TMP}/${ASSET}" | cut -d' ' -f1)
  else
    SUM=""
    echo "aviso: sin sha256sum/shasum, se omite la verificación de checksum" >&2
  fi
  if [ -n "$SUM" ]; then
    EXPECTED=$(grep " ${ASSET}\$" "${TMP}/checksums.txt" | cut -d' ' -f1)
    [ -n "$EXPECTED" ] || err "el asset ${ASSET} no aparece en checksums.txt"
    [ "$SUM" = "$EXPECTED" ] || err "checksum inválido: esperado ${EXPECTED}, obtenido ${SUM}"
    echo "Checksum verificado."
  fi
else
  echo "aviso: checksums.txt no disponible en la release, se omite la verificación" >&2
fi

tar -xzf "${TMP}/${ASSET}" -C "$TMP" "$BINARY" || err "no se pudo extraer ${BINARY} del tarball"

# --- Instalación ------------------------------------------------------------
install -m 755 "${TMP}/${BINARY}" "${BIN_DIR}/${BINARY}"

echo "Instalado en ${BIN_DIR}/${BINARY}"

case ":${PATH}:" in
  *":${BIN_DIR}:"*)
    "${BIN_DIR}/${BINARY}" version || true
    ;;
  *)
    echo "aviso: ${BIN_DIR} no está en el PATH. Añádelo:" >&2
    echo "  export PATH=\"${BIN_DIR}:\$PATH\"" >&2
    ;;
esac
