# Requirements Document

## Introduction

The run-mcp tool currently lacks proper signal handling when running MCP servers in containers. When users send terminal signals like Ctrl-C (SIGINT) or SIGTERM, these signals are not properly forwarded to the container process, causing containers to not exit gracefully and potentially leaving orphaned processes.

## Glossary

- **Signal**: An asynchronous notification sent to a process to notify it of an event
- **SIGINT**: Interrupt signal, typically sent when user presses Ctrl-C
- **SIGTERM**: Termination signal, used to request graceful shutdown
- **Container_Process**: The containerized MCP server process running inside the container
- **Host_Process**: The run-mcp binary process running on the host system
- **Signal_Handler**: Component responsible for capturing and forwarding signals
- **Graceful_Shutdown**: Proper cleanup and termination sequence allowing processes to finish current operations

## Requirements

### Requirement 1: Signal Capture and Forwarding

**User Story:** As a user running an MCP server, I want Ctrl-C to immediately stop the container, so that I can quickly terminate unresponsive or unwanted processes.

#### Acceptance Criteria

1. WHEN a user presses Ctrl-C (SIGINT), THE Signal_Handler SHALL capture the signal and forward it to the Container_Process
2. WHEN a SIGTERM is sent to the Host_Process, THE Signal_Handler SHALL forward it to the Container_Process
3. WHEN a signal is forwarded to the Container_Process, THE Host_Process SHALL wait for the container to exit gracefully
4. WHEN the Container_Process exits after receiving a signal, THE Host_Process SHALL exit with the same exit code

### Requirement 2: Signal Timeout and Force Termination

**User Story:** As a user, I want containers that don't respond to graceful shutdown signals to be forcefully terminated, so that I don't have hanging processes consuming resources.

#### Acceptance Criteria

1. WHEN a Container_Process doesn't exit within the configured timeout after receiving SIGTERM, THE Signal_Handler SHALL send SIGKILL to force termination
2. WHEN a Container_Process doesn't exit within the configured timeout after receiving SIGINT, THE Signal_Handler SHALL send SIGKILL to force termination
3. THE timeout values SHALL be configurable via MCP_SIGNAL_TIMEOUT environment variable with default of 10 seconds for SIGTERM and 5 seconds for SIGINT
4. WHEN force termination occurs, THE Host_Process SHALL log a warning message about the forced termination
5. WHEN force termination completes, THE Host_Process SHALL exit with exit code 130 (indicating interrupted by signal)

### Requirement 3: Cross-Platform Signal Support

**User Story:** As a developer using run-mcp on different operating systems, I want consistent signal handling behavior, so that my workflow is the same regardless of platform.

#### Acceptance Criteria

1. WHEN running on Linux, THE Signal_Handler SHALL handle SIGINT, SIGTERM, and SIGQUIT signals
2. WHEN running on macOS, THE Signal_Handler SHALL handle SIGINT, SIGTERM, and SIGQUIT signals  
3. WHEN running on Windows, THE Signal_Handler SHALL handle os.Interrupt (Ctrl-C and Ctrl-Break) signals
4. WHEN running on Windows, THE Signal_Handler SHALL provide equivalent graceful shutdown behavior despite platform signal limitations
5. WHEN an unsupported signal is received, THE Signal_Handler SHALL ignore it and continue normal operation

### Requirement 4: Container Runtime Compatibility

**User Story:** As a user with different container runtimes, I want signal handling to work consistently across Docker, Podman, and other supported runtimes, so that I have a uniform experience.

#### Acceptance Criteria

1. WHEN using Docker as the container runtime, THE Signal_Handler SHALL properly forward signals to docker containers
2. WHEN using Podman as the container runtime, THE Signal_Handler SHALL properly forward signals to podman containers
3. WHEN using Nerdctl as the container runtime, THE Signal_Handler SHALL properly forward signals to nerdctl containers
4. WHEN using any supported container runtime, THE Signal_Handler SHALL use the same signal forwarding mechanism

### Requirement 5: Process Group Management

**User Story:** As a system administrator, I want signal handling to work correctly in process groups and job control scenarios, so that batch operations and scripted usage work reliably.

#### Acceptance Criteria

1. WHEN the Host_Process is part of a process group, THE Signal_Handler SHALL ensure signals are handled independently of the group
2. WHEN the Container_Process spawns child processes, THE Signal_Handler SHALL ensure all child processes receive the forwarded signal
3. WHEN running in a shell with job control, THE Signal_Handler SHALL work correctly with background and foreground process management
4. WHEN the Host_Process is terminated by the parent process, THE Signal_Handler SHALL ensure the Container_Process is also terminated
5. WHEN `--ephemeral` flag is set, THE Signal_Handler SHALL ensure volume cleanup completes after container termination before Host_Process exits

### Requirement 6: Error Handling and Recovery

**User Story:** As a user, I want clear error messages when signal handling fails, so that I can understand and resolve any issues.

#### Acceptance Criteria

1. WHEN signal forwarding fails due to container runtime errors, THE Signal_Handler SHALL log a descriptive error message
2. WHEN the Container_Process is already terminated, THE Signal_Handler SHALL handle duplicate signals gracefully
3. WHEN signal handling encounters system-level errors, THE Signal_Handler SHALL provide recovery guidance to the user
4. WHEN force termination fails, THE Signal_Handler SHALL report the failure and exit with an appropriate error code

### Requirement 7: Logging and Observability

**User Story:** As a developer debugging MCP server issues, I want visibility into signal handling behavior, so that I can understand what happened during termination.

#### Acceptance Criteria

1. WHEN a signal is received, THE Signal_Handler SHALL log the signal type and timestamp at debug level
2. WHEN forwarding a signal to the container, THE Signal_Handler SHALL log the action at debug level
3. WHEN timeout occurs during graceful shutdown, THE Signal_Handler SHALL log a warning with the timeout duration
4. WHEN force termination is used, THE Signal_Handler SHALL log the action and reason at warning level

### Requirement 8: Stream Handling During Shutdown

**User Story:** As an MCP server user, I want stdio streams to be properly closed during shutdown, so that no data is lost and the process terminates cleanly.

#### Acceptance Criteria

1. WHEN a signal is received, THE Host_Process SHALL allow stdout/stderr to drain before termination
2. WHEN the Container_Process closes its streams, THE Host_Process SHALL detect EOF and begin shutdown sequence
3. WHEN stream draining takes longer than the signal timeout, THE Host_Process SHALL proceed with force termination
4. WHEN stdin is closed by the parent process, THE Signal_Handler SHALL initiate graceful shutdown of the Container_Process