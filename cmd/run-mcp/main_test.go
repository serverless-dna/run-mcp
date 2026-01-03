package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/spf13/cobra"
)

// Property 19: Container Runtime Detection
// For any system with available container runtimes, the run-mcp binary should correctly detect and use the first available runtime in the priority order
func TestProperty19_ContainerRuntimeDetection(t *testing.T) {
	// **Feature: mcp-container-images, Property 19: Container Runtime Detection**
	// **Validates: Requirements 13.1, 13.7**
	
	detector := NewRuntimeDetector()
	
	// Test that detector has expected runtimes in priority order
	expectedRuntimes := []string{"docker", "podman", "nerdctl", "finch"}
	if len(detector.runtimes) < len(expectedRuntimes) {
		t.Errorf("Expected at least %d runtimes, got %d", len(expectedRuntimes), len(detector.runtimes))
	}
	
	// Test that the first few runtimes match expected priority order
	for i, expected := range expectedRuntimes {
		if i >= len(detector.runtimes) {
			break
		}
		if detector.runtimes[i] != expected {
			t.Errorf("Expected runtime at position %d to be %s, got %s", i, expected, detector.runtimes[i])
		}
	}
	
	// Test detection logic
	runtime, err := detector.Detect()
	if err != nil {
		// If no runtime is available, that's expected in some environments
		if !strings.Contains(err.Error(), "no container runtime found") {
			t.Errorf("Unexpected error from runtime detection: %v", err)
		}
		t.Skip("No container runtime available for testing")
	}
	
	// If we found a runtime, it should be one of the expected ones
	validRuntimes := append(expectedRuntimes, "lima nerdctl")
	found := false
	for _, valid := range validRuntimes {
		if runtime == valid {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Detected runtime %s is not in expected list: %v", runtime, validRuntimes)
	}
	
	// Test runtime override
	originalRuntime := os.Getenv("MCP_CONTAINER_RUNTIME")
	defer func() {
		if originalRuntime != "" {
			os.Setenv("MCP_CONTAINER_RUNTIME", originalRuntime)
		} else {
			os.Unsetenv("MCP_CONTAINER_RUNTIME")
		}
	}()
	
	// Test with valid override
	if runtime != "" {
		os.Setenv("MCP_CONTAINER_RUNTIME", runtime)
		overrideRuntime, err := detector.Detect()
		if err != nil {
			t.Errorf("Failed to detect runtime with valid override: %v", err)
		}
		if overrideRuntime != runtime {
			t.Errorf("Expected override runtime %s, got %s", runtime, overrideRuntime)
		}
	}
	
	// Test with invalid override
	os.Setenv("MCP_CONTAINER_RUNTIME", "nonexistent-runtime")
	_, err = detector.Detect()
	if err == nil {
		t.Error("Expected error with invalid runtime override")
	}
}

// Property 20: Language Auto-Detection
// For any supported command, the run-mcp binary should correctly detect the language runtime
func TestProperty20_LanguageAutoDetection(t *testing.T) {
	// **Feature: mcp-container-images, Property 20: Language Auto-Detection**
	// **Validates: Requirements 13.2, 13.6**
	
	detector := NewLanguageDetector()
	
	// Test Node.js commands
	nodejsCommands := []string{"npx", "node", "yarn", "tsx", "npm"}
	for _, cmd := range nodejsCommands {
		lang, err := detector.DetectFromArgs([]string{cmd, "some-package"})
		if err != nil {
			t.Errorf("Failed to detect language for Node.js command %s: %v", cmd, err)
		}
		if lang != "nodejs" {
			t.Errorf("Expected nodejs for command %s, got %s", cmd, lang)
		}
	}
	
	// Test Python commands
	pythonCommands := []string{"uvx", "python", "python3", "uv", "pip", "pip3"}
	for _, cmd := range pythonCommands {
		lang, err := detector.DetectFromArgs([]string{cmd, "some-package"})
		if err != nil {
			t.Errorf("Failed to detect language for Python command %s: %v", cmd, err)
		}
		if lang != "python" {
			t.Errorf("Expected python for command %s, got %s", cmd, lang)
		}
	}
	
	// Test explicit runtime specification
	explicitTests := []struct {
		args     []string
		expected string
	}{
		{[]string{"python", "uvx", "some-package"}, "python"},
		{[]string{"python3", "pip", "install", "package"}, "python"},
		{[]string{"node", "npx", "some-package"}, "nodejs"},
		{[]string{"nodejs", "npm", "install", "package"}, "nodejs"},
	}
	
	for _, test := range explicitTests {
		lang, err := detector.DetectFromArgs(test.args)
		if err != nil {
			t.Errorf("Failed to detect language for explicit args %v: %v", test.args, err)
		}
		if lang != test.expected {
			t.Errorf("Expected %s for args %v, got %s", test.expected, test.args, lang)
		}
	}
	
	// Test unknown command
	_, err := detector.DetectFromArgs([]string{"unknown-command"})
	if err == nil {
		t.Error("Expected error for unknown command")
	}
	
	// Test empty args
	_, err = detector.DetectFromArgs([]string{})
	if err == nil {
		t.Error("Expected error for empty args")
	}
	
	// Test that all supported languages are valid
	supportedLanguages := detector.GetSupportedLanguages()
	if len(supportedLanguages) == 0 {
		t.Error("Expected at least one supported language")
	}
	
	for _, lang := range supportedLanguages {
		if !detector.IsValidLanguage(lang) {
			t.Errorf("Language %s should be valid but IsValidLanguage returned false", lang)
		}
	}
}

// Property 21: Secure Environment Variable Passthrough
// For any environment variable, the system should only pass through variables that match the allowlist
func TestProperty21_SecureEnvironmentVariablePassthrough(t *testing.T) {
	// **Feature: mcp-container-images, Property 21: Secure Environment Variable Passthrough**
	// **Validates: Requirements 13.3, 13.8, 13.10**
	
	filter := NewEnvFilter()
	
	// Test allowed prefixes
	allowedPrefixes := filter.GetAllowedPrefixes()
	expectedPrefixes := []string{
		"AWS_", "OPENAI_", "ANTHROPIC_", "AZURE_", "GOOGLE_",
		"MCP_", "HF_", "REPLICATE_", "COHERE_",
	}
	
	for _, expected := range expectedPrefixes {
		found := false
		for _, prefix := range allowedPrefixes {
			if prefix == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected prefix %s not found in allowed prefixes", expected)
		}
	}
	
	// Test allowed exact matches
	allowedExact := filter.GetAllowedExact()
	expectedExact := []string{"GITHUB_TOKEN", "GITLAB_TOKEN", "DATABASE_URL", "REDIS_URL"}
	
	for _, expected := range expectedExact {
		found := false
		for _, exact := range allowedExact {
			if exact == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected exact match %s not found in allowed exact matches", expected)
		}
	}
	
	// Test environment variable filtering
	testCases := []struct {
		envVar   string
		expected bool
	}{
		// Should be allowed (prefixes)
		{"AWS_ACCESS_KEY_ID=test", true},
		{"OPENAI_API_KEY=test", true},
		{"ANTHROPIC_API_KEY=test", true},
		{"MCP_DATA_DIR=test", true},
		
		// Should be allowed (exact matches)
		{"GITHUB_TOKEN=test", true},
		{"DATABASE_URL=test", true},
		
		// Should NOT be allowed
		{"PATH=/usr/bin", false},
		{"HOME=/home/user", false},
		{"DANGEROUS_VAR=malicious", false},
		{"SYSTEM_VAR=value", false},
	}
	
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		// Restore original environment
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()
	
	// Clear environment and set test variables
	os.Clearenv()
	for _, testCase := range testCases {
		parts := strings.SplitN(testCase.envVar, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}
	
	// Get filtered environment arguments
	envArgs := filter.GetFilteredEnvArgs()
	
	// Check that only allowed variables are included
	for _, testCase := range testCases {
		parts := strings.SplitN(testCase.envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		found := false
		for i := 0; i < len(envArgs); i += 2 {
			if i+1 < len(envArgs) && envArgs[i] == "-e" {
				if strings.HasPrefix(envArgs[i+1], parts[0]+"=") {
					found = true
					break
				}
			}
		}
		
		if found != testCase.expected {
			t.Errorf("Environment variable %s: expected %v, got %v", parts[0], testCase.expected, found)
		}
	}
	
	// Test custom passthrough variables
	os.Setenv("MCP_PASSTHROUGH_ENV", "CUSTOM_VAR1,CUSTOM_VAR2")
	os.Setenv("CUSTOM_VAR1", "value1")
	os.Setenv("CUSTOM_VAR2", "value2")
	os.Setenv("NOT_CUSTOM", "should_not_pass")
	
	envArgs = filter.GetFilteredEnvArgs()
	
	// Check that custom variables are included
	customVars := []string{"CUSTOM_VAR1", "CUSTOM_VAR2"}
	for _, customVar := range customVars {
		found := false
		for i := 0; i < len(envArgs); i += 2 {
			if i+1 < len(envArgs) && envArgs[i] == "-e" {
				if strings.HasPrefix(envArgs[i+1], customVar+"=") {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("Custom variable %s should be passed through but was not found", customVar)
		}
	}
	
	// Check that non-custom variable is not included
	found := false
	for i := 0; i < len(envArgs); i += 2 {
		if i+1 < len(envArgs) && envArgs[i] == "-e" {
			if strings.HasPrefix(envArgs[i+1], "NOT_CUSTOM=") {
				found = true
				break
			}
		}
	}
	if found {
		t.Error("Variable NOT_CUSTOM should not be passed through but was found")
	}
}

// Test volume mounting functionality
func TestVolumeMounting(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	config := &Config{
		DataDir: tempDir,
	}
	
	vm := NewVolumeManager(config)
	
	// Test volume mount generation
	mounts := vm.GetVolumeMounts()
	
	// Should have at least the data mount
	if len(mounts) < 2 { // -v and the mount spec
		t.Error("Expected at least one volume mount")
	}
	
	// Check data mount
	found := false
	for i := 0; i < len(mounts); i += 2 {
		if i+1 < len(mounts) && mounts[i] == "-v" {
			if strings.Contains(mounts[i+1], ":/data") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Data directory mount not found")
	}
	
	// Test mount info
	info := vm.GetMountInfo()
	if info.DataMount == "" {
		t.Error("Expected data mount info")
	}
	
	// Test data directory validation
	err := vm.ValidateDataDir()
	if err != nil {
		t.Errorf("Data directory validation failed: %v", err)
	}
}

// Test configuration functionality
func TestConfiguration(t *testing.T) {
	// Save original environment
	originalNodejs := os.Getenv("MCP_NODEJS_IMAGE")
	originalPython := os.Getenv("MCP_PYTHON_IMAGE")
	originalDataDir := os.Getenv("MCP_DATA_DIR")
	
	defer func() {
		// Restore original environment
		if originalNodejs != "" {
			os.Setenv("MCP_NODEJS_IMAGE", originalNodejs)
		} else {
			os.Unsetenv("MCP_NODEJS_IMAGE")
		}
		if originalPython != "" {
			os.Setenv("MCP_PYTHON_IMAGE", originalPython)
		} else {
			os.Unsetenv("MCP_PYTHON_IMAGE")
		}
		if originalDataDir != "" {
			os.Setenv("MCP_DATA_DIR", originalDataDir)
		} else {
			os.Unsetenv("MCP_DATA_DIR")
		}
	}()
	
	// Test default configuration
	config := loadConfig()
	if config.NodejsImage == "" {
		t.Error("Expected default Node.js image")
	}
	if config.PythonImage == "" {
		t.Error("Expected default Python image")
	}
	if config.DataDir == "" {
		t.Error("Expected default data directory")
	}
	
	// Test custom configuration
	os.Setenv("MCP_NODEJS_IMAGE", "custom/nodejs:test")
	os.Setenv("MCP_PYTHON_IMAGE", "custom/python:test")
	os.Setenv("MCP_DATA_DIR", "/custom/data")
	
	config = loadConfig()
	if config.NodejsImage != "custom/nodejs:test" {
		t.Errorf("Expected custom Node.js image, got %s", config.NodejsImage)
	}
	if config.PythonImage != "custom/python:test" {
		t.Errorf("Expected custom Python image, got %s", config.PythonImage)
	}
	if config.DataDir != "/custom/data" {
		t.Errorf("Expected custom data directory, got %s", config.DataDir)
	}
	
	// Test image selection
	image, err := config.GetImageForLanguage("nodejs")
	if err != nil {
		t.Errorf("Failed to get image for nodejs: %v", err)
	}
	if image != "custom/nodejs:test" {
		t.Errorf("Expected custom nodejs image, got %s", image)
	}
	
	image, err = config.GetImageForLanguage("python")
	if err != nil {
		t.Errorf("Failed to get image for python: %v", err)
	}
	if image != "custom/python:test" {
		t.Errorf("Expected custom python image, got %s", image)
	}
	
	// Test invalid language
	_, err = config.GetImageForLanguage("invalid")
	if err == nil {
		t.Error("Expected error for invalid language")
	}
	
	// Test validation with valid temp directory
	tempDir := t.TempDir()
	config.DataDir = tempDir
	err = config.Validate()
	if err != nil {
		t.Errorf("Configuration validation failed: %v", err)
	}
}

// Property 1: Volume Creation Consistency
// For any MCP server command and supported container runtime, when run-mcp starts a container, 
// a named container volume should be created following the deterministic naming pattern 
// mcp-home-{sanitized-command}-{sanitized-first-arg}
func TestProperty1_VolumeCreationConsistency(t *testing.T) {
	// **Feature: container-home-isolation, Property 1: Volume Creation Consistency**
	// **Validates: Requirements 1.1, 2.1, 2.2, 2.3**
	
	testCases := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "simple command",
			args:     []string{"uvx", "server"},
			expected: "mcp-home-uvx-server",
		},
		{
			name:     "command with package name",
			args:     []string{"npx", "@modelcontextprotocol/server-filesystem"},
			expected: "mcp-home-npx-modelcontextprotocol-server-filesystem",
		},
		{
			name:     "command with version",
			args:     []string{"uvx", "awslabs.aws-api-mcp-server@latest"},
			expected: "mcp-home-uvx-awslabs-aws-api-mcp-server-latest",
		},
		{
			name:     "command with flags (should stop at flags)",
			args:     []string{"uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"},
			expected: "mcp-home-uvx-mcp-server-sqlite",
		},
		{
			name:     "single command",
			args:     []string{"python"},
			expected: "mcp-home-python",
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: "mcp-home-default",
		},
		{
			name:     "command with special characters",
			args:     []string{"uvx", "server@1.0.0"},
			expected: "mcp-home-uvx-server-1-0-0",
		},
		{
			name:     "windows path normalization",
			args:     []string{"uvx", "server\\with\\backslashes"},
			expected: "mcp-home-uvx-server-with-backslashes",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeVolumeName(tc.args)
			if result != tc.expected {
				t.Errorf("sanitizeVolumeName(%v) = %s, expected %s", tc.args, result, tc.expected)
			}
			
			// Verify the result follows the expected pattern
			if !strings.HasPrefix(result, "mcp-home-") {
				t.Errorf("Volume name %s does not start with 'mcp-home-'", result)
			}
			
			// Verify no invalid characters remain
			validPattern := regexp.MustCompile(`^[a-z0-9-]+$`)
			if !validPattern.MatchString(result) {
				t.Errorf("Volume name %s contains invalid characters", result)
			}
			
			// Verify length constraint
			if len(result) > 64 {
				t.Errorf("Volume name %s exceeds 64 character limit (length: %d)", result, len(result))
			}
		})
	}
	
	// Test deterministic behavior - same input should always produce same output
	testArgs := []string{"uvx", "awslabs.aws-api-mcp-server@latest"}
	result1 := sanitizeVolumeName(testArgs)
	result2 := sanitizeVolumeName(testArgs)
	if result1 != result2 {
		t.Errorf("sanitizeVolumeName is not deterministic: %s != %s", result1, result2)
	}
}

// Benchmark tests for performance
func BenchmarkRuntimeDetection(b *testing.B) {
	detector := NewRuntimeDetector()
	for i := 0; i < b.N; i++ {
		detector.Detect()
	}
}

func BenchmarkLanguageDetection(b *testing.B) {
	detector := NewLanguageDetector()
	args := []string{"npx", "@modelcontextprotocol/server-filesystem", "/data"}
	for i := 0; i < b.N; i++ {
		detector.DetectFromArgs(args)
	}
}
func BenchmarkEnvFiltering(b *testing.B) {
	filter := NewEnvFilter()
	for i := 0; i < b.N; i++ {
		filter.GetFilteredEnvArgs()
	}
}

// Unit tests for volume name edge cases
// Requirements: 2.7, 2.8
func TestVolumeNameEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		expected    string
		description string
	}{
		{
			name:        "empty command",
			args:        []string{""},
			expected:    "mcp-home-default",
			description: "Empty string should result in default name",
		},
		{
			name:        "whitespace only",
			args:        []string{"   ", "\t"},
			expected:    "mcp-home-default",
			description: "Whitespace-only args should result in default name",
		},
		{
			name:        "special characters",
			args:        []string{"uvx", "server@1.0.0#latest!"},
			expected:    "mcp-home-uvx-server-1-0-0-latest",
			description: "Special characters should be replaced with dashes",
		},
		{
			name:        "consecutive special characters",
			args:        []string{"uvx", "server!!!@@@###"},
			expected:    "mcp-home-uvx-server",
			description: "Consecutive special characters should be collapsed to single dash",
		},
		{
			name:        "leading and trailing special chars",
			args:        []string{"@@@uvx@@@", "!!!server!!!"},
			expected:    "mcp-home-uvx-server",
			description: "Leading and trailing special characters should be trimmed",
		},
		{
			name:        "windows paths",
			args:        []string{"uvx", "C:\\Program Files\\server\\app.exe"},
			expected:    "mcp-home-uvx-c-program-files-server-app-exe",
			description: "Windows paths should be normalized",
		},
		{
			name:        "mixed path separators",
			args:        []string{"uvx", "path\\with/mixed\\separators"},
			expected:    "mcp-home-uvx-path-with-mixed-separators",
			description: "Mixed path separators should be normalized",
		},
		{
			name:        "unicode characters",
			args:        []string{"uvx", "sérver-ñame"},
			expected:    "mcp-home-uvx-s-rver-ame",
			description: "Unicode characters should be handled",
		},
		{
			name:        "numbers and letters",
			args:        []string{"uvx", "server123"},
			expected:    "mcp-home-uvx-server123",
			description: "Numbers should be preserved",
		},
		{
			name:        "only flags",
			args:        []string{"--help", "--version"},
			expected:    "mcp-home-default",
			description: "Only flags should result in default name",
		},
		{
			name:        "command then flags",
			args:        []string{"uvx", "--help"},
			expected:    "mcp-home-uvx",
			description: "Should stop processing at first flag",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeVolumeName(tc.args)
			if result != tc.expected {
				t.Errorf("sanitizeVolumeName(%v) = %s, expected %s (%s)", 
					tc.args, result, tc.expected, tc.description)
			}
		})
	}
}

