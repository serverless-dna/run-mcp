# Requirements Document

## Introduction

The remove-default-credential-mounts feature addresses the security concern where run-mcp automatically mounts host credential directories (`.aws`, `.config`, `.ssh`) into containers by default. This automatic mounting violates the principle of least privilege and exposes sensitive credentials to MCP servers that may not need them. The feature will remove these automatic mounts while preserving the ability for users to explicitly mount credentials when needed via the `MCP_MOUNT` environment variable.

## Glossary

- **Credential_Directories**: Host directories containing sensitive authentication data (`.aws`, `.config`, `.ssh`, platform-specific credential stores)
- **Auto_Mount**: Automatic mounting of host directories into containers without explicit user configuration
- **Explicit_Mount**: User-specified mounting via the `MCP_MOUNT` environment variable
- **Volume_Manager**: The component responsible for managing container volume mounts
- **Home_Volume**: The persistent container volume mounted at `/home/mcp` for MCP server data
- **Standard_Mounts**: The set of mounts automatically applied by the Volume_Manager

## Requirements

### Requirement 1: Remove Automatic Credential Mounting

**User Story:** As a security-conscious user, I want MCP servers to have no access to my host credentials by default, so that I can control exactly which credentials each server can access.

#### Acceptance Criteria

1. WHEN run-mcp starts a container, THE Volume_Manager SHALL NOT automatically mount any host credential directories
2. THE Volume_Manager SHALL NOT mount `~/.aws` directory automatically
3. THE Volume_Manager SHALL NOT mount `~/.config` directory automatically  
4. THE Volume_Manager SHALL NOT mount `~/.ssh` directory automatically
5. THE Volume_Manager SHALL NOT mount `~/Library/Keychains` directory automatically on macOS
6. THE Volume_Manager SHALL NOT mount any platform-specific credential directories automatically
7. THE container SHALL start with only the home volume mount and user-specified mounts from `MCP_MOUNT`

### Requirement 2: Preserve Home Volume Functionality

**User Story:** As a developer, I want MCP servers to still have persistent storage for their data, so that server configurations and cache persist between runs.

#### Acceptance Criteria

1. THE Volume_Manager SHALL continue to create and mount the home volume at `/home/mcp`
2. WHEN a container starts, THE home volume SHALL be mounted read/write for MCP server data persistence
3. THE home volume functionality SHALL remain unchanged from current behavior
4. THE Volume_Manager SHALL continue to support ephemeral volumes when `--ephemeral` flag is used

### Requirement 3: Preserve Explicit Mount Functionality

**User Story:** As a developer who needs credential access, I want to explicitly specify which credentials to mount, so that I can grant minimal necessary access to MCP servers.

#### Acceptance Criteria

1. THE Volume_Manager SHALL continue to support user-specified mounts via `MCP_MOUNT` environment variable
2. WHEN `MCP_MOUNT` includes credential directories, THE Volume_Manager SHALL mount them as specified
3. THE `MCP_MOUNT` parsing and validation SHALL remain unchanged
4. THE Volume_Manager SHALL support all existing `MCP_MOUNT` syntax including read-only mounts
5. WHEN users want credential access, THE system SHALL require explicit `MCP_MOUNT` configuration

### Requirement 4: Maintain Data Directory Mounting

**User Story:** As a developer, I want MCP servers to access my data directory when specified, so that servers can work with my project files.

#### Acceptance Criteria

1. WHEN `MCP_DATA_DIR` is specified, THE Volume_Manager SHALL continue to mount it at `/data`
2. THE data directory mounting behavior SHALL remain unchanged
3. THE Volume_Manager SHALL continue to validate data directory accessibility
4. THE data directory mount SHALL be included in standard mounts

### Requirement 5: Update Documentation and Examples

**User Story:** As a user reading documentation, I want clear examples of how to mount credentials explicitly, so that I understand the new security model.

#### Acceptance Criteria

