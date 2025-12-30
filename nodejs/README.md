# Node.js MCP Container

A Docker container image optimized for running MCP (Model Context Protocol) servers in Node.js environments. This container provides a secure, lightweight runtime with support for both CommonJS and ES modules, TypeScript compilation, and the MCP SDK.

## Features

- **Node.js LTS**: Uses the current Node.js LTS version on Alpine Linux
- **Package Managers**: Includes both npm and yarn
- **Module Systems**: Supports both CommonJS and ES modules
- **TypeScript**: Built-in TypeScript compilation support
- **MCP SDK**: Pre-installed MCP SDK for server development
- **Security**: Runs as non-root user (UID 1000)
- **Multi-Architecture**: Supports AMD64 and ARM64 architectures
- **Stdio Transport**: Optimized for MCP stdio communication

## Quick Start

### Running Published MCP Servers

```bash
# Run filesystem MCP server
docker run -i --rm \
  -v ~/Documents:/data \
  ghcr.io/owner/mcp-nodejs:node22 \
  npx @modelcontextprotocol/server-filesystem /data

# Run memory MCP server
docker run -i --rm \
  ghcr.io/owner/mcp-nodejs:node22 \
  npx @modelcontextprotocol/server-memory
```

### Running Local MCP Servers

```bash
# Run local Node.js MCP server
docker run -i --rm \
  -v ~/my-mcp-server:/app \
  -w /app \
  ghcr.io/owner/mcp-nodejs:node22 \
  node server.js

# Run local TypeScript MCP server
docker run -i --rm \
  -v ~/my-mcp-server:/app \
  -w /app \
  ghcr.io/owner/mcp-nodejs:node22 \
  npx tsx server.ts
```

## Volume Mounting

The container supports several volume mount patterns:

| Host Path | Container Path | Purpose | Mode |
|-----------|----------------|---------|------|
| `~/data` | `/data` | User data directory | read-write |
| `~/my-server` | `/app` | MCP server source code | read-write |
| `~/.aws` | `/home/mcp/.aws` | AWS credentials | read-only |
| `~/.config` | `/home/mcp/.config` | Application configs | read-only |

### Examples

```bash
# Mount user data directory
docker run -i --rm -v ~/Documents:/data ghcr.io/owner/mcp-nodejs:node22 \
  npx @modelcontextprotocol/server-filesystem /data

# Mount source code for development
docker run -i --rm -v ~/my-server:/app -w /app ghcr.io/owner/mcp-nodejs:node22 \
  npm install && node server.js

# Mount credentials for AWS access
docker run -i --rm \
  -v ~/.aws:/home/mcp/.aws:ro \
  -v ~/data:/data \
  ghcr.io/owner/mcp-nodejs:node22 \
  npx aws-mcp-server
```

## MCP Transport Configuration

This container is optimized for **stdio transport**, which is the primary MCP communication method.

### Claude Desktop Integration

Configure Claude Desktop to use the containerized MCP server:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "/Users/me/Documents:/data",
        "ghcr.io/owner/mcp-nodejs:node22",
        "npx", "@modelcontextprotocol/server-filesystem", "/data"
      ]
    },
    "memory": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "ghcr.io/owner/mcp-nodejs:node22",
        "npx", "@modelcontextprotocol/server-memory"
      ]
    }
  }
}
```

### Important Notes

- **No TTY**: Do not use `-t` flag as it breaks stdio communication
- **Interactive**: Always use `-i` flag for stdin/stdout communication
- **Remove**: Use `--rm` to automatically clean up containers
- **Unbuffered I/O**: Container automatically configures unbuffered stdio

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NODE_ENV` | `production` | Node.js environment |
| `NPM_CONFIG_UPDATE_NOTIFIER` | `false` | Disable npm update notifications |
| `NPM_CONFIG_FUND` | `false` | Disable npm funding messages |
| `FORCE_COLOR` | `0` | Disable colored output for stdio |
| `NODE_OPTIONS` | `--unhandled-rejections=strict` | Node.js runtime options |

### Custom Environment Variables

```bash
# Pass environment variables to MCP server
docker run -i --rm \
  -e OPENAI_API_KEY="your-key" \
  -e DEBUG="mcp:*" \
  ghcr.io/owner/mcp-nodejs:node22 \
  npx my-mcp-server
```