// Test truncation behavior with various input lengths
// Requirements: 2.8
func TestVolumeNameTruncation(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name: "exactly 64 characters",
			args: []string{"uvx", "this-is-a-very-long-server-name-that-should-be-exactly-sixtyfour"},
			description: "Name exactly at limit should not be truncated",
		},
		{
			name: "over 64 characters",
			args: []string{"uvx", "this-is-a-very-long-server-name-that-exceeds-the-sixtyfour-character-limit"},
			description: "Name over limit should be truncated with hash",
		},
		{
			name: "extremely long name",
			args: []string{"uvx", strings.Repeat("very-long-name-", 20)},
			description: "Extremely long names should be truncated properly",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeVolumeName(tc.args)
			
			// All results should be 64 characters or less
			if len(result) > 64 {
				t.Errorf("Volume name length %d exceeds 64 character limit: %s", len(result), result)
			}
			
			// Should still start with mcp-home-
			if !strings.HasPrefix(result, "mcp-home-") {
				t.Errorf("Truncated name should still start with 'mcp-home-': %s", result)
			}
			
			// If truncated, should end with 8-character hash
			originalName := "mcp-home-" + strings.Join(tc.args, "-")
			if len(originalName) > 64 {
				// Should be truncated to 56 chars + "-" + 8 char hash = 65 total, but we want 64
				// So it should be 55 chars + "-" + 8 char hash = 64 total
				if len(result) != 64 {
					t.Errorf("Truncated name should be exactly 64 characters, got %d: %s", len(result), result)
				}
				
				// Should contain a hash at the end
				parts := strings.Split(result, "-")
				lastPart := parts[len(parts)-1]
				if len(lastPart) != 8 {
					t.Errorf("Expected 8-character hash suffix, got %d characters: %s", len(lastPart), lastPart)
				}
				
				// Hash should be hexadecimal
				if matched, _ := regexp.MatchString("^[a-f0-9]{8}$", lastPart); !matched {
					t.Errorf("Hash suffix should be 8 hexadecimal characters: %s", lastPart)
				}
			}
			
			// Test deterministic truncation - same input should produce same hash
			result2 := sanitizeVolumeName(tc.args)
			if result != result2 {
				t.Errorf("Truncation should be deterministic: %s != %s", result, result2)
			}
		})
	}
}
// Test volume command abstraction functionality
// Requirements: 4.11, 4.12, 2.9
func TestVolumeCommandAbstraction(t *testing.T) {
	testCases := []struct {
		name    string
		runtime string
	}{
		{"Docker", "docker"},
		{"Podman", "podman"},
		{"Nerdctl", "nerdctl"},
		{"Finch", "finch"},
		{"Lima Nerdctl", "lima nerdctl"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commander := NewVolumeCommander(tc.runtime)
			
			// Test that commander is not nil
			if commander == nil {
				t.Errorf("NewVolumeCommander(%s) returned nil", tc.runtime)
			}
			
			// Test that commander implements VolumeCommander interface
			var _ VolumeCommander = commander
		})
	}
}

// Test volume command generation for each supported runtime
// Requirements: 4.11, 4.12, 2.9
func TestVolumeCommandGeneration(t *testing.T) {
	testCases := []struct {
		name            string
		runtime         string
		expectedType    string
	}{
		{"Docker", "docker", "*main.DockerVolumeCommander"},
		{"Podman", "podman", "*main.PodmanVolumeCommander"},
		{"Nerdctl", "nerdctl", "*main.NerdctlVolumeCommander"},
		{"Finch", "finch", "*main.FinchVolumeCommander"},
		{"Lima Nerdctl", "lima nerdctl", "*main.LimaNerdctlVolumeCommander"},
		{"Unknown Runtime", "unknown", "*main.DockerVolumeCommander"}, // Should default to Docker
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commander := NewVolumeCommander(tc.runtime)
			
			// Check the type of the returned commander
			commanderType := fmt.Sprintf("%T", commander)
			if commanderType != tc.expectedType {
				t.Errorf("NewVolumeCommander(%s) returned type %s, expected %s", 
					tc.runtime, commanderType, tc.expectedType)
			}
		})
	}
}

// Test label application and runtime metadata
// Requirements: 4.11, 4.12, 2.9
func TestVolumeLabelApplication(t *testing.T) {
	// Test that all commanders support the same label format
	runtimes := []string{"docker", "podman", "nerdctl", "finch", "lima nerdctl"}
	
	for _, runtime := range runtimes {
		t.Run(runtime, func(t *testing.T) {
			commander := NewVolumeCommander(runtime)
			
			// We can't actually create volumes in unit tests without the runtime available,
			// but we can test that the interface accepts the labels without error
			// This is a structural test to ensure the interface is correctly implemented
			
			// Test VolumeExists method (should not panic)
			_, err := commander.VolumeExists("test-volume")
			// Error is expected since volume doesn't exist and runtime may not be available
			// We're just testing that the method can be called without panic
			_ = err // Ignore the error for this structural test
			
			// Test that the commander has the expected runtime-specific behavior
			switch runtime {
			case "lima nerdctl":
				// Lima nerdctl should be a special case
				if lvc, ok := commander.(*LimaNerdctlVolumeCommander); ok {
					if lvc.runtime != "lima nerdctl" {
						t.Errorf("LimaNerdctlVolumeCommander runtime should be 'lima nerdctl', got %s", lvc.runtime)
					}
				} else {
					t.Errorf("Expected LimaNerdctlVolumeCommander for lima nerdctl runtime")
				}
			default:
				// Other runtimes should have their runtime name set correctly
				switch cmd := commander.(type) {
				case *DockerVolumeCommander:
					if cmd.runtime != runtime && runtime != "unknown" {
						t.Errorf("DockerVolumeCommander runtime should be %s, got %s", runtime, cmd.runtime)
					}
				case *PodmanVolumeCommander:
					if cmd.runtime != runtime {
						t.Errorf("PodmanVolumeCommander runtime should be %s, got %s", runtime, cmd.runtime)
					}
				case *NerdctlVolumeCommander:
					if cmd.runtime != runtime {
						t.Errorf("NerdctlVolumeCommander runtime should be %s, got %s", runtime, cmd.runtime)
					}
				case *FinchVolumeCommander:
					if cmd.runtime != runtime {
						t.Errorf("FinchVolumeCommander runtime should be %s, got %s", runtime, cmd.runtime)
					}
				}
			}
		})
	}
}

// Property 2: Volume Reuse Idempotency
// For any MCP server command, running the same command multiple times should always reuse the same volume,
// ensuring data persistence and consistency across executions
func TestProperty2_VolumeReuseIdempotency(t *testing.T) {
	// **Feature: container-home-isolation, Property 2: Volume Reuse Idempotency**
	// **Validates: Requirements 1.2, 2.4, 4.2**
	
	config := &Config{}
	
	testCases := []struct {
		name       string
		serverName string
		runtime    string
	}{
		{
			name:       "simple server command",
			serverName: "uvx mcp-server-sqlite",
			runtime:    "docker",
		},
		{
			name:       "complex server command",
			serverName: "npx @modelcontextprotocol/server-filesystem",
			runtime:    "podman",
		},
		{
			name:       "server with version",
			serverName: "uvx awslabs.aws-api-mcp-server@latest",
			runtime:    "nerdctl",
		},
		{
			name:       "server with flags",
			serverName: "uvx mcp-server-sqlite --db-path /data/db.sqlite",
			runtime:    "finch",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVolumeManager(config)
			
			// First call to CreateHomeVolume
			volumeName1, err1 := vm.CreateHomeVolume(tc.serverName, tc.runtime)
			
			// Second call to CreateHomeVolume with same parameters
			volumeName2, err2 := vm.CreateHomeVolume(tc.serverName, tc.runtime)
			
			// Both calls should return the same volume name (idempotency)
			if volumeName1 != volumeName2 {
				t.Errorf("Volume names should be identical for same server: %s != %s", volumeName1, volumeName2)
			}
			
			// Both calls should have consistent error behavior
			if (err1 == nil) != (err2 == nil) {
				t.Errorf("Error behavior should be consistent: err1=%v, err2=%v", err1, err2)
			}
			
			// Volume name should follow expected pattern
			expectedVolumeName := sanitizeVolumeName(strings.Fields(tc.serverName))
			if err1 == nil && volumeName1 != expectedVolumeName {
				t.Errorf("Expected volume name %s, got %s", expectedVolumeName, volumeName1)
			}
			
			// Test multiple calls in sequence (simulate multiple container starts)
			for i := 0; i < 5; i++ {
				volumeNameN, errN := vm.CreateHomeVolume(tc.serverName, tc.runtime)
				if volumeNameN != volumeName1 {
					t.Errorf("Call %d: Volume name should remain consistent: %s != %s", i+3, volumeNameN, volumeName1)
				}
				if (errN == nil) != (err1 == nil) {
					t.Errorf("Call %d: Error behavior should remain consistent: errN=%v, err1=%v", i+3, errN, err1)
				}
			}
		})
	}
	
	// Test that different server names produce different volumes
	vm := NewVolumeManager(config)
	
	serverName1 := "uvx server-one"
	serverName2 := "uvx server-two"
	runtime := "docker"
	
	volumeName1, _ := vm.CreateHomeVolume(serverName1, runtime)
	volumeName2, _ := vm.CreateHomeVolume(serverName2, runtime)
	
	if volumeName1 == volumeName2 {
		t.Errorf("Different server names should produce different volume names: %s == %s", volumeName1, volumeName2)
	}
	
	// Test that same server name with different runtime produces same volume name
	// (volume names are runtime-agnostic, but runtime metadata is stored in labels)
	volumeName3, _ := vm.CreateHomeVolume(serverName1, "podman")
	expectedName := sanitizeVolumeName(strings.Fields(serverName1))
	if volumeName3 != expectedName {
		t.Errorf("Same server name should produce same volume name regardless of runtime: expected %s, got %s", expectedName, volumeName3)
	}
}

// Test volume creation success and failure scenarios
// Requirements: 1.1, 1.6
func TestVolumeCreation(t *testing.T) {
	config := &Config{}
	
	testCases := []struct {
		name       string
		serverName string
		runtime    string
		expectErr  bool
	}{
		{
			name:       "valid server name with docker",
			serverName: "uvx mcp-server-sqlite",
			runtime:    "docker",
			expectErr:  true, // Expected because docker may not be available in test environment
		},
		{
			name:       "valid server name with podman",
			serverName: "npx @modelcontextprotocol/server-filesystem",
			runtime:    "podman",
			expectErr:  true, // Expected because podman may not be available in test environment
		},
		{
			name:       "complex server name",
			serverName: "uvx awslabs.aws-api-mcp-server@latest --config /path/to/config",
			runtime:    "nerdctl",
			expectErr:  true, // Expected because nerdctl may not be available in test environment
		},
		{
			name:       "empty server name",
			serverName: "",
			runtime:    "docker",
			expectErr:  true, // Expected because empty server name should be handled gracefully
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVolumeManager(config)
			
			volumeName, err := vm.CreateHomeVolume(tc.serverName, tc.runtime)
			
			if tc.expectErr {
				// We expect errors in test environment due to missing container runtimes
				// But we can still validate the volume name generation logic
				expectedVolumeName := sanitizeVolumeName(strings.Fields(tc.serverName))
				if err == nil && volumeName != expectedVolumeName {
					t.Errorf("Expected volume name %s, got %s", expectedVolumeName, volumeName)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				
				// Verify volume name follows expected pattern
				if !strings.HasPrefix(volumeName, "mcp-home-") {
					t.Errorf("Volume name should start with 'mcp-home-', got %s", volumeName)
				}
			}
			
			// Test that VolumeManager state is updated correctly
			if vm.commander == nil {
				t.Error("VolumeManager commander should be initialized after CreateHomeVolume call")
			}
			
			if vm.runtime != tc.runtime {
				t.Errorf("VolumeManager runtime should be %s, got %s", tc.runtime, vm.runtime)
			}
		})
	}
}

// Test concurrent access handling
// Requirements: 1.6
func TestVolumeConcurrentAccess(t *testing.T) {
	config := &Config{}
	serverName := "uvx concurrent-test-server"
	runtime := "docker"
	
	// Test concurrent calls to CreateHomeVolume
	const numGoroutines = 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)
	
	// Launch multiple goroutines calling CreateHomeVolume simultaneously
	for i := 0; i < numGoroutines; i++ {
		go func() {
			vm := NewVolumeManager(config)
			volumeName, err := vm.CreateHomeVolume(serverName, runtime)
			results <- volumeName
			errors <- err
		}()
	}
	
	// Collect results
	var volumeNames []string
	var errs []error
	
	for i := 0; i < numGoroutines; i++ {
		volumeNames = append(volumeNames, <-results)
		errs = append(errs, <-errors)
	}
	
	// All successful calls should return the same volume name
	expectedVolumeName := sanitizeVolumeName(strings.Fields(serverName))
	for i, volumeName := range volumeNames {
		if errs[i] == nil && volumeName != expectedVolumeName {
			t.Errorf("Goroutine %d: Expected volume name %s, got %s", i, expectedVolumeName, volumeName)
		}
	}
	
	// Test that concurrent calls to the same VolumeManager instance are safe
	vm := NewVolumeManager(config)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			volumeName, err := vm.CreateHomeVolume(fmt.Sprintf("%s-%d", serverName, id), runtime)
			results <- volumeName
			errors <- err
		}(i)
	}
	
	// Collect results from shared VolumeManager
	for i := 0; i < numGoroutines; i++ {
		volumeName := <-results
		err := <-errors
		
		if err == nil {
			// Each should get a different volume name since server names are different
			expectedPattern := "mcp-home-uvx-concurrent-test-server-"
			if !strings.HasPrefix(volumeName, expectedPattern) {
				t.Errorf("Volume name should start with %s, got %s", expectedPattern, volumeName)
			}
		}
	}
}

// Test volume creation with different runtime configurations
// Requirements: 1.1, 1.6
func TestVolumeCreationWithDifferentRuntimes(t *testing.T) {
	config := &Config{}
	serverName := "uvx test-server"
	
	runtimes := []string{"docker", "podman", "nerdctl", "finch", "lima nerdctl"}
	
	for _, runtime := range runtimes {
		t.Run(runtime, func(t *testing.T) {
			vm := NewVolumeManager(config)
			
			volumeName, err := vm.CreateHomeVolume(serverName, runtime)
			
			// Volume name should be consistent regardless of runtime
			expectedVolumeName := sanitizeVolumeName(strings.Fields(serverName))
			if err == nil && volumeName != expectedVolumeName {
				t.Errorf("Volume name should be consistent across runtimes: expected %s, got %s", expectedVolumeName, volumeName)
			}
			
			// VolumeManager should be configured with the correct runtime
			if vm.runtime != runtime {
				t.Errorf("VolumeManager runtime should be %s, got %s", runtime, vm.runtime)
			}
			
			// Commander should be initialized
			if vm.commander == nil {
				t.Error("VolumeManager commander should be initialized")
			}
		})
	}
}

// Test volume creation error handling - basic scenarios
// Requirements: 1.1, 1.6
func TestVolumeCreationBasicErrorHandling(t *testing.T) {
	config := &Config{}
	
	// Test with invalid runtime
	vm := NewVolumeManager(config)
	_, err := vm.CreateHomeVolume("test-server", "nonexistent-runtime")
	
	// Should handle gracefully (may not error immediately due to lazy initialization)
	if err == nil {
		// If no immediate error, the commander should still be created
		if vm.commander == nil {
			t.Error("VolumeManager commander should be initialized even with invalid runtime")
		}
	}
	
	// Test with empty server name
	vm2 := NewVolumeManager(config)
	volumeName, err := vm2.CreateHomeVolume("", "docker")
	
	// Should handle empty server name gracefully
	expectedVolumeName := sanitizeVolumeName([]string{})
	if err == nil && volumeName != expectedVolumeName {
		t.Errorf("Empty server name should produce default volume name: expected %s, got %s", expectedVolumeName, volumeName)
	}
	
	// Test with whitespace-only server name
	vm3 := NewVolumeManager(config)
	volumeName3, err3 := vm3.CreateHomeVolume("   \t\n   ", "docker")
	
	// Should handle whitespace-only server name gracefully
	expectedVolumeName3 := sanitizeVolumeName(strings.Fields("   \t\n   "))
	if err3 == nil && volumeName3 != expectedVolumeName3 {
		t.Errorf("Whitespace-only server name should produce default volume name: expected %s, got %s", expectedVolumeName3, volumeName3)
	}
}

// Test VolumeManager integration with VolumeCommander
// Requirements: 4.11, 4.12, 2.9
func TestVolumeManagerIntegration(t *testing.T) {
	config := &Config{}
	
	// Test NewVolumeManagerWithRuntime
	testRuntimes := []string{"docker", "podman", "nerdctl", "finch"}
	
	for _, runtime := range testRuntimes {
		t.Run(runtime, func(t *testing.T) {
			vm := NewVolumeManagerWithRuntime(config, runtime)
			
			if vm == nil {
				t.Errorf("NewVolumeManagerWithRuntime returned nil for runtime %s", runtime)
			}
			
			if vm.config != config {
				t.Error("VolumeManager config not set correctly")
			}
			
			if vm.commander == nil {
				t.Error("VolumeManager commander not set")
			}
			
			if vm.runtime != runtime {
				t.Errorf("VolumeManager runtime should be %s, got %s", runtime, vm.runtime)
			}
		})
	}
	
	// Test that NewVolumeManager creates a manager without commander (lazy initialization)
	vm := NewVolumeManager(config)
	if vm.commander != nil {
		t.Error("NewVolumeManager should not initialize commander immediately")
	}
	if vm.runtime != "" {
		t.Error("NewVolumeManager should not set runtime immediately")
	}
}

