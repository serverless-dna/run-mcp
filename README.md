# MCP Container Images

This repository provides pre-built container images for running MCP (Model Context Protocol) servers without requiring local development tools. The system automatically builds and publishes language-specific container images to GitHub Container Registry.

## Available Container Images

### Node.js Container (`mcp-nodejs`)
- **Base**: Node.js LTS on Alpine Linux
- **Features**: TypeScript support, ES modules, CommonJS
- **Usage**: Run published or local Node.js MCP servers

### Python Container (`mcp-python`)
- **Base**: Python 3.12-slim
- **Features**: Virtual environments, pip package support
- **Usage**: Run published or local Python MCP servers

## Container Tagging Strategy

All container images follow a consistent tagging strategy for easy version management:

### Node.js Container Tags
```
ghcr.io/owner/repo-nodejs:latest              # Latest LTS version
ghcr.io/owner/repo-nodejs:node22              # Latest Node.js 22.x
ghcr.io/owner/repo-nodejs:node22.11.0         # Specific version
ghcr.io/owner/repo-nodejs:node22.11.0-20241231 # Version with build date
```

### Python Container Tags
```
ghcr.io/owner/repo-python:latest              # Latest 3.12.x version
ghcr.io/owner/repo-python:python3.12          # Latest Python 3.12.x
ghcr.io/owner/repo-python:python3.12.8        # Specific version
ghcr.io/owner/repo-python:python3.12.8-20241231 # Version with build date
```

### Matrix Build Tags
When using matrix builds (`make build-matrix`), additional tags are created:
```bash
# Node.js matrix tags
ghcr.io/owner/repo-nodejs:node18              # Latest Node.js 18.x
ghcr.io/owner/repo-nodejs:node20              # Latest Node.js 20.x
ghcr.io/owner/repo-nodejs:node22              # Latest Node.js 22.x

# Python matrix tags  
ghcr.io/owner/repo-python:python3.11          # Latest Python 3.11.x
ghcr.io/owner/repo-python:python3.12          # Latest Python 3.12.x
ghcr.io/owner/repo-python:python3.13          # Latest Python 3.13.x
```

### Tag Usage Recommendations
- **`latest`**: Use for development and testing
- **Major version** (e.g., `node22`): Use for production when you want automatic patch updates
- **Specific version** (e.g., `node22.11.0`): Use for production when you need exact version control
- **Date-stamped**: Use for debugging or when you need to reference a specific build

## Quick Start

### Using the run-mcp Script (Recommended)

The `run-mcp` script provides a simple way to run MCP servers without Docker complexity:

**Node.js Example:**
```bash
run-mcp node /path/to/server.js
```

**Python Example:**
```bash
run-mcp python /path/to/server.py
```

#### How run-mcp Selects Container Images

The `run-mcp` script automatically selects the appropriate container image based on your command:

**Automatic Selection:**
```bash
run-mcp node server.js          # Uses: ghcr.io/owner/repo-nodejs:latest
run-mcp python server.py        # Uses: ghcr.io/owner/repo-python:latest
run-mcp npm start              # Uses: ghcr.io/owner/repo-nodejs:latest  
run-mcp pip install package   # Uses: ghcr.io/owner/repo-python:latest
```

**Override with Specific Image:**
```bash
# Use specific version
run-mcp --image ghcr.io/owner/repo-nodejs:node20 node server.js

# Use specific patch version
run-mcp --image ghcr.io/owner/repo-python:python3.11.8 python server.py

# Use date-stamped build
run-mcp --image ghcr.io/owner/repo-nodejs:node22.11.0-20241231 node server.js
```

**Environment Variable Override:**
```bash
# Set default image for session
export MCP_NODEJS_IMAGE=ghcr.io/owner/repo-nodejs:node20
export MCP_PYTHON_IMAGE=ghcr.io/owner/repo-python:python3.11

run-mcp node server.js         # Uses your custom Node.js image
run-mcp python server.py       # Uses your custom Python image
```

**Detection Rules:**
- **Node.js commands**: `node`, `npm`, `npx`, `yarn` → Uses Node.js container
- **Python commands**: `python`, `pip`, `uvx`, `python3` → Uses Python container  
- **Unknown commands**: Defaults to Node.js container (can be changed with `--image`)

#### Configuration Options

```bash
# List available container images
run-mcp list-images

# Show current configuration (including current images)
run-mcp config

# Show runtime information (container runtimes and supported languages)
run-mcp info

# Show version and build info  
run-mcp --version

# Get help
run-mcp --help
```

**Example `run-mcp list-images` output:**
```
Available Container Images
=========================
Using container runtime: docker

Node.js Images:
Repository: ghcr.io/owner/repo-nodejs
  Local images:
    ghcr.io/owner/repo-nodejs:latest
    ghcr.io/owner/repo-nodejs:node22
    ghcr.io/owner/repo-nodejs:node22.11.0
  Common tags available:
    latest, node18, node20, node22
    node18.x.x, node20.x.x, node22.x.x (specific versions)

Python Images:
Repository: ghcr.io/owner/repo-python
  No local images found. Common tags available:
    latest, python3.11, python3.12, python3.13
    python3.11.x, python3.12.x, python3.13.x (specific versions)
  Pull with: docker pull ghcr.io/owner/repo-python:<tag>

Usage:
  run-mcp --image <full-image-name> <command>
  export MCP_NODEJS_IMAGE=ghcr.io/owner/repo-nodejs:<tag>
  export MCP_PYTHON_IMAGE=ghcr.io/owner/repo-python:<tag>
```

