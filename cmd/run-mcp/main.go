package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var ephemeralMode bool
	
	rootCmd := &cobra.Command{
		Use:   "run-mcp [flags] command [args...]",
		Short: "Run MCP servers in containers with automatic runtime detection",
		Long: `run-mcp is a cross-platform binary that automatically detects container runtimes
and language requirements to run MCP servers in containers. It provides a drop-in
replacement for direct command execution with secure environment variable passthrough
and cross-platform volume mounting.

Examples:
  run-mcp uvx mcp-server-sqlite --db-path /data/db.sqlite
  run-mcp npx @modelcontextprotocol/server-filesystem /data
  run-mcp python uvx awslabs.aws-api-mcp-server@latest
  run-mcp node npx @modelcontextprotocol/server-memory
  run-mcp --ephemeral uvx mcp-server-sqlite --db-path /data/db.sqlite
  run-mcp list-images
  run-mcp config
  run-mcp info
  run-mcp doctor`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		Args:    cobra.ArbitraryArgs, // Changed from MinimumNArgs(1) to allow subcommands
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCP(cmd, args, ephemeralMode)
		},
	}

	// Add additional commands
	rootCmd.AddCommand(createInfoCommand())
	rootCmd.AddCommand(createConfigCommand())
	rootCmd.AddCommand(createListImagesCommand())
	rootCmd.AddCommand(createVolumeCommand())
	rootCmd.AddCommand(createDoctorCommand())

	rootCmd.Flags().BoolP("help", "h", false, "Help for run-mcp")
	rootCmd.Flags().BoolP("version", "v", false, "Version information")
	rootCmd.Flags().BoolVar(&ephemeralMode, "ephemeral", false, "Use ephemeral volumes that are removed when container stops")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMCP(cmd *cobra.Command, args []string, ephemeralMode bool) error {
	// If no arguments provided, show help
	if len(args) == 0 {
		return cmd.Help()
	}

	config := loadConfig()
	config.EphemeralMode = ephemeralMode

	// Validate configuration
	if err := config.Validate(); err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleFilesystemError(err, config.DataDir)
	}

	// Detect container runtime with enhanced error handling
	detector := NewRuntimeDetector()
	containerRuntime, err := detector.Detect()
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleRuntimeError(err)
	}

	// Detect language from command
	langDetector := NewLanguageDetector()
	language, err := langDetector.DetectFromArgs(args)
	if err != nil {
		return fmt.Errorf("language detection failed: %w", err)
	}

	// Build and execute container command
	containerCmd, volumeName, err := buildContainerCommand(config, containerRuntime, language, args)
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.FormatUserFriendlyError(err, "container")
	}
	
	// Set up cleanup for ephemeral volumes
	if ephemeralMode && volumeName != "" {
		defer func() {
			volumeManager := NewVolumeManagerWithRuntime(config, containerRuntime)
			if cleanupErr := volumeManager.CleanupEphemeralVolume(volumeName); cleanupErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup ephemeral volume %s: %v\n", volumeName, cleanupErr)
			}
		}()
	}
	
	// Execute with stdio passthrough
	containerCmd.Stdin = os.Stdin
	containerCmd.Stdout = os.Stdout
	containerCmd.Stderr = os.Stderr

	if err := containerCmd.Run(); err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleContainerStartupError(err, containerRuntime, config.NodejsImage)
	}
	
	return nil
}

func createDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose system configuration and requirements",
		Long:  "Check system requirements, container runtime availability, and common configuration issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func createInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show runtime information",
		Long:  "Display information about available container runtimes, configuration, and environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showInfo()
		},
	}
}

func createConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show configuration",
		Long:  "Display current configuration settings and environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showConfig()
		},
	}
}

func createListImagesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list-images",
		Short: "List available container images",
		Long:  "List available container images for Node.js and Python from the registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listImages()
		},
	}
}

