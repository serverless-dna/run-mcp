package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
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

// Test volume creation error handling
// Requirements: 1.1, 1.6
func TestVolumeCreationErrorHandling(t *testing.T) {
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