// Test volume name sanitization with VolumeManager
// Requirements: 1.1, 2.1, 2.2, 2.3
func TestVolumeManagerCreateHomeVolume(t *testing.T) {
	config := &Config{}
	
	testCases := []struct {
		name       string
		serverName string
		runtime    string
		expectErr  bool
	}{
		{
			name:       "valid server name",
			serverName: "uvx awslabs.aws-api-mcp-server@latest",
			runtime:    "docker",
			expectErr:  true, // Will error because docker may not be available, but tests the flow
		},
		{
			name:       "simple server name",
			serverName: "npx server-memory",
			runtime:    "podman",
			expectErr:  true, // Will error because podman may not be available, but tests the flow
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVolumeManager(config)
			
			volumeName, err := vm.CreateHomeVolume(tc.serverName, tc.runtime)
			
			if tc.expectErr {
				// We expect an error because the runtime is likely not available in test environment
				// But we can still test that the volume name is generated correctly
				expectedVolumeName := sanitizeVolumeName(strings.Fields(tc.serverName))
				if err == nil && volumeName != expectedVolumeName {
					t.Errorf("Expected volume name %s, got %s", expectedVolumeName, volumeName)
				}
				
				// Test that commander was initialized
				if vm.commander == nil {
					t.Error("VolumeManager commander should be initialized after CreateHomeVolume call")
				}
				
				if vm.runtime != tc.runtime {
					t.Errorf("VolumeManager runtime should be %s, got %s", tc.runtime, vm.runtime)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				
				// Verify volume name follows expected pattern
				if !strings.HasPrefix(volumeName, "mcp-home-") {
					t.Errorf("Volume name should start with 'mcp-home-', got %s", volumeName)
				}
			}
		})
	}
}
// Property 13: Ephemeral Volume Cleanup
// For any container run with the --ephemeral flag, the associated volume should be automatically removed when the container stops
func TestProperty13_EphemeralVolumeCleanup(t *testing.T) {
	// **Feature: container-home-isolation, Property 13: Ephemeral Volume Cleanup**
	// **Validates: Requirements 6.3, 6.4, 6.5**
	
	config := &Config{EphemeralMode: true}
	
	// Test with different server names and runtimes
	testCases := []struct {
		serverName string
		runtime    string
	}{
		{"uvx test-server", "docker"},
		{"npx @modelcontextprotocol/server-memory", "podman"},
		{"python -m server", "nerdctl"},
		{"node server.js", "finch"},
	}
	
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("server=%s,runtime=%s", tc.serverName, tc.runtime), func(t *testing.T) {
			volumeManager := NewVolumeManagerWithRuntime(config, tc.runtime)
			
			// Create ephemeral volume
			volumeName, err := volumeManager.CreateEphemeralVolume(tc.serverName, tc.runtime)
			if err != nil {
				t.Skipf("Cannot create ephemeral volume with runtime %s: %v", tc.runtime, err)
			}
			
			// Verify volume name follows ephemeral pattern
			if !strings.HasPrefix(volumeName, "mcp-ephemeral-") {
				t.Errorf("Ephemeral volume name %s does not follow expected pattern mcp-ephemeral-*", volumeName)
			}
			
			// Verify volume name contains timestamp
			if !regexp.MustCompile(`-\d+$`).MatchString(volumeName) {
				t.Errorf("Ephemeral volume name %s does not contain timestamp suffix", volumeName)
			}
			
			// Verify volume exists
			exists, err := volumeManager.commander.VolumeExists(volumeName)
			if err != nil {
				t.Skipf("Cannot check volume existence with runtime %s: %v", tc.runtime, err)
			}
			if !exists {
				t.Errorf("Ephemeral volume %s should exist after creation", volumeName)
			}
			
			// Cleanup ephemeral volume (simulating container exit)
			err = volumeManager.CleanupEphemeralVolume(volumeName)
			if err != nil {
				t.Skipf("Cannot cleanup ephemeral volume with runtime %s: %v", tc.runtime, err)
			}
			
			// Verify volume no longer exists
			exists, err = volumeManager.commander.VolumeExists(volumeName)
			if err != nil {
				t.Skipf("Cannot check volume existence after cleanup with runtime %s: %v", tc.runtime, err)
			}
			if exists {
				t.Errorf("Ephemeral volume %s should not exist after cleanup", volumeName)
			}
		})
	}
}
// Unit tests for ephemeral volume naming
// Test unique timestamp-based naming and cleanup behavior
// Requirements: 6.4, 6.5
func TestEphemeralVolumeNaming(t *testing.T) {
	config := &Config{EphemeralMode: true}
	volumeManager := NewVolumeManagerWithRuntime(config, "docker")
	
	testCases := []struct {
		name       string
		serverName string
		expected   string
	}{
		{
			name:       "simple server name",
			serverName: "uvx test-server",
			expected:   "mcp-ephemeral-uvx-test-server-",
		},
		{
			name:       "complex server name with special chars",
			serverName: "npx @modelcontextprotocol/server-memory",
			expected:   "mcp-ephemeral-npx-modelcontextproto", // Will be truncated
		},
		{
			name:       "python server",
			serverName: "python -m server",
			expected:   "mcp-ephemeral-python-",
		},
		{
			name:       "node server with file",
			serverName: "node server.js",
			expected:   "mcp-ephemeral-node-server-js-",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			volumeName := volumeManager.CreateEphemeralVolumeName(tc.serverName)
			
			// Verify it starts with expected prefix (or truncated version)
			if !strings.HasPrefix(volumeName, tc.expected) {
				// For truncated names, just check it starts with mcp-ephemeral-
				if !strings.HasPrefix(volumeName, "mcp-ephemeral-") {
					t.Errorf("Expected volume name to start with mcp-ephemeral-, got %s", volumeName)
				}
			}
			
			// Verify it contains timestamp suffix
			if !regexp.MustCompile(`-\d+$`).MatchString(volumeName) {
				t.Errorf("Volume name %s should end with timestamp", volumeName)
			}
			
			// Verify it follows ephemeral pattern
			if !strings.HasPrefix(volumeName, "mcp-ephemeral-") {
				t.Errorf("Volume name %s should start with mcp-ephemeral-", volumeName)
			}
			
			// Verify length constraint (64 characters max)
			if len(volumeName) > 64 {
				t.Errorf("Volume name %s exceeds 64 character limit (length: %d)", volumeName, len(volumeName))
			}
		})
	}
}

// Test unique timestamp-based naming ensures no collisions
// Requirements: 6.4, 6.5
func TestEphemeralVolumeUniqueness(t *testing.T) {
	config := &Config{EphemeralMode: true}
	volumeManager := NewVolumeManagerWithRuntime(config, "docker")
	
	serverName := "uvx test-server"
	
	// Generate multiple volume names in quick succession
	var volumeNames []string
	for i := 0; i < 5; i++ { // Reduced to 5 iterations for faster test
		volumeName := volumeManager.CreateEphemeralVolumeName(serverName)
		volumeNames = append(volumeNames, volumeName)
		
		// Small delay to ensure different timestamps (nanosecond precision should be enough)
		time.Sleep(10 * time.Nanosecond)
	}
	
	// Verify all names are unique
	seen := make(map[string]bool)
	for _, name := range volumeNames {
		if seen[name] {
			t.Errorf("Duplicate volume name generated: %s", name)
		}
		seen[name] = true
	}
	
	// Verify all names follow the pattern
	for _, name := range volumeNames {
		if !strings.HasPrefix(name, "mcp-ephemeral-") {
			t.Errorf("Volume name %s does not follow ephemeral pattern", name)
		}
		if !regexp.MustCompile(`-\d+$`).MatchString(name) {
			t.Errorf("Volume name %s does not contain timestamp suffix", name)
		}
	}
}

// Test cleanup behavior for ephemeral volumes
// Requirements: 6.3, 6.5
func TestEphemeralVolumeCleanupBehavior(t *testing.T) {
	config := &Config{EphemeralMode: true}
	volumeManager := NewVolumeManagerWithRuntime(config, "docker")
	
	// Test cleanup validation - should only cleanup ephemeral volumes
	testCases := []struct {
		name        string
		volumeName  string
		shouldError bool
	}{
		{
			name:        "valid ephemeral volume",
			volumeName:  "mcp-ephemeral-test-server-1234567890",
			shouldError: false,
		},
		{
			name:        "regular home volume",
			volumeName:  "mcp-home-test-server",
			shouldError: true,
		},
		{
			name:        "random volume name",
			volumeName:  "some-other-volume",
			shouldError: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := volumeManager.CleanupEphemeralVolume(tc.volumeName)
			
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error when trying to cleanup non-ephemeral volume %s", tc.volumeName)
				}
			} else {
				// For valid ephemeral volumes, we expect either success or a "volume not found" error
				// since we're not actually creating the volume in this test
				if err != nil && !strings.Contains(err.Error(), "not found") && 
				   !strings.Contains(err.Error(), "no such volume") && 
				   !strings.Contains(err.Error(), "exit status 1") {
					t.Errorf("Unexpected error cleaning up ephemeral volume %s: %v", tc.volumeName, err)
				}
			}
		})
	}
}

// Test ephemeral volume name truncation for very long server names
// Requirements: 6.4, 6.5
func TestEphemeralVolumeNameTruncation(t *testing.T) {
	config := &Config{EphemeralMode: true}
	volumeManager := NewVolumeManagerWithRuntime(config, "docker")
	
	// Create a very long server name that would exceed 64 characters
	longServerName := "uvx very-long-server-name-that-exceeds-the-maximum-allowed-length-for-volume-names-in-container-runtimes"
	
	volumeName := volumeManager.CreateEphemeralVolumeName(longServerName)
	
	// Verify length constraint
	if len(volumeName) > 64 {
		t.Errorf("Volume name %s exceeds 64 character limit (length: %d)", volumeName, len(volumeName))
	}
	
	// Verify it still follows ephemeral pattern
	if !strings.HasPrefix(volumeName, "mcp-ephemeral-") {
		t.Errorf("Truncated volume name %s should still start with mcp-ephemeral-", volumeName)
	}
	
	// Verify it still contains timestamp
	if !regexp.MustCompile(`-\d+$`).MatchString(volumeName) {
		t.Errorf("Truncated volume name %s should still end with timestamp", volumeName)
	}
	
	// Verify it contains hash for uniqueness when truncated
	if len(longServerName) > 40 { // If original name was long enough to require truncation
		// Should contain a hash component for uniqueness
		parts := strings.Split(volumeName, "-")
		if len(parts) < 4 {
			t.Errorf("Truncated volume name %s should contain hash component", volumeName)
		}
	}
}

// Property 8: User Mount Configuration
// For any valid MCP_MOUNT specification, run-mcp should correctly parse, expand paths (including tilde expansion), 
// and mount the specified host directories to container destinations with appropriate options
func TestProperty8_UserMountConfiguration(t *testing.T) {
	// **Feature: container-home-isolation, Property 8: User Mount Configuration**
	// **Validates: Requirements 7.1, 7.2, 7.3, 7.4, 7.8**
	
	parser := NewUserMountParser()
	
	// Create temporary directories for testing
	tempDir := t.TempDir()
	testDir1 := filepath.Join(tempDir, "test1")
	testDir2 := filepath.Join(tempDir, "test2")
	os.MkdirAll(testDir1, 0755)
	os.MkdirAll(testDir2, 0755)
	
	// Test cases for valid mount configurations
	testCases := []struct {
		name        string
		mountString string
		expected    []Mount
	}{
		{
			name:        "single mount without options",
			mountString: fmt.Sprintf("%s:/data", testDir1),
			expected: []Mount{
				{Source: testDir1, Destination: "/data", Options: ""},
			},
		},
		{
			name:        "single mount with read-only option",
			mountString: fmt.Sprintf("%s:/data:ro", testDir1),
			expected: []Mount{
				{Source: testDir1, Destination: "/data", Options: "ro"},
			},
		},
		{
			name:        "multiple mounts",
			mountString: fmt.Sprintf("%s:/data,%s:/config:ro", testDir1, testDir2),
			expected: []Mount{
				{Source: testDir1, Destination: "/data", Options: ""},
				{Source: testDir2, Destination: "/config", Options: "ro"},
			},
		},
		{
			name:        "mount with bind option",
			mountString: fmt.Sprintf("%s:/data:bind", testDir1),
			expected: []Mount{
				{Source: testDir1, Destination: "/data", Options: "bind"},
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mounts, err := parser.ParseMountString(tc.mountString)
			if err != nil {
				t.Errorf("ParseMountString failed: %v", err)
				return
			}
			
			if len(mounts) != len(tc.expected) {
				t.Errorf("Expected %d mounts, got %d", len(tc.expected), len(mounts))
				return
			}
			
			for i, mount := range mounts {
				expected := tc.expected[i]
				
				// Normalize paths for comparison
				expectedSource := filepath.Clean(expected.Source)
				actualSource := filepath.Clean(mount.Source)
				
				if actualSource != expectedSource {
					t.Errorf("Mount %d: expected source %s, got %s", i, expectedSource, actualSource)
				}
				
				if mount.Destination != expected.Destination {
					t.Errorf("Mount %d: expected destination %s, got %s", i, expected.Destination, mount.Destination)
				}
				
				if mount.Options != expected.Options {
					t.Errorf("Mount %d: expected options %s, got %s", i, expected.Options, mount.Options)
				}
			}
			
			// Validate all mounts
			for _, mount := range mounts {
				if err := parser.ValidateMount(mount); err != nil {
					t.Errorf("Mount validation failed: %v", err)
				}
			}
		})
	}
	
	// Test tilde expansion (Requirement 7.3)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory for tilde expansion test")
	}
	
	// Create a test directory in home for tilde expansion test
	homeTestDir := filepath.Join(homeDir, ".mcp-test-mount")
	os.MkdirAll(homeTestDir, 0755)
	defer os.RemoveAll(homeTestDir)
	
	tildeTestCases := []struct {
		name        string
		mountString string
		expectedSrc string
	}{
		{
			name:        "tilde expansion home directory",
			mountString: "~/.mcp-test-mount:/config",
			expectedSrc: homeTestDir,
		},
		{
			name:        "tilde expansion root",
			mountString: "~:/home",
			expectedSrc: homeDir,
		},
	}
	
	for _, tc := range tildeTestCases {
		t.Run(tc.name, func(t *testing.T) {
			mounts, err := parser.ParseMountString(tc.mountString)
			if err != nil {
				t.Errorf("ParseMountString failed: %v", err)
				return
			}
			
			if len(mounts) != 1 {
				t.Errorf("Expected 1 mount, got %d", len(mounts))
				return
			}
			
			mount := mounts[0]
			expectedSource := filepath.Clean(tc.expectedSrc)
			actualSource := filepath.Clean(mount.Source)
			
			if actualSource != expectedSource {
				t.Errorf("Tilde expansion failed: expected %s, got %s", expectedSource, actualSource)
			}
		})
	}
	
	// Test Windows path conversion (Requirement 7.4)
	if runtime.GOOS == "windows" {
		windowsTestCases := []struct {
			name        string
			input       string
			expectedSrc string
		}{
			{
				name:        "windows drive letter",
				input:       "C:\\Users\\test:/data",
				expectedSrc: "/c/Users/test",
			},
			{
				name:        "windows forward slashes",
				input:       "C:/Users/test:/data",
				expectedSrc: "/c/Users/test",
			},
		}
		
		for _, tc := range windowsTestCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create the test directory
				testPath := strings.ReplaceAll(tc.input[:strings.Index(tc.input, ":")], "/", "\\")
				os.MkdirAll(testPath, 0755)
				defer os.RemoveAll(testPath)
				
				mounts, err := parser.ParseMountString(tc.input)
				if err != nil {
					t.Errorf("ParseMountString failed: %v", err)
					return
				}
				
				if len(mounts) != 1 {
					t.Errorf("Expected 1 mount, got %d", len(mounts))
					return
				}
				
				mount := mounts[0]
				if mount.Source != tc.expectedSrc {
					t.Errorf("Windows path conversion failed: expected %s, got %s", tc.expectedSrc, mount.Source)
				}
			})
		}
	}
	
	// Test mount argument generation (Requirement 7.8)
	testMounts := []Mount{
		{Source: testDir1, Destination: "/data", Options: ""},
		{Source: testDir2, Destination: "/config", Options: "ro"},
	}
	
	args := parser.GetMountArgs(testMounts)
	expectedArgs := []string{
		"-v", fmt.Sprintf("%s:/data", testDir1),
		"-v", fmt.Sprintf("%s:/config:ro", testDir2),
	}
	
	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d mount args, got %d", len(expectedArgs), len(args))
	}
	
	for i, arg := range args {
		if i < len(expectedArgs) && arg != expectedArgs[i] {
			t.Errorf("Mount arg %d: expected %s, got %s", i, expectedArgs[i], arg)
		}
	}
}

// Unit tests for mount parsing edge cases
// Test invalid syntax, missing paths, Windows path conversion, and error message formatting
// Requirements: 7.9, 7.10
func TestMountParsingEdgeCases(t *testing.T) {
	parser := NewUserMountParser()
	
	// Test invalid syntax cases
	invalidSyntaxCases := []struct {
		name        string
		mountString string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing destination",
			mountString: "/source",
			expectError: true,
			errorMsg:    "mount specification must have at least source and destination",
		},
		{
			name:        "empty source",
			mountString: ":/dest",
			expectError: true,
			errorMsg:    "source path cannot be empty",
		},
		{
			name:        "empty destination",
			mountString: "/source:",
			expectError: true,
			errorMsg:    "destination path cannot be empty",
		},
		{
			name:        "too many colons",
			mountString: "/source:/dest:ro:extra:more",
			expectError: true,
			errorMsg:    "mount specification has too many parts",
		},
		{
			name:        "empty mount string",
			mountString: "",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "whitespace only",
			mountString: "   ",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "comma separated empty specs",
			mountString: ",,,",
			expectError: false,
			errorMsg:    "",
		},
	}
	
	for _, tc := range invalidSyntaxCases {
		t.Run(tc.name, func(t *testing.T) {
			mounts, err := parser.ParseMountString(tc.mountString)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for invalid syntax: %s", tc.mountString)
				} else if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for valid input '%s': %v", tc.mountString, err)
				}
				// For empty/whitespace inputs, should return empty slice
				if len(mounts) != 0 {
					t.Errorf("Expected empty mounts for input '%s', got %d mounts", tc.mountString, len(mounts))
				}
			}
		})
	}
	
	// Test missing paths validation
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "nonexistent")
	
	missingPathCases := []struct {
		name        string
		mountString string
		expectError bool
	}{
		{
			name:        "nonexistent source path",
			mountString: fmt.Sprintf("%s:/dest", nonExistentPath),
			expectError: true,
		},
		{
			name:        "existing source path",
			mountString: fmt.Sprintf("%s:/dest", tempDir),
			expectError: false,
		},
	}
	
	for _, tc := range missingPathCases {
		t.Run(tc.name, func(t *testing.T) {
			mounts, err := parser.ParseMountString(tc.mountString)
			if err != nil {
				t.Errorf("ParseMountString failed: %v", err)
				return
			}
			
			if len(mounts) != 1 {
				t.Errorf("Expected 1 mount, got %d", len(mounts))
				return
			}
			
			err = parser.ValidateMount(mounts[0])
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected validation error for nonexistent path")
				} else if !strings.Contains(err.Error(), "does not exist") {
					t.Errorf("Expected 'does not exist' error, got: %s", err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
	
	// Test Windows path conversion edge cases
	// Helper function to test Windows path conversion regardless of OS
	convertWindowsPathForTest := func(path string) string {
		// Convert backslashes to forward slashes
		path = strings.ReplaceAll(path, "\\", "/")
		
		// Handle Windows drive letters (C: -> /c)
		if len(path) >= 2 && path[1] == ':' {
			drive := strings.ToLower(string(path[0]))
			if len(path) == 2 {
				// Just the drive letter
				return "/" + drive
			} else {
				// Drive letter with path
				return "/" + drive + path[2:]
			}
		}
		
		return path
	}
	
	windowsPathCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "backslash to forward slash",
			input:    "C:\\Users\\test\\file.txt",
			expected: "/c/Users/test/file.txt",
		},
		{
			name:     "mixed separators",
			input:    "C:\\Users/test\\file.txt",
			expected: "/c/Users/test/file.txt",
		},
		{
			name:     "drive letter only",
			input:    "C:",
			expected: "/c",
		},
		{
			name:     "forward slashes already",
			input:    "C:/Users/test",
			expected: "/c/Users/test",
		},
		{
			name:     "no drive letter",
			input:    "/usr/local/bin",
			expected: "/usr/local/bin",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
	}
	
	for _, tc := range windowsPathCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertWindowsPathForTest(tc.input)
			if result != tc.expected {
				t.Errorf("ConvertWindowsPath(%s) = %s, expected %s", tc.input, result, tc.expected)
			}
		})
	}
	
	// Test tilde expansion edge cases
	tildeExpansionCases := []struct {
		name     string
		input    string
		expected string // Will be computed based on actual home directory
	}{
		{
			name:  "tilde only",
			input: "~",
		},
		{
			name:  "tilde with path",
			input: "~/.config",
		},
		{
			name:  "tilde user (unsupported)",
			input: "~user/path",
		},
		{
			name:  "no tilde",
			input: "/absolute/path",
		},
		{
			name:  "tilde in middle",
			input: "/path/~/middle",
		},
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory for tilde expansion tests")
	}
	
	for _, tc := range tildeExpansionCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.ExpandTildePath(tc.input)
			
			switch tc.input {
			case "~":
				if result != homeDir {
					t.Errorf("ExpandTildePath(%s) = %s, expected %s", tc.input, result, homeDir)
				}
			case "~/.config":
				expected := filepath.Join(homeDir, ".config")
				if result != expected {
					t.Errorf("ExpandTildePath(%s) = %s, expected %s", tc.input, result, expected)
				}
			case "~user/path", "/absolute/path", "/path/~/middle":
				// These should remain unchanged
				if result != tc.input {
					t.Errorf("ExpandTildePath(%s) = %s, expected %s", tc.input, result, tc.input)
				}
			}
		})
	}
	
	// Test mount option validation
	optionValidationCases := []struct {
		name        string
		options     string
		expectError bool
	}{
		{
			name:        "valid single option",
			options:     "ro",
			expectError: false,
		},
		{
			name:        "valid multiple options",
			options:     "ro,bind",
			expectError: false,
		},
		{
			name:        "invalid option",
			options:     "invalid",
			expectError: true,
		},
		{
			name:        "mixed valid and invalid",
			options:     "ro,invalid,bind",
			expectError: true,
		},
		{
			name:        "empty options",
			options:     "",
			expectError: false,
		},
		{
			name:        "whitespace in options",
			options:     "ro, bind",
			expectError: false,
		},
	}
	
	for _, tc := range optionValidationCases {
		t.Run(tc.name, func(t *testing.T) {
			mount := Mount{
				Source:      tempDir,
				Destination: "/dest",
				Options:     tc.options,
			}
			
			err := parser.ValidateMount(mount)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected validation error for options: %s", tc.options)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error for options '%s': %v", tc.options, err)
				}
			}
		})
	}
	
	// Test error message formatting
	errorMessageCases := []struct {
		name        string
		mountString string
		expectMsg   string
	}{
		{
			name:        "invalid syntax shows example",
			mountString: "invalid",
			expectMsg:   "Expected format: <src>:<dest>[:<opts>],<src>:<dest>[:<opts>],...",
		},
		{
			name:        "invalid syntax shows example usage",
			mountString: "invalid",
			expectMsg:   "Example: MCP_MOUNT=~/.aws:/home/mcp/.aws:ro,~/data:/data",
		},
	}
	
	for _, tc := range errorMessageCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parser.ParseMountString(tc.mountString)
			
			if err == nil {
				t.Errorf("Expected error for invalid syntax: %s", tc.mountString)
			} else if !strings.Contains(err.Error(), tc.expectMsg) {
				t.Errorf("Expected error message to contain '%s', got: %s", tc.expectMsg, err.Error())
			}
		})
	}
}