func showInfo() error {
	fmt.Println("run-mcp Runtime Information")
	fmt.Println("===========================")
	
	// Show available runtimes
	detector := NewRuntimeDetector()
	runtimes := detector.ListAvailableRuntimes()
	
	fmt.Println("\nAvailable Container Runtimes:")
	if len(runtimes) == 0 {
		fmt.Println("  None found")
	} else {
		for _, rt := range runtimes {
			fmt.Printf("  ✓ %s (%s)\n", rt.Name, rt.Version)
		}
	}
	
	// Show supported languages
	langDetector := NewLanguageDetector()
	languages := langDetector.GetSupportedLanguages()
	
	fmt.Println("\nSupported Languages:")
	for _, lang := range languages {
		commands := langDetector.GetCommandsForLanguage(lang)
		fmt.Printf("  %s: %s\n", lang, strings.Join(commands, ", "))
	}
	
	return nil
}

func showConfig() error {
	config := loadConfig()
	summary := config.GetConfigSummary()
	
	fmt.Println("run-mcp Configuration")
	fmt.Println("====================")
	
	fmt.Printf("Node.js Image: %s\n", summary.NodejsImage)
	fmt.Printf("Python Image:  %s\n", summary.PythonImage)
	fmt.Printf("Data Directory: %s", summary.DataDir)
	
	if !summary.DataDirExists {
		fmt.Print(" (does not exist)")
	} else if !summary.DataDirWritable {
		fmt.Print(" (not writable)")
	} else {
		fmt.Print(" (✓)")
	}
	fmt.Println()
	
	if summary.ContainerRuntime != "" {
		fmt.Printf("Container Runtime Override: %s\n", summary.ContainerRuntime)
	}
	
	// Show environment variables
	envVars := GetEnvironmentVariables()
	if len(envVars) > 0 {
		fmt.Println("\nEnvironment Variables:")
		for name, value := range envVars {
			fmt.Printf("  %s=%s\n", name, value)
		}
	}
	
	return nil
}

func listImages() error {
	fmt.Println("Available Container Images")
	fmt.Println("=========================")
	
	// Detect container runtime
	detector := NewRuntimeDetector()
	containerRuntime, err := detector.Detect()
	if err != nil {
		return fmt.Errorf("container runtime detection failed: %w", err)
	}
	
	config := loadConfig()
	
	// Extract registry and repository from current images
	nodejsRegistry, nodejsRepo := parseImageName(config.NodejsImage)
	pythonRegistry, pythonRepo := parseImageName(config.PythonImage)
	
	fmt.Printf("Using container runtime: %s\n\n", containerRuntime)
	
	// List Node.js images
	fmt.Println("Node.js Images:")
	fmt.Printf("Repository: %s/%s\n", nodejsRegistry, nodejsRepo)
	if err := listImageTags(containerRuntime, nodejsRegistry, nodejsRepo); err != nil {
		fmt.Printf("  Error listing Node.js images: %v\n", err)
		fmt.Println("  Common tags: latest, node18, node20, node22")
	}
	
	fmt.Println()
	
	// List Python images  
	fmt.Println("Python Images:")
	fmt.Printf("Repository: %s/%s\n", pythonRegistry, pythonRepo)
	if err := listImageTags(containerRuntime, pythonRegistry, pythonRepo); err != nil {
		fmt.Printf("  Error listing Python images: %v\n", err)
		fmt.Println("  Common tags: latest, python3.11, python3.12, python3.13")
	}
	
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  run-mcp --image <full-image-name> <command>")
	fmt.Printf("  export MCP_NODEJS_IMAGE=%s/%s:<tag>\n", nodejsRegistry, nodejsRepo)
	fmt.Printf("  export MCP_PYTHON_IMAGE=%s/%s:<tag>\n", pythonRegistry, pythonRepo)
	
	return nil
}

// parseImageName extracts registry and repository from a full image name
func parseImageName(imageName string) (registry, repo string) {
	// Handle cases like:
	// ghcr.io/owner/repo-nodejs:tag -> ghcr.io, owner/repo-nodejs
	// owner/repo-nodejs:tag -> docker.io, owner/repo-nodejs
	
	parts := strings.Split(imageName, "/")
	if len(parts) >= 3 && strings.Contains(parts[0], ".") {
		// Has registry (contains dot)
		registry = parts[0]
		repo = strings.Join(parts[1:], "/")
	} else {
		// No explicit registry, assume docker.io
		registry = "docker.io"
		repo = imageName
	}
	
	// Remove tag if present
	if colonIndex := strings.LastIndex(repo, ":"); colonIndex != -1 {
		repo = repo[:colonIndex]
	}
	
	return registry, repo
}

