# Requirements Document

## Introduction

The container-home-isolation feature provides isolated, persistent home directories for MCP servers running in containers. This solves the problem where MCP servers need to write configuration files, logs, and cache data to their home directory, but the container filesystem is read-only for security. The feature uses container volumes to create isolated home directories that persist between container runs while maintaining security isolation from the host filesystem.

## Glossary

- **MCP_Server**: A Model Context Protocol server that runs inside a container
- **Container_Home**: An isolated home directory for the container user, backed by a container volume
- **Volume_Manager**: The component within the run-mcp binary responsible for creating, managing, and cleaning up container volumes
- **Host_Filesystem**: The filesystem of the host machine running the containers
- **Run_MCP**: The Go binary executable (`run-mcp`) that manages container execution and volume lifecycle
- **Server_Name**: A sanitized identifier derived from the MCP server command used for volume naming
- **Container_Runtime**: The container runtime being used (Docker, Podman, Nerdctl, Finch, or Lima)

## Requirements

### Requirement 1: Isolated Home Directory Creation

**User Story:** As a developer running MCP servers, I want each server to have its own isolated home directory, so that servers can write configuration and log files without affecting my host system.

#### Acceptance Criteria

1. WHEN run-mcp starts a container, THE Volume_Manager SHALL create a named container volume for the container home directory
2. WHEN multiple containers run the same MCP server, THE Volume_Manager SHALL reuse the same volume for consistency
3. WHEN a container needs to write to its home directory, THE Container_Home SHALL be writable and persistent
4. THE Container_Home SHALL be completely isolated from the Host_Filesystem
5. THE Container_Home volume SHALL be mounted read/write to `/home/mcp` for all container types
6. WHEN multiple instances of the same MCP server run concurrently, THE Volume_Manager SHALL share the same volume (container runtime handles concurrent access)

### Requirement 2: Volume Naming and Management

**User Story:** As a system administrator, I want predictable volume names and management capabilities, so that I can understand and maintain the container volumes.

#### Acceptance Criteria

1. WHEN a volume is created, THE Volume_Manager SHALL use a deterministic naming scheme based on the MCP server command
2. THE volume name SHALL follow the pattern `mcp-home-{sanitized-command}-{sanitized-first-arg}`
3. THE sanitization SHALL convert to lowercase, replace non-alphanumeric characters with dashes, and trim leading/trailing dashes
4. WHEN the same MCP server command is run multiple times, THE Volume_Manager SHALL reuse the existing volume
5. THE Volume_Manager SHALL provide commands to list all managed volumes
6. THE Volume_Manager SHALL provide commands to clean up specific volumes
7. THE sanitization SHALL normalize path separators before processing on all platforms
8. THE Volume_Manager SHALL truncate volume names exceeding 64 characters by keeping the first 56 characters plus an 8-character hash suffix
9. THE Volume_Manager SHALL include the runtime name in volume metadata or naming to prevent cross-runtime confusion
10. WHEN listing volumes, THE Volume_Manager SHALL only show volumes for the currently detected runtime

### Requirement 3: Container Environment Configuration

**User Story:** As an MCP server, I want my environment variables to be properly configured, so that I can access necessary configuration while maintaining security.

#### Acceptance Criteria

1. THE run-mcp SHALL preserve any user-provided environment variables without modification
2. THE container SHALL have write permissions to all subdirectories under `/home/mcp`
3. WHEN users provide additional environment variables via command line, THE run-mcp SHALL pass them through to the container
4. THE run-mcp SHALL NOT make assumptions about specific MCP server environment variable requirements
5. THE run-mcp configuration environment variables (MCP_MOUNT, MCP_BIND_HOME, MCP_HOME_PATH) SHALL be consumed by run-mcp and NOT passed to the container

### Requirement 4: Volume Lifecycle Management

**User Story:** As a developer, I want to manage the lifecycle of container volumes, so that I can clean up unused volumes and understand storage usage.

#### Acceptance Criteria