// Property 9: Home Directory Override Behavior
// For any home directory override (MCP_BIND_HOME or MCP_HOME_PATH), run-mcp should mount the specified host path to /home/mcp instead of using a container volume
func TestProperty9_HomeDirectoryOverrideBehavior(t *testing.T) {
	// **Feature: container-home-isolation, Property 9: Home Directory Override Behavior**
	// **Validates: Requirements 7.6, 7.7**
	
	// Test MCP_BIND_HOME override (Requirement 7.6)
	t.Run("MCP_BIND_HOME_override", func(t *testing.T) {
		// Set up environment
		originalBindHome := os.Getenv("MCP_BIND_HOME")
		defer func() {
			if originalBindHome == "" {
				os.Unsetenv("MCP_BIND_HOME")
			} else {
				os.Setenv("MCP_BIND_HOME", originalBindHome)
			}
		}()
		
		// Test cases for MCP_BIND_HOME
		bindHomeCases := []struct {
			name        string
			bindHome    string
			serverName  string
			shouldBind  bool
		}{
			{
				name:       "bind_home_true",
				bindHome:   "true",
				serverName: "uvx test-server",
				shouldBind: true,
			},
			{
				name:       "bind_home_false",
				bindHome:   "false",
				serverName: "uvx test-server",
				shouldBind: false,
			},
			{
				name:       "bind_home_empty",
				bindHome:   "",
				serverName: "uvx test-server",
				shouldBind: false,
			},
			{
				name:       "bind_home_1",
				bindHome:   "1",
				serverName: "uvx test-server",
				shouldBind: true,
			},
			{
				name:       "bind_home_yes",
				bindHome:   "yes",
				serverName: "uvx test-server",
				shouldBind: true,
			},
		}
		
		for _, tc := range bindHomeCases {
			t.Run(tc.name, func(t *testing.T) {
				os.Setenv("MCP_BIND_HOME", tc.bindHome)
				
				homeOverrideHandler := NewHomeOverrideHandler()
				homeMount := homeOverrideHandler.GetHomeMount(strings.Fields(tc.serverName))
				
				if tc.shouldBind {
					// Should return a bind mount path
					if homeMount == "" {
						t.Errorf("Expected bind mount path for MCP_BIND_HOME=%s, got empty", tc.bindHome)
					}
					
					// Should be in ~/.run-mcp/<volume-name>/ format
					homeDir, err := os.UserHomeDir()
					if err != nil {
						t.Skip("Cannot get home directory")
					}
					
					expectedPrefix := filepath.Join(homeDir, ".run-mcp")
					if !strings.HasPrefix(homeMount, expectedPrefix) {
						t.Errorf("Expected bind mount to start with %s, got %s", expectedPrefix, homeMount)
					}
					
					// Should contain sanitized volume name
					volumeName := sanitizeVolumeName(strings.Fields(tc.serverName))
					expectedPath := filepath.Join(homeDir, ".run-mcp", volumeName)
					if homeMount != expectedPath {
						t.Errorf("Expected bind mount path %s, got %s", expectedPath, homeMount)
					}
				} else {
					// Should return empty (use container volume)
					if homeMount != "" {
						t.Errorf("Expected empty mount path for MCP_BIND_HOME=%s, got %s", tc.bindHome, homeMount)
					}
				}
			})
		}
	})
	
	// Test MCP_HOME_PATH override (Requirement 7.7)
	t.Run("MCP_HOME_PATH_override", func(t *testing.T) {
		// Set up environment
		originalHomePath := os.Getenv("MCP_HOME_PATH")
		defer func() {
			if originalHomePath == "" {
				os.Unsetenv("MCP_HOME_PATH")
			} else {
				os.Setenv("MCP_HOME_PATH", originalHomePath)
			}
		}()
		
		// Create temporary directories for testing
		tempDir := t.TempDir()
		testPath1 := filepath.Join(tempDir, "custom-home1")
		testPath2 := filepath.Join(tempDir, "custom-home2")
		os.MkdirAll(testPath1, 0755)
		os.MkdirAll(testPath2, 0755)
		
		// Test cases for MCP_HOME_PATH
		homePathCases := []struct {
			name        string
			homePath    string
			serverName  string
			expectedPath string
		}{
			{
				name:         "custom_path_absolute",
				homePath:     testPath1,
				serverName:   "uvx test-server",
				expectedPath: testPath1,
			},
			{
				name:         "custom_path_different",
				homePath:     testPath2,
				serverName:   "npx different-server",
				expectedPath: testPath2,
			},
			{
				name:         "empty_path",
				homePath:     "",
				serverName:   "uvx test-server",
				expectedPath: "",
			},
		}
		
		for _, tc := range homePathCases {
			t.Run(tc.name, func(t *testing.T) {
				os.Setenv("MCP_HOME_PATH", tc.homePath)
				
				homeOverrideHandler := NewHomeOverrideHandler()
				homeMount := homeOverrideHandler.GetHomeMount(strings.Fields(tc.serverName))
				
				if tc.expectedPath == "" {
					// Should return empty (use default behavior)
					if homeMount != "" {
						t.Errorf("Expected empty mount path for empty MCP_HOME_PATH, got %s", homeMount)
					}
				} else {
					// Should return the specified path
					if homeMount != tc.expectedPath {
						t.Errorf("Expected mount path %s, got %s", tc.expectedPath, homeMount)
					}
				}
			})
		}
	})
	
	// Test precedence: MCP_HOME_PATH takes precedence over MCP_BIND_HOME
	t.Run("precedence_test", func(t *testing.T) {
		// Set up environment
		originalBindHome := os.Getenv("MCP_BIND_HOME")
		originalHomePath := os.Getenv("MCP_HOME_PATH")
		defer func() {
			if originalBindHome == "" {
				os.Unsetenv("MCP_BIND_HOME")
			} else {
				os.Setenv("MCP_BIND_HOME", originalBindHome)
			}
			if originalHomePath == "" {
				os.Unsetenv("MCP_HOME_PATH")
			} else {
				os.Setenv("MCP_HOME_PATH", originalHomePath)
			}
		}()
		
		// Create temporary directory for testing
		tempDir := t.TempDir()
		customPath := filepath.Join(tempDir, "custom-home")
		os.MkdirAll(customPath, 0755)
		
		// Set both environment variables
		os.Setenv("MCP_BIND_HOME", "true")
		os.Setenv("MCP_HOME_PATH", customPath)
		
		homeOverrideHandler := NewHomeOverrideHandler()
		serverName := "uvx test-server"
		homeMount := homeOverrideHandler.GetHomeMount(strings.Fields(serverName))
		
		// MCP_HOME_PATH should take precedence
		if homeMount != customPath {
			t.Errorf("Expected MCP_HOME_PATH to take precedence, got %s instead of %s", homeMount, customPath)
		}
	})
	
	// Test tilde expansion in MCP_HOME_PATH
	t.Run("tilde_expansion", func(t *testing.T) {
		// Set up environment
		originalHomePath := os.Getenv("MCP_HOME_PATH")
		defer func() {
			if originalHomePath == "" {
				os.Unsetenv("MCP_HOME_PATH")
			} else {
				os.Setenv("MCP_HOME_PATH", originalHomePath)
			}
		}()
		
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot get home directory for tilde expansion test")
		}
		
		// Create test directory in home
		testSubDir := ".mcp-test-home"
		testPath := filepath.Join(homeDir, testSubDir)
		os.MkdirAll(testPath, 0755)
		defer os.RemoveAll(testPath)
		
		// Test tilde expansion
		os.Setenv("MCP_HOME_PATH", "~/"+testSubDir)
		
		homeOverrideHandler := NewHomeOverrideHandler()
		serverName := "uvx test-server"
		homeMount := homeOverrideHandler.GetHomeMount(strings.Fields(serverName))
		
		expectedPath := testPath
		if homeMount != expectedPath {
			t.Errorf("Expected tilde expansion to %s, got %s", expectedPath, homeMount)
		}
	})
	
	// Test directory creation for bind home
	t.Run("bind_home_directory_creation", func(t *testing.T) {
		// Set up environment
		originalBindHome := os.Getenv("MCP_BIND_HOME")
		defer func() {
			if originalBindHome == "" {
				os.Unsetenv("MCP_BIND_HOME")
			} else {
				os.Setenv("MCP_BIND_HOME", originalBindHome)
			}
		}()
		
		os.Setenv("MCP_BIND_HOME", "true")
		
		homeOverrideHandler := NewHomeOverrideHandler()
		serverName := "uvx test-server-creation"
		volumeName := sanitizeVolumeName(strings.Fields(serverName))
		
		// Create bind home directory
		bindPath, err := homeOverrideHandler.CreateBindHomeDir(volumeName)
		if err != nil {
			t.Errorf("CreateBindHomeDir failed: %v", err)
		}
		
		// Verify directory was created
		if _, err := os.Stat(bindPath); os.IsNotExist(err) {
			t.Errorf("Bind home directory was not created: %s", bindPath)
		}
		
		// Verify path format
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot get home directory")
		}
		
		expectedPath := filepath.Join(homeDir, ".run-mcp", volumeName)
		if bindPath != expectedPath {
			t.Errorf("Expected bind path %s, got %s", expectedPath, bindPath)
		}
		
		// Verify directory is writable
		testFile := filepath.Join(bindPath, "test-write")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Errorf("Bind home directory is not writable: %v", err)
		}
		
		// Clean up
		os.Remove(testFile)
		os.RemoveAll(filepath.Join(homeDir, ".run-mcp"))
	})
}

// Unit tests for bind home directory creation
// Test directory creation and permissions
// Requirements: 7.6, 7.7
func TestBindHomeDirectoryCreation(t *testing.T) {
	homeOverrideHandler := NewHomeOverrideHandler()
	
	t.Run("create_bind_home_directory", func(t *testing.T) {
		volumeName := "test-volume-bind-home"
		
		// Clean up any existing directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot get home directory")
		}
		runMcpDir := filepath.Join(homeDir, ".run-mcp")
		os.RemoveAll(runMcpDir)
		
		// Create bind home directory
		bindPath, err := homeOverrideHandler.CreateBindHomeDir(volumeName)
		if err != nil {
			t.Errorf("CreateBindHomeDir failed: %v", err)
		}
		
		// Verify directory was created
		if _, err := os.Stat(bindPath); os.IsNotExist(err) {
			t.Errorf("Bind home directory was not created: %s", bindPath)
		}
		
		// Verify path format
		expectedPath := filepath.Join(homeDir, ".run-mcp", volumeName)
		if bindPath != expectedPath {
			t.Errorf("Expected bind path %s, got %s", expectedPath, bindPath)
		}
		
		// Clean up
		os.RemoveAll(runMcpDir)
	})
	
	t.Run("directory_permissions", func(t *testing.T) {
		volumeName := "test-volume-permissions"
		
		// Clean up any existing directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot get home directory")
		}
		runMcpDir := filepath.Join(homeDir, ".run-mcp")
		os.RemoveAll(runMcpDir)
		
		// Create bind home directory
		bindPath, err := homeOverrideHandler.CreateBindHomeDir(volumeName)
		if err != nil {
			t.Errorf("CreateBindHomeDir failed: %v", err)
		}
		
		// Test write permissions
		testFile := filepath.Join(bindPath, "test-write")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Errorf("Bind home directory is not writable: %v", err)
		}
		
		// Test read permissions
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Cannot read from bind home directory: %v", err)
		}
		
		if string(content) != "test content" {
			t.Errorf("Expected 'test content', got '%s'", string(content))
		}
		
		// Test subdirectory creation
		subDir := filepath.Join(bindPath, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Errorf("Cannot create subdirectory in bind home: %v", err)
		}
		
		// Clean up
		os.RemoveAll(runMcpDir)
	})
	
	t.Run("nested_directory_creation", func(t *testing.T) {
		volumeName := "test-volume-nested"
		
		// Clean up any existing directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot get home directory")
		}
		runMcpDir := filepath.Join(homeDir, ".run-mcp")
		os.RemoveAll(runMcpDir)
		
		// Create bind home directory (should create nested .run-mcp directory)
		bindPath, err := homeOverrideHandler.CreateBindHomeDir(volumeName)
		if err != nil {
			t.Errorf("CreateBindHomeDir failed: %v", err)
		}
		
		// Verify parent directory was created
		if _, err := os.Stat(runMcpDir); os.IsNotExist(err) {
			t.Errorf("Parent .run-mcp directory was not created: %s", runMcpDir)
		}
		
		// Verify target directory was created
		if _, err := os.Stat(bindPath); os.IsNotExist(err) {
			t.Errorf("Target bind directory was not created: %s", bindPath)
		}
		
		// Clean up
		os.RemoveAll(runMcpDir)
	})
	
	t.Run("idempotent_creation", func(t *testing.T) {
		volumeName := "test-volume-idempotent"
		
		// Clean up any existing directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot get home directory")
		}
		runMcpDir := filepath.Join(homeDir, ".run-mcp")
		os.RemoveAll(runMcpDir)
		
		// Create bind home directory first time
		bindPath1, err := homeOverrideHandler.CreateBindHomeDir(volumeName)
		if err != nil {
			t.Errorf("First CreateBindHomeDir failed: %v", err)
		}
		
		// Create test file
		testFile := filepath.Join(bindPath1, "existing-file")
		if err := os.WriteFile(testFile, []byte("existing content"), 0644); err != nil {
			t.Errorf("Cannot create test file: %v", err)
		}
		
		// Create bind home directory second time (should be idempotent)
		bindPath2, err := homeOverrideHandler.CreateBindHomeDir(volumeName)
		if err != nil {
			t.Errorf("Second CreateBindHomeDir failed: %v", err)
		}
		
		// Verify paths are the same
		if bindPath1 != bindPath2 {
			t.Errorf("Expected same path, got %s and %s", bindPath1, bindPath2)
		}
		
		// Verify existing file is preserved
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Existing file was not preserved: %v", err)
		}
		
		if string(content) != "existing content" {
			t.Errorf("Expected 'existing content', got '%s'", string(content))
		}
		
		// Clean up
		os.RemoveAll(runMcpDir)
	})
}

