package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrorHandler provides enhanced error handling and recovery strategies
// Requirements: 5.2, 5.3, 5.5
type ErrorHandler struct{}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// HandleRuntimeError provides clear error messages and suggestions for runtime issues
// Requirements: 5.2, 5.3
func (eh *ErrorHandler) HandleRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// Handle runtime not found errors
	if strings.Contains(errMsg, "no container runtime found") {
		return fmt.Errorf(`container runtime not available: %w

Suggested solutions:
1. Install a supported container runtime:
   - Docker: https://docs.docker.com/get-docker/
   - Podman: https://podman.io/getting-started/installation
   - Nerdctl: https://github.com/containerd/nerdctl#installation
   - Finch (macOS): https://github.com/runfinch/finch#installation

2. Ensure the runtime is in your PATH
3. Verify the runtime is running (e.g., 'docker version')
4. Check runtime permissions (may need to add user to docker group)`, err)
	}
	
	// Handle specific runtime not found
	if strings.Contains(errMsg, "specified container runtime") && strings.Contains(errMsg, "not found in PATH") {
		return fmt.Errorf(`specified container runtime not available: %w

Suggested solutions:
1. Check the MCP_CONTAINER_RUNTIME environment variable
2. Install the specified runtime or use a different one
3. Verify the runtime executable is in your PATH
4. Remove MCP_CONTAINER_RUNTIME to use auto-detection`, err)
	}
	
	// Handle runtime permission errors
	if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "access denied") {
		return fmt.Errorf(`container runtime permission error: %w

Suggested solutions:
1. Add your user to the docker group: sudo usermod -aG docker $USER
2. Restart your shell session or log out and back in
3. Use sudo if appropriate for your setup
4. Check container runtime daemon is running`, err)
	}
	
	return err
}

// HandleVolumeError provides clear error messages and fallback strategies for volume issues
// Requirements: 5.2, 5.3, 5.5
func (eh *ErrorHandler) HandleVolumeError(err error, operation string) error {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// Handle volume creation failures
	if strings.Contains(errMsg, "failed to create volume") {
		return fmt.Errorf(`volume %s failed: %w

Suggested solutions:
1. Check available disk space: df -h
2. Verify container runtime is running properly
3. Check volume driver compatibility
4. Try using --ephemeral flag for temporary volumes
5. Clean up unused volumes: run-mcp volume prune`, operation, err)
	}
	
	// Handle volume existence check failures
	if strings.Contains(errMsg, "failed to check if volume exists") {
		return fmt.Errorf(`volume existence check failed: %w

Suggested solutions:
1. Verify container runtime is accessible
2. Check runtime daemon status
3. Ensure proper permissions for volume operations
4. Try restarting the container runtime service`, err)
	}
	
	// Handle volume removal failures
	if strings.Contains(errMsg, "failed to remove volume") {
		return fmt.Errorf(`volume removal failed: %w

Suggested solutions:
1. Ensure no containers are using the volume
2. Stop any running containers that might be using the volume
3. Check if volume is mounted elsewhere
4. Use force removal if safe: docker volume rm --force <volume-name>`, err)
	}
	
	// Handle volume not found during inspection
	if strings.Contains(errMsg, "failed to inspect volume") && (strings.Contains(errMsg, "exit status 1") || strings.Contains(errMsg, "volume not found")) {
		return fmt.Errorf(`volume inspection failed: %w

Suggested solutions:
1. Check if the volume exists: run-mcp volume list
2. Verify the server name is correct
3. Create the volume by running the MCP server first
4. Use the exact server command as specified in your MCP configuration`, err)
	}
	
	// Handle disk space issues
	if strings.Contains(errMsg, "no space left") || strings.Contains(errMsg, "disk full") {
		return fmt.Errorf(`insufficient disk space: %w

Suggested solutions:
1. Free up disk space: clean temporary files, logs, unused containers
2. Remove unused Docker images: docker image prune
3. Remove unused volumes: docker volume prune
4. Move Docker data directory to larger partition
5. Use --ephemeral flag to avoid persistent volumes`, err)
	}
	
	return err
}

