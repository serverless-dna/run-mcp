# Implementation Plan: MCP Container Images

## Overview

This implementation plan creates a system for building and distributing language-specific container images for MCP servers. The approach focuses on automated GitHub Actions workflows with selective building based on file changes, targeting Node.js and Python (uv) runtime environments.

## Tasks

- [x] 1. Set up repository structure and change detection
  - Create directory structure for nodejs/ and python/ containers
  - Create scripts/ directory with detect-changes.sh script
  - Set up basic project documentation
  - _Requirements: 4.3, 4.5_

- [-] 2. Implement change detection system
  - [x] 2.1 Create detect-changes.sh script with robust error handling
    - Write git diff logic with shallow clone and first commit handling
    - Set GitHub Actions outputs for conditional builds
    - Handle edge cases like deleted directories and merge conflicts
    - Add comprehensive logging for debugging build decisions
    - _Requirements: 4.1, 4.3, 4.4_

  - [ ]* 2.2 Write property test for change detection
    - **Property 9: Git-Based Change Detection**
    - **Validates: Requirements 4.3**

  - [ ]* 2.3 Write unit tests for change detection script
    - Test various git diff scenarios including shallow clones
    - Test output generation for GitHub Actions
    - Test error handling and fallback mechanisms
    - _Requirements: 4.3, 4.5_

- [ ] 3. Create Node.js container image
  - [ ] 3.1 Create Node.js Dockerfile with multi-stage build
    - Use node:lts-alpine as base image
    - Install npm and yarn package managers
    - Set up non-root user and security best practices
    - Configure multi-architecture build support
    - _Requirements: 1.1, 5.5, 6.1, 6.2, 9.1, 9.2_

  - [ ] 3.2 Create Node.js entrypoint script
    - Create simple passthrough entrypoint (exec "$@")
    - Ensure unbuffered stdio for MCP protocol
    - Set up UID 1000 for volume permission compatibility
    - _Requirements: 1.5, 5.3, 10.6, 10.7, 10.8, 10.9_

  - [ ] 3.3 Create Node.js package.json and configuration
    - Support both CommonJS and ES modules
    - Include TypeScript compilation capabilities
    - Include MCP SDK dependencies
    - Set up dependency caching
    - _Requirements: 6.3, 6.4, 6.5, 11.1_

  - [ ] 3.4 Create Node.js container README.md
    - Document usage examples and volume mounting
    - Explain MCP transport configuration
    - Provide troubleshooting guide
    - _Requirements: 10.2, 10.4_

  - [ ] 3.5 Add Dockerfile linting with hadolint
    - Integrate hadolint validation in build process
    - Configure linting rules for security and best practices
    - _Requirements: 8.4_

  - [ ]* 3.6 Write property tests for Node.js container
    - **Property 1: Language-Specific Container Creation**
    - **Property 13: LTS Version Compliance**
    - **Property 15: Node.js Module System Support**
    - **Validates: Requirements 1.1, 6.1, 6.4, 6.5**

- [ ] 4. Create Python container image
  - [ ] 4.1 Create Python Dockerfile with multi-stage build
    - Use python:3.12-slim as base image
    - Install uv package manager
    - Set up virtual environment and non-root user
    - Configure multi-architecture build support
    - _Requirements: 1.2, 5.5, 7.1, 7.2, 9.1, 9.2_

  - [ ] 4.2 Create Python entrypoint script
    - Create simple passthrough entrypoint (exec "$@")
    - Ensure unbuffered stdio for MCP protocol
    - Set up UID 1000 for volume permission compatibility
    - _Requirements: 1.5, 5.3, 10.6, 10.7, 10.8, 10.9_

  - [ ] 4.3 Create Python pyproject.toml and configuration
    - Configure uv for fast dependency resolution
    - Include common Python development tools
    - Include MCP SDK dependencies
    - Set up virtual environment isolation
    - _Requirements: 7.3, 7.4, 7.5, 11.1_

  - [ ] 4.4 Create Python container README.md
    - Document usage examples and volume mounting
    - Explain MCP transport configuration
    - Provide troubleshooting guide
    - _Requirements: 10.2, 10.4_

  - [ ]* 4.5 Write property tests for Python container
    - **Property 1: Language-Specific Container Creation**
    - **Property 13: LTS Version Compliance**
    - **Property 16: Python Environment Management**
    - **Validates: Requirements 1.2, 7.1, 7.4, 7.5**

