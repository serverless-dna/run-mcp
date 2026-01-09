#!/bin/sh
set -e

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY="run-mcp-${OS}-${ARCH}"
INSTALL_DIR="${HOME}/.local/bin"
REPO="serverless-dna/run-mcp"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download latest release
echo "Downloading run-mcp for ${OS}/${ARCH}..."
curl -fsSL "https://github.com/${REPO}/releases/latest/download/${BINARY}" -o "${INSTALL_DIR}/run-mcp"
chmod +x "${INSTALL_DIR}/run-mcp"

echo "✅ Installed run-mcp to ${INSTALL_DIR}/run-mcp"

# Check PATH
if ! echo "$PATH" | grep -q "${INSTALL_DIR}"; then
  echo ""
  echo "⚠️  Add to PATH: export PATH=\"\$HOME/.local/bin:\$PATH\""
fi