// HandleFilesystemError provides clear error messages and suggestions for filesystem issues
// Requirements: 5.2, 5.3, 5.5
func (eh *ErrorHandler) HandleFilesystemError(err error, path string) error {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// Handle directory not found
	if os.IsNotExist(err) || strings.Contains(errMsg, "does not exist") {
		return fmt.Errorf(`directory not found: %s

Suggested solutions:
1. Create the directory: mkdir -p %s
2. Check the path spelling and permissions
3. Verify parent directories exist
4. Use a different directory that exists
5. Set MCP_DATA_DIR environment variable to valid directory`, path, path)
	}
	
	// Handle permission errors
	if os.IsPermission(err) || strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "not writable") {
		return fmt.Errorf(`permission error for directory: %s

Suggested solutions:
1. Fix directory permissions: chmod 755 %s
2. Change ownership if needed: sudo chown $USER %s
3. Use a directory you have write access to
4. Set MCP_DATA_DIR to a writable directory (e.g., ~/mcp-data)`, path, path, path)
	}
	
	// Handle not a directory error
	if strings.Contains(errMsg, "not a directory") {
		return fmt.Errorf(`path is not a directory: %s

Suggested solutions:
1. Use a directory path instead of a file
2. Remove the file if it's not needed: rm %s
3. Choose a different path for the data directory
4. Set MCP_DATA_DIR to a valid directory path`, path, path)
	}
	
	// Handle filesystem full
	if strings.Contains(errMsg, "no space left") || strings.Contains(errMsg, "disk full") {
		return fmt.Errorf(`filesystem full for path: %s

Suggested solutions:
1. Free up disk space on the filesystem
2. Clean temporary files and logs
3. Move to a filesystem with more space
4. Use a different directory on another filesystem`, path)
	}
	
	return fmt.Errorf("filesystem error for path %s: %w", path, err)
}

// HandleMountError provides clear error messages for mount configuration issues
// Requirements: 7.9, 7.10
func (eh *ErrorHandler) HandleMountError(err error, mountSpec string) error {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// Handle mount syntax errors
	if strings.Contains(errMsg, "invalid MCP_MOUNT syntax") {
		return fmt.Errorf(`mount configuration error: %w

Correct MCP_MOUNT format:
  MCP_MOUNT=<source>:<destination>[:<options>],<source>:<destination>[:<options>],...

Examples:
  MCP_MOUNT=~/.aws:/home/mcp/.aws:ro
  MCP_MOUNT=~/data:/data,~/.config:/home/mcp/.config:ro
  MCP_MOUNT=/host/path:/container/path:rw

Common issues:
1. Missing colon separator between source and destination
2. Using relative paths for destination (must be absolute)
3. Source path does not exist on host
4. Invalid mount options (use: ro, rw, bind, etc.)`, err)
	}
	
	// Handle source path not found
	if strings.Contains(errMsg, "source path does not exist") {
		return fmt.Errorf(`mount source path not found: %w

Suggested solutions:
1. Create the source directory: mkdir -p <source-path>
2. Check the path spelling and case sensitivity
3. Verify the path exists and is accessible
4. Use absolute paths or proper tilde expansion
5. Remove the mount if it's not needed`, err)
	}
	
	// Handle invalid destination path
	if strings.Contains(errMsg, "destination path must be absolute") {
		return fmt.Errorf(`mount destination must be absolute path: %w

Suggested solutions:
1. Use absolute paths starting with / for destinations
2. Example: /home/mcp/.aws instead of .aws
3. Container paths must be absolute for proper mounting`, err)
	}
	
	return fmt.Errorf("mount configuration error: %w", err)
}

// SuggestFallbackStrategy provides fallback options when primary operations fail
// Requirements: 5.2, 5.3
func (eh *ErrorHandler) SuggestFallbackStrategy(operation string, err error) string {
	errMsg := err.Error()
	
	switch operation {
	case "volume_creation":
		if strings.Contains(errMsg, "disk") || strings.Contains(errMsg, "space") {
			return "Consider using --ephemeral flag for temporary volumes that don't persist"
		}
		if strings.Contains(errMsg, "permission") {
			return "Try running with appropriate permissions or use MCP_BIND_HOME=true for host directory binding"
		}
		return "Use MCP_HOME_PATH to specify a custom directory instead of container volumes"
		
	case "runtime_detection":
		return "Install a container runtime (Docker, Podman, Nerdctl, or Finch) or set MCP_CONTAINER_RUNTIME to specify one"
		
	case "mount_configuration":
		return "Simplify mount configuration or remove problematic mounts from MCP_MOUNT"
		
	case "filesystem_access":
		return "Use a different directory with proper permissions or create the required directories"
		
	default:
		return "Check system requirements and container runtime installation"
	}
}

