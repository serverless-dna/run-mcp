# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Generating changelog for version v1.0.2 (since v1.0.1)
## [v1.0.2] - 2026-01-05

### Changed
- Fix Drive mounting to containers (#10)

### Binary Releases
- Windows (AMD64): run-mcp-windows-amd64.exe
- macOS (Intel): run-mcp-darwin-amd64
- macOS (Apple Silicon): run-mcp-darwin-arm64
- Linux (AMD64): run-mcp-linux-amd64
- Linux (ARM64): run-mcp-linux-arm64

[v1.0.2]: https://github.com/modelcontextprotocol/mcp-container-images/releases/tag/vv1.0.2

## [1.0.0] - 2025-01-04

### Added
- Cross-platform `run-mcp` binary for Windows, macOS, and Linux (AMD64/ARM64)
- Automatic container runtime detection (Docker, Podman, nerdctl, Finch, lima)
- Language auto-detection for Node.js and Python from command arguments
- Secure environment variable passthrough with allowlist-based filtering (9 prefixes, 4 exact matches)
- Volume management with persistent and ephemeral modes
- Container images for Node.js (LTS) and Python (3.12) MCP servers
- Built-in diagnostics with `doctor`, `info`, `config`, and `list-images` commands
- Volume management subcommands: `list`, `clean`, `prune`, `inspect`
- Signal handling for graceful container shutdown
- Multi-architecture container support (AMD64/ARM64)
- Comprehensive test suite with Bats framework
- GitHub Actions workflows for automated building and publishing
- Makefile with complete development lifecycle targets

### Container Images
- `ghcr.io/owner/repo-nodejs:latest` - Node.js LTS with npm, yarn, TypeScript
- `ghcr.io/owner/repo-python:latest` - Python 3.12 with uv, pip, common tools
- Multi-version support with semantic tagging (e.g., `node22`, `python3.12`)
- Date-stamped builds for reproducibility

### Security Features
- Non-root container execution (UID 1000)
- Environment variable allowlist filtering:
  - Prefixes: `AWS_*`, `OPENAI_*`, `ANTHROPIC_*`, `AZURE_*`, `GOOGLE_*`, `MCP_*`, `HF_*`, `REPLICATE_*`, `COHERE_*`
  - Exact matches: `GITHUB_TOKEN`, `GITLAB_TOKEN`, `DATABASE_URL`, `REDIS_URL`
  - Custom variables via `MCP_PASSTHROUGH_ENV`
- Read-only credential directory mounting
- Configuration variables (`MCP_MOUNT`, `MCP_BIND_HOME`, `MCP_HOME_PATH`) excluded from container passthrough

### Documentation
- Complete README with installation and usage examples
- Claude Desktop integration guide
- Troubleshooting and configuration reference
- Contributing guidelines for new language runtimes
- Version support and retention policies

[1.0.0]: https://github.com/serverless-dna/run-mcp/releases/tag/v1.0.0