// listImageTags attempts to list available tags for an image repository
func listImageTags(containerRuntime, registry, repo string) error {
	// Try to use container runtime to list local images first
	localCmd := exec.Command(containerRuntime, "images", "--format", "table {{.Repository}}:{{.Tag}}", fmt.Sprintf("%s/%s", registry, repo))
	if output, err := localCmd.Output(); err == nil && len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 1 { // Skip header
			fmt.Println("  Local images:")
			for _, line := range lines[1:] {
				if strings.TrimSpace(line) != "" {
					fmt.Printf("    %s\n", strings.TrimSpace(line))
				}
			}
			return nil
		}
	}
	
	// If no local images, show common tags based on registry
	if registry == "ghcr.io" {
		fmt.Println("  No local images found. Common tags available:")
		if strings.Contains(repo, "nodejs") {
			fmt.Println("    latest, node18, node20, node22")
			fmt.Println("    node18.x.x, node20.x.x, node22.x.x (specific versions)")
		} else if strings.Contains(repo, "python") {
			fmt.Println("    latest, python3.11, python3.12, python3.13")
			fmt.Println("    python3.11.x, python3.12.x, python3.13.x (specific versions)")
		}
		fmt.Printf("  Pull with: %s pull %s/%s:<tag>\n", containerRuntime, registry, repo)
	} else {
		return fmt.Errorf("unable to list remote tags for %s/%s", registry, repo)
	}
	
	return nil
}

