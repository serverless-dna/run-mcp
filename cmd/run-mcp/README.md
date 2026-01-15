# run-mcp: Container Runner for MCP Servers

`run-mcp` is a cross-platform binary that automatically detects container runtimes and language requirements to run MCP (Model Context Protocol) servers in containers. It provides a drop-in replacement for direct command execution with secure environment variable passthrough and cross-platform volume mounting.

## Features

- **Automatic Runtime Detection**: Detects Docker, Podman, nerdctl, Finch, and lima automatically
- **Language Auto-Detection**: Automatically selects Node.js or Python containers based on command
- **Secure Environment Passthrough**: Only passes allowed environment variables to containers
- **Cross-Platform Volume Mounting**: Handles credential directories across Windows, macOS, and Linux
- **Drop-in Replacement**: Works seamlessly with existing MCP client configurations

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/modelcontextprotocol/mcp-container-images/releases):

#### Windows (AMD64)
```powershell
# Download and install
curl -L -o run-mcp.exe https://github.com/modelcontextprotocol/mcp-container-images/releases/latest/download/run-mcp-windows-amd64.exe

# Move to a directory in your PATH
move run-mcp.exe C:\Windows\System32\
```

#### macOS (Intel)
```bash
# Download and install
curl -L -o run-mcp https://github.com/modelcontextprotocol/mcp-container-images/releases/latest/download/run-mcp-darwin-amd64
chmod +x run-mcp
sudo mv run-mcp /usr/local/bin/
```

#### macOS (Apple Silicon)
```bash
# Download and install
curl -L -o run-mcp https://github.com/modelcontextprotocol/mcp-container-images/releases/latest/download/run-mcp-darwin-arm64
chmod +x run-mcp
sudo mv run-mcp /usr/local/bin/
```

#### Linux (AMD64)
```bash
# Download and install
curl -L -o run-mcp https://github.com/modelcontextprotocol/mcp-container-images/releases/latest/download/run-mcp-linux-amd64
chmod +x run-mcp
sudo mv run-mcp /usr/local/bin/
```

#### Linux (ARM64)
```bash
# Download and install
curl -L -o run-mcp https://github.com/modelcontextprotocol/mcp-container-images/releases/latest/download/run-mcp-linux-arm64
chmod +x run-mcp
sudo mv run-mcp /usr/local/bin/
```

### Package Managers

#### Homebrew (macOS/Linux)
```bash
# Coming soon
brew install modelcontextprotocol/tap/run-mcp
```

#### Chocolatey (Windows)
```powershell
# Coming soon
choco install run-mcp
```

### Build from Source

```bash
git clone https://github.com/modelcontextprotocol/mcp-container-images.git
cd mcp-container-images
make build-run-mcp
sudo cp build/run-mcp /usr/local/bin/
```

## Usage

### Basic Usage

```bash
# Run Python MCP server
run-mcp uvx mcp-server-sqlite --db-path /data/db.sqlite

# Run Node.js MCP server
run-mcp npx @modelcontextprotocol/server-filesystem /data

# Explicit runtime specification
run-mcp python uvx awslabs.aws-api-mcp-server@latest
run-mcp node npx @modelcontextprotocol/server-memory
```

### Information Commands

```bash
# Show runtime information
run-mcp info

# Show configuration
run-mcp config

# Show version
run-mcp --version
```

## Claude Desktop Integration

Replace direct command execution with `run-mcp` in your Claude Desktop configuration:

### Before (Direct Execution)
```json
{
  "mcpServers": {
    "sqlite": {
      "command": "uvx",
      "args": ["mcp-server-sqlite", "--db-path", "/Users/me/data/db.sqlite"]
    },
    "filesystem": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "/Users/me/Documents"]
    }
  }
}
```

### After (Container Execution)
```json
{
  "mcpServers": {
    "sqlite": {
      "command": "run-mcp",
      "args": ["uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"],
      "env": {
        "MCP_DATA_DIR": "/Users/me/data"
      }
    },
    "filesystem": {
      "command": "run-mcp",
      "args": ["npx", "@modelcontextprotocol/server-filesystem", "/data"],
      "env": {
        "MCP_DATA_DIR": "/Users/me/Documents"
      }
    }
  }
}
```

## Configuration

Configure `run-mcp` using environment variables:

### Container Images
```bash
export MCP_NODEJS_IMAGE="ghcr.io/modelcontextprotocol/mcp-nodejs:node22"
export MCP_PYTHON_IMAGE="ghcr.io/modelcontextprotocol/mcp-python:python3.12"
```

### Data Directory
```bash
# Default: $HOME
export MCP_DATA_DIR="/path/to/your/data"
```

