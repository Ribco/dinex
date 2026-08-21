#!/usr/bin/env bash
set -euo pipefail

REPO="Ribco/dinex"
INSTALL_DIR="/opt/dinex"

echo "=== Dinex Linux Installer ==="

command -v curl >/dev/null || { echo "ERROR: curl is required."; exit 1; }

ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) ASSET_ARCH="amd64" ;;
    aarch64|arm64) ASSET_ARCH="arm64" ;;
    *) echo "ERROR: Unsupported architecture: $ARCH"; exit 1 ;;
esac

command -v sudo >/dev/null && SUDO="sudo" || SUDO=""

$SUDO mkdir -p "$INSTALL_DIR"

BASE="https://github.com/${REPO}/releases/latest/download"

echo "Architecture: $ASSET_ARCH"
echo "Installing to: $INSTALL_DIR"

$SUDO curl -fL "$BASE/dinex-panel-linux-${ASSET_ARCH}" \
    -o "$INSTALL_DIR/dinex-panel"

$SUDO curl -fL "$BASE/dinex-agent-linux-${ASSET_ARCH}" \
    -o "$INSTALL_DIR/dinex-agent"

$SUDO chmod +x "$INSTALL_DIR/dinex-panel" "$INSTALL_DIR/dinex-agent"

echo
echo "Dinex installed successfully!"
echo "Panel: $INSTALL_DIR/dinex-panel"
echo "Agent: $INSTALL_DIR/dinex-agent"
echo
echo "Run:"
echo "  sudo $INSTALL_DIR/dinex-panel"
echo "  sudo $INSTALL_DIR/dinex-agent"