**Example `run-mcp config` output:**
```
run-mcp Configuration
====================
Node.js Image: ghcr.io/owner/repo-nodejs:node22
Python Image:  ghcr.io/owner/repo-python:python3.12
Data Directory: /home/user (✓)

Environment Variables:
  MCP_NODEJS_IMAGE=ghcr.io/owner/repo-nodejs:node20
```

**Example `run-mcp info` output:**
```
run-mcp Runtime Information
===========================

Available Container Runtimes:
  ✓ docker (24.0.7)
  ✓ podman (4.3.1)

Supported Languages:
  nodejs: node, npm, npx, yarn
  python: python, pip, uvx, python3
```

### Using Docker Directly

**Node.js Example:**
```bash
docker run -i --rm -v ~/Documents:/data ghcr.io/owner/mcp-nodejs \
  node /data/server.js
```

**Python Example:**
```bash
docker run -i --rm -v ~/data:/data ghcr.io/owner/mcp-python \
  python /data/server.py
```

### Using with Claude Desktop

**Simple Configuration (with run-mcp):**
```json
{
  "mcpServers": {
    "aws-api": {
      "command": "run-mcp",
      "args": ["python", "/path/to/aws-server.py"],
      "env": {
        "AWS_REGION": "us-east-1"
      }
    }
  }
}
```

**Custom Variables Example:**
```json
{
  "mcpServers": {
    "custom-server": {
      "command": "run-mcp",
      "args": ["node", "/path/to/my-server.js"],
      "env": {
        "MCP_PASSTHROUGH_ENV": "MY_API_KEY,CUSTOM_CONFIG",
        "MY_API_KEY": "your-api-key",
        "CUSTOM_CONFIG": "config-value"
      }
    }
  }
}
```

**Direct Docker Configuration:**
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm", 
        "-v", "/Users/me/Documents:/data",
        "ghcr.io/owner/mcp-nodejs", 
        "node", "/data/server.js"
      ]
    }
  }
}
```

## Repository Structure

```
├── nodejs/              # Node.js container source files
├── python/              # Python container source files  
├── cmd/run-mcp/         # Go binary source code
├── scripts/             # Build and utility scripts
│   ├── detect-changes.sh    # Change detection for builds
│   └── check-upstream-versions.sh  # Upstream version checking
├── .github/workflows/   # GitHub Actions workflows
└── README.md           # This file
```

## Installation

### Installing run-mcp Binary

The `run-mcp` binary simplifies running MCP servers in containers and works on all platforms:

**Download from Releases:**
```bash
# Linux/macOS
curl -L -o run-mcp https://github.com/owner/mcp-containers/releases/latest/download/run-mcp-$(uname -s)-$(uname -m)
chmod +x run-mcp
sudo mv run-mcp /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/owner/mcp-containers/releases/latest/download/run-mcp-windows-amd64.exe" -OutFile "run-mcp.exe"
# Move to a directory in your PATH
```

**Package Managers:**
```bash
# macOS (Homebrew)
brew install owner/tap/run-mcp

# Windows (Chocolatey)
choco install run-mcp

# Linux (various package managers)
# See releases page for distribution-specific packages
```

### Linux Development Setup

This project is optimized for Linux development. Use the Makefile for easy setup:

```bash
# Check current tool status
make check-tools

# Set up development environment (installs bats, hadolint)
make setup-dev

# Option 1: Use Docker Desktop (if on Windows with WSL2)
# 1. Install Docker Desktop for Windows
# 2. Enable WSL2 integration in Docker Desktop settings
# 3. Restart WSL2: wsl --shutdown && wsl

# Option 2: Install Docker natively in Linux
sudo apt update
sudo apt install docker.io
sudo usermod -aG docker $USER
# Logout and login again