// ValidateSystemRequirements checks if the system meets basic requirements
// Requirements: 5.2, 5.3
func (eh *ErrorHandler) ValidateSystemRequirements() error {
	var issues []string
	
	// Check if we can detect any container runtime
	detector := NewRuntimeDetector()
	_, err := detector.Detect()
	if err != nil {
		issues = append(issues, "No container runtime available (Docker, Podman, Nerdctl, Finch)")
	}
	
	// Check if we can access home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		issues = append(issues, "Cannot determine user home directory")
	} else {
		// Check if home directory is accessible
		if _, err := os.Stat(homeDir); err != nil {
			issues = append(issues, fmt.Sprintf("Home directory not accessible: %s", homeDir))
		}
	}
	
	// Check if we can create temporary files (basic filesystem access)
	tempFile, err := os.CreateTemp("", "mcp-test-")
	if err != nil {
		issues = append(issues, "Cannot create temporary files - filesystem may be read-only")
	} else {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}
	
	if len(issues) > 0 {
		return fmt.Errorf("system requirements not met:\n  - %s", strings.Join(issues, "\n  - "))
	}
	
	return nil
}

// RecoverFromError attempts to recover from common error conditions
// Requirements: 5.2, 5.3
func (eh *ErrorHandler) RecoverFromError(err error, context string) error {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// Attempt to recover from directory creation failures
	if strings.Contains(context, "directory") && os.IsNotExist(err) {
		if strings.Contains(errMsg, "/.run-mcp/") {
			// Try to create the .run-mcp directory structure
			homeDir, homeErr := os.UserHomeDir()
			if homeErr == nil {
				runMcpDir := filepath.Join(homeDir, ".run-mcp")
				if mkdirErr := os.MkdirAll(runMcpDir, 0755); mkdirErr == nil {
					return nil // Successfully recovered
				}
			}
		}
	}
	
	// Attempt to recover from permission issues
	if os.IsPermission(err) && strings.Contains(context, "data_directory") {
		// Suggest using home directory as fallback
		homeDir, homeErr := os.UserHomeDir()
		if homeErr == nil {
			mcpDataDir := filepath.Join(homeDir, "mcp-data")
			if mkdirErr := os.MkdirAll(mcpDataDir, 0755); mkdirErr == nil {
				return fmt.Errorf("recovered by creating fallback directory: %s (original error: %w)", mcpDataDir, err)
			}
		}
	}
	
	// Attempt to recover from volume creation failures with fallback strategies
	if strings.Contains(context, "volume") && strings.Contains(errMsg, "failed to create") {
		// Suggest ephemeral mode as fallback
		if !strings.Contains(errMsg, "ephemeral") {
			return fmt.Errorf("%w\n\nFallback suggestion: Try using --ephemeral flag for temporary volumes", err)
		}
		
		// Suggest bind home as alternative
		return fmt.Errorf("%w\n\nFallback suggestion: Try using MCP_BIND_HOME=true to use host directory instead of container volumes", err)
	}
	
	// Attempt to recover from runtime detection failures
	if strings.Contains(context, "runtime") && strings.Contains(errMsg, "not found") {
		// Try to detect any available runtime as fallback
		detector := NewRuntimeDetector()
		availableRuntimes := detector.ListAvailableRuntimes()
		if len(availableRuntimes) > 0 {
			return fmt.Errorf("%w\n\nFallback suggestion: Available runtimes found: %v. Consider using one of these", err, availableRuntimes)
		}
	}
	
	// No recovery possible
	return err
}

