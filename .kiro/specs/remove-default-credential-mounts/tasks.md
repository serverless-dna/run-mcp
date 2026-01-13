# Implementation Plan: Remove Default Credential Mounts

## Overview

This implementation plan converts the secure-by-default design into discrete coding tasks. The changes remove automatic mounting of credential directories and data directories while adding wildcard support to environment variable passthrough. Each task builds incrementally to ensure the system remains functional throughout the implementation.

## Tasks

- [x] 1. Remove automatic credential mounting functionality
  - Remove `getCredentialMounts()` and `getPlatformSpecificMounts()` methods from VolumeManager
  - Update `GetVolumeMounts()` to only return data mounts when explicitly configured
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 7.1, 7.2, 7.3_

- [x] 1.1 Write property test for no automatic credential mounting
  - **Property 1: Secure Default Container Access**
  - **Validates: Requirements 1.1, 1.6, 8.1, 8.3, 9.1, 9.3**

- [x] 2. Implement secure data directory mounting
  - Modify `getDataMount()` to only mount when `MCP_DATA_DIR` is explicitly set
  - ~~Add `isDefaultHomeDir()` helper method to detect when DataDir equals home directory~~ (removed per user feedback)
  - Update configuration loading to not default `MCP_DATA_DIR` to home directory
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

- [x] 2.1 Write property test for explicit data directory mounting
  - **Property 4: Explicit Data Directory Mounting**
  - **Validates: Requirements 4.1, 4.3, 8.2**

- [ ] 3. Add wildcard support to environment variable passthrough
  - Modify `EnvFilter` to support wildcard patterns in `MCP_PASSTHROUGH_ENV`
  - Remove hardcoded `allowedPrefixes` and `allowedExact` defaults
  - Implement pattern matching for `*` suffix wildcards
  - Update `getCustomPassthroughVars()` to parse wildcards and exact matches
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

- [ ] 3.1 Write property test for wildcard environment variable matching
  - **Property 5: Wildcard Environment Variable Matching**
  - **Validates: Requirements 9.2, 9.6**

- [ ] 3.2 Write property test for no default environment variable passthrough
  - **Property 1: Secure Default Container Access** (environment variables part)
  - **Validates: Requirements 9.1, 9.3**

- [ ] 4. Update configuration and validation
  - Modify `loadConfig()` to not default `MCP_DATA_DIR` to home directory
  - Update `Config.Validate()` to handle empty `DataDir` gracefully
  - Add `MCP_PASSTHROUGH_ENV` to configuration variable exclusion list
  - _Requirements: 8.1, 9.4_

- [ ] 4.1 Write property test for configuration variable exclusion
  - **Property 6: Configuration Variable Exclusion**
  - **Validates: Requirements 9.4**

- [ ] 5. Update mount information and cleanup methods
  - Simplify `MountInfo` struct to remove `CredentialMounts` field
  - Update `GetMountInfo()` to only include data mount information
  - Remove credential directory detection from mount info methods
  - _Requirements: 7.4, 7.5_

- [ ] 5.1 Write property test for removed method functionality
  - **Property 9: Removed Method Functionality**
  - **Validates: Requirements 7.1, 7.2, 7.3, 7.4**

- [ ] 6. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. Enhance error handling and user guidance
  - Update error messages to guide users toward explicit configuration
  - Add specific error handling for missing credential access
  - Include migration examples in error messages
  - _Requirements: 6.5_

- [ ] 7.1 Write property test for credential access error handling
  - **Property 8: Credential Access Requires Explicit Configuration**
  - **Validates: Requirements 3.5, 6.2, 6.5**

- [ ] 8. Update existing tests for new secure behavior
  - Modify existing volume mount tests to expect no automatic credential mounts
  - Update environment variable passthrough tests for new wildcard behavior
  - Fix integration tests that relied on automatic mounting
  - Add test cases for secure-by-default behavior
  - _Requirements: 7.6_

- [ ] 8.1 Write property test for home volume preservation
  - **Property 2: Home Volume Preservation**
  - **Validates: Requirements 2.1, 2.2**

- [ ] 8.2 Write property test for explicit mount functionality
  - **Property 3: Explicit Mount Functionality**
  - **Validates: Requirements 3.1, 3.3, 6.3**

- [ ] 8.3 Write property test for ephemeral volume support
  - **Property 7: Ephemeral Volume Support**
  - **Validates: Requirements 2.4**

- [ ] 8.4 Write property test for backward compatibility
  - **Property 10: Backward Compatibility for Non-Credential Users**
  - **Validates: Requirements 6.1**

- [ ] 9. Update documentation and examples
  - Update README.md examples to show explicit configuration
  - Add migration guide for users upgrading from automatic mounting
  - Document wildcard syntax for `MCP_PASSTHROUGH_ENV`
  - Update troubleshooting guide with new security model
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 6.4, 9.7_

- [ ] 10. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- The implementation maintains backward compatibility for explicit configurations while removing insecure defaults