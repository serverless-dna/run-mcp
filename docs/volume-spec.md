# run-mcp Volume Mounting Specification

## Overview

This document specifies the volume mounting strategy for `run-mcp`. The design prioritizes security (minimal host access by default) and simplicity (automatic home directory management).

## Design Principles

1. **Secure by default** — No host filesystem access unless explicitly configured
2. **Automatic home management** — Per-server isolated volumes, transparent to user
3. **Explicit opt-in** — User declares all bind mounts via `MCP_MOUNT`
4. **No special cases** — Single mechanism for all mount types (credentials, data, etc.)

---

## Automatic Home Directory Volume

Each MCP server gets an isolated Docker named volume for `/home/mcp`. This provides:

- Package cache persistence (npx/uvx downloads survive restarts)
- Server-specific state isolation (no cross-server conflicts)
- Zero configuration required

### Volume Naming

Volume names are derived from the command arguments:

```
mcp-home-<sanitized-command>-<sanitized-first-arg>
```

#### Sanitization Rules

1. Take command + first non-flag argument
2. Convert to lowercase
3. Replace non-alphanumeric characters with dashes
4. Trim leading/trailing dashes
5. Prefix with `mcp-home-`

#### Examples

| Command | Volume Name |
|---------|-------------|
| `uvx mcp-server-sqlite --db-path /data/db.sqlite` | `mcp-home-uvx-mcp-server-sqlite` |
| `uvx awslabs.aws-api-mcp-server@latest` | `mcp-home-uvx-awslabs-aws-api-mcp-server-latest` |
| `npx @modelcontextprotocol/server-filesystem /data` | `mcp-home-npx-modelcontextprotocol-server-filesystem` |
| `npx @anthropic/mcp-server-memory` | `mcp-home-npx-anthropic-mcp-server-memory` |

### Implementation

```go
import (
    "regexp"
    "strings"
)

func sanitizeVolumeName(args []string) string {
    var parts []string
    
    for i, arg := range args {
        // Only use first two args (command + server identifier)
        if i >= 2 {
            break
        }
        // Stop at flags
        if strings.HasPrefix(arg, "-") {
            break
        }
        
        // Sanitize: lowercase, replace non-alphanumeric with dash
        sanitized := strings.ToLower(arg)
        sanitized = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(sanitized, "-")
        sanitized = strings.Trim(sanitized, "-")
        
        if sanitized != "" {
            parts = append(parts, sanitized)
        }
    }
    
    if len(parts) == 0 {
        return "mcp-home-default"
    }
    
    return "mcp-home-" + strings.Join(parts, "-")
}
```

---

## Home Directory Overrides

Power users can override automatic volume management:

| Variable | Effect |
|----------|--------|
| `MCP_BIND_HOME=true` | Use `~/.mcp/<volume-name>/` on host instead of Docker volume |
| `MCP_HOME_PATH=/path` | Use explicit host path for `/home/mcp` |

### Use Cases

- **Debugging** — Inspect server state on host filesystem
- **Backup** — Server data in user-accessible location
- **Shared state** — Multiple configs pointing to same home

### Implementation

```go
func getHomeMount(args []string) string {
    volumeName := sanitizeVolumeName(args)
    
    // Override: bind mount to ~/.mcp/<name>/
    if os.Getenv("MCP_BIND_HOME") == "true" {
        home := getHomeDir()
        mcpHome := filepath.Join(home, ".mcp", volumeName)
        
        // Ensure directory exists
        if err := os.MkdirAll(mcpHome, 0755); err != nil {
            // Log warning, fall back to volume
            return volumeName + ":/home/mcp"
        }
        
        return toContainerPath(mcpHome) + ":/home/mcp"
    }
    
    // Override: explicit path
    if customPath := os.Getenv("MCP_HOME_PATH"); customPath != "" {
        return toContainerPath(expandPath(customPath)) + ":/home/mcp"
    }
    
    // Default: named Docker volume
    return volumeName + ":/home/mcp"
}
```

---

## User-Specified Mounts (MCP_MOUNT)

All additional bind mounts are configured via `MCP_MOUNT`. There are no special-case variables for credentials, config, or data directories.

### Format

```
MCP_MOUNT=<src>:<dest>[:<opts>],<src>:<dest>[:<opts>],...
```

- Comma-separated list of mounts
- Each mount follows Docker `-v` syntax: `source:destination[:options]`
- Tilde (`~`) expansion supported for source paths
- Windows paths converted automatically

### Examples

```bash
# Single mount
MCP_MOUNT=~/Documents:/data

# Multiple mounts
MCP_MOUNT=~/Documents:/docs,~/Projects:/projects

# With options (read-only)
MCP_MOUNT=~/.aws:/home/mcp/.aws:ro

# Credentials + data
MCP_MOUNT=~/.aws:/home/mcp/.aws:ro,~/.config:/home/mcp/.config:ro,~/data:/data
```

### Implementation