## Development Workflow

### Local Development

1. **Create your MCP server**:
```bash
mkdir my-mcp-server
cd my-mcp-server
npm init -y
npm install @modelcontextprotocol/sdk
```

2. **Test in container**:
```bash
docker run -i --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 \
  node server.js
```

3. **TypeScript development**:
```bash
# Install TypeScript dependencies
docker run --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 \
  npm install --save-dev typescript @types/node

# Compile TypeScript
docker run --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 \
  npx tsc

# Run with tsx (TypeScript execution)
docker run -i --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 \
  npx tsx server.ts
```

### Package Installation

```bash
# Install dependencies in container
docker run --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 \
  npm install

# Using yarn
docker run --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 \
  yarn install
```

## Troubleshooting

### Common Issues

#### 1. **Container exits immediately**
```bash
# Check if your command is correct
docker run --rm ghcr.io/owner/mcp-nodejs:node22 node --version

# Ensure you're using -i for stdio transport
docker run -i --rm ghcr.io/owner/mcp-nodejs:node22 your-command
```

#### 2. **Permission denied on volume mounts**
```bash
# The container runs as UID 1000, ensure your files are accessible
sudo chown -R 1000:1000 ~/my-mcp-server

# Or run with user override (not recommended for production)
docker run -i --rm --user $(id -u):$(id -g) -v $(pwd):/app ghcr.io/owner/mcp-nodejs:node22
```

#### 3. **Module not found errors**
```bash
# Install dependencies first
docker run --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 npm install

# Check if you're in the right directory
docker run --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 ls -la
```

#### 4. **MCP communication issues**
```bash
# Verify stdio transport (no -t flag)
docker run -i --rm ghcr.io/owner/mcp-nodejs:node22 npx @modelcontextprotocol/server-memory

# Check for proper JSON-RPC communication
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | \
docker run -i --rm ghcr.io/owner/mcp-nodejs:node22 npx @modelcontextprotocol/server-memory
```

#### 5. **TypeScript compilation errors**
```bash
# Check TypeScript configuration
docker run --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 npx tsc --noEmit

# Use tsx for direct TypeScript execution
docker run -i --rm -v $(pwd):/app -w /app ghcr.io/owner/mcp-nodejs:node22 npx tsx server.ts
```

### Debug Mode

Enable debug logging for troubleshooting:

```bash
# Enable MCP debug logging
docker run -i --rm -e DEBUG="mcp:*" ghcr.io/owner/mcp-nodejs:node22 your-command

# Enable Node.js debug logging
docker run -i --rm -e NODE_DEBUG="*" ghcr.io/owner/mcp-nodejs:node22 your-command
```

### Container Inspection

```bash
# Check container contents
docker run --rm ghcr.io/owner/mcp-nodejs:node22 ls -la /app

# Check installed packages
docker run --rm ghcr.io/owner/mcp-nodejs:node22 npm list -g --depth=0

# Check Node.js and npm versions
docker run --rm ghcr.io/owner/mcp-nodejs:node22 node --version
docker run --rm ghcr.io/owner/mcp-nodejs:node22 npm --version
docker run --rm ghcr.io/owner/mcp-nodejs:node22 yarn --version
```

## Security Considerations

- Container runs as non-root user (UID 1000)
- Minimal Alpine Linux base image
- No unnecessary packages or services
- Read-only credential mounts recommended
- No network ports exposed (stdio transport only)

## Image Tags

| Tag | Description | Node.js Version |
|-----|-------------|-----------------|
| `node22` | Current LTS (recommended) | 22.x |
| `node20` | Previous LTS | 20.x |
| `latest` | Alias for current LTS | 22.x |

## Support

For issues specific to this container image, please check:

1. **Container logs**: Use `docker logs <container-id>` if running detached
2. **MCP protocol**: Ensure proper JSON-RPC communication over stdio
3. **File permissions**: Verify UID 1000 can access mounted volumes
4. **Dependencies**: Ensure all required npm packages are installed

For MCP protocol issues, refer to the [MCP specification](https://modelcontextprotocol.io/) and [MCP SDK documentation](https://github.com/modelcontextprotocol/typescript-sdk).