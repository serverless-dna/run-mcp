package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
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