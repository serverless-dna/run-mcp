package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ProcessManager handles container process lifecycle
type ProcessManager interface {
	// StartContainer starts the container and returns immediately
	StartContainer(cmd *exec.Cmd) error
	
	// WaitForExit waits for container to exit and returns exit code
	WaitForExit() (int, error)
	
	// ForwardSignal sends a signal to the container
	ForwardSignal(signal os.Signal) error
	
	// ForceKill forcefully terminates the container
	ForceKill() error
}

// ContainerProcessManager implements ProcessManager for container runtimes
type ContainerProcessManager struct {
	cmd           *exec.Cmd
	containerName string
	runtime       string
	started       bool
}

// NewContainerProcessManager creates a new container process manager
func NewContainerProcessManager(runtime string, args []string) *ContainerProcessManager {
	// Generate unique name from server args + timestamp
	baseName := sanitizeVolumeName(args)
	containerName := fmt.Sprintf("%s-%d", baseName, time.Now().UnixNano())
	
	return &ContainerProcessManager{
		containerName: containerName,
		runtime:       runtime,
		started:       false,
	}
}

// StartContainer starts the container and returns immediately
func (pm *ContainerProcessManager) StartContainer(cmd *exec.Cmd) error {
	if pm.started {
		return fmt.Errorf("container already started")
	}
	
	pm.cmd = cmd
	
	// Start the container process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	
	pm.started = true
	return nil
}

// WaitForExit waits for container to exit and returns exit code
func (pm *ContainerProcessManager) WaitForExit() (int, error) {
	if !pm.started || pm.cmd == nil {
		return -1, fmt.Errorf("container not started")
	}
	
	// Wait for the process to complete
	err := pm.cmd.Wait()
	
	if err != nil {
		// Extract exit code from error
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			
			// Check if this was a force termination (SIGKILL = 9, exit code 128+9=137)
			// or interrupted by signal (SIGINT = 2, exit code 128+2=130)
			if exitCode == 137 {
				// Force terminated with SIGKILL - return 130 as per requirements
				return 130, nil
			} else if exitCode >= 128 && exitCode <= 165 {
				// Signal-based termination, preserve the exit code
				return exitCode, nil
			}
			
			return exitCode, nil
		}
		return -1, err
	}
	
	return 0, nil
}

// ForwardSignal sends a signal to the container
func (pm *ContainerProcessManager) ForwardSignal(sig os.Signal) error {
	if !pm.started {
		return fmt.Errorf("container not started")
	}
	
	// All runtimes use the same pattern: <runtime> kill --signal <signal> <container>
	signalName := getSignalName(sig)
	
	// Build command - works for all runtimes including "lima nerdctl"
	parts := strings.Fields(pm.runtime) // Handles both "docker" and "lima nerdctl"
	args := append(parts[1:], "kill", "--signal", signalName, pm.containerName)
	cmd := exec.Command(parts[0], args...)
	
	return cmd.Run()
}

// ForceKill forcefully terminates the container
func (pm *ContainerProcessManager) ForceKill() error {
	if !pm.started {
		return fmt.Errorf("container not started")
	}
	
	parts := strings.Fields(pm.runtime)
	args := append(parts[1:], "kill", "--signal", "SIGKILL", pm.containerName)
	cmd := exec.Command(parts[0], args...)
	
	// Execute the force kill command
	if err := cmd.Run(); err != nil {
		// Check if the error is because the container is already stopped
		if strings.Contains(err.Error(), "no such container") || 
		   strings.Contains(err.Error(), "container not found") ||
		   strings.Contains(err.Error(), "is not running") {
			// Container already stopped, this is not an error
			return nil
		}
		
		// Return the actual error for other cases
		return fmt.Errorf("failed to force kill container %s: %w", pm.containerName, err)
	}
	
	return nil
}

// GetContainerName returns the container name for this process
func (pm *ContainerProcessManager) GetContainerName() string {
	return pm.containerName
}

// BuildCommand builds the container command with the --name flag for signal forwarding
func (pm *ContainerProcessManager) BuildCommand(image string, args []string) *exec.Cmd {
	cmdArgs := []string{
		"run",
		"-i",
		"--rm",
		"--name", pm.containerName, // Key addition for signal forwarding
		image,
	}
	cmdArgs = append(cmdArgs, args...)
	
	// Handle multi-word runtimes (e.g., "lima nerdctl") consistently
	parts := strings.Fields(pm.runtime)
	if len(parts) > 1 {
		args := append(parts[1:], cmdArgs...)
		return exec.Command(parts[0], args...)
	}
	
	return exec.Command(pm.runtime, cmdArgs...)
}

// TimeoutManager handles graceful shutdown timeouts
type TimeoutManager interface {
	// StartTimeout begins timeout monitoring for a signal
	StartTimeout(signal os.Signal, duration time.Duration, callback func()) *time.Timer
	
	// CancelTimeout cancels an active timeout
	CancelTimeout(timer *time.Timer)
}

// DefaultTimeoutManager implements TimeoutManager
type DefaultTimeoutManager struct{}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager() TimeoutManager {
	return &DefaultTimeoutManager{}
}

// StartTimeout begins timeout monitoring for a signal
func (tm *DefaultTimeoutManager) StartTimeout(signal os.Signal, duration time.Duration, callback func()) *time.Timer {
	return time.AfterFunc(duration, callback)
}

// CancelTimeout cancels an active timeout
func (tm *DefaultTimeoutManager) CancelTimeout(timer *time.Timer) {
	if timer != nil {
		timer.Stop()
	}
}

// ProcessState represents the state of a container process
type ProcessState struct {
	PID           int
	ContainerName string
	Runtime       string
	StartTime     time.Time
	Status        ProcessStatus
}

// ProcessStatus represents the status of a process
type ProcessStatus int

const (
	StatusStarting ProcessStatus = iota
	StatusRunning
	StatusStopping
	StatusStopped
	StatusKilled
)

// String returns the string representation of ProcessStatus
func (ps ProcessStatus) String() string {
	switch ps {
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	case StatusStopping:
		return "stopping"
	case StatusStopped:
		return "stopped"
	case StatusKilled:
		return "killed"
	default:
		return "unknown"
	}
}