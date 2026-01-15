package main

import (
	"os"
	"strings"
)

// EnvFilter handles secure environment variable filtering
type EnvFilter struct {
	allowedPrefixes []string
	allowedExact    map[string]bool
}

// NewEnvFilter creates a new environment filter with secure defaults
func NewEnvFilter() *EnvFilter {
	return &EnvFilter{
		allowedPrefixes: []string{
			"AWS_",
			"OPENAI_",
			"ANTHROPIC_",
			"AZURE_",
			"GOOGLE_",
			"MCP_",
			"HF_",
			"REPLICATE_",
			"COHERE_",
		},
		allowedExact: map[string]bool{
			"GITHUB_TOKEN": true,
			"GITLAB_TOKEN": true,
			"DATABASE_URL": true,
			"REDIS_URL":    true,
		},
	}
}

// GetFilteredEnvArgs returns Docker environment arguments for allowed variables
func (ef *EnvFilter) GetFilteredEnvArgs() []string {
	var args []string
	seen := make(map[string]bool)
	
	// Add user-specified additional vars from MCP_PASSTHROUGH_ENV
	customVars := ef.getCustomPassthroughVars()
	for _, v := range customVars {
		ef.allowedExact[v] = true
	}
	
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		
		// Skip if already processed
		if seen[key] {
			continue
		}
		
		// Check if variable should be passed through
		if ef.shouldPassthrough(key) {
			args = append(args, "-e", env)
			seen[key] = true
		}
	}
	
	return args
}

// shouldPassthrough determines if an environment variable should be passed to the container
func (ef *EnvFilter) shouldPassthrough(key string) bool {
	// Exclude run-mcp configuration variables (Requirement 3.5)
	// These should be consumed by run-mcp and not passed to the container
	configVars := []string{
		"MCP_MOUNT",
		"MCP_BIND_HOME", 
		"MCP_HOME_PATH",
	}
	
	for _, configVar := range configVars {
		if key == configVar {
			return false
		}
	}
	
	// Check exact match first
	if ef.allowedExact[key] {
		return true
	}
	
	// Check prefix match
	for _, prefix := range ef.allowedPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	
	return false
}

// getCustomPassthroughVars parses MCP_PASSTHROUGH_ENV for additional variables
func (ef *EnvFilter) getCustomPassthroughVars() []string {
	var vars []string
	
	if extra := os.Getenv("MCP_PASSTHROUGH_ENV"); extra != "" {
		for _, v := range strings.Split(extra, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				vars = append(vars, v)
			}
		}
	}
	
	return vars
}

// GetAllowedPrefixes returns the list of allowed environment variable prefixes
func (ef *EnvFilter) GetAllowedPrefixes() []string {
	return append([]string{}, ef.allowedPrefixes...)
}

// GetAllowedExact returns the list of allowed exact environment variable names
func (ef *EnvFilter) GetAllowedExact() []string {
	var exact []string
	for key := range ef.allowedExact {
		exact = append(exact, key)
	}
	return exact
}

// AddAllowedPrefix adds a new allowed prefix (for testing or extension)
func (ef *EnvFilter) AddAllowedPrefix(prefix string) {
	ef.allowedPrefixes = append(ef.allowedPrefixes, prefix)
}

// AddAllowedExact adds a new allowed exact variable name (for testing or extension)
func (ef *EnvFilter) AddAllowedExact(name string) {
	ef.allowedExact[name] = true
}

// GetFilteredEnvCount returns the number of environment variables that would be passed through
func (ef *EnvFilter) GetFilteredEnvCount() int {
	count := 0
	seen := make(map[string]bool)
	
	// Add custom vars to exact matches temporarily
	customVars := ef.getCustomPassthroughVars()
	originalExact := make(map[string]bool)
	for k, v := range ef.allowedExact {
		originalExact[k] = v
	}
	
	for _, v := range customVars {
		ef.allowedExact[v] = true
	}
	
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		
		if !seen[key] && ef.shouldPassthrough(key) {
			count++
			seen[key] = true
		}
	}
	
	// Restore original exact matches
	ef.allowedExact = originalExact
	
	return count
}