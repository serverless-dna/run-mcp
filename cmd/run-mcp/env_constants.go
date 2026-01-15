package main

// MCP Environment Variables
// This file contains the centralized enumeration of all MCP environment variables
// used by run-mcp. This ensures consistency and makes it easy to maintain the
// complete list in one place.

const (
	// Container Configuration
	MCPNodejsImage      = "MCP_NODEJS_IMAGE"      // Container image for Node.js runtime
	MCPPythonImage      = "MCP_PYTHON_IMAGE"      // Container image for Python runtime
	MCPContainerRuntime = "MCP_CONTAINER_RUNTIME" // Override for container runtime detection

	// Data and Volume Configuration
	MCPDataDir       = "MCP_DATA_DIR"        // Data directory to mount at /data
	MCPMount         = "MCP_MOUNT"           // User-specified bind mounts
	MCPBindHome      = "MCP_BIND_HOME"       // Use host directory instead of container volume for home
	MCPHomePath      = "MCP_HOME_PATH"       // Custom path for container home directory
	MCPMaxVolumeSize = "MCP_MAX_VOLUME_SIZE" // Maximum volume size for storage warnings

	// Environment Variable Configuration
	MCPPassthroughEnv = "MCP_PASSTHROUGH_ENV" // Environment variables to pass through to container

	// Signal and Process Configuration
	MCPSignalTimeout = "MCP_SIGNAL_TIMEOUT" // Timeout for signal handling

	// Debug and Logging
	MCPDebug = "MCP_DEBUG" // Enable debug logging
)

// AllMCPEnvVars returns a slice of all MCP environment variable names
// This is useful for validation, documentation, and testing purposes
func AllMCPEnvVars() []string {
	return []string{
		MCPNodejsImage,
		MCPPythonImage,
		MCPContainerRuntime,
		MCPDataDir,
		MCPMount,
		MCPBindHome,
		MCPHomePath,
		MCPMaxVolumeSize,
		MCPPassthroughEnv,
		MCPSignalTimeout,
		MCPDebug,
	}
}

// MCPEnvVarDescriptions returns a map of environment variable names to their descriptions
// This is useful for generating documentation and help text
func MCPEnvVarDescriptions() map[string]string {
	return map[string]string{
		MCPNodejsImage:      "Container image for Node.js runtime",
		MCPPythonImage:      "Container image for Python runtime",
		MCPContainerRuntime: "Override for container runtime detection (docker/podman)",
		MCPDataDir:          "Data directory to mount at /data in container",
		MCPMount:            "User-specified bind mounts (format: host:container:options)",
		MCPBindHome:         "Use host directory instead of container volume for home",
		MCPHomePath:         "Custom path for container home directory",
		MCPMaxVolumeSize:    "Maximum volume size for storage warnings",
		MCPPassthroughEnv:   "Environment variables to pass through to container (comma-separated, supports wildcards)",
		MCPSignalTimeout:    "Timeout for signal handling (e.g., '10s', '1m')",
		MCPDebug:            "Enable debug logging (true/false or 1/0)",
	}
}

// ConfigurationMCPEnvVars returns environment variables that are consumed by run-mcp
// and should NOT be passed through to containers
func ConfigurationMCPEnvVars() []string {
	return []string{
		MCPMount,
		MCPBindHome,
		MCPHomePath,
		MCPPassthroughEnv, // This controls passthrough but shouldn't be passed itself
	}
}

// ContainerMCPEnvVars returns environment variables that may be passed to containers
// These are MCP variables that containers might need to know about
func ContainerMCPEnvVars() []string {
	return []string{
		MCPDataDir,       // Container may need to know data directory path
		MCPMaxVolumeSize, // Container may need volume size limits
		MCPSignalTimeout, // Container may need signal timeout info
		MCPDebug,         // Container may need debug flag
	}
}

// IsConfigurationEnvVar returns true if the given environment variable name
// is a run-mcp configuration variable that should not be passed to containers
func IsConfigurationEnvVar(envVar string) bool {
	configVars := ConfigurationMCPEnvVars()
	for _, configVar := range configVars {
		if envVar == configVar {
			return true
		}
	}
	return false
}
