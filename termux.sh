#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail

REPO="Ribco/dinex"
INSTALL_DIR="$HOME/.local/bin"

echo "=== Dinex Termux Installer ==="

command -v curl >/dev/null || {
    echo "ERROR: curl is required."
    echo "Install it with: pkg install curl"
    exit 1
}

ARCH="$(uname -m)"
case "$ARCH" in
    aarch64|arm64) ASSET_ARCH="arm64" ;;
    x86_64|amd64) ASSET_ARCH="amd64" ;;
    *) echo "ERROR: Unsupported architecture: $ARCH"; exit 1 ;;
esac

mkdir -p "$INSTALL_DIR"

BASE="https://github.com/${REPO}/releases/latest/download"

echo "Architecture: $ASSET_ARCH"
echo "Installing to: $INSTALL_DIR"

curl -fL "$BASE/dinex-panel-linux-${ASSET_ARCH}" \
    -o "$INSTALL_DIR/dinex-panel"

curl -fL "$BASE/dinex-agent-linux-${ASSET_ARCH}" \
    -o "$INSTALL_DIR/dinex-agent"

chmod +x "$INSTALL_DIR/dinex-panel" "$INSTALL_DIR/dinex-agent"

if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
    echo
    echo "Add this to your ~/.bashrc:"
    echo 'export PATH="$HOME/.local/bin:$PATH"'
fi

echo
echo "Dinex installed successfully!"
echo "Panel: $INSTALL_DIR/dinex-panel"
echo "Agent: $INSTALL_DIR/dinex-agent"
