# Troubleshooting

Common issues and solutions.

## Container Runtime Not Found

**Error:** `No container runtime found`

**Solution:** Install Docker, Podman, or another supported runtime:

- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- [Podman](https://podman.io/getting-started/installation)

Ensure the runtime is running:

```bash
docker ps
# or
podman ps
```

## Permission Denied

**Error:** `Permission denied` when accessing mounted files

**Cause:** The container runs as UID 1000. Your files may have different permissions.

**Solution:**

```bash
# Make files readable
chmod -R a+rX ~/data

# Or change ownership (if you're UID 1000)
ls -n ~/data  # Check your UID
```

## Mount Path Not Found

**Error:** `Mount path does not exist`

**Solution:** Ensure the host path exists:

```bash
ls -la ~/data
```

Create it if needed:

```bash
mkdir -p ~/data
```

## Server Fails to Start

**Symptoms:** Claude Desktop shows server as disconnected

**Debug steps:**

1. Enable debug logging:
   ```json
   {
     "env": {
       "MCP_DEBUG": "true"
     }
   }
   ```

2. Test manually:
   ```bash
   run-mcp uvx mcp-server-sqlite --help
   ```

3. Check container logs:
   ```bash
   docker logs <container-id>
   ```

## Image Pull Fails

**Error:** `Failed to pull image`

**Causes:**
- Network issues
- Registry authentication required

**Solution:**

```bash
# Test manually
docker pull ghcr.io/serverless-dna/run-mcp-python:latest

# If authentication required
docker login ghcr.io
```

## Server Timeout

**Symptoms:** Server starts but Claude Desktop times out

**Cause:** First run pulls container images which can be slow.

**Solution:** Pre-pull images:

```bash
docker pull ghcr.io/serverless-dna/run-mcp-python:latest
docker pull ghcr.io/serverless-dna/run-mcp-nodejs:latest
```

## Wrong Python/Node Version

**Symptoms:** MCP server requires specific runtime version

**Solution:** Use a specific image tag:

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

## Environment Variables Not Passed

**Symptoms:** Server doesn't see expected environment variables

**Remember:** Variables prefixed with `MCP_` are consumed by run-mcp, not passed to the container.

**Wrong:**
```json
{
  "env": {
    "MCP_AWS_REGION": "us-east-1"
  }
}
```

**Correct:**
```json
{
  "env": {
    "AWS_REGION": "us-east-1"
  }
}
```

## Windows-Specific Issues

### Path Format

Use forward slashes:

```json
{
  "env": {
    "MCP_MOUNT": "C:/Users/me/data:/data"
  }
}
```

### Docker Desktop Not Running

Ensure Docker Desktop is started before launching Claude Desktop.

### WSL Integration

If using WSL, ensure Docker Desktop has WSL integration enabled in settings.

## macOS-Specific Issues

### Docker Desktop Permissions

Grant Docker Desktop access to the directories you want to mount in Docker Desktop → Settings → Resources → File Sharing.

### Apple Silicon

Images support ARM64 natively. If you see slow performance, ensure you're not running x86 emulation.

## Still Stuck?

1. Check [GitHub Issues](https://github.com/serverless-dna/run-mcp/issues)
2. Open a new issue with:
   - Your OS and architecture
   - Container runtime and version
   - The MCP server you're trying to run
   - Full error message
   - Debug logs (with `MCP_DEBUG=true`)