// FormatUserFriendlyError formats errors in a user-friendly way with actionable suggestions
// Requirements: 5.2, 5.3, 5.5
func (eh *ErrorHandler) FormatUserFriendlyError(err error, context string) error {
	if err == nil {
		return nil
	}
	
	// Try specific error handlers first
	switch context {
	case "runtime":
		return eh.HandleRuntimeError(err)
	case "volume":
		return eh.HandleVolumeError(err, "operation")
	case "filesystem":
		return eh.HandleFilesystemError(err, "")
	case "mount":
		return eh.HandleMountError(err, "")
	}
	
	// Generic user-friendly formatting
	errMsg := err.Error()
	
	// Remove technical stack traces and internal details
	if strings.Contains(errMsg, "exec:") {
		errMsg = strings.ReplaceAll(errMsg, "exec: ", "")
	}
	
	// Add context if not already present
	if !strings.Contains(errMsg, "run-mcp") {
		errMsg = fmt.Sprintf("run-mcp error: %s", errMsg)
	}
	
	return fmt.Errorf("%s\n\nFor more help, run: run-mcp --help", errMsg)
}

// HandleContainerStartupError provides specific error handling for container startup failures
// Requirements: 5.2, 5.3, 5.5
func (eh *ErrorHandler) HandleContainerStartupError(err error, containerRuntime, image string) error {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// Handle image not found errors
	if strings.Contains(errMsg, "pull access denied") || strings.Contains(errMsg, "not found") {
		return fmt.Errorf(`container image not available: %w

Suggested solutions:
1. Check if the image name is correct: %s
2. Pull the image manually: %s pull %s
3. Check your internet connection
4. Verify you have access to the registry
5. Use a different image with MCP_NODEJS_IMAGE or MCP_PYTHON_IMAGE environment variables`, err, image, containerRuntime, image)
	}
	
	// Handle container runtime daemon not running
	if strings.Contains(errMsg, "daemon") || strings.Contains(errMsg, "connection refused") {
		return fmt.Errorf(`container runtime daemon not running: %w

Suggested solutions:
1. Start the container runtime daemon:
   - Docker: sudo systemctl start docker (Linux) or start Docker Desktop
   - Podman: sudo systemctl start podman (Linux)
2. Check daemon status: %s version
3. Restart the daemon if needed
4. Check if you have permission to access the daemon`, err, containerRuntime)
	}
	
	// Handle resource constraints
	if strings.Contains(errMsg, "no space left") || strings.Contains(errMsg, "disk full") {
		return fmt.Errorf(`insufficient resources for container: %w

Suggested solutions:
1. Free up disk space: clean temporary files, logs, unused containers
2. Remove unused images: %s image prune
3. Remove unused volumes: %s volume prune
4. Use --ephemeral flag to avoid persistent volumes
5. Check available disk space: df -h`, err, containerRuntime, containerRuntime)
	}
	
	// Handle network issues
	if strings.Contains(errMsg, "network") || strings.Contains(errMsg, "timeout") {
		return fmt.Errorf(`container networking issue: %w

Suggested solutions:
1. Check your internet connection
2. Verify container runtime network configuration
3. Try restarting the container runtime daemon
4. Check firewall settings
5. Use a different network configuration if available`, err)
	}
	
	return fmt.Errorf("container startup failed: %w", err)
}