# Verify setup
make check-tools
```

**Linux Docker Options:**
- **Docker Desktop**: Easy setup for Windows users with WSL2
- **Native Docker**: Better performance, runs entirely in Linux
- **Podman**: Alternative container runtime, docker-compatible
- **Finch**: AWS's container runtime, optimized for development

### Configuration

Configure via environment variables:

```bash
export MCP_NODEJS_IMAGE="ghcr.io/owner/mcp-nodejs:node22"
export MCP_PYTHON_IMAGE="ghcr.io/owner/mcp-python:python3.12"
export MCP_DATA_DIR="$HOME/Documents"
```

## Environment Variables

### Automatic Passthrough

These patterns are automatically passed to the container:

| Pattern | Examples |
|---------|----------|
| `AWS_*` | `AWS_REGION`, `AWS_PROFILE`, `AWS_ACCESS_KEY_ID` |
| `OPENAI_*` | `OPENAI_API_KEY` |
| `ANTHROPIC_*` | `ANTHROPIC_API_KEY` |
| `AZURE_*` | `AZURE_OPENAI_API_KEY` |
| `GOOGLE_*` | `GOOGLE_API_KEY` |
| `MCP_*` | `MCP_DATA_DIR` |
| `HF_*` | `HF_TOKEN` |
| `REPLICATE_*` | `REPLICATE_API_TOKEN` |
| `COHERE_*` | `COHERE_API_KEY` |
| `GITHUB_TOKEN` | (exact match) |
| `GITLAB_TOKEN` | (exact match) |
| `DATABASE_URL` | (exact match) |
| `REDIS_URL` | (exact match) |

### Custom Variables

Pass additional variables using `MCP_PASSTHROUGH_ENV`:

```json
{
  "mcpServers": {
    "custom": {
      "command": "run-mcp",
      "args": ["uvx", "my-server"],
      "env": {
        "MCP_PASSTHROUGH_ENV": "MY_API_KEY,MY_SECRET",
        "MY_API_KEY": "xxx",
        "MY_SECRET": "yyy"
      }
    }
  }
}
```

### Security

Only explicitly allowed variables are passed through. System variables like `PATH`, `HOME`, `USER`, etc. are filtered out for security.

## Development

This project uses a Makefile for development workflows. Here's how to build and publish containers from the CLI:

### Prerequisites
```bash
# Check if you have required tools
make check-tools

# Install missing tools (bats, jq, curl)
make setup-dev
```

### Building Containers

```bash
# Build latest versions of all containers
make build

# Build specific containers
make build-nodejs      # Node.js container only
make build-python      # Python container only

# Build all supported versions (matrix build)
make build-matrix      # Builds Node.js 18,20,22 + Python 3.11,3.12,3.13
```

### Publishing to Registry

```bash
# Login to GitHub Container Registry (requires GITHUB_TOKEN)
export GITHUB_TOKEN=your_token_here
make login

# Push latest versions
make push

# Push all matrix versions  
make push-matrix

# Combined build and push
make build-and-push           # Latest versions
make build-and-push-matrix    # All matrix versions
```

### Complete Container Lifecycle

The proper order is: **cleanup old versions → build → push**

```bash
# Complete lifecycle for latest versions
make containers              # cleanup → build → push

# Complete lifecycle for all matrix versions  
make containers-matrix       # cleanup → build-matrix → push-matrix

# Full release process
make release                 # ci → containers → build-run-mcp-all
make release-matrix          # ci → containers-matrix → build-run-mcp-all
```

### Building the run-mcp Binary

```bash
# Build for current platform
make build-run-mcp

# Build for all platforms (Windows, macOS, Linux - amd64/arm64)
make build-run-mcp-all

# Install locally
make install-run-mcp         # Copies to /usr/local/bin/
```

### Registry Management

```bash
# Clean up old container versions (follows retention policy)
make cleanup-versions

# Check what would be built based on git changes
make check-changes

# Show build configuration
make info
```

### Testing and Validation

```bash
# Run all tests
make test

# Run CI checks locally
make ci

# Validate Dockerfiles
make lint
```

### Available Make Targets
Run `make help` to see all available targets with descriptions.

### Upstream Version Detection

The system automatically detects new runtime versions:
- **Dynamic Detection**: Always fetches latest versions from nodejs.org and endoflife.date APIs
- **Matrix Support**: Builds multiple versions (Node.js 18,20,22 + Python 3.11,3.12,3.13)  
- **Scheduled Checks**: Weekly GitHub Actions workflow checks for updates
- **Manual Triggers**: Use `make check-upstream` or GitHub Actions workflow_dispatch
- Manual version checks available via GitHub Actions workflow_dispatch

### Manual Builds

You can manually trigger builds using the GitHub Actions workflow_dispatch event in the repository's Actions tab.

## Container Features

### run-mcp Binary Features
- **Cross-platform**: Single binary works on Windows, macOS, and Linux
- **Auto-detection**: Automatically detects Docker, Podman, nerdctl, or Finch
- **Language inference**: Detects Node.js (npx, node) or Python (uvx, python) from commands
- **Secure environment passthrough**: Allowlist-based environment variable filtering
- **Custom variables**: Support for additional variables via `MCP_PASSTHROUGH_ENV`
- **Smart mounting**: Auto-mounts ~/.aws and ~/.config for credentials
- **Drop-in replacement**: Works as a simple replacement for direct command execution

### Standardized Interface
- All containers use stdio transport for MCP protocol
- Consistent volume mounting at `/data` for user data
- Run as UID 1000 for permission compatibility
- Unbuffered I/O for real-time communication

### Security
- Non-root user execution
- Minimal attack surface
- Regular security scanning
- Multi-architecture support (AMD64, ARM64)

### Performance
- Optimized image sizes (Node.js < 200MB, Python < 300MB)
- Docker layer caching for faster builds
- Multi-stage builds for production efficiency

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines and contribution process.

## Versioning

See [VERSIONING.md](VERSIONING.md) for version support and retention policies.