```go
func expandPath(path string) string {
    if strings.HasPrefix(path, "~/") {
        home := getHomeDir()
        return filepath.Join(home, path[2:])
    }
    if strings.HasPrefix(path, "~\\") { // Windows
        home := getHomeDir()
        return filepath.Join(home, path[2:])
    }
    return path
}

func getHomeDir() string {
    if home := os.Getenv("HOME"); home != "" {
        return home
    }
    if home := os.Getenv("USERPROFILE"); home != "" { // Windows
        return home
    }
    return ""
}

func toContainerPath(hostPath string) string {
    if runtime.GOOS != "windows" {
        return hostPath
    }
    // Convert C:\Users\name to /c/Users/name (Docker Desktop format)
    if len(hostPath) >= 2 && hostPath[1] == ':' {
        drive := strings.ToLower(string(hostPath[0]))
        rest := strings.ReplaceAll(hostPath[2:], "\\", "/")
        return "/" + drive + rest
    }
    return strings.ReplaceAll(hostPath, "\\", "/")
}

func parseUserMounts() []string {
    mountList := os.Getenv("MCP_MOUNT")
    if mountList == "" {
        return nil
    }
    
    var mounts []string
    
    for _, m := range strings.Split(mountList, ",") {
        m = strings.TrimSpace(m)
        if m == "" {
            continue
        }
        
        // Split src:dest[:opts] - handle Windows paths with drive letters
        var parts []string
        if runtime.GOOS == "windows" && len(m) > 2 && m[1] == ':' {
            // Windows path like C:\path:/dest
            colonIdx := strings.Index(m[2:], ":")
            if colonIdx != -1 {
                parts = append(parts, m[:colonIdx+2])
                remainder := m[colonIdx+3:]
                parts = append(parts, strings.SplitN(remainder, ":", 2)...)
            } else {
                parts = []string{m}
            }
        } else {
            parts = strings.SplitN(m, ":", 3)
        }
        
        if len(parts) >= 2 {
            // Expand and convert source path
            parts[0] = toContainerPath(expandPath(parts[0]))
            m = strings.Join(parts, ":")
        }
        
        mounts = append(mounts, m)
    }
    
    return mounts
}
```

---

## Complete Volume Mount Assembly

```go
func getVolumeMounts(args []string) []string {
    var mounts []string
    
    // 1. Home directory (always present)
    mounts = append(mounts, "-v", getHomeMount(args))
    
    // 2. User-specified mounts (optional)
    for _, m := range parseUserMounts() {
        mounts = append(mounts, "-v", m)
    }
    
    return mounts
}
```

---

## Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_MOUNT` | (none) | Comma-separated bind mounts (`src:dest[:opts]`) |
| `MCP_BIND_HOME` | `false` | Use `~/.mcp/<name>/` instead of Docker volume |
| `MCP_HOME_PATH` | (none) | Explicit host path for `/home/mcp` |

---

## Usage Examples

### Minimal (no host access)

```json
{
  "mcpServers": {
    "memory": {
      "command": "run-mcp",
      "args": ["npx", "@modelcontextprotocol/server-memory"]
    }
  }
}
```

Container gets isolated home volume only. No host filesystem access.

### Filesystem server

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "run-mcp",
      "args": ["npx", "@modelcontextprotocol/server-filesystem", "/docs"],
      "env": {
        "MCP_MOUNT": "~/Documents:/docs:ro"
      }
    }
  }
}
```

### AWS server with credentials

```json
{
  "mcpServers": {
    "aws-api": {
      "command": "run-mcp",
      "args": ["uvx", "awslabs.aws-api-mcp-server"],
      "env": {
        "MCP_MOUNT": "~/.aws:/home/mcp/.aws:ro",
        "AWS_REGION": "us-east-1"
      }
    }
  }
}
```

### SQLite server with data directory

```json
{
  "mcpServers": {
    "sqlite": {
      "command": "run-mcp",
      "args": ["uvx", "mcp-server-sqlite", "--db-path", "/data/mydb.sqlite"],
      "env": {
        "MCP_MOUNT": "~/databases:/data"
      }
    }
  }
}
```

### Multiple mounts (credentials + data)

```json
{
  "mcpServers": {
    "dev-tools": {
      "command": "run-mcp",
      "args": ["uvx", "some-dev-server"],
      "env": {
        "MCP_MOUNT": "~/.aws:/home/mcp/.aws:ro,~/.config:/home/mcp/.config:ro,~/projects:/projects"
      }
    }
  }
}
```

### Debug mode (home on host filesystem)

```json
{
  "mcpServers": {
    "debug-server": {
      "command": "run-mcp",
      "args": ["uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"],
      "env": {
        "MCP_BIND_HOME": "true",
        "MCP_MOUNT": "~/data:/data"
      }
    }
  }
}
```

Home directory at `~/.mcp/mcp-home-uvx-mcp-server-sqlite/` for inspection.

---

## Volume Management Commands

Users can manage volumes with standard Docker commands:

```bash
# List MCP volumes
docker volume ls | grep mcp-home

# Inspect specific volume
docker volume inspect mcp-home-uvx-mcp-server-sqlite

# Remove specific server's volume (reset state)
docker volume rm mcp-home-uvx-mcp-server-sqlite

# Remove all MCP volumes (reset all servers)
docker volume ls -q | grep mcp-home | xargs docker volume rm

# Check volume disk usage
docker system df -v | grep mcp-home
```

---

## Security Considerations

1. **No default host access** — Only the isolated home volume is mounted by default
2. **Explicit opt-in** — All bind mounts require user configuration
3. **Read-only option** — Users can mount credentials as read-only (`:ro`)
4. **Isolated per-server** — Each server has separate home volume, no cross-contamination
5. **No credential auto-discovery** — `~/.aws`, `~/.config` not mounted automatically