- [ ] 5. Checkpoint - Ensure container images build locally
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 6. Create GitHub Actions workflow
  - [ ] 6.1 Create build-containers.yml workflow file
    - Set up triggers for push to main and pull requests
    - Add workflow_dispatch for manual triggers
    - Configure GITHUB_TOKEN authentication for GHCR
    - Implement conditional job execution based on change detection
    - _Requirements: 2.1, 3.3, 4.1, 4.2_

  - [ ] 6.2 Implement multi-architecture build matrix
    - Configure buildx for AMD64 and ARM64 builds
    - Set up build matrix for parallel architecture builds
    - Create multi-architecture manifests
    - _Requirements: 9.1, 9.2, 9.3_

  - [ ] 6.3 Implement container build jobs
    - Create separate jobs for Node.js and Python containers
    - Configure Docker layer caching for faster builds
    - Add hadolint linting validation
    - Implement proper error handling and isolation
    - _Requirements: 2.3, 2.4, 8.4_

  - [ ] 6.4 Implement registry publishing
    - Push images to GitHub Container Registry (ghcr.io)
    - Configure runtime-based tagging (node22, python3.12)
    - Create pinned tags with date and commit SHA
    - Make images publicly accessible
    - Add vulnerability scanning before publish
    - _Requirements: 2.2, 2.5, 3.1, 3.2, 3.4, 8.4_

  - [ ] 6.5 Add integration testing
    - Test container startup behavior
    - Validate MCP protocol functionality
    - Test volume mounting and entrypoint behavior
    - _Requirements: 10.1, 10.3, 11.2, 11.4_

  - [ ] 6.6 Test stdio mode with Claude Desktop
    - Configure Claude Desktop to use container via stdio
    - Verify bidirectional JSON-RPC communication
    - Test graceful shutdown on SIGTERM
    - Validate no TTY allocation breaks communication
    - _Requirements: 10.6, 10.7, 10.8, 10.9, 11.3_

  - [ ] 6.7 Add image size validation
    - Fail build if Node.js image > 200MB
    - Fail build if Python image > 300MB
    - Report size in build logs
    - _Requirements: 8.1, 8.2_

  - [ ]* 6.8 Write property tests for GitHub Actions workflow
    - **Property 4: Change-Triggered Builds**
    - **Property 5: Successful Build Publishing**
    - **Property 7: Runtime-Based Tagging**
    - **Validates: Requirements 2.1, 2.2, 2.5, 4.1**

- [ ] 7. Implement standardized container interfaces
  - [ ] 7.1 Ensure consistent stdio behavior across containers
    - Standardize passthrough entrypoint behavior
    - Implement consistent signal forwarding
    - _Requirements: 1.4, 10.6, 10.7, 10.8_

  - [ ]* 7.2 Write property tests for container interfaces
    - **Property 2: Standardized Container Interface**
    - **Property 3: Container Startup Readiness**
    - **Property 17: stdio Transport Integrity**
    - **Validates: Requirements 1.4, 1.5, 10.6, 10.7, 10.8, 10.9**

