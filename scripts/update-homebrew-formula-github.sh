#!/bin/bash
set -euo pipefail

# Script to update homebrew formula from GitHub Actions with downloaded binaries
# Usage: ./scripts/update-homebrew-formula-github.sh <version> <tag>

VERSION="${1:-}"
TAG="${2:-}"

if [ -z "$VERSION" ] || [ -z "$TAG" ]; then
    echo "Error: VERSION and TAG are required"
    echo "Usage: $0 <version> <tag>"
    echo "Example: $0 1.0.0 v1.0.0"
    exit 1
fi

echo "Updating homebrew formula for version: $VERSION (from git tag: $TAG)"

# Check required directories
if [ ! -d "homebrew-tap" ]; then
    echo "Error: homebrew-tap directory not found"
    exit 1
fi

if [ ! -d "build" ]; then
    echo "Error: build directory not found"
    exit 1
fi

# Calculate SHA256 for each binary
echo "Calculating SHA256 checksums..."
SHA_DARWIN_ARM64=$(sha256sum build/run-mcp-darwin-arm64 | cut -d' ' -f1)
SHA_DARWIN_AMD64=$(sha256sum build/run-mcp-darwin-amd64 | cut -d' ' -f1)
SHA_LINUX_AMD64=$(sha256sum build/run-mcp-linux-amd64 | cut -d' ' -f1)
SHA_LINUX_ARM64=$(sha256sum build/run-mcp-linux-arm64 | cut -d' ' -f1)

echo "SHA256 checksums:"
echo "  Darwin ARM64: $SHA_DARWIN_ARM64"
echo "  Darwin AMD64: $SHA_DARWIN_AMD64"
echo "  Linux AMD64:  $SHA_LINUX_AMD64"
echo "  Linux ARM64:  $SHA_LINUX_ARM64"

# Create the homebrew formula file
echo "Creating homebrew formula..."
cat > homebrew-tap/Formula/run-mcp.rb << EOF
class RunMcp < Formula
  desc "Cross-platform binary for running MCP servers in containers"
  homepage "https://github.com/serverless-dna/run-mcp"
  version "$VERSION"
  license "MIT"
  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/serverless-dna/run-mcp/releases/download/$TAG/run-mcp-darwin-arm64"
      sha256 "$SHA_DARWIN_ARM64"
    else
      url "https://github.com/serverless-dna/run-mcp/releases/download/$TAG/run-mcp-darwin-amd64"
      sha256 "$SHA_DARWIN_AMD64"
    end
  end
  
  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/serverless-dna/run-mcp/releases/download/$TAG/run-mcp-linux-arm64"
      sha256 "$SHA_LINUX_ARM64"
    else
      url "https://github.com/serverless-dna/run-mcp/releases/download/$TAG/run-mcp-linux-amd64"
      sha256 "$SHA_LINUX_AMD64"
    end
  end
  
  def install
    bin.install Dir["*"].first => "run-mcp"
  end
  
  test do
    assert_match version.to_s, shell_output("#{bin}/run-mcp --version")
    assert_match "Show this help message", shell_output("#{bin}/run-mcp --help")
  end
end
EOF

echo "✅ Homebrew formula updated successfully!"