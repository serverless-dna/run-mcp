package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// RuntimeDetector handles container runtime detection
type RuntimeDetector struct {
	runtimes []string
}

// NewRuntimeDetector creates a new runtime detector with platform-specific runtimes
func NewRuntimeDetector() *RuntimeDetector {
	// Priority order for runtime detection
	runtimes := []string{"docker", "podman", "nerdctl", "finch"}
	
	// Add macOS-specific runtime
	if runtime.GOOS == "darwin" {
		runtimes = append(runtimes, "lima")
	}
	
	return &RuntimeDetector{runtimes: runtimes}
}

// Detect finds the first available container runtime
func (rd *RuntimeDetector) Detect() (string, error) {
	// Check for explicit override
	if rt := os.Getenv("MCP_CONTAINER_RUNTIME"); rt != "" {
		if rd.isAvailable(rt) {
			return rt, nil
		}
		return "", fmt.Errorf("specified container runtime '%s' not found in PATH", rt)
	}

	// Try runtimes in priority order
	for _, rt := range rd.runtimes {
		if rd.isAvailable(rt) {
			// Special handling for lima on macOS
			if rt == "lima" && runtime.GOOS == "darwin" {
				// Check if lima nerdctl is available
				if rd.isLimaNerdctlAvailable() {
					return "lima nerdctl", nil
				}
				continue
			}
			return rt, nil
		}
	}

	return "", fmt.Errorf("no container runtime found. Please install one of: %s", strings.Join(rd.runtimes, ", "))
}

// isAvailable checks if a runtime command is available in PATH
func (rd *RuntimeDetector) isAvailable(runtime string) bool {
	parts := strings.Fields(runtime)
	if len(parts) == 0 {
		return false
	}
	
	_, err := exec.LookPath(parts[0])
	return err == nil
}

// isLimaNerdctlAvailable checks if lima nerdctl is available and working
func (rd *RuntimeDetector) isLimaNerdctlAvailable() bool {
	cmd := exec.Command("lima", "nerdctl", "--version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// GetRuntimeInfo returns information about the detected runtime
func (rd *RuntimeDetector) GetRuntimeInfo(runtime string) (RuntimeInfo, error) {
	info := RuntimeInfo{
		Name:      runtime,
		Available: rd.isAvailable(runtime),
	}
	
	if !info.Available {
		return info, fmt.Errorf("runtime %s is not available", runtime)
	}
	
	// Get version information
	var cmd *exec.Cmd
	if strings.HasPrefix(runtime, "lima") {
		cmd = exec.Command("lima", "nerdctl", "--version")
	} else {
		cmd = exec.Command(runtime, "--version")
	}
	
	output, err := cmd.Output()
	if err != nil {
		info.Version = "unknown"
	} else {
		info.Version = strings.TrimSpace(string(output))
	}
	
	return info, nil
}

// RuntimeInfo contains information about a container runtime
type RuntimeInfo struct {
	Name      string
	Available bool
	Version   string
}

// ListAvailableRuntimes returns information about all available runtimes
func (rd *RuntimeDetector) ListAvailableRuntimes() []RuntimeInfo {
	var runtimes []RuntimeInfo
	
	for _, rt := range rd.runtimes {
		info, _ := rd.GetRuntimeInfo(rt)
		if info.Available {
			runtimes = append(runtimes, info)
		}
	}
	
	// Special case for lima nerdctl on macOS
	if runtime.GOOS == "darwin" && rd.isLimaNerdctlAvailable() {
		info, _ := rd.GetRuntimeInfo("lima nerdctl")
		if info.Available {
			runtimes = append(runtimes, info)
		}
	}
	
	return runtimes
}