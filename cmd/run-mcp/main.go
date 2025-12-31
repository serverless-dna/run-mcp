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
  run-mcp node npx @modelcontextprotocol/server-memory`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		Args:    cobra.MinimumNArgs(1),
		RunE:    runMCP,
	}

	// Add additional commands
	rootCmd.AddCommand(createInfoCommand())
	rootCmd.AddCommand(createConfigCommand())

	rootCmd.Flags().BoolP("help", "h", false, "Help for run-mcp")
	rootCmd.Flags().BoolP("version", "v", false, "Version information")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMCP(cmd *cobra.Command, args []string) error {
	config := loadConfig()

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
	containerCmd, err := buildContainerCommand(config, containerRuntime, language, args)
	if err != nil {
		return fmt.Errorf("failed to build container command: %w", err)
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

// buildContainerCommand constructs the container execution command
func buildContainerCommand(config *Config, containerRuntime, language string, args []string) (*exec.Cmd, error) {
	// Get the appropriate image for the language
	image, err := config.GetImageForLanguage(language)
	if err != nil {
		return nil, err
	}

	// Build container command arguments
	containerArgs := []string{"run", "-i", "--rm"}
	
	// Add environment variables
	envFilter := NewEnvFilter()
	containerArgs = append(containerArgs, envFilter.GetFilteredEnvArgs()...)
	
	// Add volume mounts
	volumeManager := NewVolumeManager(config)
	containerArgs = append(containerArgs, volumeManager.GetVolumeMounts()...)
	
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
		return exec.Command(parts[0], append(parts[1:], containerArgs...)...), nil
	}

	return exec.Command(containerRuntime, containerArgs...), nil
}