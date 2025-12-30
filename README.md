# MCP Container Images

This repository provides pre-built container images for running MCP (Model Context Protocol) servers without requiring local development tools. The system automatically builds and publishes language-specific container images to GitHub Container Registry.

## Available Container Images

### Node.js Container (`mcp-nodejs`)
- **Base**: Node.js LTS on Alpine Linux
- **Package Managers**: npm, yarn, npx
- **Features**: TypeScript support, ES modules, CommonJS
- **Usage**: Run published or local Node.js MCP servers

### Python Container (`mcp-python`)
- **Base**: Python 3.12-slim
- **Package Manager**: uv (fast Python package manager)
- **Features**: Virtual environments, uvx for running packages
- **Usage**: Run published or local Python MCP servers

## Quick Start

### Using the run-mcp Script (Recommended)

The `run-mcp` script provides a simple way to run MCP servers without Docker complexity:

**Node.js Example:**
```bash
run-mcp npx @modelcontextprotocol/server-filesystem /data
```

**Python Example:**
```bash
run-mcp uvx mcp-server-sqlite --db-path /data/mydb.sqlite
```

### Using Docker Directly

**Node.js Example:**
```bash
docker run -i --rm -v ~/Documents:/data ghcr.io/owner/mcp-nodejs \
  npx @modelcontextprotocol/server-filesystem /data
```

**Python Example:**
```bash
docker run -i --rm -v ~/data:/data ghcr.io/owner/mcp-python \
  uvx mcp-server-sqlite --db-path /data/mydb.sqlite
```

### Using with Claude Desktop

**Simple Configuration (with run-mcp):**
```json
{
  "mcpServers": {
    "aws-api": {
      "command": "run-mcp",
      "args": ["uvx", "awslabs.aws-api-mcp-server"],
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
      "args": ["uvx", "my-custom-server"],
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
        "npx", "@modelcontextprotocol/server-filesystem", "/data"
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

The build system uses GitHub Actions to automatically build and publish container images when source files change. Only affected containers are rebuilt for efficiency.

### Change Detection

The system uses git diff to detect which container directories have changed:
- Changes in `nodejs/` trigger Node.js container builds
- Changes in `python/` trigger Python container builds
- Changes in common files (workflows, scripts) trigger all container builds

### Upstream Version Detection

The system automatically checks for new upstream runtime versions:
- Weekly scheduled checks for new Node.js LTS and Python releases
- Automatic rebuilds when new patch or minor versions are detected
- Version tracking in `versions.json` to avoid unnecessary rebuilds
- Manual version checks available via GitHub Actions workflow_dispatch

### Manual Builds

You can manually trigger builds using the GitHub Actions workflow_dispatch event in the repository's Actions tab.

## Container Features

### run-mcp Binary Features
- **Cross-platform**: Single binary works on Windows, macOS, and Linux
- **Auto-detection**: Automatically detects Docker, Podman, or nerdctl
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