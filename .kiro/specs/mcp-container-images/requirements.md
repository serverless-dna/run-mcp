# Requirements Document

## Introduction

A system for building and distributing language-specific container images that enable users to run MCP (Model Context Protocol) servers without requiring local development tools. The system will provide pre-built container images for Node.js and Python (uv) environments, automatically built and published to GitHub Container Registry through GitHub Actions.

## Glossary

- **MCP_Server**: A Model Context Protocol server that provides specific functionality
- **Container_Image**: A Docker container image containing a language runtime and MCP server dependencies
- **GitHub_Container_Registry**: GitHub's container registry service (ghcr.io)
- **Build_System**: The GitHub Actions workflow that builds and publishes container images
- **Language_Runtime**: The specific programming language environment (Node.js or Python with uv)

## Requirements

### Requirement 1: Container Image Creation

**User Story:** As a user, I want pre-built container images for different programming languages, so that I can run MCP servers without installing development tools locally.

#### Acceptance Criteria

1. THE Build_System SHALL create container images for Node.js runtime environments
2. THE Build_System SHALL create container images for Python runtime environments with uv package manager
3. WHEN a container image is built, THE Build_System SHALL include all necessary dependencies for running MCP servers
4. THE Container_Image SHALL provide a standardized interface for running MCP servers regardless of language
5. WHEN a container starts, THE Container_Image SHALL be ready to execute MCP server code immediately

### Requirement 2: Automated Build Pipeline

**User Story:** As a developer, I want container images to be automatically built when source files change, so that the latest versions are always available.

#### Acceptance Criteria

1. WHEN source files for a container image are modified, THE Build_System SHALL trigger an automated build
2. WHEN a build completes successfully, THE Build_System SHALL publish the image to GitHub Container Registry
3. THE Build_System SHALL build only the container images whose source files have changed
4. WHEN a build fails, THE Build_System SHALL provide clear error messages and stop the pipeline
5. THE Build_System SHALL tag images with appropriate version information

### Requirement 3: GitHub Container Registry Integration

**User Story:** As a user, I want to pull container images from a reliable registry, so that I can easily access and use the MCP server containers.

#### Acceptance Criteria

1. THE Build_System SHALL publish all container images to GitHub Container Registry (ghcr.io)
2. WHEN an image is published, THE GitHub_Container_Registry SHALL make it publicly accessible
3. THE Build_System SHALL use proper authentication to push images to the registry
4. WHEN publishing images, THE Build_System SHALL follow semantic versioning conventions
5. THE GitHub_Container_Registry SHALL retain versions according to the version support limits defined in VERSIONING.md

### Requirement 4: Change Detection and Selective Building

**User Story:** As a maintainer, I want only affected container images to be rebuilt when changes occur, so that build times are minimized and resources are used efficiently.

#### Acceptance Criteria

1. WHEN files in a specific container directory are modified, THE Build_System SHALL build only that container image
2. WHEN common files are modified, THE Build_System SHALL build all affected container images
3. THE Build_System SHALL detect file changes using Git diff or similar mechanisms
4. WHEN no relevant files have changed, THE Build_System SHALL skip the build process
5. THE Build_System SHALL provide clear logging about which images are being built and why

### Requirement 5: Container Image Structure

**User Story:** As a developer, I want consistent container image structure across languages, so that I can easily switch between different runtime environments.

#### Acceptance Criteria

1. THE Container_Image SHALL provide a standardized interface for running MCP servers via stdio transport
2. WHEN a container starts, THE Container_Image SHALL log startup information clearly
3. THE Container_Image SHALL support environment variable configuration
4. THE Container_Image SHALL follow Docker best practices for security and efficiency

### Requirement 6: Node.js Container Specifications

**User Story:** As a Node.js developer, I want a container image optimized for Node.js MCP servers, so that I can run JavaScript/TypeScript MCP servers efficiently.

#### Acceptance Criteria

1. THE Node_Container SHALL use the current LTS version of Node.js
2. THE Node_Container SHALL include npm and yarn package managers
3. WHEN Node.js packages are installed, THE Node_Container SHALL cache dependencies appropriately
4. THE Node_Container SHALL support both CommonJS and ES modules
5. THE Node_Container SHALL include TypeScript compilation capabilities

### Requirement 7: Python Container Specifications

**User Story:** As a Python developer, I want a container image optimized for Python MCP servers with uv package management, so that I can run Python MCP servers with fast dependency resolution.

#### Acceptance Criteria