- [ ] 8. Add error handling and logging
  - [ ] 8.1 Implement build failure handling
    - Add clear error reporting for failed builds
    - Ensure container isolation (one failure doesn't block others)
    - _Requirements: 2.4_

  - [ ] 8.2 Add comprehensive logging
    - Log which images are being built and why
    - Provide clear startup information for containers
    - _Requirements: 4.5, 5.3_

  - [ ]* 8.3 Write property tests for error handling
    - **Property 6: Build Failure Handling**
    - **Property 11: Build Logging Transparency**
    - **Validates: Requirements 2.4, 4.5, 5.3**

- [ ] 9. Add version management and registry features
  - [ ] 9.1 Implement version retention in registry
    - Configure multiple version maintenance
    - Set up proper tagging strategy
    - _Requirements: 3.5_

  - [ ]* 9.2 Write property tests for version management
    - **Property 8: Version Retention**
    - **Property 10: Build Skipping Logic**
    - **Validates: Requirements 3.5, 4.4**

- [ ] 10. Final integration and security hardening
  - [ ] 10.1 Implement Docker security best practices
    - Ensure all containers run as non-root users
    - Minimize attack surface in container images
    - _Requirements: 5.5_

  - [ ]* 10.2 Write property tests for security practices
    - **Property 12: Docker Security Best Practices**
    - **Property 14: Language-Specific Package Managers**
    - **Validates: Requirements 5.5, 6.2, 6.3, 7.2, 7.3**

- [ ] 11. Implement upstream version detection
  - [ ] 11.1 Create versions.json tracking file
    - Store current built versions for each runtime
    - Include last check timestamp
    - Add JSON schema validation for versions.json structure
    - _Requirements: 12.6_

  - [ ] 11.2 Create check-upstream-versions.sh script
    - Query nodejs.org API for all tracked Node.js major versions (22, 20)
    - Query endoflife.date API for all tracked Python major versions (3.12, 3.11)
    - Compare against versions.json
    - Output detected updates for GitHub Actions
    - _Requirements: 12.1, 12.2, 12.5_

  - [ ] 11.3 Create check-upstream.yml workflow
    - Schedule weekly cron (Sunday 00:00 UTC)
    - Add workflow_dispatch for manual checks
    - Trigger build-containers.yml when updates detected
    - _Requirements: 12.1, 12.2, 12.3, 12.4_

  - [ ] 11.4 Update build workflow to accept version parameters
    - Accept target Node.js version as input
    - Accept target Python version as input
    - Update versions.json after successful build
    - _Requirements: 12.3, 12.4, 12.6_

  - [ ]* 11.5 Write property tests for upstream version detection
    - **Property 18: Upstream Version Detection**
    - **Validates: Requirements 12.1, 12.2, 12.3, 12.4, 12.6**

- [ ] 12. Implement run-mcp container runner binary
  - [ ] 12.1 Create Go project structure and dependencies
    - Initialize Go module with proper dependencies
    - Set up cross-platform build configuration
    - Create main.go with CLI argument parsing
    - _Requirements: 13.9_

  - [ ] 12.2 Implement container runtime detection
    - Implement cross-platform runtime detection (docker, podman, nerdctl)
    - Add platform-specific handling (lima nerdctl for macOS)
    - Support MCP_CONTAINER_RUNTIME override
    - Provide clear error messages when no runtime found
    - _Requirements: 13.1, 13.7_

  - [ ] 12.3 Implement language auto-detection
    - Create command-to-language mapping (npx→Node.js, uvx→Python)
    - Support explicit runtime specification (run-mcp python uvx ...)
    - Handle unknown commands with clear error messages
    - _Requirements: 13.2, 13.6_

  - [ ] 12.4 Implement secure environment variable passthrough
    - Implement allowlist-based environment filtering with predefined prefixes
    - Support exact matches (GITHUB_TOKEN, GITLAB_TOKEN, DATABASE_URL, REDIS_URL)
    - Handle MCP_PASSTHROUGH_ENV comma-separated custom variables
    - Ensure no system variables leak through (security)
    - _Requirements: 13.3, 13.8, 13.10_

  - [ ] 12.5 Implement cross-platform volume mounting
    - Handle cross-platform path resolution for credential directories
    - Mount ~/.aws and ~/.config with proper permissions
    - Support MCP_DATA_DIR configuration with platform-specific defaults
    - _Requirements: 13.4, 13.5_

  - [ ] 12.6 Add configuration and image selection
    - Support MCP_NODEJS_IMAGE and MCP_PYTHON_IMAGE overrides
    - Implement image selection based on detected language
    - Add configuration validation and defaults
    - _Requirements: 13.5_

  - [ ] 12.7 Create cross-platform build system
    - Set up GitHub Actions for multi-platform builds
    - Build for Windows (AMD64), macOS (AMD64, ARM64), Linux (AMD64, ARM64)
    - Create release artifacts with checksums
    - _Requirements: 13.9_

  - [ ] 12.8 Create installation documentation
    - Document binary installation for all platforms
    - Provide Claude Desktop configuration examples
    - Document environment variable configuration
    - Create usage examples and troubleshooting guide
    - _Requirements: 13.5_

  - [ ] 12.9 Test with Claude Desktop integration
    - Verify drop-in replacement works on all platforms
    - Test environment variable passthrough with real credentials
    - Test with AWS, OpenAI, and other service credentials
    - Validate cross-platform volume mounting
    - _Requirements: 13.3, 13.4, 13.5_

  - [ ]* 12.10 Write property tests for run-mcp binary
    - **Property 19: Container Runtime Detection**
    - **Property 20: Language Auto-Detection**
    - **Property 21: Secure Environment Variable Passthrough**
    - **Validates: Requirements 13.1, 13.2, 13.3, 13.8, 13.10**

- [ ] 13. Final checkpoint - Complete system validation
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at key milestones
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples and edge cases
- The implementation focuses on GitHub Actions automation with selective building for efficiency