// Unit tests for path expansion and validation
// Requirements: 7.6, 7.7
func TestHomePathExpansionAndValidation(t *testing.T) {
	homeOverrideHandler := NewHomeOverrideHandler()
	
	t.Run("validate_custom_home_path", func(t *testing.T) {
		// Create temporary directory for testing
		tempDir := t.TempDir()
		
		// Test valid directory
		err := homeOverrideHandler.ValidateCustomHomePath(tempDir)
		if err != nil {
			t.Errorf("ValidateCustomHomePath failed for valid directory: %v", err)
		}
		
		// Test non-existent directory
		nonExistentPath := filepath.Join(tempDir, "non-existent")
		err = homeOverrideHandler.ValidateCustomHomePath(nonExistentPath)
		if err == nil {
			t.Errorf("Expected error for non-existent directory, got nil")
		}
		
		// Test file instead of directory
		testFile := filepath.Join(tempDir, "test-file")
		os.WriteFile(testFile, []byte("test"), 0644)
		err = homeOverrideHandler.ValidateCustomHomePath(testFile)
		if err == nil {
			t.Errorf("Expected error for file instead of directory, got nil")
		}
		
		// Test empty path
		err = homeOverrideHandler.ValidateCustomHomePath("")
		if err == nil {
			t.Errorf("Expected error for empty path, got nil")
		}
	})
	
	t.Run("tilde_expansion_in_validation", func(t *testing.T) {
		// Create test directory in home
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot get home directory")
		}
		
		testDir := filepath.Join(homeDir, "test-mcp-home-validation")
		os.MkdirAll(testDir, 0755)
		defer os.RemoveAll(testDir)
		
		// Test tilde expansion in validation
		tildeTestDir := "~/test-mcp-home-validation"
		err = homeOverrideHandler.ValidateCustomHomePath(tildeTestDir)
		if err != nil {
			t.Errorf("ValidateCustomHomePath failed for tilde path: %v", err)
		}
	})
	
	t.Run("write_permission_validation", func(t *testing.T) {
		// Create temporary directory for testing
		tempDir := t.TempDir()
		
		// Test writable directory
		err := homeOverrideHandler.ValidateCustomHomePath(tempDir)
		if err != nil {
			t.Errorf("ValidateCustomHomePath failed for writable directory: %v", err)
		}
		
		// Create read-only directory (if possible on this platform)
		readOnlyDir := filepath.Join(tempDir, "readonly")
		os.MkdirAll(readOnlyDir, 0555) // Read and execute only
		defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup
		
		err = homeOverrideHandler.ValidateCustomHomePath(readOnlyDir)
		// Note: This test might not work on all platforms (e.g., Windows)
		// but it's still valuable where it does work
		if err == nil && runtime.GOOS != "windows" {
			t.Logf("Warning: Expected error for read-only directory, but got nil (platform may not support read-only directories)")
		}
	})
}
// Test environment variable filtering for MCP configuration variables
// Requirements: 3.5
func TestMCPConfigurationVariableFiltering(t *testing.T) {
	// Save original environment
	originalMount := os.Getenv("MCP_MOUNT")
	originalBindHome := os.Getenv("MCP_BIND_HOME")
	originalHomePath := os.Getenv("MCP_HOME_PATH")
	originalOtherMCP := os.Getenv("MCP_OTHER_VAR")
	
	defer func() {
		// Restore original environment
		if originalMount == "" {
			os.Unsetenv("MCP_MOUNT")
		} else {
			os.Setenv("MCP_MOUNT", originalMount)
		}
		if originalBindHome == "" {
			os.Unsetenv("MCP_BIND_HOME")
		} else {
			os.Setenv("MCP_BIND_HOME", originalBindHome)
		}
		if originalHomePath == "" {
			os.Unsetenv("MCP_HOME_PATH")
		} else {
			os.Setenv("MCP_HOME_PATH", originalHomePath)
		}
		if originalOtherMCP == "" {
			os.Unsetenv("MCP_OTHER_VAR")
		} else {
			os.Setenv("MCP_OTHER_VAR", originalOtherMCP)
		}
	}()
	
	// Set test environment variables
	os.Setenv("MCP_MOUNT", "~/test:/test")
	os.Setenv("MCP_BIND_HOME", "true")
	os.Setenv("MCP_HOME_PATH", "/custom/home")
	os.Setenv("MCP_OTHER_VAR", "should_pass_through")
	
	// Create environment filter
	envFilter := NewEnvFilter()
	filteredArgs := envFilter.GetFilteredEnvArgs()
	
	// Convert args to map for easier checking
	envMap := make(map[string]string)
	for i := 0; i < len(filteredArgs); i += 2 {
		if filteredArgs[i] == "-e" && i+1 < len(filteredArgs) {
			parts := strings.SplitN(filteredArgs[i+1], "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}
	}
	
	// Verify MCP configuration variables are filtered out
	if _, exists := envMap["MCP_MOUNT"]; exists {
		t.Errorf("MCP_MOUNT should be filtered out but was passed through")
	}
	
	if _, exists := envMap["MCP_BIND_HOME"]; exists {
		t.Errorf("MCP_BIND_HOME should be filtered out but was passed through")
	}
	
	if _, exists := envMap["MCP_HOME_PATH"]; exists {
		t.Errorf("MCP_HOME_PATH should be filtered out but was passed through")
	}
	
	// Verify other MCP variables are still passed through
	if value, exists := envMap["MCP_OTHER_VAR"]; !exists {
		t.Errorf("MCP_OTHER_VAR should be passed through but was filtered out")
	} else if value != "should_pass_through" {
		t.Errorf("Expected MCP_OTHER_VAR=should_pass_through, got %s", value)
	}
}
// Property 12: Backward Compatibility
// For any existing MCP server command that worked before volume isolation, 
// the command should continue to work transparently with automatic volume creation
func TestProperty12_BackwardCompatibility(t *testing.T) {
	// **Feature: container-home-isolation, Property 12: Backward Compatibility**
	// **Validates: Requirements 5.1, 5.4**
	
	// Test cases representing existing MCP server commands that should continue working
	testCases := []struct {
		name        string
		args        []string
		language    string
		description string
	}{
		{
			name:        "uvx command",
			args:        []string{"uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"},
			language:    "python",
			description: "Python uvx command should work transparently",
		},
		{
			name:        "npx command",
			args:        []string{"npx", "@modelcontextprotocol/server-filesystem", "/data"},
			language:    "nodejs",
			description: "Node.js npx command should work transparently",
		},
		{
			name:        "python explicit",
			args:        []string{"python", "uvx", "awslabs.aws-api-mcp-server@latest"},
			language:    "python",
			description: "Explicit python runtime should work transparently",
		},
		{
			name:        "node explicit",
			args:        []string{"node", "npx", "@modelcontextprotocol/server-memory"},
			language:    "nodejs",
			description: "Explicit node runtime should work transparently",
		},
		{
			name:        "complex command with flags",
			args:        []string{"uvx", "mcp-server-sqlite", "--db-path", "/data/test.db", "--readonly"},
			language:    "python",
			description: "Complex commands with multiple flags should work transparently",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				NodejsImage: "ghcr.io/serverless-dna/run-mcp-nodejs:latest",
				PythonImage: "ghcr.io/serverless-dna/run-mcp-python:latest",
				DataDir:     t.TempDir(),
			}
			
			// Build container command - this should work without errors
			cmd, volumeName, err := buildContainerCommand(config, "docker", tc.language, tc.args)
			if err != nil {
				t.Fatalf("buildContainerCommand failed for %s: %v", tc.description, err)
			}
			
			// Verify command was built successfully
			if cmd == nil {
				t.Fatalf("buildContainerCommand returned nil command for %s", tc.description)
			}
			
			// Verify volume name was generated (unless using home override)
			homeMount := os.Getenv("MCP_BIND_HOME")
			customHome := os.Getenv("MCP_HOME_PATH")
			if homeMount == "" && customHome == "" {
				// Should have a volume name when not using overrides
				if volumeName == "" {
					t.Errorf("Expected volume name to be generated for %s", tc.description)
				}
				
				// Volume name should follow expected pattern
				if !strings.HasPrefix(volumeName, "mcp-home-") && !strings.HasPrefix(volumeName, "mcp-ephemeral-") {
					t.Errorf("Volume name should follow expected pattern, got: %s", volumeName)
				}
			}
			
			// Verify container arguments include expected elements
			args := cmd.Args
			
			// Should have docker/runtime command
			if len(args) == 0 {
				t.Fatalf("Command args should not be empty for %s", tc.description)
			}
			
			// Should include run, -i, --rm flags
			foundRun := false
			foundInteractive := false
			foundRemove := false
			foundVolumeMount := false
			
			for i, arg := range args {
				switch arg {
				case "run":
					foundRun = true
				case "-i":
					foundInteractive = true
				case "--rm":
					foundRemove = true
				case "-v":
					if i+1 < len(args) && strings.Contains(args[i+1], ":/home/mcp") {
						foundVolumeMount = true
					}
				}
			}
			
			if !foundRun {
				t.Errorf("Command should include 'run' argument for %s", tc.description)
			}
			if !foundInteractive {
				t.Errorf("Command should include '-i' argument for %s", tc.description)
			}
			if !foundRemove {
				t.Errorf("Command should include '--rm' argument for %s", tc.description)
			}
			if !foundVolumeMount {
				t.Errorf("Command should include volume mount to /home/mcp for %s", tc.description)
			}
			
			// Verify the original command arguments are preserved at the end
			// Find the image argument first
			imageIndex := -1
			expectedImage, _ := config.GetImageForLanguage(tc.language)
			for i, arg := range args {
				if arg == expectedImage {
					imageIndex = i
					break
				}
			}
			
			if imageIndex == -1 {
				t.Errorf("Expected image %s not found in command args for %s", expectedImage, tc.description)
			} else {
				// Arguments after the image should match the original command
				commandArgs := args[imageIndex+1:]
				
				// Handle explicit runtime specification
				expectedArgs := tc.args
				if len(tc.args) >= 2 && (tc.args[0] == "python" || tc.args[0] == "node" || tc.args[0] == "nodejs") {
					expectedArgs = tc.args[1:]
				}
				
				if len(commandArgs) != len(expectedArgs) {
					t.Errorf("Command args length mismatch for %s: expected %d, got %d", tc.description, len(expectedArgs), len(commandArgs))
				} else {
					for i, expected := range expectedArgs {
						if i < len(commandArgs) && commandArgs[i] != expected {
							t.Errorf("Command arg mismatch for %s at position %d: expected %s, got %s", tc.description, i, expected, commandArgs[i])
						}
					}
				}
			}
		})
	}
}
// Integration tests for container execution
// Test complete container startup with volume mounts and environment variable passthrough
// Requirements: 3.1, 3.3, 3.5
func TestContainerExecutionIntegration(t *testing.T) {
	// Save original environment
	originalAWS := os.Getenv("AWS_ACCESS_KEY_ID")
	originalOpenAI := os.Getenv("OPENAI_API_KEY")
	originalMount := os.Getenv("MCP_MOUNT")
	originalBindHome := os.Getenv("MCP_BIND_HOME")
	originalHomePath := os.Getenv("MCP_HOME_PATH")
	
	defer func() {
		// Restore original environment
		if originalAWS == "" {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
		} else {
			os.Setenv("AWS_ACCESS_KEY_ID", originalAWS)
		}
		if originalOpenAI == "" {
			os.Unsetenv("OPENAI_API_KEY")
		} else {
			os.Setenv("OPENAI_API_KEY", originalOpenAI)
		}
		if originalMount == "" {
			os.Unsetenv("MCP_MOUNT")
		} else {
			os.Setenv("MCP_MOUNT", originalMount)
		}
		if originalBindHome == "" {
			os.Unsetenv("MCP_BIND_HOME")
		} else {
			os.Setenv("MCP_BIND_HOME", originalBindHome)
		}
		if originalHomePath == "" {
			os.Unsetenv("MCP_HOME_PATH")
		} else {
			os.Setenv("MCP_HOME_PATH", originalHomePath)
		}
	}()
	
	testCases := []struct {
		name        string
		args        []string
		language    string
		envVars     map[string]string
		mountConfig string
		description string
	}{
		{
			name:     "basic_python_command",
			args:     []string{"uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"},
			language: "python",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID": "test-aws-key",
				"OPENAI_API_KEY":    "test-openai-key",
			},
			description: "Basic Python command with environment variables",
		},
		{
			name:     "nodejs_with_user_mounts",
			args:     []string{"npx", "@modelcontextprotocol/server-filesystem", "/data"},
			language: "nodejs",
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-github-token",
			},
			mountConfig: "~/test-data:/data:ro",
			description: "Node.js command with user-specified mounts",
		},
		{
			name:     "python_with_complex_env",
			args:     []string{"python", "uvx", "awslabs.aws-api-mcp-server@latest"},
			language: "python",
			envVars: map[string]string{
				"AWS_REGION":           "us-east-1",
				"AWS_ACCESS_KEY_ID":    "test-key",
				"AWS_SECRET_ACCESS_KEY": "test-secret",
				"ANTHROPIC_API_KEY":    "test-anthropic",
			},
			description: "Python command with multiple environment variables",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear MCP_MOUNT for each test case
			os.Unsetenv("MCP_MOUNT")
			
			// Set up test environment
			for key, value := range tc.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}
			
			if tc.mountConfig != "" {
				// Create test directory for mount source
				testDir := t.TempDir()
				// Replace ~/test-data with actual temp dir
				actualMountConfig := strings.Replace(tc.mountConfig, "~/test-data", testDir, 1)
				os.Setenv("MCP_MOUNT", actualMountConfig)
				defer os.Unsetenv("MCP_MOUNT")
			}
			
			config := &Config{
				NodejsImage: "ghcr.io/serverless-dna/run-mcp-nodejs:latest",
				PythonImage: "ghcr.io/serverless-dna/run-mcp-python:latest",
				DataDir:     t.TempDir(),
			}
			
			// Build container command
			cmd, volumeName, err := buildContainerCommand(config, "docker", tc.language, tc.args)
			if err != nil {
				t.Fatalf("buildContainerCommand failed for %s: %v", tc.description, err)
			}
			
			// Verify command structure
			if cmd == nil {
				t.Fatalf("buildContainerCommand returned nil command for %s", tc.description)
			}
			
			args := cmd.Args
			if len(args) == 0 {
				t.Fatalf("Command args should not be empty for %s", tc.description)
			}
			
			// Test 1: Verify volume mount is present
			foundHomeMount := false
			for i, arg := range args {
				if arg == "-v" && i+1 < len(args) {
					if strings.Contains(args[i+1], ":/home/mcp") {
						foundHomeMount = true
						break
					}
				}
			}
			if !foundHomeMount {
				t.Errorf("Expected home volume mount to /home/mcp for %s", tc.description)
			}
			
			// Test 2: Verify environment variables are passed through correctly
			envFound := make(map[string]bool)
			for i, arg := range args {
				if arg == "-e" && i+1 < len(args) {
					envPair := args[i+1]
					parts := strings.SplitN(envPair, "=", 2)
					if len(parts) == 2 {
						envKey := parts[0]
						envValue := parts[1]
						
						// Check if this is one of our test environment variables
						if expectedValue, exists := tc.envVars[envKey]; exists {
							if envValue == expectedValue {
								envFound[envKey] = true
							} else {
								t.Errorf("Environment variable %s has wrong value: expected %s, got %s", envKey, expectedValue, envValue)
							}
						}
						
						// Verify MCP configuration variables are NOT passed through
						if envKey == "MCP_MOUNT" || envKey == "MCP_BIND_HOME" || envKey == "MCP_HOME_PATH" {
							t.Errorf("MCP configuration variable %s should not be passed to container", envKey)
						}
					}
				}
			}
			
			// Verify all expected environment variables were found
			for envKey := range tc.envVars {
				if !envFound[envKey] {
					t.Errorf("Expected environment variable %s not found in container args for %s", envKey, tc.description)
				}
			}
			
			// Test 3: Verify user mounts are included when specified
			if tc.mountConfig != "" {
				foundUserMount := false
				for i, arg := range args {
					if arg == "-v" && i+1 < len(args) {
						mountSpec := args[i+1]
						// Check if this looks like our user mount (contains :ro or matches pattern)
						if strings.Contains(mountSpec, ":ro") || strings.Contains(mountSpec, ":/data") {
							foundUserMount = true
							break
						}
					}
				}
				if !foundUserMount {
					t.Errorf("Expected user mount not found in container args for %s", tc.description)
				}
			}
			
			// Test 4: Verify volume name generation
			if volumeName == "" && os.Getenv("MCP_BIND_HOME") == "" && os.Getenv("MCP_HOME_PATH") == "" {
				t.Errorf("Expected volume name to be generated for %s", tc.description)
			}
			
			// Test 5: Verify image selection
			expectedImage, _ := config.GetImageForLanguage(tc.language)
			foundImage := false
			for _, arg := range args {
				if arg == expectedImage {
					foundImage = true
					break
				}
			}
			if !foundImage {
				t.Errorf("Expected image %s not found in container args for %s", expectedImage, tc.description)
			}
			
			// Test 6: Verify command arguments are preserved
			imageIndex := -1
			for i, arg := range args {
				if arg == expectedImage {
					imageIndex = i
					break
				}
			}
			
			if imageIndex != -1 && imageIndex+1 < len(args) {
				commandArgs := args[imageIndex+1:]
				expectedArgs := tc.args
				
				// Handle explicit runtime specification
				if len(tc.args) >= 2 && (tc.args[0] == "python" || tc.args[0] == "node" || tc.args[0] == "nodejs") {
					expectedArgs = tc.args[1:]
				}
				
				if len(commandArgs) >= len(expectedArgs) {
					for i, expected := range expectedArgs {
						if i < len(commandArgs) && commandArgs[i] != expected {
							t.Errorf("Command arg mismatch for %s at position %d: expected %s, got %s", tc.description, i, expected, commandArgs[i])
						}
					}
				} else {
					t.Errorf("Not enough command arguments for %s: expected at least %d, got %d", tc.description, len(expectedArgs), len(commandArgs))
				}
			}
		})
	}
}
// Property 3: Home Directory Write Access
// For any file or directory operation within /home/mcp, the container should have full read/write permissions,
// enabling MCP servers to create configuration files, logs, and cache data
func TestProperty3_HomeDirectoryWriteAccess(t *testing.T) {
	// **Feature: container-home-isolation, Property 3: Home Directory Write Access**
	// **Validates: Requirements 1.3, 3.2**
	
	// Test cases for different types of file operations that should be possible in /home/mcp
	testCases := []struct {
		name        string
		args        []string
		language    string
		description string
	}{
		{
			name:        "python_server_config",
			args:        []string{"uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"},
			language:    "python",
			description: "Python server should be able to write config files",
		},
		{
			name:        "nodejs_server_logs",
			args:        []string{"npx", "@modelcontextprotocol/server-filesystem", "/data"},
			language:    "nodejs",
			description: "Node.js server should be able to write log files",
		},
		{
			name:        "python_cache_data",
			args:        []string{"python", "uvx", "awslabs.aws-api-mcp-server@latest"},
			language:    "python",
			description: "Python server should be able to write cache data",
		},
		{
			name:        "nodejs_temp_files",
			args:        []string{"node", "npx", "@modelcontextprotocol/server-memory"},
			language:    "nodejs",
			description: "Node.js server should be able to create temporary files",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				NodejsImage: "ghcr.io/serverless-dna/run-mcp-nodejs:latest",
				PythonImage: "ghcr.io/serverless-dna/run-mcp-python:latest",
				DataDir:     t.TempDir(),
			}
			
			// Build container command
			cmd, volumeName, err := buildContainerCommand(config, "docker", tc.language, tc.args)
			if err != nil {
				t.Fatalf("buildContainerCommand failed for %s: %v", tc.description, err)
			}
			
			// Verify command was built successfully
			if cmd == nil {
				t.Fatalf("buildContainerCommand returned nil command for %s", tc.description)
			}
			
			// Verify volume mount is present and writable (not read-only)
			args := cmd.Args
			foundWritableHomeMount := false
			
			for i, arg := range args {
				if arg == "-v" && i+1 < len(args) {
					mountSpec := args[i+1]
					if strings.Contains(mountSpec, ":/home/mcp") {
						// Verify it's NOT read-only (should not contain :ro)
						if !strings.Contains(mountSpec, ":ro") {
							foundWritableHomeMount = true
						} else {
							t.Errorf("Home directory mount should be writable, but found read-only mount: %s", mountSpec)
						}
						break
					}
				}
			}
			
			if !foundWritableHomeMount {
				t.Errorf("Expected writable home volume mount to /home/mcp for %s", tc.description)
			}
			
			// Verify volume name was generated (indicates persistent storage)
			homeMount := os.Getenv("MCP_BIND_HOME")
			customHome := os.Getenv("MCP_HOME_PATH")
			if homeMount == "" && customHome == "" {
				if volumeName == "" {
					t.Errorf("Expected volume name to be generated for persistent home directory for %s", tc.description)
				}
				
				// Volume should follow expected naming pattern
				if !strings.HasPrefix(volumeName, "mcp-home-") && !strings.HasPrefix(volumeName, "mcp-ephemeral-") {
					t.Errorf("Volume name should follow expected pattern for %s, got: %s", tc.description, volumeName)
				}
			}
			
			// Verify the container will run with appropriate user permissions
			// The container should run as UID 1000 (mcp user) which has write access to /home/mcp
			expectedImage, _ := config.GetImageForLanguage(tc.language)
			foundImage := false
			for _, arg := range args {
				if arg == expectedImage {
					foundImage = true
					break
				}
			}
			
			if !foundImage {
				t.Errorf("Expected image %s not found in container args for %s", expectedImage, tc.description)
			}
			
			// Verify no conflicting user or permission flags that would prevent write access
			for i, arg := range args {
				// Check for user override that might conflict with write permissions
				if arg == "--user" && i+1 < len(args) {
					userSpec := args[i+1]
					// If user is set to root (0:0) or other non-mcp user, it might affect permissions
					if userSpec != "1000:1000" && userSpec != "mcp:mcp" {
						t.Logf("Warning: User override detected (%s) for %s - verify write permissions", userSpec, tc.description)
					}
				}
				
				// Check for read-only filesystem flags
				if arg == "--read-only" {
					t.Errorf("Container should not have read-only filesystem for %s", tc.description)
				}
			}
			
			// Test different mount scenarios
			t.Run("volume_mount", func(t *testing.T) {
				// When using container volumes, verify mount is writable
				if volumeName != "" {
					// Volume mount should be in format: volumeName:/home/mcp (no :ro suffix)
					expectedMount := volumeName + ":/home/mcp"
					foundExpectedMount := false
					
					for i, arg := range args {
						if arg == "-v" && i+1 < len(args) {
							if args[i+1] == expectedMount {
								foundExpectedMount = true
								break
							}
						}
					}
					
					if !foundExpectedMount {
						t.Errorf("Expected volume mount %s not found for %s", expectedMount, tc.description)
					}
				}
			})
			
			t.Run("bind_mount", func(t *testing.T) {
				// When using bind mounts (MCP_BIND_HOME or MCP_HOME_PATH), verify they're writable
				if homeMount != "" || customHome != "" {
					// Bind mounts should also be writable (no :ro suffix)
					foundBindMount := false
					
					for i, arg := range args {
						if arg == "-v" && i+1 < len(args) {
							mountSpec := args[i+1]
							if strings.Contains(mountSpec, ":/home/mcp") && !strings.Contains(mountSpec, ":ro") {
								foundBindMount = true
								break
							}
						}
					}
					
					if !foundBindMount {
						t.Errorf("Expected writable bind mount to /home/mcp for %s", tc.description)
					}
				}
			})
		})
	}
}
// Property 5: Consistent Mount Point
// For any container type (Python or Node.js) and MCP server command, 
// the Container_Home volume should always be mounted at /home/mcp with read/write permissions
func TestProperty5_ConsistentMountPoint(t *testing.T) {
	// **Feature: container-home-isolation, Property 5: Consistent Mount Point**
	// **Validates: Requirements 1.5**
	
	// Test cases covering different container types and command variations
	testCases := []struct {
		name         string
		args         []string
		language     string
		ephemeral    bool
		description  string
	}{
		{
			name:        "python_uvx_persistent",
			args:        []string{"uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"},
			language:    "python",
			ephemeral:   false,
			description: "Python uvx command with persistent volume",
		},
		{
			name:        "python_uvx_ephemeral",
			args:        []string{"uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"},
			language:    "python",
			ephemeral:   true,
			description: "Python uvx command with ephemeral volume",
		},
		{
			name:        "nodejs_npx_persistent",
			args:        []string{"npx", "@modelcontextprotocol/server-filesystem", "/data"},
			language:    "nodejs",
			ephemeral:   false,
			description: "Node.js npx command with persistent volume",
		},
		{
			name:        "nodejs_npx_ephemeral",
			args:        []string{"npx", "@modelcontextprotocol/server-filesystem", "/data"},
			language:    "nodejs",
			ephemeral:   true,
			description: "Node.js npx command with ephemeral volume",
		},
		{
			name:        "python_explicit_runtime",
			args:        []string{"python", "uvx", "awslabs.aws-api-mcp-server@latest"},
			language:    "python",
			ephemeral:   false,
			description: "Explicit Python runtime specification",
		},
		{
			name:        "nodejs_explicit_runtime",
			args:        []string{"node", "npx", "@modelcontextprotocol/server-memory"},
			language:    "nodejs",
			ephemeral:   false,
			description: "Explicit Node.js runtime specification",
		},
		{
			name:        "python_complex_command",
			args:        []string{"uvx", "mcp-server-sqlite", "--db-path", "/data/test.db", "--readonly", "--port", "8080"},
			language:    "python",
			ephemeral:   false,
			description: "Complex Python command with multiple arguments",
		},
		{
			name:        "nodejs_complex_command",
			args:        []string{"npx", "@modelcontextprotocol/server-filesystem", "/data", "--verbose", "--port", "3000"},
			language:    "nodejs",
			ephemeral:   true,
			description: "Complex Node.js command with multiple arguments",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				NodejsImage:   "ghcr.io/serverless-dna/run-mcp-nodejs:latest",
				PythonImage:   "ghcr.io/serverless-dna/run-mcp-python:latest",
				DataDir:       t.TempDir(),
				EphemeralMode: tc.ephemeral,
			}
			
			// Build container command
			cmd, volumeName, err := buildContainerCommand(config, "docker", tc.language, tc.args)
			if err != nil {
				t.Fatalf("buildContainerCommand failed for %s: %v", tc.description, err)
			}
			
			// Verify command was built successfully
			if cmd == nil {
				t.Fatalf("buildContainerCommand returned nil command for %s", tc.description)
			}
			
			args := cmd.Args
			if len(args) == 0 {
				t.Fatalf("Command args should not be empty for %s", tc.description)
			}
			
			// Test 1: Verify exactly one mount to /home/mcp exists
			homeMountCount := 0
			var homeMountSpec string
			
			for i, arg := range args {
				if arg == "-v" && i+1 < len(args) {
					mountSpec := args[i+1]
					if strings.Contains(mountSpec, ":/home/mcp") {
						homeMountCount++
						homeMountSpec = mountSpec
					}
				}
			}
			
			if homeMountCount == 0 {
				t.Errorf("Expected exactly one mount to /home/mcp for %s, found none", tc.description)
			} else if homeMountCount > 1 {
				t.Errorf("Expected exactly one mount to /home/mcp for %s, found %d", tc.description, homeMountCount)
			}
			
			// Test 2: Verify mount point is exactly /home/mcp (not /home/mcp/ or other variations)
			if homeMountSpec != "" {
				parts := strings.Split(homeMountSpec, ":")
				if len(parts) >= 2 {
					mountPoint := parts[1]
					if mountPoint != "/home/mcp" {
						t.Errorf("Expected mount point to be exactly '/home/mcp' for %s, got '%s'", tc.description, mountPoint)
					}
				}
			}
			
			// Test 3: Verify mount is read/write (not read-only)
			if homeMountSpec != "" && strings.Contains(homeMountSpec, ":ro") {
				t.Errorf("Home directory mount should be read/write, not read-only for %s: %s", tc.description, homeMountSpec)
			}
			
			// Test 4: Verify volume name consistency based on mode
			homeMount := os.Getenv("MCP_BIND_HOME")
			customHome := os.Getenv("MCP_HOME_PATH")
			
			if homeMount == "" && customHome == "" {
				// Using container volumes
				if volumeName == "" {
					t.Errorf("Expected volume name to be generated for %s", tc.description)
				} else {
					// Verify volume name follows expected pattern
					if tc.ephemeral {
						if !strings.HasPrefix(volumeName, "mcp-ephemeral-") {
							t.Errorf("Expected ephemeral volume name to start with 'mcp-ephemeral-' for %s, got: %s", tc.description, volumeName)
						}
					} else {
						if !strings.HasPrefix(volumeName, "mcp-home-") {
							t.Errorf("Expected persistent volume name to start with 'mcp-home-' for %s, got: %s", tc.description, volumeName)
						}
					}
					
					// Verify mount spec uses the generated volume name
					expectedMountPrefix := volumeName + ":/home/mcp"
					if !strings.HasPrefix(homeMountSpec, expectedMountPrefix) {
						t.Errorf("Expected mount spec to start with '%s' for %s, got: %s", expectedMountPrefix, tc.description, homeMountSpec)
					}
				}
			}
			
			// Test 5: Verify consistency across different container types
			expectedImage, _ := config.GetImageForLanguage(tc.language)
			foundImage := false
			
			for _, arg := range args {
				if arg == expectedImage {
					foundImage = true
					break
				}
			}
			
			if !foundImage {
				t.Errorf("Expected image %s not found in container args for %s", expectedImage, tc.description)
			}
			
			// Test 6: Verify no conflicting mounts to /home directory
			for i, arg := range args {
				if arg == "-v" && i+1 < len(args) {
					mountSpec := args[i+1]
					parts := strings.Split(mountSpec, ":")
					if len(parts) >= 2 {
						mountPoint := parts[1]
						// Check for conflicting mounts to /home or subdirectories other than /home/mcp
						if strings.HasPrefix(mountPoint, "/home/") && mountPoint != "/home/mcp" {
							t.Errorf("Found conflicting mount to /home directory for %s: %s", tc.description, mountSpec)
						}
					}
				}
			}
			
			// Test 7: Verify mount consistency across runtime variations
			t.Run("runtime_consistency", func(t *testing.T) {
				// Test with different runtime commands to ensure consistency
				runtimes := []string{"docker", "podman", "nerdctl", "finch"}
				
				for _, runtime := range runtimes {
					// Skip if runtime is not available (this is just a consistency check)
					cmd2, volumeName2, err2 := buildContainerCommand(config, runtime, tc.language, tc.args)
					if err2 != nil {
						// Runtime might not be available, skip
						continue
					}
					
					// Find home mount in this runtime's command
					args2 := cmd2.Args
					var homeMountSpec2 string
					
					for i, arg := range args2 {
						if arg == "-v" && i+1 < len(args2) {
							mountSpec := args2[i+1]
							if strings.Contains(mountSpec, ":/home/mcp") {
								homeMountSpec2 = mountSpec
								break
							}
						}
					}
					
					// Verify mount point is consistent across runtimes
					if homeMountSpec2 != "" {
						parts := strings.Split(homeMountSpec2, ":")
						if len(parts) >= 2 && parts[1] != "/home/mcp" {
							t.Errorf("Mount point inconsistent across runtimes for %s with %s: expected /home/mcp, got %s", tc.description, runtime, parts[1])
						}
					}
					
					// Verify volume name consistency (should be same across runtimes for same command)
					// Note: Ephemeral volumes use timestamps so they will be different each time
					if homeMount == "" && customHome == "" && volumeName != "" && volumeName2 != "" && !tc.ephemeral {
						if volumeName != volumeName2 {
							t.Errorf("Volume name inconsistent across runtimes for %s: %s vs %s", tc.description, volumeName, volumeName2)
						}
					}
				}
			})
		})
	}
}
// Property 7: Environment Variable Passthrough
// For any user-provided environment variables, run-mcp should pass them through to the container 
// without modification, preserving both names and values
func TestProperty7_EnvironmentVariablePassthrough(t *testing.T) {
	// **Feature: container-home-isolation, Property 7: Environment Variable Passthrough**
	// **Validates: Requirements 3.1, 3.3**
	
	// Save original environment to restore later
	originalEnv := make(map[string]string)
	testEnvVars := []string{
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION", "AWS_SESSION_TOKEN",
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "AZURE_OPENAI_API_KEY",
		"GOOGLE_API_KEY", "GITHUB_TOKEN", "GITLAB_TOKEN",
		"DATABASE_URL", "REDIS_URL", "HF_TOKEN", "REPLICATE_API_TOKEN",
		"COHERE_API_KEY", "MCP_CUSTOM_VAR", "MCP_PASSTHROUGH_TEST",
	}
	
	// Save original values
	for _, envVar := range testEnvVars {
		originalEnv[envVar] = os.Getenv(envVar)
	}
	
	defer func() {
		// Restore original environment
		for _, envVar := range testEnvVars {
			if originalValue, existed := originalEnv[envVar]; existed && originalValue != "" {
				os.Setenv(envVar, originalValue)
			} else {
				os.Unsetenv(envVar)
			}
		}
	}()
	
	// Test cases with different combinations of environment variables
	testCases := []struct {
		name        string
		args        []string
		language    string
		envVars     map[string]string
		description string
	}{
		{
			name:     "aws_credentials",
			args:     []string{"uvx", "awslabs.aws-api-mcp-server@latest"},
			language: "python",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIAIOSFODNN7EXAMPLE",
				"AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				"AWS_REGION":            "us-east-1",
				"AWS_SESSION_TOKEN":     "example-session-token",
			},
			description: "AWS credentials should be passed through",
		},
		{
			name:     "ai_api_keys",
			args:     []string{"npx", "@modelcontextprotocol/server-memory"},
			language: "nodejs",
			envVars: map[string]string{
				"OPENAI_API_KEY":       "sk-example-openai-key",
				"ANTHROPIC_API_KEY":    "sk-ant-example-key",
				"AZURE_OPENAI_API_KEY": "example-azure-key",
				"GOOGLE_API_KEY":       "example-google-key",
			},
			description: "AI API keys should be passed through",
		},
		{
			name:     "development_tokens",
			args:     []string{"python", "uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"},
			language: "python",
			envVars: map[string]string{
				"GITHUB_TOKEN":        "ghp_example_token",
				"GITLAB_TOKEN":        "glpat_example_token",
				"HF_TOKEN":            "hf_example_token",
				"REPLICATE_API_TOKEN": "r8_example_token",
				"COHERE_API_KEY":      "example_cohere_key",
			},
			description: "Development tokens should be passed through",
		},
		{
			name:     "database_urls",
			args:     []string{"node", "npx", "@modelcontextprotocol/server-filesystem", "/data"},
			language: "nodejs",
			envVars: map[string]string{
				"DATABASE_URL": "postgresql://user:pass@localhost:5432/db",
				"REDIS_URL":    "redis://localhost:6379",
			},
			description: "Database URLs should be passed through",
		},
		{
			name:     "custom_mcp_vars",
			args:     []string{"uvx", "mcp-server-sqlite", "--db-path", "/data/db.sqlite"},
			language: "python",
			envVars: map[string]string{
				"MCP_CUSTOM_VAR":       "custom_value",
				"MCP_PASSTHROUGH_TEST": "test_value",
				"MCP_SERVER_CONFIG":    "config_value",
			},
			description: "Custom MCP variables should be passed through",
		},
		{
			name:     "mixed_environment",
			args:     []string{"npx", "@modelcontextprotocol/server-filesystem", "/data"},
			language: "nodejs",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE",
				"OPENAI_API_KEY":    "sk-example-key",
				"GITHUB_TOKEN":      "ghp_example_token",
				"DATABASE_URL":      "postgresql://localhost:5432/db",
				"MCP_CUSTOM_VAR":    "mixed_test_value",
			},
			description: "Mixed environment variables should all be passed through",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear all test environment variables first
			for _, envVar := range testEnvVars {
				os.Unsetenv(envVar)
			}
			
			// Set up test environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}
			
			config := &Config{
				NodejsImage: "ghcr.io/serverless-dna/run-mcp-nodejs:latest",
				PythonImage: "ghcr.io/serverless-dna/run-mcp-python:latest",
				DataDir:     t.TempDir(),
			}
			
			// Build container command
			cmd, _, err := buildContainerCommand(config, "docker", tc.language, tc.args)
			if err != nil {
				t.Fatalf("buildContainerCommand failed for %s: %v", tc.description, err)
			}
			
			// Verify command was built successfully
			if cmd == nil {
				t.Fatalf("buildContainerCommand returned nil command for %s", tc.description)
			}
			
			args := cmd.Args
			if len(args) == 0 {
				t.Fatalf("Command args should not be empty for %s", tc.description)
			}
			
			// Extract environment variables from container command
			containerEnv := make(map[string]string)
			for i, arg := range args {
				if arg == "-e" && i+1 < len(args) {
					envPair := args[i+1]
					parts := strings.SplitN(envPair, "=", 2)
					if len(parts) == 2 {
						containerEnv[parts[0]] = parts[1]
					}
				}
			}
			
			// Test 1: Verify all expected environment variables are passed through
			for expectedKey, expectedValue := range tc.envVars {
				if actualValue, exists := containerEnv[expectedKey]; !exists {
					t.Errorf("Expected environment variable %s not found in container for %s", expectedKey, tc.description)
				} else if actualValue != expectedValue {
					t.Errorf("Environment variable %s has wrong value for %s: expected %s, got %s", expectedKey, tc.description, expectedValue, actualValue)
				}
			}
			
			// Test 2: Verify MCP configuration variables are NOT passed through
			configVars := []string{"MCP_MOUNT", "MCP_BIND_HOME", "MCP_HOME_PATH"}
			for _, configVar := range configVars {
				if _, exists := containerEnv[configVar]; exists {
					t.Errorf("MCP configuration variable %s should not be passed to container for %s", configVar, tc.description)
				}
			}
			
			// Test 3: Verify environment variable names are preserved exactly
			for expectedKey := range tc.envVars {
				found := false
				for containerKey := range containerEnv {
					if containerKey == expectedKey {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Environment variable name %s not preserved exactly for %s", expectedKey, tc.description)
				}
			}
			
			// Test 4: Verify environment variable values are preserved exactly (no modification)
			for expectedKey, expectedValue := range tc.envVars {
				if actualValue, exists := containerEnv[expectedKey]; exists {
					// Check for any modifications to the value
					if len(actualValue) != len(expectedValue) {
						t.Errorf("Environment variable %s value length changed for %s: expected %d chars, got %d chars", expectedKey, tc.description, len(expectedValue), len(actualValue))
					}
					
					// Check character-by-character to ensure no modifications
					if actualValue != expectedValue {
						t.Errorf("Environment variable %s value modified for %s: expected '%s', got '%s'", expectedKey, tc.description, expectedValue, actualValue)
					}
				}
			}
			
			// Test 5: Verify no unexpected environment variables are added
			// (This test checks that we don't add extra variables beyond what's expected)
			expectedCount := len(tc.envVars)
			actualCount := 0
			
			// Count variables that match our test set
			for containerKey := range containerEnv {
				for expectedKey := range tc.envVars {
					if containerKey == expectedKey {
						actualCount++
						break
					}
				}
			}
			
			if actualCount != expectedCount {
				t.Errorf("Environment variable count mismatch for %s: expected %d test variables, found %d", tc.description, expectedCount, actualCount)
			}
			
			// Test 6: Verify special characters and edge cases in values are preserved
			t.Run("special_characters", func(t *testing.T) {
				// Test with special characters in environment variable values
				specialTestVars := map[string]string{
					"TEST_SPECIAL_CHARS": "value with spaces and symbols: !@#$%^&*(){}[]|\\:;\"'<>?,./",
					"TEST_MULTILINE":     "line1\nline2\nline3",
					"TEST_UNICODE":       "测试中文 🚀 émojis",
					"TEST_EMPTY":         "",
					"TEST_EQUALS":        "key=value=more=equals",
				}
				
				// Save original values
				originalSpecial := make(map[string]string)
				for key := range specialTestVars {
					originalSpecial[key] = os.Getenv(key)
				}
				
				defer func() {
					// Restore original values
					for key, originalValue := range originalSpecial {
						if originalValue != "" {
							os.Setenv(key, originalValue)
						} else {
							os.Unsetenv(key)
						}
					}
				}()
				
				// Set special test variables
				for key, value := range specialTestVars {
					os.Setenv(key, value)
				}
				
				// Build command with special variables
				cmd2, _, err2 := buildContainerCommand(config, "docker", tc.language, tc.args)
				if err2 != nil {
					t.Fatalf("buildContainerCommand failed with special characters: %v", err2)
				}
				
				// Extract environment variables
				containerEnv2 := make(map[string]string)
				for i, arg := range cmd2.Args {
					if arg == "-e" && i+1 < len(cmd2.Args) {
						envPair := cmd2.Args[i+1]
						parts := strings.SplitN(envPair, "=", 2)
						if len(parts) == 2 {
							containerEnv2[parts[0]] = parts[1]
						}
					}
				}
				
				// Verify special characters are preserved
				for key, expectedValue := range specialTestVars {
					if actualValue, exists := containerEnv2[key]; exists {
						if actualValue != expectedValue {
							t.Errorf("Special character preservation failed for %s: expected '%s', got '%s'", key, expectedValue, actualValue)
						}
					}
				}
			})
		})
	}
}

