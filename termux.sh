#!/data/data/com.termux/files/usr/bin/bash

set -euo pipefail

REPO="Ribco/dinex"
INSTALL_DIR="$HOME/.local/bin"
BASE="https://github.com/${REPO}/releases/latest/download"

echo "=== Dinex Termux Installer ==="

command -v curl >/dev/null || {
    echo "ERROR: curl is required."
    echo "Install it with: pkg install curl"
    exit 1
}

ARCH="$(uname -m)"

case "$ARCH" in
    aarch64|arm64)
        ASSET_ARCH="arm64"
        ;;
    x86_64|amd64)
        ASSET_ARCH="amd64"
        ;;
    *)
        echo "ERROR: Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

mkdir -p "$INSTALL_DIR"

echo "Architecture: $ASSET_ARCH"
echo "Installing to: $INSTALL_DIR"
echo

download() {
    local name="$1"
    local output="$2"

    echo "Downloading $name..."

    curl -fL \
        --retry 10 \
        --retry-delay 3 \
        --retry-max-time 300 \
        --connect-timeout 30 \
        --continue-at - \
        "$BASE/$name" \
        -o "$output"
}

download "dinex-panel-linux-${ASSET_ARCH}" "$INSTALL_DIR/dinex-panel"
download "dinex-agent-linux-${ASSET_ARCH}" "$INSTALL_DIR/dinex-agent"

chmod +x "$INSTALL_DIR/dinex-panel" "$INSTALL_DIR/dinex-agent"

echo
echo "Dinex installed successfully!"
echo
echo "Panel: $INSTALL_DIR/dinex-panel"
echo "Agent: $INSTALL_DIR/dinex-agent"

if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
    echo
    echo "Add this to your ~/.bashrc:"
    echo 'export PATH="$HOME/.local/bin:$PATH"'
fi

echo
echo "Done!"
