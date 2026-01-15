# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

<<<<<<< HEAD
## [Unreleased]

### Fixed
- Fixed container test suite to align with simplified runtime dependency model
  - Updated Property 3 test to verify Python readiness without pre-installed MCP SDK
  - Updated Property 1 Python test to check uvx availability instead of pre-installed dependencies
  - Updated Property 14 Node.js test to verify /app directory writability for runtime installations
  - Fixed Property 15 Node.js test to use piped input instead of volume mounts for better portability
  - All 30 container tests now pass consistently across Docker and Podman

### Changed
- Added .gitattributes file to enforce consistent Unix line endings across all platforms
  - Ensures Makefile, shell scripts, and YAML files use LF line endings
  - Prevents CRLF issues in GitHub Actions and cross-platform development
- Enhanced Makefile linting to detect CRLF line endings
  - `make lint-makefile` now checks for Windows line endings in Makefile
  - `make lint-workflows` now checks for Windows line endings in workflow files
  - Provides helpful fix commands when CRLF is detected

## [v1.0.2] - 2026-01-05

### Fixed
- Fix MCP_DATA_DIR and credential directories not being mounted to containers (#6)
  - Added missing `volumeManager.GetVolumeMounts()` calls to container command builders
  - MCP_DATA_DIR now properly mounts to `/data` in containers as documented
  - Credential directories (~/.aws, ~/.config) now mount correctly to container home directory
  
- Fix MCP_MOUNT validation failing on Windows due to incorrect path conversion order (#7)
  - Path validation now occurs before Windows path conversion in `parseSingleMount`
  - Windows users can now use forward slashes in MCP_MOUNT paths (e.g., `C:/Users/...`)
  - Fixes "mount source path does not exist" errors on Windows with valid paths
=======
Generating changelog for version 1.0.3 (since v1.0.2)
## [1.0.3] - 2026-01-15

### Added
- Windows Install via winget for windows users (#19)
- add GitHub Actions workflow linting with actionlint (#18)
- Setup HomeBrew package publishing (#14)

### Fixed
- fix broken blog link (#22)
- add workflow for homebrew update and quality gates (#17)
- reset to PAT token for homebrew repo and resolve scripting error in makefile (#16)
- reset token secret for Homebrew Tap repo steps to standard GITHUB_TOKEN. (#15)

### Container Images
- Fix: Ensure no auto-mounts in containers occur (#29)

### Documentation
- Feat logo (#13)
- Feat logo (#12)

### Maintenance
- add github templates to help community raise issues or ideas (#23)
- Cleanup documentation (#21)
- fix release url reference (#20)

### Container Images
- Updated base images and dependencies
- Multi-architecture builds (AMD64/ARM64)
>>>>>>> main

### Binary Releases
- Windows (AMD64): run-mcp-windows-amd64.exe
- macOS (Intel): run-mcp-darwin-amd64
- macOS (Apple Silicon): run-mcp-darwin-arm64
- Linux (AMD64): run-mcp-linux-amd64
- Linux (ARM64): run-mcp-linux-arm64

<<<<<<< HEAD
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

=======
[1.0.3]: https://github.com/modelcontextprotocol/mcp-container-images/releases/tag/v1.0.3

## [v1.0.2] - 2026-01-05

### Fixed
- Fix MCP_DATA_DIR and credential directories not being mounted to containers (#6)
  - Added missing `volumeManager.GetVolumeMounts()` calls to container command builders
  - MCP_DATA_DIR now properly mounts to `/data` in containers as documented
  - Credential directories (~/.aws, ~/.config) now mount correctly to container home directory
  
- Fix MCP_MOUNT validation failing on Windows due to incorrect path conversion order (#7)
  - Path validation now occurs before Windows path conversion in `parseSingleMount`
  - Windows users can now use forward slashes in MCP_MOUNT paths (e.g., `C:/Users/...`)
  - Fixes "mount source path does not exist" errors on Windows with valid paths

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

>>>>>>> main
[1.0.0]: https://github.com/serverless-dna/run-mcp/releases/tag/v1.0.0