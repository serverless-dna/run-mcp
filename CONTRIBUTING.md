# Contributing to MCP Container Images

## Adding New Language Runtimes

### Prerequisites
- Docker installed and configured
- GitHub account with repository access
- Basic understanding of MCP (Model Context Protocol)

### Step-by-Step Process

#### 1. Create Language Directory Structure
```bash
mkdir {language-name}/
cd {language-name}/
```

Required files:
- `Dockerfile` - Multi-stage build configuration
- `entrypoint.sh` - Container entrypoint script
- `README.md` - Language-specific documentation
- Configuration file (e.g., `package.json`, `pyproject.toml`, `go.mod`)

#### 2. Dockerfile Requirements

Your Dockerfile MUST:
- Use multi-stage builds (build + runtime stages)
- Run as non-root user (UID 1000)
- Support multi-architecture (AMD64 + ARM64)
- Include language-specific package manager
- Install MCP SDK for the language
- Be smaller than 400MB final image size
- Pass hadolint linting

Example structure:
```dockerfile
# Build stage
FROM {base-image} AS builder
# Install build dependencies
# Compile/prepare application

# Runtime stage  
FROM {base-image}-slim AS runtime
# Copy artifacts from builder
# Set up non-root user
# Configure entrypoint
```

#### 3. Entrypoint Script Requirements

Your entrypoint script MUST:
- Accept MCP transport configuration (stdio/HTTP)
- Support volume mounts at `/app/mcp-server`
- Provide health check endpoint
- Log startup information clearly
- Handle graceful shutdown

Template:
```bash
#!/bin/bash
set -euo pipefail

SCRIPT_PATH=${1:-"main.{ext}"}
TRANSPORT=${MCP_TRANSPORT:-"stdio"}
PORT=${MCP_PORT:-{default-port}}

# Language-specific setup
cd /app/mcp-server
exec {runtime} "$SCRIPT_PATH" --transport="$TRANSPORT" --port="$PORT" "${@:2}"
```

#### 4. Update Build System

1. **Add to change detection** (`scripts/detect-changes.sh`):
```bash
NEWLANG_CHANGED=$(git diff --name-only HEAD~1 HEAD | grep "^{language-name}/" || echo "")
echo "{language-name}-changed=${NEWLANG_CHANGED:+true}" >> $GITHUB_OUTPUT
```

2. **Add to GitHub Actions** (`.github/workflows/build-containers.yml`):
```yaml
- container: {language-name}
  platform: linux/amd64,linux/arm64
  context: ./{language-name}
```

3. **Update requirements and design documents** with new language specifications

#### 5. Testing Requirements

Create tests for:
- Container builds successfully
- Multi-architecture support works
- MCP protocol compliance
- Volume mounting functionality
- Health check endpoints
- Security best practices

#### 6. Documentation

Your README.md should include:
- Quick start guide
- Volume mounting examples
- MCP transport configuration
- Troubleshooting section
- Language-specific considerations

### Quality Standards

All contributions must meet:
- **Security**: Non-root user, minimal attack surface
- **Performance**: Build time < 5 minutes, image size targets
- **Reliability**: Proper error handling, graceful shutdown
- **Compatibility**: Multi-architecture support
- **Standards**: MCP protocol compliance

### Review Process

1. Create feature branch: `feature/add-{language}-runtime`
2. Implement all required components
3. Test locally with provided test scripts
4. Submit pull request with:
   - Description of language runtime added
   - Test results and build logs
   - Updated documentation
5. Address review feedback
6. Merge after approval

### Getting Help

- Check existing language implementations for examples
- Review MCP protocol documentation
- Ask questions in GitHub issues
- Join community discussions

### Maintenance Responsibilities

When adding a new runtime, you commit to:
- Keeping base images updated
- Responding to security vulnerabilities
- Maintaining MCP SDK compatibility
- Supporting community questions

Thank you for contributing to the MCP ecosystem!