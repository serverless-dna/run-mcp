# Configuration Guide

run-mcp is configured through environment variables in your MCP client config (e.g., Claude Desktop).

## Environment Variables

### MCP_MOUNT

Mount host directories into the container. Format: `<host-path>:<container-path>[:<options>]`

```json
{
  "env": {
    "MCP_MOUNT": "~/data:/data"
  }
}
```

Multiple mounts separated by commas:
```json
{
  "env": {
    "MCP_MOUNT": "~/data:/data,~/.config:/config:ro"
  }
}
```

Options:
- `ro` — Read-only access
- `rw` — Read-write access (default)

See [mounts.md](./mounts.md) for detailed mount configuration.

### MCP_RUNTIME

Override the auto-detected container runtime.

```json
{
  "env": {
    "MCP_RUNTIME": "podman"
  }
}
```

Supported values: `docker`, `podman`, `nerdctl`, `finch`

### MCP_IMAGE_PYTHON

Override the Python container image.

```json
{
  "env": {
    "MCP_IMAGE_PYTHON": "ghcr.io/serverless-dna/run-mcp-python:latest"
  }
}
```

### MCP_IMAGE_NODE

Override the Node.js container image.

```json
{
  "env": {
    "MCP_IMAGE_NODE": "ghcr.io/serverless-dna/run-mcp-nodejs:latest"
  }
}
```

### MCP_DEBUG

Enable debug logging.

```json
{
  "env": {
    "MCP_DEBUG": "true"
  }
}
```

### MCP_PASSTHROUGH_ENV

Pass additional custom environment variables to the container.

```json
{
  "env": {
    "MCP_PASSTHROUGH_ENV": "MY_API_KEY,MY_SECRET",
    "MY_API_KEY": "xxx",
    "MY_SECRET": "yyy"
  }
}
```

Comma-separated list of variable names to pass through.

## Passing Environment Variables to the Server

Only specific environment variable patterns are passed through to the container for security. System variables like `PATH`, `HOME`, `USER` are filtered out.

### Automatic Passthrough

These patterns are automatically passed to the container:

| Pattern | Examples |
|---------|----------|
| `AWS_*` | `AWS_REGION`, `AWS_PROFILE`, `AWS_ACCESS_KEY_ID` |
| `OPENAI_*` | `OPENAI_API_KEY` |
| `ANTHROPIC_*` | `ANTHROPIC_API_KEY` |
| `AZURE_*` | `AZURE_OPENAI_API_KEY` |
| `GOOGLE_*` | `GOOGLE_API_KEY` |
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
  "env": {
    "MCP_PASSTHROUGH_ENV": "MY_API_KEY,CUSTOM_CONFIG",
    "MY_API_KEY": "your-api-key",
    "CUSTOM_CONFIG": "config-value"
  }
}
```

Variables listed in `MCP_PASSTHROUGH_ENV` (comma-separated) will be passed to the container along with the automatic patterns.

## Command Line Options

```bash
run-mcp [options] <command> [args...]
```

### --version

Print version information.

```bash
run-mcp --version
```

### --help

Show help message.

```bash
run-mcp --help
```

### --runtime

Override container runtime (same as MCP_RUNTIME).

```bash
run-mcp --runtime podman uvx mcp-server-sqlite
```

## Config File Location

Claude Desktop config file locations:

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux:** `~/.config/Claude/claude_desktop_config.json`