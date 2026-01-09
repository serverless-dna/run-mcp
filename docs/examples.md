# Examples

Common MCP server configurations with run-mcp.

## Filesystem Server

Read-only access to Documents:

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

Read-write access to a project:

```json
{
  "mcpServers": {
    "project-files": {
      "command": "run-mcp",
      "args": ["npx", "@modelcontextprotocol/server-filesystem", "/project"],
      "env": {
        "MCP_MOUNT": "~/projects/my-app:/project"
      }
    }
  }
}
```

## SQLite Server

```json
{
  "mcpServers": {
    "sqlite": {
      "command": "run-mcp",
      "args": ["uvx", "mcp-server-sqlite", "--db-path", "/data/mydb.sqlite"],
      "env": {
        "MCP_MOUNT": "~/data:/data"
      }
    }
  }
}
```

## Memory Server

No mounts needed — data persists in an isolated container volume:

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

## AWS MCP Servers

### AWS API Server

```json
{
  "mcpServers": {
    "aws-api": {
      "command": "run-mcp",
      "args": ["uvx", "awslabs.aws-api-mcp-server"],
      "env": {
        "MCP_MOUNT": "~/.aws:/home/mcp/.aws:ro",
        "AWS_REGION": "us-east-1",
        "AWS_PROFILE": "default"
      }
    }
  }
}
```

### AWS Documentation Server

```json
{
  "mcpServers": {
    "aws-docs": {
      "command": "run-mcp",
      "args": ["uvx", "awslabs.aws-documentation-mcp-server"],
      "env": {
        "MCP_MOUNT": "~/.aws:/home/mcp/.aws:ro"
      }
    }
  }
}
```

### Powertools for AWS MCP

```json
{
  "mcpServers": {
    "powertools": {
      "command": "run-mcp",
      "args": ["npx", "powertools-for-aws-mcp"]
    }
  }
}
```

## Git Server

```json
{
  "mcpServers": {
    "git": {
      "command": "run-mcp",
      "args": ["uvx", "mcp-server-git", "--repository", "/repo"],
      "env": {
        "MCP_MOUNT": "~/projects/my-repo:/repo"
      }
    }
  }
}
```

## Fetch Server

No mounts needed for web fetching:

```json
{
  "mcpServers": {
    "fetch": {
      "command": "run-mcp",
      "args": ["uvx", "mcp-server-fetch"]
    }
  }
}
```

## GitHub Server

```json
{
  "mcpServers": {
    "github": {
      "command": "run-mcp",
      "args": ["npx", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_your_token_here"
      }
    }
  }
}
```

## Brave Search Server

```json
{
  "mcpServers": {
    "brave-search": {
      "command": "run-mcp",
      "args": ["npx", "@modelcontextprotocol/server-brave-search"],
      "env": {
        "BRAVE_API_KEY": "your_api_key"
      }
    }
  }
}
```

## Multiple Servers

You can configure multiple servers, each isolated:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "run-mcp",
      "args": ["npx", "@modelcontextprotocol/server-filesystem", "/docs"],
      "env": {
        "MCP_MOUNT": "~/Documents:/docs:ro"
      }
    },
    "sqlite": {
      "command": "run-mcp",
      "args": ["uvx", "mcp-server-sqlite", "--db-path", "/data/notes.db"],
      "env": {
        "MCP_MOUNT": "~/data:/data"
      }
    },
    "memory": {
      "command": "run-mcp",
      "args": ["npx", "@modelcontextprotocol/server-memory"]
    }
  }
}
```

Each server runs in its own isolated container with only the access you've granted.