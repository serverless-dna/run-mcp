# Implementation Plan: Container Signal Handling

## Overview

This implementation plan transforms the current blocking container execution (`containerCmd.Run()`) into a signal-aware system that properly forwards terminal signals (Ctrl-C, SIGTERM) to containerized MCP server processes. The approach uses container naming with the `--name` flag to enable signal forwarding via `docker kill`, `podman kill`, etc.

## Tasks

- [x] 1. Create signal handling infrastructure
  - Create `cmd/run-mcp/signal.go` with SignalHandler interface and implementation
  - Create `cmd/run-mcp/process.go` with ProcessManager interface and implementation
  - Add signal configuration loading to existing config system
  - _Requirements: 1.1, 1.2, 1.3, 2.3, 3.1, 3.2, 3.3, 7.1, 7.2_

- [x] 1.1 Write property test for signal forwarding consistency
  - **Property 1: Signal Forwarding Consistency**
  - **Validates: Requirements 1.1, 1.2, 4.1, 4.2, 4.3, 4.4**

- [x] 1.2 Write property test for cross-platform signal support
  - **Property 6: Platform Signal Support**
  - **Validates: Requirements 3.1, 3.2, 3.3**

- [ ] 2. Implement container naming strategy
  - Modify `buildContainerCommand` to use ProcessManager with container naming
  - Add `--name` flag generation using existing `sanitizeVolumeName` function
  - Update container command building to include unique container names
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [ ] 2.1 Write property test for configuration loading
  - **Property 5: Configuration Loading**
  - **Validates: Requirements 2.3**

- [ ] 3. Replace blocking execution with signal-aware execution
  - Replace `containerCmd.Run()` in main.go with `Start()` + signal handling + `Wait()`
  - Integrate SignalHandler and ProcessManager into main execution flow
  - Ensure stdio passthrough continues to work correctly
  - _Requirements: 1.3, 1.4, 8.1, 8.2_

- [ ] 3.1 Write property test for exit code propagation
  - **Property 2: Exit Code Propagation**
  - **Validates: Requirements 1.4**

- [ ] 3.2 Write property test for graceful shutdown wait
  - **Property 3: Graceful Shutdown Wait**
  - **Validates: Requirements 1.3**

- [ ] 4. Implement timeout and force termination
  - Add timeout configuration with MCP_SIGNAL_TIMEOUT environment variable support
  - Implement progressive timeout handling (graceful → force → absolute)
  - Add force termination using SIGKILL when graceful shutdown fails
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [ ] 4.1 Write property test for timeout-based force termination
  - **Property 4: Timeout-Based Force Termination**
  - **Validates: Requirements 2.1, 2.2**

- [ ] 4.2 Write unit test for force termination exit code
  - Test that force termination exits with code 130
  - _Requirements: 2.5_

- [ ] 5. Add comprehensive error handling and logging
  - Implement error handling for signal forwarding failures
  - Add debug and warning level logging for signal operations
  - Handle duplicate signals gracefully for already-terminated containers
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 7.1, 7.2, 7.3, 7.4_

- [ ] 5.1 Write property test for error handling resilience
  - **Property 12: Error Handling Resilience**
  - **Validates: Requirements 6.1, 6.3**

- [ ] 5.2 Write property test for comprehensive logging
  - **Property 15: Comprehensive Logging**
  - **Validates: Requirements 2.4, 7.1, 7.2, 7.3, 7.4**

- [ ] 5.3 Write property test for force termination error handling
  - **Property 14: Force Termination Error Handling**
  - **Validates: Requirements 6.4**

- [ ] 6. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 7. Integrate with existing volume management
  - Ensure ephemeral volume cleanup works correctly with signal handling
  - Update volume cleanup to occur after container termination during signal handling
  - Test integration with existing `--ephemeral` flag functionality
  - _Requirements: 5.5_

- [ ] 7.1 Write property test for ephemeral volume cleanup
  - **Property 11: Ephemeral Volume Cleanup**
  - **Validates: Requirements 5.5**

- [ ] 8. Add advanced signal handling features
  - Implement process group independence for signal handling
  - Add support for child process signal propagation
  - Handle host process termination cleanup
  - _Requirements: 5.1, 5.2, 5.4_

- [ ] 8.1 Write property test for process group independence
  - **Property 8: Process Group Independence**
  - **Validates: Requirements 5.1**

- [ ] 8.2 Write property test for child process signal propagation
  - **Property 9: Child Process Signal Propagation**
  - **Validates: Requirements 5.2**

- [ ] 8.3 Write property test for host termination cleanup
  - **Property 10: Host Termination Cleanup**
  - **Validates: Requirements 5.4**

- [ ] 9. Implement stream handling during shutdown
  - Add stdout/stderr draining before termination
  - Implement EOF detection for container stream closure
  - Ensure stream draining doesn't prevent force termination
  - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [ ] 9.1 Write property test for stream draining
  - **Property 16: Stream Draining**
  - **Validates: Requirements 8.1**

- [ ] 9.2 Write property test for EOF detection and shutdown
  - **Property 17: EOF Detection and Shutdown**
  - **Validates: Requirements 8.2**

- [ ] 9.3 Write property test for stream timeout override
  - **Property 18: Stream Timeout Override**
  - **Validates: Requirements 8.3**

- [ ] 9.4 Write property test for stdin EOF handling
  - **Property 19: Stdin EOF Handling**
  - **Validates: Requirements 8.4**

- [ ] 10. Add unsupported signal resilience and duplicate signal handling
  - Implement graceful handling of unsupported signals
  - Add duplicate signal handling for already-terminated containers
  - Ensure system continues normal operation when receiving unsupported signals
  - _Requirements: 3.5, 6.2_

- [ ] 10.1 Write property test for unsupported signal resilience
  - **Property 7: Unsupported Signal Resilience**
  - **Validates: Requirements 3.5**

- [ ] 10.2 Write property test for duplicate signal handling
  - **Property 13: Duplicate Signal Handling**
  - **Validates: Requirements 6.2**

- [ ] 11. Final integration and testing
  - Test signal handling with all supported container runtimes (Docker, Podman, Nerdctl, Finch, Lima)
  - Verify backward compatibility with existing functionality
  - Test cross-platform behavior on available platforms
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [ ] 11.1 Write integration tests for all container runtimes
  - Test signal forwarding works consistently across all supported runtimes
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [ ] 12. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- All tasks are required for comprehensive signal handling implementation
- Each task references specific requirements for traceability
- Property tests validate universal correctness properties across all inputs
- Unit tests validate specific examples and edge cases
- The implementation maintains full backward compatibility with existing functionality
- Signal handling uses container naming strategy to avoid complex container ID extraction