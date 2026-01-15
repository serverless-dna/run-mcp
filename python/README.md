# Python MCP Server Container

A containerized Python environment for running Model Context Protocol (MCP) servers with uv package management.

## Features

- **Python 3.12-slim**: Lightweight base image with Python 3.12
- **uv Package Manager**: Fast Python package installation and dependency resolution
- **MCP SDK**: Pre-installed MCP SDK for Python server development
- **Virtual Environment**: Isolated Python environment for dependencies
- **Security**: Runs as non-root user (UID 1000)
- **Multi-architecture**: Supports AMD64 and ARM64 architectures

## Quick Start

### Running a Python MCP Server

```bash
# Run the sample server included in the container
docker run --rm -i ghcr.io/serverless-dna/mcp-python:python3.12 python server.py

# Run an MCP server from your local directory
docker run --rm -i -v "$(pwd):/data" ghcr.io/serverless-dna/mcp-python:python3.12 python /data/my_server.py

# Install and run an MCP server using uvx
docker run --rm -i ghcr.io/serverless-dna/mcp-python:python3.12 uvx mcp-server-example
```

### Volume Mounting

Mount your Python MCP server code to `/data`:

```bash
docker run --rm -i \
  -v "$(pwd):/data" \
  ghcr.io/serverless-dna/mcp-python:python3.12 \
  python /data/server.py
```

### Installing Dependencies

If your server has dependencies, you can install them at runtime:

```bash
# Using uv (recommended)
docker run --rm -i \
  -v "$(pwd):/data" \
  ghcr.io/serverless-dna/mcp-python:python3.12 \
  sh -c "cd /data && uv pip install -r requirements.txt && python server.py"

# Using pip
docker run --rm -i \
  -v "$(pwd):/data" \
  ghcr.io/serverless-dna/mcp-python:python3.12 \
  sh -c "cd /data && pip install -r requirements.txt && python server.py"
```

## Configuration with Claude Desktop

Add to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "my-python-server": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-v", "/path/to/your/server:/data",
        "ghcr.io/serverless-dna/mcp-python:python3.12",
        "python", "/data/server.py"
      ]
    }
  }
}
```

## Environment Variables

- `PYTHONUNBUFFERED=1`: Ensures unbuffered output for MCP protocol
- `PYTHONDONTWRITEBYTECODE=1`: Prevents .pyc file creation
- `VIRTUAL_ENV=/app/venv`: Virtual environment path
- `PATH=/app/venv/bin:$PATH`: Includes virtual environment in PATH

## Development

### Building Locally

```bash
# Build the container
make build-python

# Validate with hadolint
make validate-dockerfiles
```

### Testing the Container

```bash
# Test basic functionality
docker run --rm ghcr.io/serverless-dna/mcp-python:python3.12 python --version

# Test the sample server
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test", "version": "1.0.0"}}}' | \
docker run --rm -i ghcr.io/serverless-dna/mcp-python:python3.12 python server.py

# Test uv functionality
docker run --rm ghcr.io/serverless-dna/mcp-python:python3.12 uv --version
```

## Troubleshooting

### Common Issues

**Container exits immediately:**
- Ensure you're using the `-i` flag for interactive mode
- Check that your Python script is executable and has proper shebang

**Permission denied errors:**
- The container runs as UID 1000 by default
- Ensure your mounted volumes have appropriate permissions

**Module not found errors:**
- Install dependencies using `uv pip install` or `pip install`
- Ensure your Python path is correct within the container

**MCP protocol issues:**
- Verify your server implements the MCP protocol correctly
- Check that stdin/stdout are not buffered (PYTHONUNBUFFERED=1 is set)
- Ensure no debug output is sent to stdout (use stderr for logging)

### Debugging

```bash
# Get a shell in the container
docker run --rm -it ghcr.io/serverless-dna/mcp-python:python3.12 sh

# Check Python environment
docker run --rm ghcr.io/serverless-dna/mcp-python:python3.12 python -c "import sys; print(sys.path)"

# List installed packages
docker run --rm ghcr.io/serverless-dna/mcp-python:python3.12 uv pip list
```

## Image Tags

- `python3.12`: Latest Python 3.12 build
- `python3.12-YYYYMMDD-<commit>`: Specific date and commit build
- `latest`: Latest stable build

## Security

- Runs as non-root user (UID 1000)
- Minimal base image (python:3.12-slim)
- No unnecessary packages installed
- Virtual environment isolation
- Read-only filesystem recommended for production use