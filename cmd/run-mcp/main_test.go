package main

import (
	"os"
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