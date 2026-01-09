# Mount Options

By default, run-mcp containers have no access to your host filesystem. Use `MCP_MOUNT` to grant explicit access.

## Basic Syntax

```
MCP_MOUNT=<host-path>:<container-path>[:<options>]
```

## Examples

### Simple Mount

```json
{
  "env": {
    "MCP_MOUNT": "~/data:/data"
  }
}
```

Host `~/data` is accessible at `/data` inside the container.

### Read-Only Mount

```json
{
  "env": {
    "MCP_MOUNT": "~/Documents:/docs:ro"
  }
}
```

The container can read but not write to the mounted directory.

### Multiple Mounts

Separate multiple mounts with commas:

```json
{
  "env": {
    "MCP_MOUNT": "~/data:/data,~/.config/myapp:/config:ro"
  }
}
```

### Credentials Mount

For AWS credentials:

```json
{
  "env": {
    "MCP_MOUNT": "~/.aws:/home/mcp/.aws:ro"
  }
}
```

For SSH keys:

```json
{
  "env": {
    "MCP_MOUNT": "~/.ssh:/home/mcp/.ssh:ro"
  }
}
```

## Path Expansion

- `~` expands to your home directory
- Relative paths are relative to your home directory
- Absolute paths work as expected

## Container Paths

The container user is `mcp` with home directory `/home/mcp`.

Common container paths:
- `/data` — General data access
- `/home/mcp/.aws` — AWS credentials
- `/home/mcp/.ssh` — SSH keys
- `/home/mcp/.config` — Application config

## Security Best Practices

### Principle of Least Privilege

Only mount what the server needs:

```json
{
  "env": {
    "MCP_MOUNT": "~/projects/myproject:/project:ro"
  }
}
```

Not:

```json
{
  "env": {
    "MCP_MOUNT": "~:/home/mcp"
  }
}
```

### Use Read-Only When Possible

If the server only needs to read files, use `:ro`:

```json
{
  "env": {
    "MCP_MOUNT": "~/Documents:/docs:ro"
  }
}
```

### Separate Credentials

Mount credentials separately with read-only access:

```json
{
  "env": {
    "MCP_MOUNT": "~/data:/data,~/.aws:/home/mcp/.aws:ro"
  }
}
```

## Windows Paths

On Windows, use forward slashes or escape backslashes:

```json
{
  "env": {
    "MCP_MOUNT": "C:/Users/me/data:/data"
  }
}
```

## Troubleshooting

### Permission Denied

The container runs as UID 1000. Ensure mounted directories are readable:

```bash
chmod -R a+rX ~/data
```

### Mount Not Working

Check the path exists on the host:

```bash
ls -la ~/data
```

### Changes Not Visible

Ensure you're mounting the correct path and the container path matches what the MCP server expects.