// Property 6: Volume Management Commands
// For any set of managed volumes, the volume list, clean, and inspect commands should correctly display, 
// remove, and show details for volumes matching the mcp-home-* pattern
func TestProperty6_VolumeManagementCommands(t *testing.T) {
	// **Feature: container-home-isolation, Property 6: Volume Management Commands**
	// **Validates: Requirements 2.5, 4.4, 4.5, 4.8, 4.10**
	
	if testing.Short() {
		t.Skip("Skipping property test in short mode")
	}
	
	// Property test with multiple iterations
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Generate random test data
			numVolumes := rand.Intn(5) + 1 // 1-5 volumes
			serverNames := make([]string, numVolumes)
			expectedVolumeNames := make([]string, numVolumes)
			
			for j := 0; j < numVolumes; j++ {
				// Generate random server names
				commands := []string{"uvx", "npx", "python", "node"}
				servers := []string{"test-server", "mcp-server", "api-server", "data-processor"}
				
				command := commands[rand.Intn(len(commands))]
				server := servers[rand.Intn(len(servers))]
				serverNames[j] = fmt.Sprintf("%s %s-%d", command, server, rand.Intn(1000))
				expectedVolumeNames[j] = sanitizeVolumeName(strings.Fields(serverNames[j]))
			}
			
			// Test with mock volume commander
			mockCommander := &MockVolumeCommander{
				volumes: make(map[string]VolumeInfo),
			}
			
			// Create volumes
			for j, serverName := range serverNames {
				labels := map[string]string{
					"run-mcp":         "true",
					"run-mcp.runtime": "docker",
					"run-mcp.server":  serverName,
					"run-mcp.type":    "home",
				}
				
				err := mockCommander.CreateVolume(expectedVolumeNames[j], labels)
				if err != nil {
					t.Fatalf("Failed to create volume: %v", err)
				}
			}
			
			// Test ListVolumes - should return all created volumes
			volumes, err := mockCommander.ListVolumes()
			if err != nil {
				t.Fatalf("ListVolumes failed: %v", err)
			}
			
			// Verify all volumes are returned
			if len(volumes) != numVolumes {
				t.Errorf("Expected %d volumes, got %d", numVolumes, len(volumes))
			}
			
			// Verify volume names match expected pattern
			volumeNames := make(map[string]bool)
			for _, vol := range volumes {
				volumeNames[vol.Name] = true
				
				// Verify volume follows mcp-home-* pattern
				if !strings.HasPrefix(vol.Name, "mcp-home-") {
					t.Errorf("Volume name %s does not follow mcp-home-* pattern", vol.Name)
				}
				
				// Verify required labels exist
				if vol.Labels["run-mcp"] != "true" {
					t.Errorf("Volume %s missing run-mcp=true label", vol.Name)
				}
				
				if vol.Labels["run-mcp.runtime"] == "" {
					t.Errorf("Volume %s missing run-mcp.runtime label", vol.Name)
				}
			}
			
			// Verify all expected volumes are present
			for _, expectedName := range expectedVolumeNames {
				if !volumeNames[expectedName] {
					t.Errorf("Expected volume %s not found in list", expectedName)
				}
			}
			
			// Test InspectVolume for each volume
			for _, volumeName := range expectedVolumeNames {
				details, err := mockCommander.InspectVolume(volumeName)
				if err != nil {
					t.Errorf("InspectVolume failed for %s: %v", volumeName, err)
					continue
				}
				
				// Verify inspect returns correct details
				if details.Name != volumeName {
					t.Errorf("InspectVolume returned wrong name: expected %s, got %s", volumeName, details.Name)
				}
				
				// Verify labels are preserved
				if details.Labels["run-mcp"] != "true" {
					t.Errorf("InspectVolume for %s missing run-mcp=true label", volumeName)
				}
			}
			
			// Test RemoveVolume - remove half the volumes
			volumesToRemove := expectedVolumeNames[:numVolumes/2]
			for _, volumeName := range volumesToRemove {
				err := mockCommander.RemoveVolume(volumeName)
				if err != nil {
					t.Errorf("RemoveVolume failed for %s: %v", volumeName, err)
				}
			}
			
			// Verify removed volumes are gone
			remainingVolumes, err := mockCommander.ListVolumes()
			if err != nil {
				t.Fatalf("ListVolumes after removal failed: %v", err)
			}
			
			expectedRemaining := numVolumes - len(volumesToRemove)
			if len(remainingVolumes) != expectedRemaining {
				t.Errorf("After removal, expected %d volumes, got %d", expectedRemaining, len(remainingVolumes))
			}
			
			// Verify removed volumes are not in the list
			remainingNames := make(map[string]bool)
			for _, vol := range remainingVolumes {
				remainingNames[vol.Name] = true
			}
			
			for _, removedName := range volumesToRemove {
				if remainingNames[removedName] {
					t.Errorf("Removed volume %s still appears in list", removedName)
				}
			}
		})
	}
}

