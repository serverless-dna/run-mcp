# Implementation Plan: Container Home Isolation

## Overview

This implementation plan converts the container home isolation design into incremental coding tasks. Each task builds on previous work and focuses on discrete functionality that can be tested independently. The implementation extends the existing `run-mcp` Go binary with volume management capabilities.

## Tasks

- [x] 1. Update Node.js container to use `/home/mcp` (Container Images Dependency)
  - **Note**: This task belongs in the mcp-container-images repository
  - Modify `nodejs/Dockerfile` to use `/home/mcp` instead of `/home/node`
  - Update `ENV HOME=/home/mcp` and directory creation commands
  - Ensure `node` user (UID 1000) has proper permissions for `/home/mcp`
  - _Requirements: 1.5_

- [x] 1.1 Write unit tests for Node.js container home directory
  - Test that container starts with correct HOME environment variable
  - Test that `/home/mcp` directory exists and is writable
  - _Requirements: 1.5_

- [x] 2. Implement volume name sanitization with truncation
  - Create `sanitizeVolumeName()` function in `cmd/run-mcp/volumes.go`
  - Implement command argument parsing and sanitization logic
  - Add truncation with hash suffix for names exceeding 64 characters
  - _Requirements: 2.1, 2.2, 2.3, 2.7, 2.8_

- [x] 2.1 Write property test for volume name sanitization
  - **Property 1: Volume Creation Consistency**
  - **Validates: Requirements 1.1, 2.1, 2.2, 2.3**

- [x] 2.2 Write unit tests for volume name edge cases
  - Test empty commands, special characters, Windows paths
  - Test truncation behavior with various input lengths
  - _Requirements: 2.7, 2.8_

- [x] 3. Implement container runtime abstraction for volume commands
  - Create `VolumeCommander` interface in `cmd/run-mcp/volumes.go`
  - Implement runtime-specific volume command builders (Docker, Podman, Nerdctl, Finch)
  - Add volume creation with runtime-specific labels
  - _Requirements: 4.11, 4.12, 2.9_

- [x] 3.1 Write unit tests for volume command abstraction
  - Test command generation for each supported runtime
  - Test label application and runtime metadata
  - _Requirements: 4.11, 4.12, 2.9_

- [x] 4. Implement home volume creation and management
  - Extend `VolumeManager` with `CreateHomeVolume()` method
  - Add persistent volume creation with proper labels
  - Implement volume existence checking and reuse logic
  - _Requirements: 1.1, 1.2, 1.6_

- [x] 4.1 Write property test for volume reuse idempotency
  - **Property 2: Volume Reuse Idempotency**
  - **Validates: Requirements 1.2, 2.4, 4.2**

- [x] 4.2 Write unit tests for volume creation
  - Test volume creation success and failure scenarios
  - Test concurrent access handling
  - _Requirements: 1.1, 1.6_

- [x] 5. Implement ephemeral volume support
  - Add `CreateEphemeralVolume()` method with timestamp-based naming
  - Implement `--ephemeral` flag parsing in CLI
  - Add ephemeral volume cleanup on container exit
  - _Requirements: 6.3, 6.4, 6.5_

- [x] 5.1 Write property test for ephemeral volume cleanup
  - **Property 13: Ephemeral Volume Cleanup**
  - **Validates: Requirements 6.3, 6.4, 6.5**

- [x] 5.2 Write unit tests for ephemeral volume naming
  - Test unique timestamp-based naming
  - Test cleanup behavior
  - _Requirements: 6.4, 6.5_

- [x] 6. Checkpoint - Ensure basic volume functionality works
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Implement MCP_MOUNT parsing and validation
  - Create `UserMountParser` struct with parsing methods
  - Implement tilde expansion and Windows path conversion
  - Add source path validation and syntax error handling
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.9, 7.10_

- [x] 7.1 Write property test for user mount configuration
  - **Property 8: User Mount Configuration**
  - **Validates: Requirements 7.1, 7.2, 7.3, 7.4, 7.8**

- [x] 7.2 Write unit tests for mount parsing edge cases
  - Test invalid syntax, missing paths, Windows path conversion
  - Test error message formatting
  - _Requirements: 7.9, 7.10_

- [x] 8. Implement home directory override support
  - Add `MCP_BIND_HOME` and `MCP_HOME_PATH` handling
  - Create bind mount directory creation for `~/.run-mcp/<volume-name>/`
  - Implement override precedence logic
  - _Requirements: 7.6, 7.7_