// buildContainerCommand constructs the container execution command with ProcessManager integration
// Requirements: 1.5, 5.1, 5.4, 4.1, 4.2, 4.3, 4.4
func buildContainerCommand(config *Config, containerRuntime, language string, args []string) (*exec.Cmd, string, error) {
	// Get the appropriate image for the language
	image, err := config.GetImageForLanguage(language)
	if err != nil {
		return nil, "", err
	}

	// Create ProcessManager with container naming for signal forwarding
	// Requirements: 4.1, 4.2, 4.3, 4.4
	processManager := NewContainerProcessManager(containerRuntime, args)
	
	// Build base container command arguments
	containerArgs := []string{"run", "-i", "--rm"}
	
	// Add unique container name for signal forwarding
	containerArgs = append(containerArgs, "--name", processManager.GetContainerName())
	
	// Add environment variables (filtered to exclude MCP configuration variables)
	// Requirements: 3.1, 3.3, 3.5
	envFilter := NewEnvFilter()
	containerArgs = append(containerArgs, envFilter.GetFilteredEnvArgs()...)
	
	// Handle home directory override support first
	// Requirements: 7.6, 7.7
	homeOverrideHandler := NewHomeOverrideHandler()
	homeMount := homeOverrideHandler.GetHomeMount(args)
	
	var volumeName string
	serverName := strings.Join(args, " ")
	
	if homeMount != "" {
		// Use home directory override (MCP_BIND_HOME or MCP_HOME_PATH)
		// For MCP_BIND_HOME, create the bind directory if needed
		bindHomeValue := os.Getenv("MCP_BIND_HOME")
		if bindHomeValue != "" {
			// Check if MCP_BIND_HOME is truthy
			lower := strings.ToLower(strings.TrimSpace(bindHomeValue))
			isTruthy := lower == "true" || lower == "1" || lower == "yes" || lower == "on"
			
			if isTruthy {
				// Create bind home directory with enhanced error handling
				volumeNameForBind := sanitizeVolumeName(args)
				bindPath, err := homeOverrideHandler.CreateBindHomeDir(volumeNameForBind)
				if err != nil {
					errorHandler := NewErrorHandler()
					return nil, "", errorHandler.HandleFilesystemError(err, "~/.run-mcp/"+volumeNameForBind)
				}
				homeMount = bindPath
			}
		}
		
		// Add home directory override mount
		// Requirements: 1.5 - Mount at /home/mcp consistently
		containerArgs = append(containerArgs, "-v", fmt.Sprintf("%s:/home/mcp", homeMount))
		volumeName = "" // No volume name when using override
	} else {
		// Use container volume (default behavior)
		// Requirements: 1.1, 1.2, 1.5
		volumeManager := NewVolumeManagerWithRuntime(config, containerRuntime)
		
		if config.EphemeralMode {
			volumeName, err = volumeManager.CreateEphemeralVolume(serverName, containerRuntime)
			if err != nil {
				errorHandler := NewErrorHandler()
				return nil, "", errorHandler.HandleVolumeError(err, "ephemeral volume creation")
			}
		} else {
			volumeName, err = volumeManager.CreateHomeVolume(serverName, containerRuntime)
			if err != nil {
				errorHandler := NewErrorHandler()
				return nil, "", errorHandler.HandleVolumeError(err, "home volume creation")
			}
		}
		
		// Add home volume mount - Requirements: 1.5
		containerArgs = append(containerArgs, "-v", fmt.Sprintf("%s:/home/mcp", volumeName))
	}
	
	// Add user-specified mounts from MCP_MOUNT with enhanced error handling
	// Requirements: 7.1, 7.2, 7.3, 7.4, 7.8
	userMountParser := NewUserMountParser()
	userMounts, err := userMountParser.ParseUserMounts()
	if err != nil {
		errorHandler := NewErrorHandler()
		return nil, "", errorHandler.HandleMountError(err, os.Getenv("MCP_MOUNT"))
	}
	
	if len(userMounts) > 0 {
		userMountArgs := userMountParser.GetMountArgs(userMounts)
		containerArgs = append(containerArgs, userMountArgs...)
	}
	
	// Add image
	containerArgs = append(containerArgs, image)
	
	// Handle explicit runtime specification - Requirements: 5.1, 5.4 (backward compatibility)
	if len(args) >= 2 && (args[0] == "python" || args[0] == "node" || args[0] == "nodejs") {
		containerArgs = append(containerArgs, args[1:]...)
	} else {
		containerArgs = append(containerArgs, args...)
	}

	// Build final command using ProcessManager for consistent runtime handling
	// This handles both single-word runtimes ("docker") and multi-word runtimes ("lima nerdctl")
	parts := strings.Fields(containerRuntime)
	if len(parts) > 1 {
		// Multi-word runtime (e.g., "lima nerdctl")
		finalArgs := append(parts[1:], containerArgs...)
		return exec.Command(parts[0], finalArgs...), volumeName, nil
	}
	
	// Single-word runtime (e.g., "docker", "podman")
	return exec.Command(containerRuntime, containerArgs...), volumeName, nil
}
// createVolumeCommand creates the volume management command with subcommands
// Requirements: 4.4, 4.5, 4.6, 4.8, 4.9, 4.13, 2.10
func createVolumeCommand() *cobra.Command {
	volumeCmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage container volumes",
		Long:  "Manage container volumes used by MCP servers for persistent home directories",
	}

	// Add subcommands
	volumeCmd.AddCommand(createVolumeListCommand())
	volumeCmd.AddCommand(createVolumeCleanCommand())
	volumeCmd.AddCommand(createVolumePruneCommand())
	volumeCmd.AddCommand(createVolumeInspectCommand())

	return volumeCmd
}

// createVolumeListCommand creates the volume list subcommand
// Requirements: 4.4, 4.5, 2.10
func createVolumeListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all managed volumes",
		Long:  "List all container volumes managed by run-mcp with creation date and size information",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listVolumes()
		},
	}
}

// createVolumeCleanCommand creates the volume clean subcommand
// Requirements: 4.5, 4.9
func createVolumeCleanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clean <server-name>",
		Short: "Remove a specific volume",
		Long:  "Remove the container volume for a specific MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanVolume(args[0])
		},
	}
}

