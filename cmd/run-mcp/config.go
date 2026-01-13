package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds runtime configuration
type Config struct {
	NodejsImage      string
	PythonImage      string
	DataDir          string
	ContainerRuntime string
	EphemeralMode    bool          // New field for ephemeral volume support
	MaxVolumeSize    string        // Maximum volume size for storage warnings (Requirements: 6.6)
	SignalConfig     *SignalConfig // Signal handling configuration
}

// loadConfig loads configuration from environment variables with defaults
func loadConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	config := &Config{
		NodejsImage:      getEnvWithDefault(MCPNodejsImage, "ghcr.io/serverless-dna/run-mcp-nodejs:latest"),
		PythonImage:      getEnvWithDefault(MCPPythonImage, "ghcr.io/serverless-dna/run-mcp-python:latest"),
		DataDir:          getEnvWithDefault(MCPDataDir, homeDir),
		ContainerRuntime: os.Getenv(MCPContainerRuntime), // Optional override
		MaxVolumeSize:    os.Getenv(MCPMaxVolumeSize),    // Optional storage limit for warnings
		SignalConfig:     LoadSignalConfig(),             // Load signal handling configuration
	}

	return config
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetImageForLanguage returns the appropriate container image for a language
func (c *Config) GetImageForLanguage(language string) (string, error) {
	switch strings.ToLower(language) {
	case "python":
		return c.PythonImage, nil
	case "nodejs", "node":
		return c.NodejsImage, nil
	default:
		return "", fmt.Errorf("unsupported language: %s (supported: nodejs, python)", language)
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate images are specified
	if c.NodejsImage == "" {
		return fmt.Errorf("%s cannot be empty", MCPNodejsImage)
	}
	if c.PythonImage == "" {
		return fmt.Errorf("%s cannot be empty", MCPPythonImage)
	}

	// Validate data directory
	if c.DataDir == "" {
		return fmt.Errorf("%s cannot be empty", MCPDataDir)
	}

	// Check if data directory exists and is accessible
	if err := c.validateDataDir(); err != nil {
		return fmt.Errorf("data directory validation failed: %w", err)
	}

	return nil
}

// validateDataDir checks if the data directory is valid
func (c *Config) validateDataDir() error {
	info, err := os.Stat(c.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("data directory does not exist: %s", c.DataDir)
		}
		return fmt.Errorf("cannot access data directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("data directory path is not a directory: %s", c.DataDir)
	}

	return nil
}

// GetConfigSummary returns a summary of the current configuration
func (c *Config) GetConfigSummary() ConfigSummary {
	return ConfigSummary{
		NodejsImage:      c.NodejsImage,
		PythonImage:      c.PythonImage,
		DataDir:          c.DataDir,
		ContainerRuntime: c.ContainerRuntime,
		DataDirExists:    c.dataDirExists(),
		DataDirWritable:  c.dataDirWritable(),
	}
}

// dataDirExists checks if data directory exists
func (c *Config) dataDirExists() bool {
	_, err := os.Stat(c.DataDir)
	return err == nil
}

// dataDirWritable checks if data directory is writable
func (c *Config) dataDirWritable() bool {
	if !c.dataDirExists() {
		return false
	}

	testFile := filepath.Join(c.DataDir, ".mcp-write-test")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		return false
	}

	os.Remove(testFile)
	return true
}

// ConfigSummary contains configuration information for display
type ConfigSummary struct {
	NodejsImage      string
	PythonImage      string
	DataDir          string
	ContainerRuntime string
	DataDirExists    bool
	DataDirWritable  bool
}

// SetImageForLanguage sets the container image for a specific language
func (c *Config) SetImageForLanguage(language, image string) error {
	switch strings.ToLower(language) {
	case "python":
		c.PythonImage = image
		return nil
	case "nodejs", "node":
		c.NodejsImage = image
		return nil
	default:
		return fmt.Errorf("unsupported language: %s", language)
	}
}

// GetSupportedLanguages returns the list of supported languages
func (c *Config) GetSupportedLanguages() []string {
	return []string{"nodejs", "python"}
}

// GetDefaultImages returns the default images for each language
func GetDefaultImages() map[string]string {
	return map[string]string{
		"nodejs": "ghcr.io/serverless-dna/run-mcp-nodejs:latest",
		"python": "ghcr.io/serverless-dna/run-mcp-python:latest",
	}
}

// GetEnvironmentVariables returns all MCP-related environment variables
func GetEnvironmentVariables() map[string]string {
	vars := make(map[string]string)

	// Use centralized list of all MCP environment variables
	for _, varName := range AllMCPEnvVars() {
		if value := os.Getenv(varName); value != "" {
			vars[varName] = value
		}
	}

	return vars
}
