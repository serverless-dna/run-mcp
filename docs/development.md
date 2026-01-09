# Development

Guide for building, testing, and contributing to run-mcp.

**Note:** Primary development environment has been on Windows using WSL2 so dev setup is in an Ubuntu Linux style right now.

## Prerequisites

- Go 1.21+
- Docker or Podman
- Make
- Git

## Clone the Repository

```bash
git clone https://github.com/serverless-dna/run-mcp.git
cd run-mcp
```
## Setup Dev Environment

```bash
make setup-dev
```

## Building

### Build for Current Platform

```bash
make build-run-mcp
```

Binary output: `build/run-mcp`

### Build for All Platforms

```bash
make build-run-mcp-all
```

Outputs:
- `build/run-mcp-linux-amd64`
- `build/run-mcp-linux-arm64`
- `build/run-mcp-darwin-amd64`
- `build/run-mcp-darwin-arm64`
- `build/run-mcp-windows-amd64.exe`

### Build Container Images

```bash
make build
```

Builds both Python and Node.js images.

## Testing

### Run All Tests

```bash
make test
```

### Run Specific Tests

```bash
# Go tests
make test-run-mcp

# Container tests
make test-containers

# Script tests
make test-scripts
```

### Test Manually

```bash
# Build and test
make build-run-mcp
./build/run-mcp --version
./build/run-mcp uvx mcp-server-sqlite --help
```

## Project Structure

```
run-mcp/
├── cmd/
│   └── run-mcp/          # Go source code
│       ├── main.go
│       ├── config.go
│       ├── runtime.go
│       └── ...
├── python/               # Python container image
│   └── Dockerfile
├── nodejs/               # Node.js container image
│   └── Dockerfile
├── scripts/              # Build and utility scripts
├── tests/                # Test files
├── docs/                 # Documentation
├── install/              # install scripts
└── Makefile
```

## Making Changes

### Code Style

- Go: Follow standard Go conventions, run `go fmt`
- Shell: Use shellcheck for scripts
- Dockerfiles: Use hadolint

### Linting

```bash
make lint
```

### Pre-Commit Checks

```bash
make ci
```

Runs linting, tests, and validation.

## Container Images

### Build Images Locally

```bash
# Latest versions
make build

# All supported versions
make build-matrix
```

### Push Images

Requires GitHub Container Registry authentication:

```bash
export GITHUB_TOKEN=your_token
make login
make push
```

## Release Process

Releases are automated via GitHub Actions when a tag is pushed:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The workflow:
1. Builds all platform binaries
2. Builds container images
3. Creates GitHub release with assets
4. Updates Homebrew formula

### Manual Release

```bash
make release VERSION=1.0.0
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes and test: `make ci`
4. Commit with clear messages
5. Push and open a Pull Request

### Commit Messages

Follow conventional commits:

```
feat: add support for custom images
fix: handle spaces in mount paths
docs: update installation instructions
chore: update dependencies
```

### Pull Request Guidelines

- Include tests for new features
- Update documentation if needed
- Ensure CI passes
- Keep changes focused and atomic

## Architecture

### run-mcp Binary

The Go binary:
1. Parses command and environment variables
2. Detects container runtime
3. Builds container run command with appropriate flags
4. Executes container with stdio passthrough

### Container Images

Minimal images with:
- Runtime (Python/Node.js)
- Package manager (uv/npm)
- Non-root user `mcp` (UID 1000)

### Security Model

- Containers run as non-root
- No host filesystem access by default
- Explicit mounts via `MCP_MOUNT`
- Allowlist-based environment variable filtering (only specific patterns like `AWS_*`, `OPENAI_*` pass through)
- Custom variables via `MCP_PASSTHROUGH_ENV`

## Getting Help

- [GitHub Issues](https://github.com/serverless-dna/run-mcp/issues)
- [Discussions](https://github.com/serverless-dna/run-mcp/discussions)