// createVolumePruneCommand creates the volume prune subcommand
// Requirements: 4.6, 4.9, 4.13
func createVolumePruneCommand() *cobra.Command {
	var force bool
	
	pruneCmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove all managed volumes",
		Long:  "Remove all container volumes managed by run-mcp with user confirmation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return pruneVolumes(force)
		},
	}
	
	pruneCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	
	return pruneCmd
}

// createVolumeInspectCommand creates the volume inspect subcommand
// Requirements: 4.8, 4.13
func createVolumeInspectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <server-name>",
		Short: "Show volume details",
		Long:  "Show detailed information about a specific volume including mount point and contents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return inspectVolume(args[0])
		},
	}
}

// listVolumes lists all managed volumes with enhanced error handling
// Requirements: 4.4, 4.5, 2.10
func listVolumes() error {
	// Detect container runtime
	detector := NewRuntimeDetector()
	containerRuntime, err := detector.Detect()
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleRuntimeError(err)
	}
	
	// Create volume commander
	commander := NewVolumeCommander(containerRuntime)
	
	// List volumes
	volumes, err := commander.ListVolumes()
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleVolumeError(err, "volume listing")
	}
	
	if len(volumes) == 0 {
		fmt.Println("No managed volumes found.")
		return nil
	}
	
	fmt.Printf("Managed Volumes (Runtime: %s)\n", containerRuntime)
	fmt.Println("================================")
	
	for _, vol := range volumes {
		fmt.Printf("Name: %s\n", vol.Name)
		if serverName, exists := vol.Labels["run-mcp.server"]; exists {
			fmt.Printf("  Server: %s\n", serverName)
		}
		fmt.Printf("  Created: %s\n", vol.CreatedAt.Format("2006-01-02 15:04:05"))
		if vol.Size != "" {
			fmt.Printf("  Size: %s\n", vol.Size)
		}
		fmt.Printf("  Runtime: %s\n", vol.Runtime)
		fmt.Println()
	}
	
	return nil
}

// cleanVolume removes a specific volume with enhanced error handling
// Requirements: 4.5, 4.9
func cleanVolume(serverName string) error {
	// Detect container runtime
	detector := NewRuntimeDetector()
	containerRuntime, err := detector.Detect()
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleRuntimeError(err)
	}
	
	// Create volume commander
	commander := NewVolumeCommander(containerRuntime)
	
	// Generate volume name from server name
	volumeName := sanitizeVolumeName(strings.Fields(serverName))
	
	// Check if volume exists
	exists, err := commander.VolumeExists(volumeName)
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleVolumeError(err, "volume existence check")
	}
	
	if !exists {
		fmt.Printf("Volume for server '%s' not found (expected volume name: %s)\n", serverName, volumeName)
		return nil
	}
	
	// Confirm deletion
	if !promptConfirmation(fmt.Sprintf("Are you sure you want to remove volume '%s'? This will permanently delete all data", volumeName)) {
		fmt.Println("Operation cancelled.")
		return nil
	}
	
	// Remove volume
	if err := commander.RemoveVolume(volumeName); err != nil {
		errorMsg := err.Error()
		// Check if error indicates volume is in use
		if strings.Contains(errorMsg, "volume is in use") || 
		   strings.Contains(errorMsg, "device or resource busy") ||
		   strings.Contains(errorMsg, "exit status 1") {
			fmt.Printf("⚠️  Cannot remove volume '%s': currently in use by a running container\n", volumeName)
			fmt.Println("\n💡 To resolve this:")
			fmt.Println("   1. Find the running container: docker ps")
			fmt.Println("   2. Stop the container: docker stop <container-id>")
			fmt.Println("   3. Retry the volume removal")
			return nil
		} else {
			errorHandler := NewErrorHandler()
			return errorHandler.HandleVolumeError(err, "volume removal")
		}
	}
	
	fmt.Printf("✅ Volume '%s' removed successfully.\n", volumeName)
	return nil
}

