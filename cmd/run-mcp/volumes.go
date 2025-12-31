package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// VolumeManager handles cross-platform volume mounting
type VolumeManager struct {
	config *Config
}

// NewVolumeManager creates a new volume manager
func NewVolumeManager(config *Config) *VolumeManager {
	return &VolumeManager{config: config}
}

// GetVolumeMounts returns volume mount arguments for the container
func (vm *VolumeManager) GetVolumeMounts() []string {
	var mounts []string
	
	// Data directory mount (primary mount point)
	dataMount := vm.getDataMount()
	if dataMount != "" {
		mounts = append(mounts, "-v", dataMount)
	}
	
	// Credential directory mounts (read-only)
	credMounts := vm.getCredentialMounts()
	mounts = append(mounts, credMounts...)
	
	return mounts
}

// getDataMount returns the data directory mount specification
func (vm *VolumeManager) getDataMount() string {
	dataDir := vm.config.DataDir
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dataDir = homeDir
	}
	
	// Normalize path for cross-platform compatibility
	dataDir = vm.normalizePath(dataDir)
	
	return fmt.Sprintf("%s:/data", dataDir)
}

// getCredentialMounts returns credential directory mounts
func (vm *VolumeManager) getCredentialMounts() []string {
	var mounts []string
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return mounts
	}
	
	// AWS credentials
	awsDir := filepath.Join(homeDir, ".aws")
	if vm.dirExists(awsDir) {
		awsMount := fmt.Sprintf("%s:/home/mcp/.aws:ro", vm.normalizePath(awsDir))
		mounts = append(mounts, "-v", awsMount)
	}
	
	// General config directory
	configDir := filepath.Join(homeDir, ".config")
	if vm.dirExists(configDir) {
		configMount := fmt.Sprintf("%s:/home/mcp/.config:ro", vm.normalizePath(configDir))
		mounts = append(mounts, "-v", configMount)
	}
	
	// Platform-specific credential directories
	platformMounts := vm.getPlatformSpecificMounts(homeDir)
	mounts = append(mounts, platformMounts...)
	
	return mounts
}

// getPlatformSpecificMounts returns platform-specific credential mounts
func (vm *VolumeManager) getPlatformSpecificMounts(homeDir string) []string {
	var mounts []string
	
	switch runtime.GOOS {
	case "darwin":
		// macOS Keychain access (if needed)
		keychainDir := filepath.Join(homeDir, "Library", "Keychains")
		if vm.dirExists(keychainDir) {
			keychainMount := fmt.Sprintf("%s:/home/mcp/Library/Keychains:ro", vm.normalizePath(keychainDir))
			mounts = append(mounts, "-v", keychainMount)
		}
		
	case "windows":
		// Windows credential store (if accessible)
		// Note: This might not work in all Windows container scenarios
		
	case "linux":
		// Linux-specific credential directories
		// SSH keys
		sshDir := filepath.Join(homeDir, ".ssh")
		if vm.dirExists(sshDir) {
			sshMount := fmt.Sprintf("%s:/home/mcp/.ssh:ro", vm.normalizePath(sshDir))
			mounts = append(mounts, "-v", sshMount)
		}
	}
	
	return mounts
}

// normalizePath normalizes file paths for cross-platform compatibility
func (vm *VolumeManager) normalizePath(path string) string {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	
	// On Windows, convert backslashes to forward slashes for Docker
	if runtime.GOOS == "windows" {
		absPath = strings.ReplaceAll(absPath, "\\", "/")
		
		// Handle Windows drive letters (C: -> /c)
		if len(absPath) >= 2 && absPath[1] == ':' {
			drive := strings.ToLower(string(absPath[0]))
			absPath = "/" + drive + absPath[2:]
		}
	}
	
	return absPath
}

// dirExists checks if a directory exists and is accessible
func (vm *VolumeManager) dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetMountInfo returns information about what will be mounted
func (vm *VolumeManager) GetMountInfo() MountInfo {
	info := MountInfo{
		DataMount:       vm.getDataMount(),
		CredentialMounts: make(map[string]string),
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return info
	}
	
	// Check each potential credential directory
	credDirs := map[string]string{
		"aws":    filepath.Join(homeDir, ".aws"),
		"config": filepath.Join(homeDir, ".config"),
		"ssh":    filepath.Join(homeDir, ".ssh"),
	}
	
	for name, dir := range credDirs {
		if vm.dirExists(dir) {
			info.CredentialMounts[name] = vm.normalizePath(dir)
		}
	}
	
	return info
}

// MountInfo contains information about volume mounts
type MountInfo struct {
	DataMount        string
	CredentialMounts map[string]string
}

// ValidateDataDir validates that the data directory is accessible
func (vm *VolumeManager) ValidateDataDir() error {
	dataDir := vm.config.DataDir
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		dataDir = homeDir
	}
	
	if !vm.dirExists(dataDir) {
		return fmt.Errorf("data directory does not exist or is not accessible: %s", dataDir)
	}
	
	// Test write access
	testFile := filepath.Join(dataDir, ".mcp-test-write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("data directory is not writable: %s", dataDir)
	}
	
	// Clean up test file
	os.Remove(testFile)
	
	return nil
}