1. WHEN an MCP server runs for the first time, THE Volume_Manager SHALL create a new container volume
2. WHEN an MCP server runs subsequently, THE Volume_Manager SHALL reuse the existing volume to preserve data
3. THE volumes SHALL persist indefinitely until explicitly cleaned up by the user
4. THE run-mcp binary SHALL provide a `run-mcp volume list` command to show all managed volumes with creation date and size
5. THE run-mcp binary SHALL provide a `run-mcp volume clean <server-name>` command to remove specific volumes
6. THE run-mcp binary SHALL provide a `run-mcp volume prune` command to remove all volumes with user confirmation
7. WHEN a volume is cleaned, THE Volume_Manager SHALL remove the container volume and all its data permanently
8. THE run-mcp binary SHALL provide a `run-mcp volume inspect <server-name>` command to show volume details and contents
9. WHEN cleaning volumes, THE run-mcp binary SHALL prompt for confirmation before deletion
10. THE volume list command SHALL show the container runtime's built-in volume metadata (creation time, size) when available
11. THE Volume_Manager SHALL abstract container runtime differences when executing volume commands
12. THE Volume_Manager SHALL support volume operations across Docker, Podman, Finch, and nerdctl runtimes
13. THE volume inspect command SHALL show the physical storage location when supported by the runtime

### Requirement 5: Backward Compatibility and Error Handling

**User Story:** As an existing user, I want the new volume feature to work transparently, so that my existing MCP server commands continue to work without changes.

#### Acceptance Criteria

1. WHEN existing MCP server commands are run, THE run-mcp SHALL automatically create volumes without user intervention
2. WHEN volume creation fails, THE run-mcp SHALL provide clear error messages and fallback options
3. WHEN the container runtime is not available, THE run-mcp SHALL provide helpful error messages about container runtime requirements
4. THE run-mcp SHALL maintain backward compatibility with existing command-line arguments
5. WHEN a container exits due to filesystem errors, THE run-mcp SHALL suggest volume-related solutions

### Requirement 6: Data Persistence and Ephemeral Mode

**User Story:** As a developer, I want control over whether container data persists between runs, so that I can choose between consistency and clean-slate execution.

#### Acceptance Criteria

1. THE volumes SHALL use a "persistent by default" strategy - data persists until explicitly removed
2. WHEN a container stops or crashes, THE volume data SHALL remain intact for the next run
3. THE run-mcp SHALL provide an `--ephemeral` flag to use temporary volumes that are removed when the container stops
4. WHEN using ephemeral mode, THE Volume_Manager SHALL create a unique temporary volume name to avoid conflicts with persistent volumes
5. THE run-mcp SHALL warn users about storage usage when volumes exceed configurable size limits
6. THE ephemeral volume name SHALL follow the pattern `mcp-ephemeral-{sanitized-server}-{timestamp}` to ensure uniqueness

### Requirement 7: User-Specified Mount Configuration

**User Story:** As a developer, I want to configure additional bind mounts for credentials and data access, so that MCP servers can access necessary host resources while maintaining security.

#### Acceptance Criteria

1. THE run-mcp SHALL support user-specified bind mounts via the `MCP_MOUNT` environment variable
2. THE `MCP_MOUNT` format SHALL be `<src>:<dest>[:<opts>],<src>:<dest>[:<opts>],...` (comma-separated Docker mount syntax)
3. THE run-mcp SHALL support tilde (`~`) expansion for source paths in `MCP_MOUNT`
4. THE run-mcp SHALL convert Windows paths automatically for cross-platform compatibility
5. THE run-mcp SHALL provide no default host filesystem access except the isolated home volume
6. WHEN `MCP_BIND_HOME=true` is set, THE run-mcp SHALL use `~/.run-mcp/<volume-name>/` on host instead of container volume
7. WHEN `MCP_HOME_PATH` is set, THE run-mcp SHALL use the specified host path for `/home/mcp`
8. THE run-mcp SHALL support read-only mounts using `:ro` option syntax
9. WHEN `MCP_MOUNT` source path does not exist, THE run-mcp SHALL fail with clear error message before starting container
10. WHEN `MCP_MOUNT` syntax is invalid, THE run-mcp SHALL fail with example of correct syntax

### Requirement 8: Security and Isolation

**User Story:** As a security-conscious user, I want container home directories to be completely isolated from my host system, so that MCP servers cannot access or modify my personal files.

#### Acceptance Criteria

1. THE Container_Home SHALL NOT have access to any Host_Filesystem directories
2. THE Container_Home SHALL NOT be able to escape the container volume sandbox
3. WHEN containers are removed, THE Host_Filesystem SHALL remain unchanged
4. THE Volume_Manager SHALL use the container runtime's built-in security mechanisms for isolation
5. THE container user SHALL have appropriate permissions within the Container_Home but no host access