- [x] 8.1 Write property test for home directory overrides
  - **Property 9: Home Directory Override Behavior**
  - **Validates: Requirements 7.6, 7.7**

- [x] 8.2 Write unit tests for bind home directory creation
  - Test directory creation and permissions
  - Test path expansion and validation
  - _Requirements: 7.6, 7.7_

- [x] 9. Integrate volume mounting into container execution
  - Update `buildContainerCommand()` to include home volume mounts
  - Add user-specified mount integration
  - Ensure backward compatibility with existing commands
  - _Requirements: 1.5, 5.1, 5.4_

- [x] 9.1 Write property test for backward compatibility
  - **Property 12: Backward Compatibility**
  - **Validates: Requirements 5.1, 5.4**

- [x] 9.2 Write integration tests for container execution
  - Test complete container startup with volume mounts
  - Test environment variable passthrough
  - _Requirements: 3.1, 3.3, 3.5_

- [x] 9.3 Implement environment variable filtering
  - Filter out MCP_MOUNT, MCP_BIND_HOME, MCP_HOME_PATH from container env
  - Pass through all other user-provided variables
  - _Requirements: 3.5_

- [x] 9.4 Write property test for home directory write access
  - **Property 3: Home Directory Write Access**
  - **Validates: Requirements 1.3, 3.2**

- [x] 9.5 Write property test for consistent mount point
  - **Property 5: Consistent Mount Point**
  - **Validates: Requirements 1.5**

- [x] 9.6 Write property test for environment variable passthrough
  - **Property 7: Environment Variable Passthrough**
  - **Validates: Requirements 3.1, 3.3**

- [x] 10. Implement volume management CLI commands
  - Add `run-mcp volume` subcommand with list, clean, prune, inspect
  - Implement confirmation prompts for destructive operations
  - Add cross-runtime volume filtering
  - _Requirements: 4.4, 4.5, 4.6, 4.8, 4.9, 4.13, 2.10_

- [x] 10.1 Write property test for volume management commands
  - **Property 6: Volume Management Commands**
  - **Validates: Requirements 2.5, 4.4, 4.5, 4.8, 4.10**

- [x] 10.2 Write unit tests for CLI command parsing
  - Test subcommand routing and argument validation
  - Test confirmation prompt behavior
  - _Requirements: 4.9, 4.13_

- [x] 10.3 Implement volume inspect command
  - Show volume details, mount point, labels
  - Show physical storage location when supported
  - _Requirements: 4.8, 4.13_

- [x] 11. Implement volume prune functionality
  - Add `PruneHomeVolumes()` method with confirmation
  - Implement runtime-specific volume filtering
  - Add storage usage warnings
  - _Requirements: 4.6, 4.7, 6.6_

- [x] 11.1 Write property test for volume prune operation
  - **Property 11: Volume Prune Operation**
  - **Validates: Requirements 4.6, 4.7**

- [x] 11.2 Write unit tests for storage warnings
  - Test size limit detection and warning messages
  - _Requirements: 6.6_

- [x] 11.3 Write property test for volume persistence
  - **Property 10: Volume Persistence**
  - **Validates: Requirements 4.3, 6.1, 6.2**

- [x] 12. Implement error handling and recovery
  - Add clear error messages for runtime unavailable scenarios
  - Implement fallback strategies for volume creation failures
  - Add filesystem error detection and suggestions
  - _Requirements: 5.2, 5.3, 5.5_

- [x] 12.1 Write unit tests for error scenarios
  - Test runtime detection failures
  - Test volume creation error handling
  - _Requirements: 5.2, 5.3_

- [ ] 13. Add security isolation validation
  - Implement host filesystem access prevention checks
  - Add container permission boundary validation
  - Ensure volume sandbox isolation
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 13.1 Write property test for host filesystem isolation
  - **Property 4: Host Filesystem Isolation**
  - **Validates: Requirements 1.4, 7.5, 8.1, 8.2, 8.3**

- [ ] 13.2 Write property test for container permission boundaries
  - **Property 14: Container Permission Boundaries**
  - **Validates: Requirements 8.5**

- [ ] 14. Final checkpoint and integration testing
  - Ensure all tests pass, ask the user if questions arise.
  - Run complete integration tests across all supported runtimes
  - Validate property-based tests with 100+ iterations each

## Notes

- All tasks are required for comprehensive implementation from the start
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Implementation builds incrementally on existing `run-mcp` codebase
- All volume operations work across Docker, Podman, Finch, and nerdctl runtimes