// ProvideRecoveryGuidance provides step-by-step recovery guidance for common issues
// Requirements: 5.2, 5.3, 5.5
func (eh *ErrorHandler) ProvideRecoveryGuidance(err error, operation string) string {
	if err == nil {
		return ""
	}
	
	errMsg := err.Error()
	
	switch operation {
	case "runtime_detection":
		return `Recovery Steps for Runtime Detection:
1. Install a container runtime:
   - Docker: https://docs.docker.com/get-docker/
   - Podman: https://podman.io/getting-started/installation
   - Nerdctl: https://github.com/containerd/nerdctl#installation
   - Finch (macOS): https://github.com/runfinch/finch#installation

2. Verify installation:
   - Run: docker --version (or podman --version, etc.)
   - Ensure the runtime is in your PATH

3. Check permissions:
   - Add user to docker group: sudo usermod -aG docker $USER
   - Log out and back in, or restart your shell

4. Start the runtime daemon:
   - Docker: sudo systemctl start docker
   - Podman: sudo systemctl start podman`
	
	case "volume_creation":
		if strings.Contains(errMsg, "space") || strings.Contains(errMsg, "disk") {
			return `Recovery Steps for Volume Creation (Disk Space):
1. Check available disk space: df -h
2. Clean up unused containers: docker container prune
3. Clean up unused images: docker image prune
4. Clean up unused volumes: docker volume prune
5. Use --ephemeral flag for temporary volumes
6. Set MCP_BIND_HOME=true to use host directories instead`
		}
		
		if strings.Contains(errMsg, "permission") {
			return `Recovery Steps for Volume Creation (Permissions):
1. Check Docker daemon permissions:
   - Add user to docker group: sudo usermod -aG docker $USER
   - Restart your shell session
2. Try with sudo (if appropriate for your setup)
3. Use MCP_BIND_HOME=true to use host directories
4. Set MCP_HOME_PATH to a writable directory`
		}
		
		return `Recovery Steps for Volume Creation:
1. Check container runtime status: docker version
2. Restart container runtime daemon
3. Try --ephemeral flag for temporary volumes
4. Use MCP_BIND_HOME=true as alternative
5. Check available disk space and permissions`
	
	case "filesystem_access":
		return `Recovery Steps for Filesystem Access:
1. Check directory permissions: ls -la <directory>
2. Create missing directories: mkdir -p <directory>
3. Fix ownership: sudo chown $USER <directory>
4. Fix permissions: chmod 755 <directory>
5. Use a different directory with MCP_DATA_DIR
6. Use default home directory if custom path fails`
	
	case "mount_configuration":
		return `Recovery Steps for Mount Configuration:
1. Check MCP_MOUNT syntax: <source>:<destination>[:<options>]
2. Verify source paths exist: ls -la <source-path>
3. Use absolute paths for destinations
4. Create missing source directories: mkdir -p <source-path>
5. Simplify mount configuration (remove problematic mounts)
6. Test with a single mount first`
	
	default:
		return `General Recovery Steps:
1. Check system requirements: run-mcp doctor
2. Verify container runtime installation
3. Check available disk space and permissions
4. Review error messages for specific guidance
5. Try with --ephemeral flag if volume issues persist
6. Use MCP_BIND_HOME=true as alternative to container volumes`
	}
}

// DiagnoseSystemIssues performs comprehensive system diagnostics
// Requirements: 5.2, 5.3, 5.5
func (eh *ErrorHandler) DiagnoseSystemIssues() []string {
	var issues []string
	
	// Check container runtime availability
	detector := NewRuntimeDetector()
	_, err := detector.Detect()
	if err != nil {
		issues = append(issues, fmt.Sprintf("Container runtime: %v", err))
	}
	
	// Check home directory access
	homeDir, err := os.UserHomeDir()
	if err != nil {
		issues = append(issues, fmt.Sprintf("Home directory access: %v", err))
	} else {
		// Test home directory write access
		testFile := filepath.Join(homeDir, ".mcp-test-write")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			issues = append(issues, fmt.Sprintf("Home directory not writable: %v", err))
		} else {
			os.Remove(testFile)
		}
	}
	
	// Check configuration
	config := loadConfig()
	if err := config.Validate(); err != nil {
		issues = append(issues, fmt.Sprintf("Configuration: %v", err))
	}
	
	// Check mount configuration
	if mountStr := os.Getenv("MCP_MOUNT"); mountStr != "" {
		parser := NewUserMountParser()
		if _, err := parser.ParseUserMounts(); err != nil {
			issues = append(issues, fmt.Sprintf("Mount configuration: %v", err))
		}
	}
	
	// Check home directory overrides
	if homePath := os.Getenv("MCP_HOME_PATH"); homePath != "" {
		handler := NewHomeOverrideHandler()
		if err := handler.ValidateCustomHomePath(homePath); err != nil {
			issues = append(issues, fmt.Sprintf("Custom home path: %v", err))
		}
	}
	
	// Check disk space in common locations
	locations := []string{homeDir}
	
	// Add temp directory based on OS
	if tempDir := os.TempDir(); tempDir != "" {
		locations = append(locations, tempDir)
	}
	
	for _, location := range locations {
		if location != "" {
			if err := eh.checkDiskSpace(location); err != nil {
				issues = append(issues, fmt.Sprintf("Disk space in %s: %v", location, err))
			}
		}
	}
	
	return issues
}

// checkDiskSpace checks available disk space in a directory
func (eh *ErrorHandler) checkDiskSpace(path string) error {
	// Create a small test file to check if we can write
	testFile := filepath.Join(path, ".mcp-space-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		if strings.Contains(err.Error(), "no space left") {
			return fmt.Errorf("insufficient disk space")
		}
		return fmt.Errorf("cannot write to directory: %w", err)
	}
	
	os.Remove(testFile)
	return nil
}