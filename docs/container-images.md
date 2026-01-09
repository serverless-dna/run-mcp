# Container Images

run-mcp uses pre-built container images for Python and Node.js MCP servers.

## Available Images

| Image | Runtime | Registry |
|-------|---------|----------|
| `run-mcp-python` | Python (uvx) | `ghcr.io/serverless-dna/run-mcp-python` |
| `run-mcp-nodejs` | Node.js (npx) | `ghcr.io/serverless-dna/run-mcp-nodejs` |

## Tags

### Latest Stable

```
ghcr.io/serverless-dna/run-mcp-python:latest
ghcr.io/serverless-dna/run-mcp-nodejs:latest
```

### Specific Versions

Python:
```
ghcr.io/serverless-dna/run-mcp-python:python3.12
ghcr.io/serverless-dna/run-mcp-python:python3.11
ghcr.io/serverless-dna/run-mcp-python:python3.10
```

Node.js:
```
ghcr.io/serverless-dna/run-mcp-nodejs:node22
ghcr.io/serverless-dna/run-mcp-nodejs:node20
ghcr.io/serverless-dna/run-mcp-nodejs:node18
```

## Auto-Detection

run-mcp automatically selects the correct image based on the command:

| Command | Image Selected |
|---------|----------------|
| `uvx` | `run-mcp-python:latest` |
| `npx` | `run-mcp-nodejs:latest` |

## Custom Images

Override the default images with environment variables:

```json
{
  "env": {
    "MCP_IMAGE_PYTHON": "ghcr.io/serverless-dna/run-mcp-python:python3.11"
  }
}
```

Or for Node.js:

```json
{
  "env": {
    "MCP_IMAGE_NODE": "ghcr.io/serverless-dna/run-mcp-nodejs:node20"
  }
}
```

## Image Contents

### Python Image

- Python 3.12 (or specified version)
- uv package manager
- uvx for running Python packages
- Non-root user `mcp` (UID 1000)

### Node.js Image

- Node.js 22 LTS (or specified version)
- npm and npx
- Non-root user `mcp` (UID 1000)

## Building Custom Images

If you need additional dependencies, create a custom Dockerfile:

```dockerfile
FROM ghcr.io/serverless-dna/run-mcp-python:latest

# Add system dependencies
RUN apt-get update && apt-get install -y \
    libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# Add Python packages
RUN pip install psycopg2-binary
```

Build and use:

```bash
docker build -t my-custom-mcp-python .
```

```json
{
  "env": {
    "MCP_IMAGE_PYTHON": "my-custom-mcp-python:latest"
  }
}
```

## Multi-Architecture Support

Images are built for:
- `linux/amd64` (Intel/AMD)
- `linux/arm64` (Apple Silicon, ARM servers)

The correct architecture is selected automatically.

## Image Updates

Images are automatically rebuilt when:
- New Python or Node.js versions are released
- Security updates are available
- Base image updates are published

Check the [GitHub Packages page](https://github.com/orgs/serverless-dna/packages) for the latest versions.