// pruneVolumes removes all managed volumes with enhanced error handling
// Requirements: 4.6, 4.9, 4.13
func pruneVolumes(force bool) error {
	// Detect container runtime
	detector := NewRuntimeDetector()
	containerRuntime, err := detector.Detect()
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleRuntimeError(err)
	}
	
	// Create volume commander
	commander := NewVolumeCommander(containerRuntime)
	
	// List volumes
	volumes, err := commander.ListVolumes()
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleVolumeError(err, "volume listing")
	}
	
	if len(volumes) == 0 {
		fmt.Println("No managed volumes found to prune.")
		return nil
	}
	
	// Show what will be removed
	fmt.Printf("The following %d volume(s) will be removed:\n", len(volumes))
	for _, vol := range volumes {
		fmt.Printf("  - %s", vol.Name)
		if serverName, exists := vol.Labels["run-mcp.server"]; exists {
			fmt.Printf(" (server: %s)", serverName)
		}
		fmt.Println()
	}
	fmt.Println()
	
	// Confirm deletion unless force flag is used
	if !force {
		if !promptConfirmation("Are you sure you want to remove ALL managed volumes? This will permanently delete all data") {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}
	
	// Remove all volumes
	var errors []string
	var inUseVolumes []string
	successCount := 0
	
	for _, vol := range volumes {
		if err := commander.RemoveVolume(vol.Name); err != nil {
			errorMsg := err.Error()
			// Check if error indicates volume is in use
			if strings.Contains(errorMsg, "volume is in use") || 
			   strings.Contains(errorMsg, "device or resource busy") ||
			   strings.Contains(errorMsg, "exit status 1") {
				inUseVolumes = append(inUseVolumes, vol.Name)
				fmt.Printf("⚠️  Skipped volume (in use): %s\n", vol.Name)
			} else {
				errors = append(errors, fmt.Sprintf("%s: %v", vol.Name, err))
				fmt.Printf("❌ Failed to remove volume: %s (%v)\n", vol.Name, err)
			}
		} else {
			fmt.Printf("✅ Removed volume: %s\n", vol.Name)
			successCount++
		}
	}
	
	// Print summary
	totalFailed := len(errors) + len(inUseVolumes)
	if totalFailed > 0 {
		fmt.Printf("\nSummary: %d volume(s) removed successfully, %d could not be removed\n", successCount, totalFailed)
		
		if len(inUseVolumes) > 0 {
			fmt.Printf("\n⚠️  %d volume(s) skipped (currently in use by running containers):\n", len(inUseVolumes))
			for _, volName := range inUseVolumes {
				fmt.Printf("  - %s\n", volName)
			}
			fmt.Println("\n💡 Tip: Stop running containers first, then retry:")
			fmt.Println("   docker ps                    # List running containers")
			fmt.Println("   docker stop <container-id>   # Stop specific container")
			fmt.Println("   docker stop $(docker ps -q)  # Stop all running containers")
		}
		
		if len(errors) > 0 {
			fmt.Printf("\n❌ %d volume(s) failed with other errors:\n", len(errors))
			for _, errMsg := range errors {
				fmt.Printf("  - %s\n", errMsg)
			}
		}
		
		return nil // Don't return error to avoid double printing
	}
	
	fmt.Printf("\n✅ Successfully removed all %d volume(s).\n", len(volumes))
	return nil
}