### Container Runtime Override
```bash
# Auto-detected by default
export MCP_CONTAINER_RUNTIME="podman"  # or docker, nerdctl, finch
```

### Environment Variable Passthrough
```bash
# Pass additional environment variables (comma-separated)
export MCP_PASSTHROUGH_ENV="CUSTOM_VAR,ANOTHER_VAR"
```

## Environment Variable Security

`run-mcp` uses an allowlist approach for environment variable passthrough:

### Allowed Prefixes
- `AWS_*` - AWS credentials and configuration
- `OPENAI_*` - OpenAI API keys and settings
- `ANTHROPIC_*` - Anthropic API keys and settings
- `AZURE_*` - Azure credentials and configuration
- `GOOGLE_*` - Google Cloud credentials and configuration
- `MCP_*` - MCP-specific configuration
- `HF_*` - Hugging Face tokens
- `REPLICATE_*` - Replicate API tokens
- `COHERE_*` - Cohere API keys

### Allowed Exact Matches
- `GITHUB_TOKEN` - GitHub personal access tokens
- `GITLAB_TOKEN` - GitLab access tokens
- `DATABASE_URL` - Database connection strings
- `REDIS_URL` - Redis connection strings

### Custom Variables
Use `MCP_PASSTHROUGH_ENV` to pass additional variables:
```bash
export MCP_PASSTHROUGH_ENV="MY_API_KEY,CUSTOM_CONFIG"
```

## Volume Mounts

`run-mcp` automatically mounts directories for data and credentials:

### Data Mount
- Host: `$MCP_DATA_DIR` (default: `$HOME`)
- Container: `/data`
- Mode: Read-write

### Credential Mounts (Read-only)
- `~/.aws` → `/home/mcp/.aws` (AWS credentials)
- `~/.config` → `/home/mcp/.config` (General configuration)
- `~/.ssh` → `/home/mcp/.ssh` (SSH keys, Linux only)

## Supported Container Runtimes

`run-mcp` detects container runtimes in this priority order:

1. **Docker** - Most common, works with Docker Desktop
2. **Podman** - Docker-compatible, rootless containers
3. **nerdctl** - containerd-based Docker alternative
4. **Finch** - AWS's container runtime
5. **lima nerdctl** - macOS-specific via Lima VM

## Supported Languages

### Node.js
Detected from commands: `npx`, `node`, `yarn`, `tsx`, `npm`
- Container: `ghcr.io/modelcontextprotocol/mcp-nodejs:node22`
- Includes: npm, yarn, TypeScript support

### Python
Detected from commands: `uvx`, `python`, `python3`, `uv`, `pip`, `pip3`
- Container: `ghcr.io/modelcontextprotocol/mcp-python:python3.12`
- Includes: uv package manager, common development tools

## Troubleshooting

### Container Runtime Not Found
```bash
# Check available runtimes
run-mcp info

# Install Docker Desktop (recommended)
# Or install alternative: podman, nerdctl, finch
```

### Permission Denied
```bash
# Ensure user is in docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker

# Or use rootless containers with Podman
```

### Data Directory Issues
```bash
# Check data directory permissions
run-mcp config

# Set custom data directory
export MCP_DATA_DIR="/path/to/accessible/directory"
```

### Environment Variables Not Passed
```bash
# Check current configuration
run-mcp config

# Add custom variables
export MCP_PASSTHROUGH_ENV="YOUR_VAR,ANOTHER_VAR"
```

### Container Image Pull Issues
```bash
# Test container runtime directly
docker pull ghcr.io/modelcontextprotocol/mcp-nodejs:node22

# Use custom images
export MCP_NODEJS_IMAGE="your-registry/nodejs-image:tag"
```

## Examples

### AWS MCP Server
```bash
# Set up AWS credentials
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
export AWS_REGION="us-east-1"

# Run AWS MCP server
run-mcp uvx awslabs.aws-api-mcp-server@latest
```

### File System Server with Custom Data Directory
```bash
# Set custom data directory
export MCP_DATA_DIR="/Users/me/projects"

# Run filesystem server
run-mcp npx @modelcontextprotocol/server-filesystem /data
```

### Memory Server with Custom Image
```bash
# Use custom Node.js image
export MCP_NODEJS_IMAGE="my-registry/custom-nodejs:latest"

# Run memory server
run-mcp npx @modelcontextprotocol/server-memory
```

## Development

### Building
```bash
# Build for current platform
make build-run-mcp

# Build for all platforms
make build-run-mcp-all

# Test binary
make test-run-mcp
```

### Testing
```bash
# Run tests
go test ./cmd/run-mcp/...

# Test with real containers
run-mcp info
run-mcp config
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.