1. THE Python_Container SHALL use Python 3.12-slim as the base image
2. THE Python_Container SHALL include the uv package manager for fast dependency management
3. WHEN Python packages are installed, THE Python_Container SHALL use uv for dependency resolution
4. THE Python_Container SHALL support virtual environment creation and management
5. THE Python_Container SHALL include common Python development tools and libraries

### Requirement 8: Container Performance and Quality

**User Story:** As a user, I want container images that are efficient and reliable, so that I can run MCP servers with predictable performance.

#### Acceptance Criteria

1. THE Node_Container SHALL be smaller than 200MB in final image size
2. THE Python_Container SHALL be smaller than 300MB in final image size
3. WHEN building any container, THE Build_System SHALL complete within 5 minutes
4. THE Build_System SHALL scan all images for security vulnerabilities before publishing
5. THE GitHub_Container_Registry SHALL retain versions according to the limits defined in VERSIONING.md

### Requirement 9: Multi-Architecture Support

**User Story:** As a user on different hardware platforms, I want container images that work on both Intel and ARM processors, so that I can run MCP servers regardless of my system architecture.

#### Acceptance Criteria

1. THE Build_System SHALL create container images for AMD64 architecture
2. THE Build_System SHALL create container images for ARM64 architecture
3. WHEN publishing images, THE Build_System SHALL create multi-architecture manifests
4. THE Container_Image SHALL run identically across all supported architectures
5. THE Build_System SHALL test images on both architectures before publishing

### Requirement 10: Container Runtime Interface

**User Story:** As a developer, I want a clear interface for running MCP servers in containers, so that I can easily mount my code and configure the runtime environment.

#### Acceptance Criteria

1. THE Container_Image SHALL accept user data via volume mounts at /data
2. THE Container_Image SHALL support stdio MCP transport protocol as the primary mode
3. WHEN started, THE Container_Image SHALL execute the provided command and arguments directly
4. THE Container_Image SHALL run as UID 1000 by default for volume permission compatibility
5. THE Container_Image SHALL support UID/GID override via environment variables
6. WHEN using stdio transport, THE Container_Image SHALL NOT buffer stdin or stdout
7. WHEN using stdio transport, THE Container_Image SHALL forward SIGTERM/SIGINT to the child process
8. WHEN the MCP server process exits, THE Container_Image SHALL exit with the same code
9. THE Container_Image SHALL NOT allocate a pseudo-TTY by default

### Requirement 11: MCP Protocol Compliance

**User Story:** As an MCP developer, I want containers that properly support MCP protocol standards, so that my servers work correctly with MCP clients.

#### Acceptance Criteria

1. THE Container_Image SHALL include the latest stable MCP SDK for the respective language
2. THE Container_Image SHALL support MCP protocol version negotiation
3. WHEN using stdio transport, THE Container_Image SHALL properly handle stdin/stdout communication without buffering
4. THE Container_Image SHALL support running published MCP servers via npx (Node.js) or uvx (Python)
5. THE Container_Image SHALL validate that the executed MCP server responds to protocol initialization

### Requirement 12: Automated Upstream Version Detection

**User Story:** As a user, I want container images to be automatically updated when new runtime versions are released, so that I receive security patches without manual intervention.

#### Acceptance Criteria

1. THE Build_System SHALL check for new Node.js LTS releases at least weekly
2. THE Build_System SHALL check for new Python releases at least weekly
3. WHEN a new patch version is detected, THE Build_System SHALL automatically trigger a container rebuild
4. WHEN a new minor version is detected, THE Build_System SHALL automatically trigger a container rebuild
5. THE Build_System SHALL log all version checks and detected updates
6. THE Build_System SHALL NOT rebuild if the upstream version is already built

### Requirement 13: Simple Container Runner Script

**User Story:** As a user, I want a simple command to run MCP servers in containers without writing verbose Docker commands, so that I can easily switch from local execution to containerized execution.

#### Acceptance Criteria

1. THE run-mcp script SHALL auto-detect available container runtimes (Docker, Podman, nerdctl)
2. THE run-mcp script SHALL auto-detect the required language runtime from the command (npx→Node.js, uvx→Python)
3. THE run-mcp script SHALL pass environment variables from the MCP client to the container using a secure allowlist approach
4. THE run-mcp script SHALL mount common credential directories (~/.aws, ~/.config) read-only
5. THE run-mcp script SHALL work as a drop-in replacement for direct command execution in MCP configs
6. THE run-mcp script SHALL support explicit runtime specification (run-mcp python uvx ...)
7. THE run-mcp script SHALL provide clear error messages when no container runtime is found
8. THE run-mcp script SHALL support custom environment variables via MCP_PASSTHROUGH_ENV configuration