// inspectVolume shows detailed information about a volume with enhanced error handling
// Requirements: 4.8, 4.13
func inspectVolume(serverName string) error {
	// Detect container runtime
	detector := NewRuntimeDetector()
	containerRuntime, err := detector.Detect()
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleRuntimeError(err)
	}
	
	// Create volume commander
	commander := NewVolumeCommander(containerRuntime)
	
	// Generate volume name from server name
	volumeName := sanitizeVolumeName(strings.Fields(serverName))
	
	// Inspect volume
	details, err := commander.InspectVolume(volumeName)
	if err != nil {
		errorHandler := NewErrorHandler()
		return errorHandler.HandleVolumeError(err, "volume inspection")
	}
	
	fmt.Printf("Volume Details: %s\n", details.Name)
	fmt.Println("========================")
	
	if serverName, exists := details.Labels["run-mcp.server"]; exists {
		fmt.Printf("Server: %s\n", serverName)
	}
	
	fmt.Printf("Created: %s\n", details.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Runtime: %s\n", details.Runtime)
	
	if details.Size != "" {
		fmt.Printf("Size: %s\n", details.Size)
	}
	
	if details.MountPoint != "" {
		fmt.Printf("Mount Point: %s\n", details.MountPoint)
	}
	
	// Show labels
	if len(details.Labels) > 0 {
		fmt.Println("\nLabels:")
		for key, value := range details.Labels {
			fmt.Printf("  %s=%s\n", key, value)
		}
	}
	
	// Show options if available
	if len(details.Options) > 0 {
		fmt.Println("\nOptions:")
		for key, value := range details.Options {
			fmt.Printf("  %s=%s\n", key, value)
		}
	}
	
	return nil
}

