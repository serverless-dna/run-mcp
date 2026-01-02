package main

import (
	"fmt"
	"strings"
)

// LanguageDetector handles language detection from commands
type LanguageDetector struct {
	commandMap map[string]string
}

// NewLanguageDetector creates a new language detector with command mappings
func NewLanguageDetector() *LanguageDetector {
	return &LanguageDetector{
		commandMap: map[string]string{
			// Node.js commands
			"npx":     "nodejs",
			"node":    "nodejs",
			"yarn":    "nodejs",
			"tsx":     "nodejs",
			"npm":     "nodejs",
			
			// Python commands
			"uvx":     "python",
			"python":  "python",
			"python3": "python",
			"uv":      "python",
			"pip":     "python",
			"pip3":    "python",
		},
	}
}

// DetectFromArgs detects the language runtime from command arguments
func (ld *LanguageDetector) DetectFromArgs(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	// Check for explicit runtime specification (run-mcp python uvx ...)
	if len(args) >= 2 {
		firstArg := strings.ToLower(args[0])
		if firstArg == "python" || firstArg == "python3" {
			return "python", nil
		}
		if firstArg == "node" || firstArg == "nodejs" {
			return "nodejs", nil
		}
	}

	// Auto-detect from first command
	command := strings.ToLower(args[0])
	if lang, exists := ld.commandMap[command]; exists {
		return lang, nil
	}

	// Return error for unknown commands instead of defaulting
	return "", fmt.Errorf("unknown command '%s'. Supported commands: %s", 
		args[0], ld.getSupportedCommands())
}

// getSupportedCommands returns a formatted string of supported commands
func (ld *LanguageDetector) getSupportedCommands() string {
	var nodejs, python []string
	
	for cmd, lang := range ld.commandMap {
		switch lang {
		case "nodejs":
			nodejs = append(nodejs, cmd)
		case "python":
			python = append(python, cmd)
		}
	}
	
	return fmt.Sprintf("Node.js: %s; Python: %s", 
		strings.Join(nodejs, ", "), 
		strings.Join(python, ", "))
}

// GetSupportedLanguages returns the list of supported languages
func (ld *LanguageDetector) GetSupportedLanguages() []string {
	languages := make(map[string]bool)
	for _, lang := range ld.commandMap {
		languages[lang] = true
	}
	
	var result []string
	for lang := range languages {
		result = append(result, lang)
	}
	
	return result
}

// GetCommandsForLanguage returns all commands that map to a specific language
func (ld *LanguageDetector) GetCommandsForLanguage(language string) []string {
	var commands []string
	for cmd, lang := range ld.commandMap {
		if lang == language {
			commands = append(commands, cmd)
		}
	}
	return commands
}

// IsValidLanguage checks if a language is supported
func (ld *LanguageDetector) IsValidLanguage(language string) bool {
	for _, lang := range ld.commandMap {
		if lang == language {
			return true
		}
	}
	return false
}