1. THE documentation SHALL show examples of explicit credential mounting using `MCP_MOUNT`
2. THE examples SHALL demonstrate mounting AWS credentials with `MCP_MOUNT=~/.aws:/home/mcp/.aws:ro`
3. THE examples SHALL demonstrate mounting SSH keys with `MCP_MOUNT=~/.ssh:/home/mcp/.ssh:ro`
4. THE documentation SHALL explain the security benefits of explicit mounting
5. THE migration guide SHALL help users update existing configurations

### Requirement 6: Backward Compatibility Considerations

**User Story:** As an existing user, I want to understand how this change affects my current setup, so that I can update my configuration appropriately.

#### Acceptance Criteria

1. THE system SHALL continue to work for users who don't rely on automatic credential mounting
2. WHEN users previously relied on automatic credential mounting, THE containers SHALL start successfully but without credential access
3. THE system SHALL NOT break existing `MCP_MOUNT` configurations
4. THE change SHALL be documented as a security improvement requiring user action for credential access
5. THE system SHALL provide clear error messages when credential access is needed but not configured

### Requirement 7: Remove Credential Mount Code

**User Story:** As a maintainer, I want the credential mounting code removed cleanly, so that the codebase is simpler and more secure by default.

#### Acceptance Criteria

1. THE `getCredentialMounts()` method SHALL be removed from VolumeManager
2. THE `getPlatformSpecificMounts()` method SHALL be removed from VolumeManager  
3. THE credential directory detection code SHALL be removed (including `.aws`, `.config`, `.ssh`, and `~/Library/Keychains` on macOS)
4. THE `GetVolumeMounts()` method SHALL only return data directory mounts when `MCP_DATA_DIR` is explicitly set
4. THE credential directory detection code SHALL be removed
5. THE `GetMountInfo()` method SHALL be updated to not include credential mount information
6. THE related test code SHALL be updated to reflect the new behavior

### Requirement 8: Remove Default Data Directory Mounting

**User Story:** As a security-conscious user, I want no host directories mounted by default, so that containers have minimal host access unless explicitly configured.

#### Acceptance Criteria

1. WHEN `MCP_DATA_DIR` is not explicitly set, THE Volume_Manager SHALL NOT mount any data directory
2. WHEN `MCP_DATA_DIR` is explicitly set, THE Volume_Manager SHALL mount it at `/data` as before
3. THE default behavior SHALL be no host filesystem access except the home volume
4. THE `getDataMount()` method SHALL return empty string when `MCP_DATA_DIR` is not set
5. THE `GetVolumeMounts()` method SHALL return empty array when no explicit data directory is configured
6. THE containers SHALL start successfully with only the home volume mount by default

### Requirement 9: Add Wildcard Support to MCP_PASSTHROUGH_ENV

**User Story:** As a security-conscious user, I want to use wildcards in MCP_PASSTHROUGH_ENV to specify environment variable patterns, so that I can efficiently control which variables are passed to containers without hardcoded defaults.

#### Acceptance Criteria

1. THE hardcoded environment variable prefixes SHALL be removed from the default allowlist
2. THE system SHALL support wildcard patterns in `MCP_PASSTHROUGH_ENV` (e.g., `MCP_PASSTHROUGH_ENV=AWS_*,OPENAI_*,DATABASE_URL`)
3. WHEN `MCP_PASSTHROUGH_ENV` is not set, THE system SHALL pass through no environment variables by default
4. THE wildcard syntax SHALL support `*` at the end of patterns to match prefixes (e.g., `AWS_*` matches `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
5. THE exact match allowlist (`GITHUB_TOKEN`, `DATABASE_URL`, etc.) SHALL be removed from hardcoded defaults
6. THE system SHALL support mixing exact names and wildcard patterns in the same `MCP_PASSTHROUGH_ENV` value
7. THE documentation SHALL provide examples of common wildcard configurations for different use cases