// promptConfirmation prompts the user for confirmation
// Requirements: 4.9, 4.13
func promptConfirmation(message string) bool {
	fmt.Printf("%s [y/N]: ", message)
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}
// runDoctor performs system diagnostics and provides troubleshooting information
// Requirements: 5.2, 5.3, 5.5
func runDoctor() error {
	fmt.Println("run-mcp System Diagnostics")
	fmt.Println("=========================")
	
	errorHandler := NewErrorHandler()
	var issues []string
	var warnings []string
	
	// Check system requirements
	fmt.Println("\n1. Checking system requirements...")
	if err := errorHandler.ValidateSystemRequirements(); err != nil {
		issues = append(issues, fmt.Sprintf("System requirements: %v", err))
		fmt.Printf("   ❌ %v\n", err)
	} else {
		fmt.Println("   ✅ System requirements met")
	}
	
	// Check container runtime
	fmt.Println("\n2. Checking container runtime...")
	detector := NewRuntimeDetector()
	runtime, err := detector.Detect()
	if err != nil {
		issues = append(issues, fmt.Sprintf("Container runtime: %v", err))
		fmt.Printf("   ❌ %v\n", err)
		
		// Show available runtimes
		fmt.Println("\n   Available runtimes:")
		availableRuntimes := detector.ListAvailableRuntimes()
		if len(availableRuntimes) == 0 {
			fmt.Println("     None found")
		} else {
			for _, rt := range availableRuntimes {
				fmt.Printf("     ✅ %s (%s)\n", rt.Name, rt.Version)
			}
		}
		
		// Provide recovery guidance
		fmt.Println("\n   Recovery guidance:")
		guidance := errorHandler.ProvideRecoveryGuidance(err, "runtime_detection")
		fmt.Println(guidance)
	} else {
		fmt.Printf("   ✅ Container runtime: %s\n", runtime)
		
		// Test runtime functionality
		fmt.Println("   Testing runtime functionality...")
		commander := NewVolumeCommander(runtime)
		_, err := commander.ListVolumes()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Runtime test failed: %v", err))
			fmt.Printf("   ⚠️  Runtime test failed: %v\n", err)
		} else {
			fmt.Println("   ✅ Runtime is functional")
		}
	}
	
	// Check configuration
	fmt.Println("\n3. Checking configuration...")
	config := loadConfig()
	if err := config.Validate(); err != nil {
		issues = append(issues, fmt.Sprintf("Configuration: %v", err))
		fmt.Printf("   ❌ %v\n", err)
		
		// Provide recovery guidance for configuration issues
		fmt.Println("\n   Recovery guidance:")
		guidance := errorHandler.ProvideRecoveryGuidance(err, "filesystem_access")
		fmt.Println(guidance)
	} else {
		fmt.Println("   ✅ Configuration is valid")
		
		// Check data directory permissions
		fmt.Printf("   Data directory: %s\n", config.DataDir)
		vm := NewVolumeManager(config)
		if err := vm.ValidateDataDir(); err != nil {
			warnings = append(warnings, fmt.Sprintf("Data directory: %v", err))
			fmt.Printf("   ⚠️  %v\n", err)
		} else {
			fmt.Println("   ✅ Data directory is accessible and writable")
		}
	}
	
	// Check environment variables
	fmt.Println("\n4. Checking environment configuration...")
	envVars := GetEnvironmentVariables()
	if len(envVars) > 0 {
		fmt.Println("   Environment variables:")
		for name, value := range envVars {
			fmt.Printf("     %s=%s\n", name, value)
		}
	} else {
		fmt.Println("   No MCP environment variables set (using defaults)")
	}
	
	// Check mount configuration
	fmt.Println("\n5. Checking mount configuration...")
	if mountStr := os.Getenv("MCP_MOUNT"); mountStr != "" {
		fmt.Printf("   MCP_MOUNT: %s\n", mountStr)
		parser := NewUserMountParser()
		mounts, err := parser.ParseUserMounts()
		if err != nil {
			issues = append(issues, fmt.Sprintf("Mount configuration: %v", err))
			fmt.Printf("   ❌ %v\n", err)
			
			// Provide recovery guidance for mount issues
			fmt.Println("\n   Recovery guidance:")
			guidance := errorHandler.ProvideRecoveryGuidance(err, "mount_configuration")
			fmt.Println(guidance)
		} else {
			fmt.Printf("   ✅ Mount configuration is valid (%d mounts)\n", len(mounts))
			for _, mount := range mounts {
				fmt.Printf("     %s -> %s", mount.Source, mount.Destination)
				if mount.Options != "" {
					fmt.Printf(" (%s)", mount.Options)
				}
				fmt.Println()
			}
		}
	} else {
		fmt.Println("   No custom mounts configured")
	}
	
	// Check home directory overrides
	if bindHome := os.Getenv("MCP_BIND_HOME"); bindHome != "" {
		fmt.Printf("   MCP_BIND_HOME: %s\n", bindHome)
	}
	if homePath := os.Getenv("MCP_HOME_PATH"); homePath != "" {
		fmt.Printf("   MCP_HOME_PATH: %s\n", homePath)
		handler := NewHomeOverrideHandler()
		if err := handler.ValidateCustomHomePath(homePath); err != nil {
			warnings = append(warnings, fmt.Sprintf("Custom home path: %v", err))
			fmt.Printf("   ⚠️  %v\n", err)
		} else {
			fmt.Println("   ✅ Custom home path is valid")
		}
	}
	
	// Perform comprehensive system diagnostics
	fmt.Println("\n6. Running comprehensive diagnostics...")
	systemIssues := errorHandler.DiagnoseSystemIssues()
	for _, issue := range systemIssues {
		if !containsIssue(issues, issue) && !containsIssue(warnings, issue) {
			warnings = append(warnings, issue)
			fmt.Printf("   ⚠️  %s\n", issue)
		}
	}
	
	if len(systemIssues) == 0 {
		fmt.Println("   ✅ No additional issues detected")
	}
	
	// Summary
	fmt.Println("\n" + strings.Repeat("=", 50))
	if len(issues) == 0 {
		fmt.Println("✅ All checks passed! run-mcp should work correctly.")
	} else {
		fmt.Printf("❌ Found %d issue(s) that need attention:\n", len(issues))
		for i, issue := range issues {
			fmt.Printf("   %d. %s\n", i+1, issue)
		}
	}
	
	if len(warnings) > 0 {
		fmt.Printf("\n⚠️  Found %d warning(s):\n", len(warnings))
		for i, warning := range warnings {
			fmt.Printf("   %d. %s\n", i+1, warning)
		}
	}
	
	if len(issues) > 0 || len(warnings) > 0 {
		fmt.Println("\nFor help resolving these issues:")
		fmt.Println("  - Review the recovery guidance above")
		fmt.Println("  - Run: run-mcp --help")
		fmt.Println("  - Check container runtime documentation")
		fmt.Println("  - Visit project documentation")
	}
	
	return nil
}

// containsIssue checks if an issue is already in the list
func containsIssue(issues []string, issue string) bool {
	for _, existing := range issues {
		if strings.Contains(existing, issue) || strings.Contains(issue, existing) {
			return true
		}
	}
	return false
}