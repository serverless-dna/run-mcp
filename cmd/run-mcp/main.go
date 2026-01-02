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
  run-mcp info`,
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
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Detect container runtime
	detector := NewRuntimeDetector()
	containerRuntime, err := detector.Detect()
	if err != nil {
		return fmt.Errorf("container runtime detection failed: %w", err)
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
		return fmt.Errorf("failed to build container command: %w", err)
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

	return containerCmd.Run()
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

// buildContainerCommand constructs the container execution command
func buildContainerCommand(config *Config, containerRuntime, language string, args []string) (*exec.Cmd, string, error) {
	// Get the appropriate image for the language
	image, err := config.GetImageForLanguage(language)
	if err != nil {
		return nil, "", err
	}

	// Build container command arguments
	containerArgs := []string{"run", "-i", "--rm"}
	
	// Add environment variables
	envFilter := NewEnvFilter()
	containerArgs = append(containerArgs, envFilter.GetFilteredEnvArgs()...)
	
	// Add volume mounts (including home volume)
	volumeManager := NewVolumeManagerWithRuntime(config, containerRuntime)
	volumeMounts := volumeManager.GetVolumeMounts()
	containerArgs = append(containerArgs, volumeMounts...)
	
	// Create and add home volume mount
	var volumeName string
	serverName := strings.Join(args, " ")
	
	if config.EphemeralMode {
		volumeName, err = volumeManager.CreateEphemeralVolume(serverName, containerRuntime)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create ephemeral volume: %w", err)
		}
	} else {
		volumeName, err = volumeManager.CreateHomeVolume(serverName, containerRuntime)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create home volume: %w", err)
		}
	}
	
	// Add home volume mount
	containerArgs = append(containerArgs, "-v", fmt.Sprintf("%s:/home/mcp", volumeName))
	
	// Add image
	containerArgs = append(containerArgs, image)
	
	// Handle explicit runtime specification
	if len(args) >= 2 && (args[0] == "python" || args[0] == "node" || args[0] == "nodejs") {
		containerArgs = append(containerArgs, args[1:]...)
	} else {
		containerArgs = append(containerArgs, args...)
	}

	// Handle lima nerdctl special case
	if strings.HasPrefix(containerRuntime, "lima") {
		parts := strings.Fields(containerRuntime)
		return exec.Command(parts[0], append(parts[1:], containerArgs...)...), volumeName, nil
	}

	return exec.Command(containerRuntime, containerArgs...), volumeName, nil
}