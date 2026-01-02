package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// VolumeCommander interface abstracts container runtime volume operations
// Requirements: 4.11, 4.12, 2.9
type VolumeCommander interface {
	CreateVolume(name string, labels map[string]string) error
	ListVolumes() ([]VolumeInfo, error)
	RemoveVolume(name string) error
	InspectVolume(name string) (*VolumeDetails, error)
	VolumeExists(name string) (bool, error)
}

// VolumeInfo contains basic information about a volume
type VolumeInfo struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"created_at"`
	Size      string            `json:"size,omitempty"`
	Runtime   string            `json:"runtime"`
}

// VolumeDetails contains detailed information about a volume
type VolumeDetails struct {
	VolumeInfo
	MountPoint string            `json:"mount_point"`
	Options    map[string]string `json:"options"`
}

// DockerVolumeCommander implements VolumeCommander for Docker
type DockerVolumeCommander struct {
	runtime string
}

// NewDockerVolumeCommander creates a new Docker volume commander
func NewDockerVolumeCommander() *DockerVolumeCommander {
	return &DockerVolumeCommander{runtime: "docker"}
}

// CreateVolume creates a new Docker volume with labels
func (dvc *DockerVolumeCommander) CreateVolume(name string, labels map[string]string) error {
	args := []string{"volume", "create"}
	
	// Add labels
	for key, value := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}
	
	args = append(args, name)
	
	cmd := exec.Command(dvc.runtime, args...)
	return cmd.Run()
}

// ListVolumes lists all volumes with run-mcp labels
func (dvc *DockerVolumeCommander) ListVolumes() ([]VolumeInfo, error) {
	cmd := exec.Command(dvc.runtime, "volume", "ls", "--filter", "label=run-mcp=true", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	
	var volumes []VolumeInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var dockerVolume struct {
			Name       string            `json:"Name"`
			Labels     map[string]string `json:"Labels"`
			CreatedAt  string            `json:"CreatedAt"`
			Size       string            `json:"Size"`
		}
		
		if err := json.Unmarshal([]byte(line), &dockerVolume); err != nil {
			continue // Skip malformed entries
		}
		
		createdAt, _ := time.Parse(time.RFC3339, dockerVolume.CreatedAt)
		
		volumes = append(volumes, VolumeInfo{
			Name:      dockerVolume.Name,
			Labels:    dockerVolume.Labels,
			CreatedAt: createdAt,
			Size:      dockerVolume.Size,
			Runtime:   dvc.runtime,
		})
	}
	
	return volumes, nil
}

// RemoveVolume removes a Docker volume
func (dvc *DockerVolumeCommander) RemoveVolume(name string) error {
	cmd := exec.Command(dvc.runtime, "volume", "rm", name)
	return cmd.Run()
}

// InspectVolume inspects a Docker volume
func (dvc *DockerVolumeCommander) InspectVolume(name string) (*VolumeDetails, error) {
	cmd := exec.Command(dvc.runtime, "volume", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}
	
	var dockerVolumes []struct {
		Name       string            `json:"Name"`
		Labels     map[string]string `json:"Labels"`
		CreatedAt  string            `json:"CreatedAt"`
		Mountpoint string            `json:"Mountpoint"`
		Options    map[string]string `json:"Options"`
	}
	
	if err := json.Unmarshal(output, &dockerVolumes); err != nil {
		return nil, fmt.Errorf("failed to parse volume inspect output: %w", err)
	}
	
	if len(dockerVolumes) == 0 {
		return nil, fmt.Errorf("volume not found: %s", name)
	}
	
	vol := dockerVolumes[0]
	createdAt, _ := time.Parse(time.RFC3339, vol.CreatedAt)
	
	return &VolumeDetails{
		VolumeInfo: VolumeInfo{
			Name:      vol.Name,
			Labels:    vol.Labels,
			CreatedAt: createdAt,
			Runtime:   dvc.runtime,
		},
		MountPoint: vol.Mountpoint,
		Options:    vol.Options,
	}, nil
}

// VolumeExists checks if a Docker volume exists
func (dvc *DockerVolumeCommander) VolumeExists(name string) (bool, error) {
	cmd := exec.Command(dvc.runtime, "volume", "inspect", name)
	err := cmd.Run()
	return err == nil, nil
}

// PodmanVolumeCommander implements VolumeCommander for Podman
type PodmanVolumeCommander struct {
	runtime string
}

// NewPodmanVolumeCommander creates a new Podman volume commander
func NewPodmanVolumeCommander() *PodmanVolumeCommander {
	return &PodmanVolumeCommander{runtime: "podman"}
}

// CreateVolume creates a new Podman volume with labels
func (pvc *PodmanVolumeCommander) CreateVolume(name string, labels map[string]string) error {
	args := []string{"volume", "create"}
	
	// Add labels
	for key, value := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}
	
	args = append(args, name)
	
	cmd := exec.Command(pvc.runtime, args...)
	return cmd.Run()
}

// ListVolumes lists all volumes with run-mcp labels
func (pvc *PodmanVolumeCommander) ListVolumes() ([]VolumeInfo, error) {
	cmd := exec.Command(pvc.runtime, "volume", "ls", "--filter", "label=run-mcp=true", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	
	var volumes []VolumeInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var podmanVolume struct {
			Name       string            `json:"Name"`
			Labels     map[string]string `json:"Labels"`
			CreatedAt  string            `json:"CreatedAt"`
		}
		
		if err := json.Unmarshal([]byte(line), &podmanVolume); err != nil {
			continue // Skip malformed entries
		}
		
		createdAt, _ := time.Parse(time.RFC3339, podmanVolume.CreatedAt)
		
		volumes = append(volumes, VolumeInfo{
			Name:      podmanVolume.Name,
			Labels:    podmanVolume.Labels,
			CreatedAt: createdAt,
			Runtime:   pvc.runtime,
		})
	}
	
	return volumes, nil
}

// RemoveVolume removes a Podman volume
func (pvc *PodmanVolumeCommander) RemoveVolume(name string) error {
	cmd := exec.Command(pvc.runtime, "volume", "rm", name)
	return cmd.Run()
}

// InspectVolume inspects a Podman volume
func (pvc *PodmanVolumeCommander) InspectVolume(name string) (*VolumeDetails, error) {
	cmd := exec.Command(pvc.runtime, "volume", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}
	
	var podmanVolumes []struct {
		Name       string            `json:"Name"`
		Labels     map[string]string `json:"Labels"`
		CreatedAt  string            `json:"CreatedAt"`
		Mountpoint string            `json:"Mountpoint"`
		Options    map[string]string `json:"Options"`
	}
	
	if err := json.Unmarshal(output, &podmanVolumes); err != nil {
		return nil, fmt.Errorf("failed to parse volume inspect output: %w", err)
	}
	
	if len(podmanVolumes) == 0 {
		return nil, fmt.Errorf("volume not found: %s", name)
	}
	
	vol := podmanVolumes[0]
	createdAt, _ := time.Parse(time.RFC3339, vol.CreatedAt)
	
	return &VolumeDetails{
		VolumeInfo: VolumeInfo{
			Name:      vol.Name,
			Labels:    vol.Labels,
			CreatedAt: createdAt,
			Runtime:   pvc.runtime,
		},
		MountPoint: vol.Mountpoint,
		Options:    vol.Options,
	}, nil
}

// VolumeExists checks if a Podman volume exists
func (pvc *PodmanVolumeCommander) VolumeExists(name string) (bool, error) {
	cmd := exec.Command(pvc.runtime, "volume", "inspect", name)
	err := cmd.Run()
	return err == nil, nil
}

// NerdctlVolumeCommander implements VolumeCommander for Nerdctl
type NerdctlVolumeCommander struct {
	runtime string
}

// NewNerdctlVolumeCommander creates a new Nerdctl volume commander
func NewNerdctlVolumeCommander() *NerdctlVolumeCommander {
	return &NerdctlVolumeCommander{runtime: "nerdctl"}
}

// CreateVolume creates a new Nerdctl volume with labels
func (nvc *NerdctlVolumeCommander) CreateVolume(name string, labels map[string]string) error {
	args := []string{"volume", "create"}
	
	// Add labels
	for key, value := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}
	
	args = append(args, name)
	
	cmd := exec.Command(nvc.runtime, args...)
	return cmd.Run()
}

// ListVolumes lists all volumes with run-mcp labels
func (nvc *NerdctlVolumeCommander) ListVolumes() ([]VolumeInfo, error) {
	cmd := exec.Command(nvc.runtime, "volume", "ls", "--filter", "label=run-mcp=true", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	
	var volumes []VolumeInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var nerdctlVolume struct {
			Name       string            `json:"Name"`
			Labels     map[string]string `json:"Labels"`
			CreatedAt  string            `json:"CreatedAt"`
		}
		
		if err := json.Unmarshal([]byte(line), &nerdctlVolume); err != nil {
			continue // Skip malformed entries
		}
		
		createdAt, _ := time.Parse(time.RFC3339, nerdctlVolume.CreatedAt)
		
		volumes = append(volumes, VolumeInfo{
			Name:      nerdctlVolume.Name,
			Labels:    nerdctlVolume.Labels,
			CreatedAt: createdAt,
			Runtime:   nvc.runtime,
		})
	}
	
	return volumes, nil
}

// RemoveVolume removes a Nerdctl volume
func (nvc *NerdctlVolumeCommander) RemoveVolume(name string) error {
	cmd := exec.Command(nvc.runtime, "volume", "rm", name)
	return cmd.Run()
}

// InspectVolume inspects a Nerdctl volume
func (nvc *NerdctlVolumeCommander) InspectVolume(name string) (*VolumeDetails, error) {
	cmd := exec.Command(nvc.runtime, "volume", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}
	
	var nerdctlVolumes []struct {
		Name       string            `json:"Name"`
		Labels     map[string]string `json:"Labels"`
		CreatedAt  string            `json:"CreatedAt"`
		Mountpoint string            `json:"Mountpoint"`
		Options    map[string]string `json:"Options"`
	}
	
	if err := json.Unmarshal(output, &nerdctlVolumes); err != nil {
		return nil, fmt.Errorf("failed to parse volume inspect output: %w", err)
	}
	
	if len(nerdctlVolumes) == 0 {
		return nil, fmt.Errorf("volume not found: %s", name)
	}
	
	vol := nerdctlVolumes[0]
	createdAt, _ := time.Parse(time.RFC3339, vol.CreatedAt)
	
	return &VolumeDetails{
		VolumeInfo: VolumeInfo{
			Name:      vol.Name,
			Labels:    vol.Labels,
			CreatedAt: createdAt,
			Runtime:   nvc.runtime,
		},
		MountPoint: vol.Mountpoint,
		Options:    vol.Options,
	}, nil
}

// VolumeExists checks if a Nerdctl volume exists
func (nvc *NerdctlVolumeCommander) VolumeExists(name string) (bool, error) {
	cmd := exec.Command(nvc.runtime, "volume", "inspect", name)
	err := cmd.Run()
	return err == nil, nil
}

// FinchVolumeCommander implements VolumeCommander for Finch
type FinchVolumeCommander struct {
	runtime string
}

// NewFinchVolumeCommander creates a new Finch volume commander
func NewFinchVolumeCommander() *FinchVolumeCommander {
	return &FinchVolumeCommander{runtime: "finch"}
}

// CreateVolume creates a new Finch volume with labels
func (fvc *FinchVolumeCommander) CreateVolume(name string, labels map[string]string) error {
	args := []string{"volume", "create"}
	
	// Add labels
	for key, value := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}
	
	args = append(args, name)
	
	cmd := exec.Command(fvc.runtime, args...)
	return cmd.Run()
}

// ListVolumes lists all volumes with run-mcp labels
func (fvc *FinchVolumeCommander) ListVolumes() ([]VolumeInfo, error) {
	cmd := exec.Command(fvc.runtime, "volume", "ls", "--filter", "label=run-mcp=true", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	
	var volumes []VolumeInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var finchVolume struct {
			Name       string            `json:"Name"`
			Labels     map[string]string `json:"Labels"`
			CreatedAt  string            `json:"CreatedAt"`
		}
		
		if err := json.Unmarshal([]byte(line), &finchVolume); err != nil {
			continue // Skip malformed entries
		}
		
		createdAt, _ := time.Parse(time.RFC3339, finchVolume.CreatedAt)
		
		volumes = append(volumes, VolumeInfo{
			Name:      finchVolume.Name,
			Labels:    finchVolume.Labels,
			CreatedAt: createdAt,
			Runtime:   fvc.runtime,
		})
	}
	
	return volumes, nil
}

// RemoveVolume removes a Finch volume
func (fvc *FinchVolumeCommander) RemoveVolume(name string) error {
	cmd := exec.Command(fvc.runtime, "volume", "rm", name)
	return cmd.Run()
}

// InspectVolume inspects a Finch volume
func (fvc *FinchVolumeCommander) InspectVolume(name string) (*VolumeDetails, error) {
	cmd := exec.Command(fvc.runtime, "volume", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}
	
	var finchVolumes []struct {
		Name       string            `json:"Name"`
		Labels     map[string]string `json:"Labels"`
		CreatedAt  string            `json:"CreatedAt"`
		Mountpoint string            `json:"Mountpoint"`
		Options    map[string]string `json:"Options"`
	}
	
	if err := json.Unmarshal(output, &finchVolumes); err != nil {
		return nil, fmt.Errorf("failed to parse volume inspect output: %w", err)
	}
	
	if len(finchVolumes) == 0 {
		return nil, fmt.Errorf("volume not found: %s", name)
	}
	
	vol := finchVolumes[0]
	createdAt, _ := time.Parse(time.RFC3339, vol.CreatedAt)
	
	return &VolumeDetails{
		VolumeInfo: VolumeInfo{
			Name:      vol.Name,
			Labels:    vol.Labels,
			CreatedAt: createdAt,
			Runtime:   fvc.runtime,
		},
		MountPoint: vol.Mountpoint,
		Options:    vol.Options,
	}, nil
}

// VolumeExists checks if a Finch volume exists
func (fvc *FinchVolumeCommander) VolumeExists(name string) (bool, error) {
	cmd := exec.Command(fvc.runtime, "volume", "inspect", name)
	err := cmd.Run()
	return err == nil, nil
}

// LimaNerdctlVolumeCommander implements VolumeCommander for Lima Nerdctl
type LimaNerdctlVolumeCommander struct {
	runtime string
}

// NewLimaNerdctlVolumeCommander creates a new Lima Nerdctl volume commander
func NewLimaNerdctlVolumeCommander() *LimaNerdctlVolumeCommander {
	return &LimaNerdctlVolumeCommander{runtime: "lima nerdctl"}
}

// CreateVolume creates a new Lima Nerdctl volume with labels
func (lvc *LimaNerdctlVolumeCommander) CreateVolume(name string, labels map[string]string) error {
	args := []string{"nerdctl", "volume", "create"}
	
	// Add labels
	for key, value := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}
	
	args = append(args, name)
	
	cmd := exec.Command("lima", args...)
	return cmd.Run()
}

// ListVolumes lists all volumes with run-mcp labels
func (lvc *LimaNerdctlVolumeCommander) ListVolumes() ([]VolumeInfo, error) {
	cmd := exec.Command("lima", "nerdctl", "volume", "ls", "--filter", "label=run-mcp=true", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	
	var volumes []VolumeInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var limaVolume struct {
			Name       string            `json:"Name"`
			Labels     map[string]string `json:"Labels"`
			CreatedAt  string            `json:"CreatedAt"`
		}
		
		if err := json.Unmarshal([]byte(line), &limaVolume); err != nil {
			continue // Skip malformed entries
		}
		
		createdAt, _ := time.Parse(time.RFC3339, limaVolume.CreatedAt)
		
		volumes = append(volumes, VolumeInfo{
			Name:      limaVolume.Name,
			Labels:    limaVolume.Labels,
			CreatedAt: createdAt,
			Runtime:   lvc.runtime,
		})
	}
	
	return volumes, nil
}

// RemoveVolume removes a Lima Nerdctl volume
func (lvc *LimaNerdctlVolumeCommander) RemoveVolume(name string) error {
	cmd := exec.Command("lima", "nerdctl", "volume", "rm", name)
	return cmd.Run()
}

// InspectVolume inspects a Lima Nerdctl volume
func (lvc *LimaNerdctlVolumeCommander) InspectVolume(name string) (*VolumeDetails, error) {
	cmd := exec.Command("lima", "nerdctl", "volume", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}
	
	var limaVolumes []struct {
		Name       string            `json:"Name"`
		Labels     map[string]string `json:"Labels"`
		CreatedAt  string            `json:"CreatedAt"`
		Mountpoint string            `json:"Mountpoint"`
		Options    map[string]string `json:"Options"`
	}
	
	if err := json.Unmarshal(output, &limaVolumes); err != nil {
		return nil, fmt.Errorf("failed to parse volume inspect output: %w", err)
	}
	
	if len(limaVolumes) == 0 {
		return nil, fmt.Errorf("volume not found: %s", name)
	}
	
	vol := limaVolumes[0]
	createdAt, _ := time.Parse(time.RFC3339, vol.CreatedAt)
	
	return &VolumeDetails{
		VolumeInfo: VolumeInfo{
			Name:      vol.Name,
			Labels:    vol.Labels,
			CreatedAt: createdAt,
			Runtime:   lvc.runtime,
		},
		MountPoint: vol.Mountpoint,
		Options:    vol.Options,
	}, nil
}

// VolumeExists checks if a Lima Nerdctl volume exists
func (lvc *LimaNerdctlVolumeCommander) VolumeExists(name string) (bool, error) {
	cmd := exec.Command("lima", "nerdctl", "volume", "inspect", name)
	err := cmd.Run()
	return err == nil, nil
}

// NewVolumeCommander creates a VolumeCommander for the specified runtime
func NewVolumeCommander(runtime string) VolumeCommander {
	switch runtime {
	case "docker":
		return NewDockerVolumeCommander()
	case "podman":
		return NewPodmanVolumeCommander()
	case "nerdctl":
		return NewNerdctlVolumeCommander()
	case "finch":
		return NewFinchVolumeCommander()
	case "lima nerdctl":
		return NewLimaNerdctlVolumeCommander()
	default:
		// Default to Docker-compatible interface
		return &DockerVolumeCommander{runtime: runtime}
	}
}

// VolumeManager handles cross-platform volume mounting and home directory isolation
type VolumeManager struct {
	config    *Config
	commander VolumeCommander
	runtime   string
}

// NewVolumeManager creates a new volume manager
func NewVolumeManager(config *Config) *VolumeManager {
	return &VolumeManager{config: config}
}

// NewVolumeManagerWithRuntime creates a new volume manager with a specific runtime
func NewVolumeManagerWithRuntime(config *Config, runtime string) *VolumeManager {
	return &VolumeManager{
		config:    config,
		commander: NewVolumeCommander(runtime),
		runtime:   runtime,
	}
}

// CreateHomeVolume creates a home directory volume for an MCP server
// Requirements: 1.1, 1.2, 1.6
func (vm *VolumeManager) CreateHomeVolume(serverName, runtime string) (string, error) {
	if vm.commander == nil {
		vm.commander = NewVolumeCommander(runtime)
		vm.runtime = runtime
	}
	
	volumeName := sanitizeVolumeName(strings.Fields(serverName))
	
	// Check if volume already exists
	exists, err := vm.commander.VolumeExists(volumeName)
	if err != nil {
		return "", fmt.Errorf("failed to check if volume exists: %w", err)
	}
	
	if exists {
		return volumeName, nil // Reuse existing volume (Requirement 1.2)
	}
	
	// Create volume with runtime-specific labels (Requirements 2.9, 4.11, 4.12)
	labels := map[string]string{
		"run-mcp":         "true",
		"run-mcp.runtime": runtime,
		"run-mcp.server":  serverName,
		"run-mcp.type":    "home",
	}
	
	if err := vm.commander.CreateVolume(volumeName, labels); err != nil {
		return "", fmt.Errorf("failed to create volume %s: %w", volumeName, err)
	}
	
	return volumeName, nil
}

// CreateEphemeralVolume creates a temporary volume that will be cleaned up on container exit
// Requirements: 6.3, 6.4, 6.5
func (vm *VolumeManager) CreateEphemeralVolume(serverName, runtime string) (string, error) {
	if vm.commander == nil {
		vm.commander = NewVolumeCommander(runtime)
		vm.runtime = runtime
	}
	
	volumeName := vm.CreateEphemeralVolumeName(serverName)
	
	// Create ephemeral volume with runtime-specific labels
	labels := map[string]string{
		"run-mcp":           "true",
		"run-mcp.ephemeral": "true",
		"run-mcp.runtime":   runtime,
		"run-mcp.server":    serverName,
		"run-mcp.type":      "ephemeral",
	}
	
	if err := vm.commander.CreateVolume(volumeName, labels); err != nil {
		return "", fmt.Errorf("failed to create ephemeral volume %s: %w", volumeName, err)
	}
	
	return volumeName, nil
}

// CreateEphemeralVolumeName generates a unique ephemeral volume name with timestamp
// Requirements: 6.4, 6.5
func (vm *VolumeManager) CreateEphemeralVolumeName(serverName string) string {
	sanitizedName := sanitizeVolumeName(strings.Fields(serverName))
	// Remove the "mcp-home-" prefix and replace with "mcp-ephemeral-"
	if strings.HasPrefix(sanitizedName, "mcp-home-") {
		sanitizedName = strings.TrimPrefix(sanitizedName, "mcp-home-")
	}
	
	// Use nanosecond timestamp for better uniqueness
	timestamp := time.Now().UnixNano()
	name := fmt.Sprintf("mcp-ephemeral-%s-%d", sanitizedName, timestamp)
	
	// Apply same truncation logic for consistency
	if len(name) > 64 {
		// Keep first 47 characters plus "-" plus 8-character hash suffix plus "-" plus timestamp = 64 total
		hash := fmt.Sprintf("%08x", md5.Sum([]byte(name)))[:8]
		baseLength := 64 - len(fmt.Sprintf("-%s-%d", hash, timestamp))
		if baseLength < 1 {
			baseLength = 1
		}
		name = name[:baseLength] + "-" + hash + fmt.Sprintf("-%d", timestamp)
	}
	
	return name
}

// CleanupEphemeralVolume removes an ephemeral volume
// Requirements: 6.3, 6.4, 6.5
func (vm *VolumeManager) CleanupEphemeralVolume(volumeName string) error {
	if vm.commander == nil {
		return fmt.Errorf("volume commander not initialized")
	}
	
	// Verify this is actually an ephemeral volume before removing
	if !strings.HasPrefix(volumeName, "mcp-ephemeral-") {
		return fmt.Errorf("volume %s is not an ephemeral volume", volumeName)
	}
	
	return vm.commander.RemoveVolume(volumeName)
}

// sanitizeVolumeName creates a sanitized volume name from command arguments
// Following the pattern: mcp-home-{sanitized-command}-{sanitized-first-arg}
// Requirements: 2.1, 2.2, 2.3, 2.7, 2.8
func sanitizeVolumeName(args []string) string {
	if len(args) == 0 {
		return "mcp-home-default"
	}
	
	var parts []string
	
	// Extract up to 2 parts (command + first non-flag argument)
	for i, arg := range args {
		// Only use first two args (command + server identifier)
		if i >= 2 {
			break
		}
		// Stop at flags
		if strings.HasPrefix(arg, "-") {
			break
		}
		
		// Normalize path separators before processing (Requirement 2.7)
		normalizedArg := strings.ReplaceAll(arg, "\\", "/")
		
		// Sanitize: lowercase, replace non-alphanumeric with dash
		sanitized := strings.ToLower(normalizedArg)
		sanitized = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(sanitized, "-")
		sanitized = strings.Trim(sanitized, "-")
		
		if sanitized != "" {
			parts = append(parts, sanitized)
		}
	}
	
	if len(parts) == 0 {
		return "mcp-home-default"
	}
	
	name := "mcp-home-" + strings.Join(parts, "-")
	
	// Truncate if exceeds 64 characters (Requirement 2.8)
	if len(name) > 64 {
		// Keep first 55 characters plus "-" plus 8-character hash suffix = 64 total
		hash := fmt.Sprintf("%08x", md5.Sum([]byte(name)))[:8]
		name = name[:55] + "-" + hash
	}
	
	return name
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