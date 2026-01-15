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

// NewEnvFilter creates a new environment filter with secure defaults (no hardcoded allowlists)
func NewEnvFilter() *EnvFilter {
	return &EnvFilter{
		allowedPrefixes: []string{},            // Empty by default - secure by default
		allowedExact:    make(map[string]bool), // Empty by default - secure by default
	}
}

// GetFilteredEnvArgs returns Docker environment arguments for allowed variables
func (ef *EnvFilter) GetFilteredEnvArgs() []string {
	var args []string
	seen := make(map[string]bool)

	// Parse MCP_PASSTHROUGH_ENV for exact vars and patterns
	exactVars, patterns := ef.getCustomPassthroughVars()

	// Add exact variables to allowlist
	for _, v := range exactVars {
		ef.allowedExact[v] = true
	}

	// Store patterns for prefix matching
	ef.allowedPrefixes = patterns

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
	// Exclude run-mcp configuration variables (Requirement 9.4)
	// These should be consumed by run-mcp and not passed to the container
	if IsConfigurationEnvVar(key) {
		return false
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

// getCustomPassthroughVars parses MCP_PASSTHROUGH_ENV for wildcard patterns and exact variables
func (ef *EnvFilter) getCustomPassthroughVars() ([]string, []string) {
	var exactVars []string
	var patterns []string

	if extra := os.Getenv(MCPPassthroughEnv); extra != "" {
		for _, v := range strings.Split(extra, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				if strings.HasSuffix(v, "*") {
					// Wildcard pattern - remove the * suffix
					patterns = append(patterns, strings.TrimSuffix(v, "*"))
				} else {
					// Exact match
					exactVars = append(exactVars, v)
				}
			}
		}
	}

	return exactVars, patterns
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

	// Parse MCP_PASSTHROUGH_ENV for exact vars and patterns
	exactVars, patterns := ef.getCustomPassthroughVars()

	// Store original state
	originalExact := make(map[string]bool)
	for k, v := range ef.allowedExact {
		originalExact[k] = v
	}
	originalPrefixes := append([]string{}, ef.allowedPrefixes...)

	// Add exact variables to allowlist temporarily
	for _, v := range exactVars {
		ef.allowedExact[v] = true
	}

	// Store patterns for prefix matching temporarily
	ef.allowedPrefixes = patterns

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

	// Restore original state
	ef.allowedExact = originalExact
	ef.allowedPrefixes = originalPrefixes

	return count
}
