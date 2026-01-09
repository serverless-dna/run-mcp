#!/bin/bash
set -euo pipefail

# Script to update winget manifest from GitHub Actions with downloaded binaries
# Usage: ./scripts/update-winget-manifest-github.sh <version> <tag>

VERSION="${1:-}"
TAG="${2:-}"

if [ -z "$VERSION" ] || [ -z "$TAG" ]; then
    echo "Error: VERSION and TAG are required"
    echo "Usage: $0 <version> <tag>"
    echo "Example: $0 1.0.0 v1.0.0"
    exit 1
fi

echo "Updating winget manifest for version: $VERSION (from git tag: $TAG)"

# Check required directories
if [ ! -d "winget-pkgs" ]; then
    echo "Error: winget-pkgs directory not found"
    echo "Please clone your winget-pkgs repository first:"
    echo "  git clone https://github.com/serverless-dna/winget-pkgs.git"
    exit 1
fi

if [ ! -d "build" ]; then
    echo "Error: build directory not found"
    exit 1
fi

# Calculate SHA256 for Windows binaries
echo "Calculating SHA256 checksums for Windows binaries..."
SHA_WINDOWS_AMD64=$(sha256sum build/run-mcp-windows-amd64.exe | cut -d' ' -f1)

echo "SHA256 checksums:"
echo "  Windows AMD64: $SHA_WINDOWS_AMD64"

# Create manifest directory structure
MANIFEST_DIR="winget-pkgs/manifests/s/ServerlessDNA/RunMCP/$VERSION"
mkdir -p "$MANIFEST_DIR"

# Create the main manifest file
echo "Creating main manifest..."
cat > "$MANIFEST_DIR/ServerlessDNA.RunMCP.yaml" << EOF
# Created with YamlCreate.ps1 v2.4.1
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.version.1.6.0.schema.json

PackageIdentifier: ServerlessDNA.RunMCP
PackageVersion: $VERSION
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
EOF

# Create the installer manifest
echo "Creating installer manifest..."
cat > "$MANIFEST_DIR/ServerlessDNA.RunMCP.installer.yaml" << EOF
# Created with YamlCreate.ps1 v2.4.1
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.installer.1.6.0.schema.json

PackageIdentifier: ServerlessDNA.RunMCP
PackageVersion: $VERSION
InstallerType: portable
Commands:
- run-mcp
Installers:
- Architecture: x64
  InstallerUrl: https://github.com/modelcontextprotocol/mcp-container-images/releases/download/$TAG/run-mcp-windows-amd64.exe
  InstallerSha256: $SHA_WINDOWS_AMD64
ManifestType: installer
ManifestVersion: 1.6.0
EOF

# Create the locale manifest
echo "Creating locale manifest..."
cat > "$MANIFEST_DIR/ServerlessDNA.RunMCP.locale.en-US.yaml" << EOF
# Created with YamlCreate.ps1 v2.4.1
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.defaultLocale.1.6.0.schema.json

PackageIdentifier: ServerlessDNA.RunMCP
PackageVersion: $VERSION
PackageLocale: en-US
Publisher: ServerlessDNA
PublisherUrl: https://github.com/serverless-dna
PublisherSupportUrl: https://github.com/modelcontextprotocol/mcp-container-images/issues
PackageName: run-mcp
PackageUrl: https://github.com/modelcontextprotocol/mcp-container-images
License: MIT
LicenseUrl: https://github.com/modelcontextprotocol/mcp-container-images/blob/main/LICENSE
ShortDescription: Run MCP servers in containers with automatic runtime detection
Description: |-
  run-mcp is a cross-platform binary that automatically detects container runtimes
  and language requirements to run MCP servers in containers. It provides a drop-in
  replacement for direct command execution with secure environment variable passthrough
  and cross-platform volume mounting.
Moniker: run-mcp
Tags:
- containers
- development
- docker
- mcp
- model-context-protocol
- podman
ManifestType: defaultLocale
ManifestVersion: 1.6.0
EOF

echo "✅ Winget manifest created successfully!"
echo "📁 Manifest files created in: $MANIFEST_DIR"
echo ""
echo "Next steps:"
echo "1. Review the generated manifest files"
echo "2. Commit and push to your winget-pkgs repository:"
echo "   cd winget-pkgs"
echo "   git add manifests/s/ServerlessDNA/RunMCP/$VERSION/"
echo "   git commit -m \"Add ServerlessDNA.RunMCP version $VERSION\""
echo "   git push origin main"
echo ""
echo "3. Users can then install with:"
echo "   winget source add ServerlessDNA https://github.com/serverless-dna/winget-pkgs"
echo "   winget install ServerlessDNA.RunMCP --source ServerlessDNA"