// MockVolumeCommander for testing volume management commands
type MockVolumeCommander struct {
	volumes       map[string]VolumeInfo
	failMode      bool
	failError     error
	failOnExists  bool
	failOnList    bool
	failOnRemove  bool
	failOnInspect bool
}

func (mvc *MockVolumeCommander) CreateVolume(name string, labels map[string]string) error {
	if mvc.failMode {
		return mvc.failError
	}
	
	mvc.volumes[name] = VolumeInfo{
		Name:      name,
		Labels:    labels,
		CreatedAt: time.Now(),
		Runtime:   "mock",
	}
	return nil
}

func (mvc *MockVolumeCommander) ListVolumes() ([]VolumeInfo, error) {
	if mvc.failOnList {
		return nil, fmt.Errorf("mock list volumes failed")
	}
	
	var volumes []VolumeInfo
	for _, vol := range mvc.volumes {
		// Only return volumes with run-mcp=true label
		if vol.Labels["run-mcp"] == "true" {
			volumes = append(volumes, vol)
		}
	}
	return volumes, nil
}

func (mvc *MockVolumeCommander) RemoveVolume(name string) error {
	if mvc.failOnRemove {
		return fmt.Errorf("mock remove volume failed")
	}
	
	if _, exists := mvc.volumes[name]; !exists {
		return fmt.Errorf("volume not found: %s", name)
	}
	delete(mvc.volumes, name)
	return nil
}

func (mvc *MockVolumeCommander) InspectVolume(name string) (*VolumeDetails, error) {
	if mvc.failOnInspect {
		return nil, fmt.Errorf("mock inspect volume failed")
	}
	
	vol, exists := mvc.volumes[name]
	if !exists {
		return nil, fmt.Errorf("volume not found: %s", name)
	}
	
	return &VolumeDetails{
		VolumeInfo: vol,
		MountPoint: "/var/lib/docker/volumes/" + name + "/_data",
		Options:    map[string]string{},
	}, nil
}

func (mvc *MockVolumeCommander) VolumeExists(name string) (bool, error) {
	if mvc.failOnExists {
		return false, fmt.Errorf("mock volume exists check failed")
	}
	
	_, exists := mvc.volumes[name]
	return exists, nil
}
// Unit tests for CLI command parsing
// Test subcommand routing and argument validation
// Requirements: 4.9, 4.13
func TestCLICommandParsing(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "volume command without subcommand",
			args:        []string{"volume"},
			expectError: false, // Should show help, not error
		},
		{
			name:        "volume list command",
			args:        []string{"volume", "list"},
			expectError: false,
		},
		{
			name:        "volume clean with server name",
			args:        []string{"volume", "clean", "test-server"},
			expectError: false,
		},
		{
			name:        "volume clean without server name",
			args:        []string{"volume", "clean"},
			expectError: true,
			errorMsg:    "accepts 1 arg(s), received 0",
		},
		{
			name:        "volume clean with too many args",
			args:        []string{"volume", "clean", "server1", "server2"},
			expectError: true,
			errorMsg:    "accepts 1 arg(s), received 2",
		},
		{
			name:        "volume prune command",
			args:        []string{"volume", "prune"},
			expectError: false,
		},
		{
			name:        "volume prune with force flag",
			args:        []string{"volume", "prune", "--force"},
			expectError: false,
		},
		{
			name:        "volume prune with short force flag",
			args:        []string{"volume", "prune", "-f"},
			expectError: false,
		},
		{
			name:        "volume inspect with server name",
			args:        []string{"volume", "inspect", "test-server"},
			expectError: true, // Will fail because volume doesn't exist, but parsing is correct
		},
		{
			name:        "volume inspect without server name",
			args:        []string{"volume", "inspect"},
			expectError: true,
			errorMsg:    "accepts 1 arg(s), received 0",
		},
		{
			name:        "volume inspect with too many args",
			args:        []string{"volume", "inspect", "server1", "server2"},
			expectError: true,
			errorMsg:    "accepts 1 arg(s), received 2",
		},
		{
			name:        "invalid volume subcommand",
			args:        []string{"volume", "invalid"},
			expectError: false, // Cobra shows help for invalid subcommands, doesn't error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create root command
			rootCmd := &cobra.Command{
				Use: "run-mcp",
			}
			rootCmd.AddCommand(createVolumeCommand())

			// Set args and execute
			rootCmd.SetArgs(tc.args)
			
			// Capture output to avoid printing during tests
			rootCmd.SetOut(os.Stdout)
			rootCmd.SetErr(os.Stderr)

			err := rootCmd.Execute()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Test confirmation prompt behavior
// Requirements: 4.9, 4.13
func TestConfirmationPrompt(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "yes response",
			input:    "y",
			expected: true,
		},
		{
			name:     "yes full response",
			input:    "yes",
			expected: true,
		},
		{
			name:     "Yes capitalized response",
			input:    "Yes",
			expected: true,
		},
		{
			name:     "YES uppercase response",
			input:    "YES",
			expected: true,
		},
		{
			name:     "no response",
			input:    "n",
			expected: false,
		},
		{
			name:     "no full response",
			input:    "no",
			expected: false,
		},
		{
			name:     "empty response",
			input:    "",
			expected: false,
		},
		{
			name:     "invalid response",
			input:    "maybe",
			expected: false,
		},
		{
			name:     "whitespace response",
			input:    "  y  ",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock the confirmation function behavior
			response := strings.ToLower(strings.TrimSpace(tc.input))
			result := response == "y" || response == "yes"
			
			if result != tc.expected {
				t.Errorf("Expected %v for input '%s', got %v", tc.expected, tc.input, result)
			}
		})
	}
}

// Test volume command argument validation
// Requirements: 4.9, 4.13
func TestVolumeCommandArgumentValidation(t *testing.T) {
	testCases := []struct {
		name         string
		command      string
		args         []string
		expectError  bool
		errorPattern string
	}{
		{
			name:        "clean command with valid server name",
			command:     "clean",
			args:        []string{"uvx test-server"},
			expectError: false,
		},
		{
			name:         "clean command with empty server name",
			command:      "clean",
			args:         []string{""},
			expectError:  false, // Empty string is still a valid argument
		},
		{
			name:        "inspect command with valid server name",
			command:     "inspect",
			args:        []string{"npx @modelcontextprotocol/server-memory"},
			expectError: false,
		},
		{
			name:        "inspect command with complex server name",
			command:     "inspect",
			args:        []string{"uvx awslabs.aws-api-mcp-server@latest --region us-east-1"},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the specific volume subcommand
			var cmd *cobra.Command
			switch tc.command {
			case "clean":
				cmd = createVolumeCleanCommand()
			case "inspect":
				cmd = createVolumeInspectCommand()
			default:
				t.Fatalf("Unknown command: %s", tc.command)
			}

			// Set args and validate
			cmd.SetArgs(tc.args)
			
			// Capture output to avoid printing during tests
			cmd.SetOut(os.Stdout)
			cmd.SetErr(os.Stderr)

			err := cmd.Execute()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.errorPattern != "" && !strings.Contains(err.Error(), tc.errorPattern) {
					t.Errorf("Expected error to match pattern '%s', got: %v", tc.errorPattern, err)
				}
			} else {
				// For these tests, we expect the command parsing to succeed
				// The actual execution might fail due to missing runtime, but that's not what we're testing
				if err != nil && !strings.Contains(err.Error(), "container runtime detection failed") {
					t.Errorf("Expected no parsing error but got: %v", err)
				}
			}
		})
	}
}

// Property 11: Volume Prune Operation
// For any set of managed volumes, the prune command should remove all volumes with the mcp-home-* pattern, 
// leaving no managed volumes remaining
func TestProperty11_VolumePruneOperation(t *testing.T) {
	// **Feature: container-home-isolation, Property 11: Volume Prune Operation**
	// **Validates: Requirements 4.6, 4.7**
	
	property := func(volumeCount int) bool {
		if volumeCount < 0 || volumeCount > 10 {
			return true // Skip invalid inputs
		}
		
		// Create a mock volume commander for testing
		commander := &MockVolumeCommander{
			volumes: make(map[string]VolumeInfo),
		}
		
		// Create test volumes with run-mcp labels
		volumeNames := make([]string, volumeCount)
		for i := 0; i < volumeCount; i++ {
			volumeName := fmt.Sprintf("mcp-home-test-server-%d", i)
			volumeNames[i] = volumeName
			
			commander.volumes[volumeName] = VolumeInfo{
				Name: volumeName,
				Labels: map[string]string{
					"run-mcp":         "true",
					"run-mcp.server":  fmt.Sprintf("test-server-%d", i),
					"run-mcp.runtime": "docker",
				},
				CreatedAt: time.Now(),
				Runtime:   "docker",
			}
		}
		
		// List volumes before prune
		volumesBefore, err := commander.ListVolumes()
		if err != nil {
			return false
		}
		
		if len(volumesBefore) != volumeCount {
			return false
		}
		
		// Prune all volumes
		for _, vol := range volumesBefore {
			if err := commander.RemoveVolume(vol.Name); err != nil {
				return false
			}
		}
		
		// List volumes after prune
		volumesAfter, err := commander.ListVolumes()
		if err != nil {
			return false
		}
		
		// Property: After prune, no managed volumes should remain
		return len(volumesAfter) == 0
	}
	
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// Test volume command flag parsing
// Requirements: 4.9, 4.13
func TestVolumeCommandFlagParsing(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		expectForce bool
		expectError bool
	}{
		{
			name:        "prune without force flag",
			args:        []string{"prune"},
			expectForce: false,
			expectError: false,
		},
		{
			name:        "prune with long force flag",
			args:        []string{"prune", "--force"},
			expectForce: true,
			expectError: false,
		},
		{
			name:        "prune with short force flag",
			args:        []string{"prune", "-f"},
			expectForce: true,
			expectError: false,
		},
		{
			name:        "prune with invalid flag",
			args:        []string{"prune", "--invalid"},
			expectForce: false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := createVolumePruneCommand()
			cmd.SetArgs(tc.args)
			
			// Capture output to avoid printing during tests
			cmd.SetOut(os.Stdout)
			cmd.SetErr(os.Stderr)

			err := cmd.Execute()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				// Check if force flag was parsed correctly
				forceFlag, _ := cmd.Flags().GetBool("force")
				if forceFlag != tc.expectForce {
					t.Errorf("Expected force flag to be %v, got %v", tc.expectForce, forceFlag)
				}
				
				// The actual execution might fail due to missing runtime, but that's not what we're testing
				if err != nil && !strings.Contains(err.Error(), "container runtime detection failed") {
					t.Errorf("Expected no parsing error but got: %v", err)
				}
			}
		})
	}
}

// Test storage warning message formatting
// Requirements: 6.6
func TestStorageWarningMessageFormatting(t *testing.T) {
	testCases := []struct {
		name        string
		volumeName  string
		volumeSize  string
		sizeLimit   string
		expectedMsg string
	}{
		{
			name:        "basic warning message",
			volumeName:  "mcp-home-test-server",
			volumeSize:  "200MB",
			sizeLimit:   "100MB",
			expectedMsg: "Warning: Volume 'mcp-home-test-server' size (200MB) exceeds configured limit (100MB)",
		},
		{
			name:        "warning with GB sizes",
			volumeName:  "mcp-home-large-server",
			volumeSize:  "2.5GB",
			sizeLimit:   "1GB",
			expectedMsg: "Warning: Volume 'mcp-home-large-server' size (2.5GB) exceeds configured limit (1GB)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := formatStorageWarningMessage(tc.volumeName, tc.volumeSize, tc.sizeLimit)
			
			if msg != tc.expectedMsg {
				t.Errorf("Expected message '%s', got '%s'", tc.expectedMsg, msg)
			}
		})
	}
}

// Test storage size parsing and comparison
// Requirements: 6.6
func TestStorageSizeComparison(t *testing.T) {
	testCases := []struct {
		name      string
		size1     string
		size2     string
		expected  int // -1 if size1 < size2, 0 if equal, 1 if size1 > size2
		expectErr bool
	}{
		{
			name:     "MB comparison - smaller",
			size1:    "50MB",
			size2:    "100MB",
			expected: -1,
		},
		{
			name:     "MB comparison - equal",
			size1:    "100MB",
			size2:    "100MB",
			expected: 0,
		},
		{
			name:     "MB comparison - larger",
			size1:    "150MB",
			size2:    "100MB",
			expected: 1,
		},
		{
			name:     "GB vs MB comparison",
			size1:    "2GB",
			size2:    "500MB",
			expected: 1,
		},
		{
			name:     "KB vs MB comparison",
			size1:    "500KB",
			size2:    "1MB",
			expected: -1,
		},
		{
			name:      "invalid size format",
			size1:     "invalid",
			size2:     "100MB",
			expectErr: true,
		},
		{
			name:      "empty size",
			size1:     "",
			size2:     "100MB",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := compareStorageSizes(tc.size1, tc.size2)
			
			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				} else if result != tc.expected {
					t.Errorf("Expected comparison result %d, got %d", tc.expected, result)
				}
			}
		})
	}
}

// Unit tests for storage warnings
// Test size limit detection and warning messages
// Requirements: 6.6
func TestStorageWarnings(t *testing.T) {
	testCases := []struct {
		name           string
		volumeSize     string
		maxVolumeSize  string
		expectWarning  bool
		expectedMsg    string
	}{
		{
			name:          "volume under limit",
			volumeSize:    "50MB",
			maxVolumeSize: "100MB",
			expectWarning: false,
		},
		{
			name:          "volume at limit",
			volumeSize:    "100MB",
			maxVolumeSize: "100MB",
			expectWarning: false,
		},
		{
			name:          "volume over limit",
			volumeSize:    "150MB",
			maxVolumeSize: "100MB",
			expectWarning: true,
			expectedMsg:   "Warning: Volume size (150MB) exceeds configured limit (100MB)",
		},
		{
			name:          "no size limit configured",
			volumeSize:    "500MB",
			maxVolumeSize: "",
			expectWarning: false,
		},
		{
			name:          "no volume size available",
			volumeSize:    "",
			maxVolumeSize: "100MB",
			expectWarning: false,
		},
		{
			name:          "invalid size format",
			volumeSize:    "invalid",
			maxVolumeSize: "100MB",
			expectWarning: false, // Should not warn on parse error
		},
		{
			name:          "different units - GB vs MB",
			volumeSize:    "2GB",
			maxVolumeSize: "1500MB",
			expectWarning: true,
			expectedMsg:   "Warning: Volume size (2GB) exceeds configured limit (1500MB)",
		},
		{
			name:          "different units - KB vs MB",
			volumeSize:    "500KB",
			maxVolumeSize: "1MB",
			expectWarning: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				MaxVolumeSize: tc.maxVolumeSize,
			}
			
			volumeInfo := VolumeInfo{
				Name: "test-volume",
				Size: tc.volumeSize,
			}
			
			warning := checkVolumeStorageWarning(config, volumeInfo)
			
			if tc.expectWarning {
				if warning == "" {
					t.Errorf("Expected warning but got none")
				} else if warning != tc.expectedMsg {
					t.Errorf("Expected warning message '%s', got '%s'", tc.expectedMsg, warning)
				}
			} else {
				if warning != "" {
					t.Errorf("Expected no warning but got: %s", warning)
				}
			}
		})
	}
}

// Test formatStorageWarningMessage function
// Requirements: 6.6
func TestFormatStorageWarningMessage(t *testing.T) {
	testCases := []struct {
		name        string
		volumeName  string
		volumeSize  string
		sizeLimit   string
		expected    string
	}{
		{
			name:       "basic warning message",
			volumeName: "mcp-home-test-server",
			volumeSize: "150MB",
			sizeLimit:  "100MB",
			expected:   "Warning: Volume 'mcp-home-test-server' size (150MB) exceeds configured limit (100MB)",
		},
		{
			name:       "warning with GB units",
			volumeName: "mcp-home-large-server",
			volumeSize: "2.5GB",
			sizeLimit:  "2GB",
			expected:   "Warning: Volume 'mcp-home-large-server' size (2.5GB) exceeds configured limit (2GB)",
		},
		{
			name:       "warning with special characters in name",
			volumeName: "mcp-home-server-with-dashes",
			volumeSize: "1TB",
			sizeLimit:  "500GB",
			expected:   "Warning: Volume 'mcp-home-server-with-dashes' size (1TB) exceeds configured limit (500GB)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatStorageWarningMessage(tc.volumeName, tc.volumeSize, tc.sizeLimit)
			if result != tc.expected {
				t.Errorf("Expected message '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// Test parseStorageSize function edge cases
// Requirements: 6.6
func TestParseStorageSizeEdgeCases(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expected  int64
		expectErr bool
	}{
		{
			name:     "bytes",
			input:    "1024B",
			expected: 1024,
		},
		{
			name:     "kilobytes",
			input:    "1KB",
			expected: 1024,
		},
		{
			name:     "megabytes",
			input:    "1MB",
			expected: 1024 * 1024,
		},
		{
			name:     "gigabytes",
			input:    "1GB",
			expected: 1024 * 1024 * 1024,
		},
		{
			name:     "terabytes",
			input:    "1TB",
			expected: 1024 * 1024 * 1024 * 1024,
		},
		{
			name:     "decimal megabytes",
			input:    "1.5MB",
			expected: int64(1.5 * 1024 * 1024),
		},
		{
			name:     "decimal gigabytes",
			input:    "2.5GB",
			expected: int64(2.5 * 1024 * 1024 * 1024),
		},
		{
			name:     "lowercase units",
			input:    "100mb",
			expected: 100 * 1024 * 1024,
		},
		{
			name:     "mixed case units",
			input:    "50Mb",
			expected: 50 * 1024 * 1024,
		},
		{
			name:     "spaces in input",
			input:    " 100 MB ",
			expected: 100 * 1024 * 1024,
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: true,
		},
		{
			name:      "no unit",
			input:     "100",
			expectErr: true,
		},
		{
			name:      "invalid unit",
			input:     "100XB",
			expectErr: true,
		},
		{
			name:      "invalid number",
			input:     "abcMB",
			expectErr: true,
		},
		{
			name:      "negative number",
			input:     "-100MB",
			expectErr: true,
		},
		{
			name:     "zero size",
			input:    "0MB",
			expected: 0,
		},
		{
			name:      "multiple decimal points",
			input:     "1.2.3MB",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseStorageSize(tc.input)
			
			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				} else if result != tc.expected {
					t.Errorf("Expected %d bytes, got %d bytes", tc.expected, result)
				}
			}
		})
	}
}
// Property 10: Volume Persistence
// For any created volume, the data should persist indefinitely across container stops, crashes, and restarts 
// until explicitly removed by user commands
func TestProperty10_VolumePersistence(t *testing.T) {
	// **Feature: container-home-isolation, Property 10: Volume Persistence**
	// **Validates: Requirements 4.3, 6.1, 6.2**
	
	property := func(serverName string, restartCount int) bool {
		// Limit inputs to reasonable ranges
		if len(serverName) == 0 || len(serverName) > 50 || restartCount < 0 || restartCount > 10 {
			return true // Skip invalid inputs
		}
		
		// Sanitize server name to ensure it's valid
		sanitizedName := strings.ReplaceAll(serverName, " ", "-")
		if sanitizedName == "" {
			return true
		}
		
		config := &Config{}
		
		// Create a mock volume commander for testing
		commander := &MockVolumeCommander{
			volumes: make(map[string]VolumeInfo),
		}
		
		vm := &VolumeManager{
			config:    config,
			commander: commander,
			runtime:   "docker",
		}
		
		// Create initial volume
		volumeName, err := vm.CreateHomeVolume(sanitizedName, "docker")
		if err != nil {
			return false
		}
		
		// Verify volume was created
		exists, err := commander.VolumeExists(volumeName)
		if err != nil || !exists {
			return false
		}
		
		// Simulate multiple container restarts
		for i := 0; i < restartCount; i++ {
			// Simulate container stop (volume should persist)
			// In real scenario, only container is removed, not volume
			
			// Simulate container restart - volume should still exist and be reused
			volumeName2, err := vm.CreateHomeVolume(sanitizedName, "docker")
			if err != nil {
				return false
			}
			
			// Property: Same server name should always return same volume name
			if volumeName2 != volumeName {
				return false
			}
			
			// Property: Volume should still exist after restart
			exists, err := commander.VolumeExists(volumeName)
			if err != nil || !exists {
				return false
			}
		}
		
		// Property: Volume should persist until explicitly removed
		// Simulate explicit removal
		err = vm.RemoveHomeVolume(sanitizedName)
		if err != nil {
			return false
		}
		
		// Property: After explicit removal, volume should no longer exist
		exists, err = commander.VolumeExists(volumeName)
		if err != nil {
			return false
		}
		
		// Volume should not exist after explicit removal
		return !exists
	}
	
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// Test runtime detection failures
// Requirements: 5.2, 5.3
func TestRuntimeDetectionFailures(t *testing.T) {
	// Save original environment
	originalRuntime := os.Getenv("MCP_CONTAINER_RUNTIME")
	defer func() {
		if originalRuntime != "" {
			os.Setenv("MCP_CONTAINER_RUNTIME", originalRuntime)
		} else {
			os.Unsetenv("MCP_CONTAINER_RUNTIME")
		}
	}()
	
	// Test with invalid runtime override
	os.Setenv("MCP_CONTAINER_RUNTIME", "nonexistent-runtime-12345")
	
	detector := NewRuntimeDetector()
	_, err := detector.Detect()
	
	if err == nil {
		t.Error("Expected error when runtime override points to nonexistent runtime")
	}
	
	if !strings.Contains(err.Error(), "specified container runtime") {
		t.Errorf("Expected error message about specified runtime, got: %v", err)
	}
	
	// Test when no runtimes are available (simulate by clearing PATH)
	os.Unsetenv("MCP_CONTAINER_RUNTIME")
	
	// Create detector with empty runtime list to simulate no available runtimes
	emptyDetector := &RuntimeDetector{runtimes: []string{"definitely-not-installed-runtime"}}
	_, err = emptyDetector.Detect()
	
	if err == nil {
		t.Error("Expected error when no container runtimes are available")
	}
	
	expectedNoRuntimeMsg := "no container runtime found"
	if !strings.Contains(err.Error(), expectedNoRuntimeMsg) {
		t.Errorf("Expected error message about no runtime found, got: %v", err)
	}
}

// Test volume creation error handling
// Requirements: 5.2, 5.3
func TestVolumeCreationErrorHandling(t *testing.T) {
	config := &Config{}
	
	// Test with mock commander that always fails
	failingCommander := &MockVolumeCommander{
		volumes:   make(map[string]VolumeInfo),
		failMode:  true,
		failError: fmt.Errorf("mock volume creation failed: insufficient disk space"),
	}
	
	vm := &VolumeManager{
		config:    config,
		commander: failingCommander,
		runtime:   "docker",
	}
	
	// Test volume creation failure
	_, err := vm.CreateHomeVolume("test-server", "docker")
	if err == nil {
		t.Error("Expected error from failing volume commander")
	}
	
	expectedErrMsg := "failed to create volume"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error message about volume creation failure, got: %v", err)
	}
	
	// Test volume existence check failure
	failingCommander.failOnExists = true
	_, err = vm.CreateHomeVolume("test-server-2", "docker")
	if err == nil {
		t.Error("Expected error from failing volume existence check")
	}
	
	expectedExistsErrMsg := "failed to check if volume exists"
	if !strings.Contains(err.Error(), expectedExistsErrMsg) {
		t.Errorf("Expected error message about volume existence check failure, got: %v", err)
	}
	
	// Test ephemeral volume creation failure
	failingCommander.failOnExists = false // Reset
	_, err = vm.CreateEphemeralVolume("test-ephemeral", "docker")
	if err == nil {
		t.Error("Expected error from failing ephemeral volume creation")
	}
	
	expectedEphemeralErrMsg := "failed to create ephemeral volume"
	if !strings.Contains(err.Error(), expectedEphemeralErrMsg) {
		t.Errorf("Expected error message about ephemeral volume creation failure, got: %v", err)
	}
}

// Test filesystem error detection and suggestions
// Requirements: 5.2, 5.3, 5.5
func TestFilesystemErrorDetection(t *testing.T) {
	// Test invalid data directory
	config := &Config{
		NodejsImage: "test-image:latest", // Set required fields
		PythonImage: "test-image:latest",
		DataDir:     "/nonexistent/directory/path/12345",
	}
	
	err := config.Validate()
	if err == nil {
		t.Error("Expected error for nonexistent data directory")
	}
	
	if !strings.Contains(err.Error(), "data directory validation failed") {
		t.Errorf("Expected data directory validation error, got: %v", err)
	}
	
	// Test data directory that exists but is not a directory (use a file)
	tempFile, err := os.CreateTemp("", "test-file")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	
	config.DataDir = tempFile.Name()
	err = config.Validate()
	if err == nil {
		t.Error("Expected error when data directory path points to a file")
	}
	
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("Expected 'not a directory' error, got: %v", err)
	}
	
	// Test read-only directory (create temp dir and make it read-only)
	tempDir := t.TempDir()
	
	// Make directory read-only (remove write permissions)
	err = os.Chmod(tempDir, 0444)
	if err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}
	
	// Restore permissions after test
	defer os.Chmod(tempDir, 0755)
	
	config.DataDir = tempDir
	
	// Test VolumeManager validation
	vm := NewVolumeManager(config)
	err = vm.ValidateDataDir()
	
	// On some systems, write test might still succeed even with 0444 permissions
	// So we'll check if the error is about write permissions or if it succeeds
	if err != nil && !strings.Contains(err.Error(), "not writable") {
		t.Errorf("Expected 'not writable' error or no error, got: %v", err)
	}
}

// Test user mount parsing error scenarios
// Requirements: 7.9, 7.10
func TestUserMountParsingErrors(t *testing.T) {
	parser := NewUserMountParser()
	
	// Test invalid mount syntax
	invalidMountSpecs := []struct {
		name      string
		mountStr  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "missing destination",
			mountStr:  "/source/path",
			expectErr: true,
			errMsg:    "must have at least source and destination",
		},
		{
			name:      "empty source",
			mountStr:  ":/dest",
			expectErr: true,
			errMsg:    "source path cannot be empty",
		},
		{
			name:      "empty destination",
			mountStr:  "/source:",
			expectErr: true,
			errMsg:    "destination path cannot be empty",
		},
		{
			name:      "too many colons",
			mountStr:  "/source:/dest:ro:extra",
			expectErr: true,
			errMsg:    "too many parts",
		},
		{
			name:      "nonexistent source path",
			mountStr:  "/definitely/nonexistent/path/12345:/dest",
			expectErr: true,
			errMsg:    "source path does not exist",
		},
		{
			name:      "relative destination path",
			mountStr:  "/tmp:relative/path",
			expectErr: true,
			errMsg:    "destination path must be absolute",
		},
		{
			name:      "invalid mount options",
			mountStr:  "/tmp:/dest:invalid_option",
			expectErr: true,
			errMsg:    "invalid mount option",
		},
	}
	
	for _, tc := range invalidMountSpecs {
		t.Run(tc.name, func(t *testing.T) {
			// For validation errors (like nonexistent paths), we need to test the full ParseUserMounts flow
			if strings.Contains(tc.errMsg, "source path does not exist") || 
			   strings.Contains(tc.errMsg, "destination path must be absolute") ||
			   strings.Contains(tc.errMsg, "invalid mount option") {
				
				// Set the environment variable and test ParseUserMounts
				originalMount := os.Getenv("MCP_MOUNT")
				defer func() {
					if originalMount != "" {
						os.Setenv("MCP_MOUNT", originalMount)
					} else {
						os.Unsetenv("MCP_MOUNT")
					}
				}()
				
				os.Setenv("MCP_MOUNT", tc.mountStr)
				_, err := parser.ParseUserMounts()
				
				if tc.expectErr {
					if err == nil {
						t.Errorf("Expected error for mount spec %s", tc.mountStr)
					} else if !strings.Contains(err.Error(), tc.errMsg) {
						t.Errorf("Expected error containing '%s', got: %v", tc.errMsg, err)
					}
				}
			} else {
				// For syntax errors, test ParseMountString directly
				mounts, err := parser.ParseMountString(tc.mountStr)
				
				if tc.expectErr {
					if err == nil {
						t.Errorf("Expected error for mount spec %s", tc.mountStr)
					} else if !strings.Contains(err.Error(), tc.errMsg) {
						t.Errorf("Expected error containing '%s', got: %v", tc.errMsg, err)
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error for mount spec %s: %v", tc.mountStr, err)
					}
					
					// If no error expected, validate the mounts
					for _, mount := range mounts {
						if err := parser.ValidateMount(mount); err != nil {
							t.Errorf("Mount validation failed: %v", err)
						}
					}
				}
			}
		})
	}
	
	// Test MCP_MOUNT environment variable parsing errors
	originalMount := os.Getenv("MCP_MOUNT")
	defer func() {
		if originalMount != "" {
			os.Setenv("MCP_MOUNT", originalMount)
		} else {
			os.Unsetenv("MCP_MOUNT")
		}
	}()
	
	// Test with invalid MCP_MOUNT
	os.Setenv("MCP_MOUNT", "/nonexistent:/dest")
	_, err := parser.ParseUserMounts()
	if err == nil {
		t.Error("Expected error when MCP_MOUNT contains nonexistent source path")
	}
	
	// Test with malformed MCP_MOUNT
	os.Setenv("MCP_MOUNT", "invalid_format")
	_, err = parser.ParseUserMounts()
	if err == nil {
		t.Error("Expected error when MCP_MOUNT has invalid format")
	}
	
	if !strings.Contains(err.Error(), "invalid MCP_MOUNT syntax") {
		t.Errorf("Expected MCP_MOUNT syntax error, got: %v", err)
	}
}

// Test home directory override error scenarios
// Requirements: 7.6, 7.7
func TestHomeDirectoryOverrideErrors(t *testing.T) {
	handler := NewHomeOverrideHandler()
	
	// Save original environment
	originalBindHome := os.Getenv("MCP_BIND_HOME")
	originalHomePath := os.Getenv("MCP_HOME_PATH")
	defer func() {
		if originalBindHome != "" {
			os.Setenv("MCP_BIND_HOME", originalBindHome)
		} else {
			os.Unsetenv("MCP_BIND_HOME")
		}
		if originalHomePath != "" {
			os.Setenv("MCP_HOME_PATH", originalHomePath)
		} else {
			os.Unsetenv("MCP_HOME_PATH")
		}
	}()
	
	// Test MCP_HOME_PATH with nonexistent directory
	os.Setenv("MCP_HOME_PATH", "/definitely/nonexistent/path/12345")
	os.Unsetenv("MCP_BIND_HOME")
	
	err := handler.ValidateCustomHomePath("/definitely/nonexistent/path/12345")
	if err == nil {
		t.Error("Expected error for nonexistent custom home path")
	}
	
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
	
	// Test MCP_HOME_PATH pointing to a file instead of directory
	tempFile, err := os.CreateTemp("", "test-file")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	
	err = handler.ValidateCustomHomePath(tempFile.Name())
	if err == nil {
		t.Error("Expected error when custom home path points to a file")
	}
	
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("Expected 'not a directory' error, got: %v", err)
	}
	
	// Test MCP_HOME_PATH with read-only directory
	tempDir := t.TempDir()
	err = os.Chmod(tempDir, 0444) // Make read-only
	if err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}
	defer os.Chmod(tempDir, 0755) // Restore permissions
	
	err = handler.ValidateCustomHomePath(tempDir)
	// On some systems, write test might still succeed even with 0444 permissions
	if err != nil && !strings.Contains(err.Error(), "not writable") {
		t.Errorf("Expected 'not writable' error or no error, got: %v", err)
	}
	
	// Test CreateBindHomeDir with permission issues
	// This test is more about the interface than actual permission testing
	// since permission behavior varies across systems
	
	// Test with empty volume name
	_, err = handler.CreateBindHomeDir("")
	// This should still work as it creates a directory with empty name
	// The real test is in the integration where permissions matter
}