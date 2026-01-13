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
			Name   string `json:"Name"`
			Labels string `json:"Labels"`
			Size   string `json:"Size"`
		}

		if err := json.Unmarshal([]byte(line), &dockerVolume); err != nil {
			continue // Skip malformed entries
		}

		// Parse labels string into map
		labels := parseLabelsString(dockerVolume.Labels)

		// Use current time as CreatedAt since Docker volume ls doesn't provide it
		createdAt := time.Now()

		volumes = append(volumes, VolumeInfo{
			Name:      dockerVolume.Name,
			Labels:    labels,
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
			Name      string            `json:"Name"`
			Labels    map[string]string `json:"Labels"`
			CreatedAt string            `json:"CreatedAt"`
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
			Name      string            `json:"Name"`
			Labels    map[string]string `json:"Labels"`
			CreatedAt string            `json:"CreatedAt"`
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
			Name      string            `json:"Name"`
			Labels    map[string]string `json:"Labels"`
			CreatedAt string            `json:"CreatedAt"`
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
			Name      string            `json:"Name"`
			Labels    map[string]string `json:"Labels"`
			CreatedAt string            `json:"CreatedAt"`
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

	// Data directory mount (only when explicitly configured)
	dataMount := vm.getDataMount()
	if dataMount != "" {
		mounts = append(mounts, "-v", dataMount)
	}

	// No automatic credential mounts - removed for security

	return mounts
}

// getDataMount returns the data directory mount specification
func (vm *VolumeManager) getDataMount() string {
	dataDir := vm.config.DataDir

	// Only mount if MCP_DATA_DIR is explicitly set (not defaulted to home)
	if dataDir == "" {
		return ""
	}

	// Check if this is the default home directory (which means it wasn't explicitly set)
	if vm.isDefaultHomeDir(dataDir) {
		return ""
	}

	// Normalize path for cross-platform compatibility
	dataDir = vm.normalizePath(dataDir)

	return fmt.Sprintf("%s:/data", dataDir)
}

// isDefaultHomeDir checks if the data directory is the default home directory
func (vm *VolumeManager) isDefaultHomeDir(dataDir string) bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	// Normalize both paths for comparison
	normalizedDataDir := vm.normalizePath(dataDir)
	normalizedHomeDir := vm.normalizePath(homeDir)

	return normalizedDataDir == normalizedHomeDir
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
		DataMount: vm.getDataMount(),
	}

	// No credential mount detection - removed for security

	return info
}

// MountInfo contains information about volume mounts
type MountInfo struct {
	DataMount string
	// CredentialMounts field removed for security
}

// ValidateDataDir validates that the data directory is accessible
func (vm *VolumeManager) ValidateDataDir() error {
	dataDir := vm.config.DataDir

	// Only validate if data directory is explicitly set
	if dataDir == "" {
		return nil // No data directory to validate
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

// Mount represents a user-specified mount configuration
// Requirements: 7.1, 7.2, 7.3, 7.4
type Mount struct {
	Source      string
	Destination string
	Options     string
}

// UserMountParser handles parsing and validation of MCP_MOUNT environment variable
// Requirements: 7.1, 7.2, 7.3, 7.4, 7.9, 7.10
type UserMountParser struct{}

// NewUserMountParser creates a new user mount parser
func NewUserMountParser() *UserMountParser {
	return &UserMountParser{}
}

// ParseMountString parses the MCP_MOUNT environment variable into Mount structs
// Format: <src>:<dest>[:<opts>],<src>:<dest>[:<opts>],...
// Requirements: 7.1, 7.2, 7.9, 7.10
func (ump *UserMountParser) ParseMountString(mountStr string) ([]Mount, error) {
	if mountStr == "" {
		return []Mount{}, nil
	}

	var mounts []Mount

	// Split by comma to get individual mount specifications
	mountSpecs := strings.Split(mountStr, ",")

	for _, spec := range mountSpecs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}

		mount, err := ump.parseSingleMount(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid MCP_MOUNT syntax: %s\n\nExpected format: <src>:<dest>[:<opts>],<src>:<dest>[:<opts>],...\nExample: MCP_MOUNT=~/.aws:/home/mcp/.aws:ro,~/data:/data\nError: %w", spec, err)
		}

		mounts = append(mounts, mount)
	}

	return mounts, nil
}

// parseSingleMount parses a single mount specification
// Requirements: 7.1, 7.2, 7.3, 7.4
func (ump *UserMountParser) parseSingleMount(spec string) (Mount, error) {
	// Split by colon, but be careful about Windows paths (C:\path)
	parts := ump.splitMountSpec(spec)

	if len(parts) < 2 {
		return Mount{}, fmt.Errorf("mount specification must have at least source and destination: %s", spec)
	}

	if len(parts) > 3 {
		return Mount{}, fmt.Errorf("mount specification has too many parts: %s", spec)
	}

	source := strings.TrimSpace(parts[0])
	destination := strings.TrimSpace(parts[1])
	options := ""

	if len(parts) == 3 {
		options = strings.TrimSpace(parts[2])
		// Check if options contain additional colons (indicating too many parts)
		if strings.Contains(options, ":") {
			return Mount{}, fmt.Errorf("mount specification has too many parts: %s", spec)
		}
	}

	if source == "" {
		return Mount{}, fmt.Errorf("source path cannot be empty")
	}

	if destination == "" {
		return Mount{}, fmt.Errorf("destination path cannot be empty")
	}

	// Expand tilde in source path (Requirement 7.3)
	expandedSource := ump.ExpandTildePath(source)

	// Validate the original source path BEFORE conversion (Requirement 7.9)
	if _, err := os.Stat(expandedSource); os.IsNotExist(err) {
		return Mount{}, fmt.Errorf("mount source path does not exist: %s", expandedSource)
	}

	// Convert Windows paths (Requirement 7.4)
	normalizedSource := ump.ConvertWindowsPath(expandedSource)

	return Mount{
		Source:      normalizedSource,
		Destination: destination,
		Options:     options,
	}, nil
}

// splitMountSpec splits a mount specification by colons, handling Windows paths
// Requirements: 7.4
func (ump *UserMountParser) splitMountSpec(spec string) []string {
	// Handle Windows absolute paths (C:\path or C:/path)
	if runtime.GOOS == "windows" && len(spec) >= 2 && spec[1] == ':' {
		// This is a Windows absolute path, find the next colon
		colonIndex := strings.Index(spec[2:], ":")
		if colonIndex == -1 {
			// No destination specified
			return []string{spec}
		}

		// Adjust index to account for the offset
		colonIndex += 2

		source := spec[:colonIndex]
		remainder := spec[colonIndex+1:]

		// Split the remainder normally
		parts := strings.SplitN(remainder, ":", 2)
		result := []string{source}
		result = append(result, parts...)

		return result
	}

	// Normal case: split by colon
	return strings.SplitN(spec, ":", 3)
}

// ExpandTildePath expands ~ to the user's home directory
// Requirements: 7.3
func (ump *UserMountParser) ExpandTildePath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home directory, return path unchanged
		return path
	}

	if path == "~" {
		return homeDir
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}

	// Handle ~user syntax (not supported, return unchanged)
	return path
}

// ConvertWindowsPath converts Windows paths for cross-platform compatibility
// Requirements: 7.4
func (ump *UserMountParser) ConvertWindowsPath(path string) string {
	if runtime.GOOS != "windows" {
		return path
	}

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

// ValidateMount validates a mount configuration
// Requirements: 7.9, 7.10
func (ump *UserMountParser) ValidateMount(mount Mount) error {
	// Note: Source path validation is now done in parseSingleMount before path conversion

	// Validate destination path format
	if !strings.HasPrefix(mount.Destination, "/") {
		return fmt.Errorf("destination path must be absolute: %s", mount.Destination)
	}

	// Validate options if specified
	if mount.Options != "" {
		validOptions := map[string]bool{
			"ro":      true,
			"rw":      true,
			"bind":    true,
			"rbind":   true,
			"shared":  true,
			"slave":   true,
			"private": true,
		}

		options := strings.Split(mount.Options, ",")
		for _, opt := range options {
			opt = strings.TrimSpace(opt)
			if opt != "" && !validOptions[opt] {
				return fmt.Errorf("invalid mount option: %s", opt)
			}
		}
	}

	return nil
}

// ParseUserMounts parses the MCP_MOUNT environment variable and validates all mounts
// Requirements: 7.1, 7.2, 7.3, 7.4, 7.9, 7.10
func (ump *UserMountParser) ParseUserMounts() ([]Mount, error) {
	mountStr := os.Getenv("MCP_MOUNT")
	if mountStr == "" {
		return []Mount{}, nil
	}

	mounts, err := ump.ParseMountString(mountStr)
	if err != nil {
		return nil, err
	}

	// Validate all mounts
	for _, mount := range mounts {
		if err := ump.ValidateMount(mount); err != nil {
			return nil, err
		}
	}

	return mounts, nil
}

// GetMountArgs converts Mount structs to Docker mount arguments
// Requirements: 7.1, 7.2, 7.8
func (ump *UserMountParser) GetMountArgs(mounts []Mount) []string {
	var args []string

	for _, mount := range mounts {
		mountSpec := fmt.Sprintf("%s:%s", mount.Source, mount.Destination)
		if mount.Options != "" {
			mountSpec += ":" + mount.Options
		}

		args = append(args, "-v", mountSpec)
	}

	return args
}

// HomeOverrideHandler manages MCP_BIND_HOME and MCP_HOME_PATH overrides
// Requirements: 7.6, 7.7
type HomeOverrideHandler struct{}

// NewHomeOverrideHandler creates a new home override handler
func NewHomeOverrideHandler() *HomeOverrideHandler {
	return &HomeOverrideHandler{}
}

// GetHomeMount returns the home mount path based on environment variable overrides
// Returns empty string if no override is set (use container volume)
// Requirements: 7.6, 7.7
func (hoh *HomeOverrideHandler) GetHomeMount(args []string) string {
	// MCP_HOME_PATH takes precedence over MCP_BIND_HOME (Requirement 7.7)
	if homePath := os.Getenv("MCP_HOME_PATH"); homePath != "" {
		// Expand tilde in custom home path
		parser := NewUserMountParser()
		expandedPath := parser.ExpandTildePath(homePath)
		return expandedPath
	}

	// Check MCP_BIND_HOME (Requirement 7.6)
	if bindHome := os.Getenv("MCP_BIND_HOME"); hoh.isTruthy(bindHome) {
		// Use ~/.run-mcp/<volume-name>/ format
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}

		volumeName := sanitizeVolumeName(args)
		return filepath.Join(homeDir, ".run-mcp", volumeName)
	}

	// No override, use container volume
	return ""
}

// CreateBindHomeDir creates the bind home directory for MCP_BIND_HOME
// Requirements: 7.6
func (hoh *HomeOverrideHandler) CreateBindHomeDir(volumeName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot get home directory: %w", err)
	}

	bindPath := filepath.Join(homeDir, ".run-mcp", volumeName)

	// Create directory with proper permissions
	if err := os.MkdirAll(bindPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create bind home directory %s: %w", bindPath, err)
	}

	return bindPath, nil
}

// ValidateCustomHomePath validates a custom home path from MCP_HOME_PATH
// Requirements: 7.7
func (hoh *HomeOverrideHandler) ValidateCustomHomePath(path string) error {
	if path == "" {
		return fmt.Errorf("custom home path cannot be empty")
	}

	// Expand tilde
	parser := NewUserMountParser()
	expandedPath := parser.ExpandTildePath(path)

	// Check if path exists
	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("custom home path does not exist: %s", expandedPath)
		}
		return fmt.Errorf("cannot access custom home path: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("custom home path is not a directory: %s", expandedPath)
	}

	// Test write access
	testFile := filepath.Join(expandedPath, ".mcp-write-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("custom home path is not writable: %s", expandedPath)
	}

	// Clean up test file
	os.Remove(testFile)

	return nil
}

// isTruthy checks if a string value should be considered true
// Supports: "true", "1", "yes", "on" (case insensitive)
func (hoh *HomeOverrideHandler) isTruthy(value string) bool {
	if value == "" {
		return false
	}

	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
}

// Storage warning functionality for requirement 6.6

// checkVolumeStorageWarning checks if a volume exceeds the configured size limit and returns a warning message
// Requirements: 6.6
func checkVolumeStorageWarning(config *Config, volumeInfo VolumeInfo) string {
	if config.MaxVolumeSize == "" || volumeInfo.Size == "" {
		return ""
	}

	comparison, err := compareStorageSizes(volumeInfo.Size, config.MaxVolumeSize)
	if err != nil {
		// If we can't parse sizes, don't show warning
		return ""
	}

	if comparison > 0 {
		return fmt.Sprintf("Warning: Volume size (%s) exceeds configured limit (%s)", volumeInfo.Size, config.MaxVolumeSize)
	}

	return ""
}

// formatStorageWarningMessage formats a storage warning message with volume name
// Requirements: 6.6
func formatStorageWarningMessage(volumeName, volumeSize, sizeLimit string) string {
	return fmt.Sprintf("Warning: Volume '%s' size (%s) exceeds configured limit (%s)", volumeName, volumeSize, sizeLimit)
}

// parseLabelsString parses a comma-separated labels string into a map
// Format: "key1=value1,key2=value2,key3=value3"
func parseLabelsString(labelsStr string) map[string]string {
	labels := make(map[string]string)
	if labelsStr == "" {
		return labels
	}

	// Split by comma
	pairs := strings.Split(labelsStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split by first equals sign
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			labels[key] = value
		}
	}

	return labels
}

// compareStorageSizes compares two storage size strings
// Returns -1 if size1 < size2, 0 if equal, 1 if size1 > size2
// Requirements: 6.6
func compareStorageSizes(size1, size2 string) (int, error) {
	bytes1, err := parseStorageSize(size1)
	if err != nil {
		return 0, fmt.Errorf("invalid size format for '%s': %w", size1, err)
	}

	bytes2, err := parseStorageSize(size2)
	if err != nil {
		return 0, fmt.Errorf("invalid size format for '%s': %w", size2, err)
	}

	if bytes1 < bytes2 {
		return -1, nil
	} else if bytes1 > bytes2 {
		return 1, nil
	}
	return 0, nil
}

// parseStorageSize parses a storage size string (e.g., "100MB", "2GB") into bytes
// Requirements: 6.6
func parseStorageSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Remove all spaces and convert to uppercase
	sizeStr = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(sizeStr), " ", ""))

	// Define size multipliers
	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	// Find the unit suffix - check longer units first to avoid partial matches
	units := []string{"TB", "GB", "MB", "KB", "B"}
	var unit string
	var numberPart string

	for _, suffix := range units {
		if strings.HasSuffix(sizeStr, suffix) {
			unit = suffix
			numberPart = strings.TrimSuffix(sizeStr, suffix)
			break
		}
	}

	if unit == "" {
		return 0, fmt.Errorf("no valid unit found (B, KB, MB, GB, TB)")
	}

	// Parse the numeric part
	var number float64
	var err error

	if strings.Contains(numberPart, ".") {
		number, err = parseFloat(numberPart)
	} else {
		var intNumber int64
		intNumber, err = parseInt(numberPart)
		number = float64(intNumber)
	}

	if err != nil {
		return 0, fmt.Errorf("invalid number format: %s", numberPart)
	}

	if number < 0 {
		return 0, fmt.Errorf("negative size not allowed")
	}

	bytes := int64(number * float64(multipliers[unit]))
	return bytes, nil
}

// parseInt parses an integer string
func parseInt(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	var result int64
	var sign int64 = 1

	if s[0] == '-' {
		sign = -1
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}

	for _, char := range s {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid character: %c", char)
		}
		result = result*10 + int64(char-'0')
	}

	return result * sign, nil
}

// parseFloat parses a float string
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	var result float64
	var sign float64 = 1
	var decimalPlaces int
	var afterDecimal bool

	if s[0] == '-' {
		sign = -1
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}

	for _, char := range s {
		if char == '.' {
			if afterDecimal {
				return 0, fmt.Errorf("multiple decimal points")
			}
			afterDecimal = true
			continue
		}

		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid character: %c", char)
		}

		digit := float64(char - '0')

		if afterDecimal {
			decimalPlaces++
			result += digit / pow10(decimalPlaces)
		} else {
			result = result*10 + digit
		}
	}

	return result * sign, nil
}

// pow10 returns 10^n
func pow10(n int) float64 {
	result := 1.0
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}

// PruneHomeVolumes removes all managed volumes with confirmation
// Requirements: 4.6, 4.7, 6.6
func (vm *VolumeManager) PruneHomeVolumes() error {
	if vm.commander == nil {
		return fmt.Errorf("volume commander not initialized")
	}

	// List all managed volumes
	volumes, err := vm.commander.ListVolumes()
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	if len(volumes) == 0 {
		return nil // No volumes to prune
	}

	// Filter volumes to only include home volumes (runtime-specific filtering)
	// Requirements: 4.6, 4.7
	var homeVolumes []VolumeInfo
	for _, vol := range volumes {
		// Check if this is a home volume (has mcp-home- prefix or run-mcp.type=home label)
		if strings.HasPrefix(vol.Name, "mcp-home-") || vol.Labels["run-mcp.type"] == "home" {
			// Additional runtime filtering - only include volumes for current runtime
			if vm.runtime == "" || vol.Labels["run-mcp.runtime"] == vm.runtime {
				homeVolumes = append(homeVolumes, vol)
			}
		}
	}

	if len(homeVolumes) == 0 {
		return nil // No home volumes to prune
	}

	// Check for storage warnings before pruning
	// Requirements: 6.6
	var warnings []string
	for _, vol := range homeVolumes {
		if warning := checkVolumeStorageWarning(vm.config, vol); warning != "" {
			warnings = append(warnings, warning)
		}
	}

	// Display storage warnings if any
	if len(warnings) > 0 {
		fmt.Println("Storage Warnings:")
		for _, warning := range warnings {
			fmt.Printf("  %s\n", warning)
		}
		fmt.Println()
	}

	// Remove all home volumes
	var errors []string
	for _, vol := range homeVolumes {
		if err := vm.commander.RemoveVolume(vol.Name); err != nil {
			errors = append(errors, fmt.Sprintf("failed to remove volume %s: %v", vol.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some volumes could not be removed:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// ListHomeVolumes lists all managed home volumes
// Requirements: 4.4, 4.5, 2.10
func (vm *VolumeManager) ListHomeVolumes() ([]VolumeInfo, error) {
	if vm.commander == nil {
		return nil, fmt.Errorf("volume commander not initialized")
	}

	// List all managed volumes
	volumes, err := vm.commander.ListVolumes()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	// Filter to only include home volumes for current runtime
	var homeVolumes []VolumeInfo
	for _, vol := range volumes {
		// Check if this is a home volume
		if strings.HasPrefix(vol.Name, "mcp-home-") || vol.Labels["run-mcp.type"] == "home" {
			// Runtime-specific filtering
			if vm.runtime == "" || vol.Labels["run-mcp.runtime"] == vm.runtime {
				homeVolumes = append(homeVolumes, vol)
			}
		}
	}

	return homeVolumes, nil
}

// RemoveHomeVolume removes a specific home volume by server name
// Requirements: 4.5, 4.9
func (vm *VolumeManager) RemoveHomeVolume(serverName string) error {
	if vm.commander == nil {
		return fmt.Errorf("volume commander not initialized")
	}

	// Generate volume name from server name
	volumeName := sanitizeVolumeName(strings.Fields(serverName))

	// Check if volume exists
	exists, err := vm.commander.VolumeExists(volumeName)
	if err != nil {
		return fmt.Errorf("failed to check if volume exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("volume for server '%s' not found (expected volume name: %s)", serverName, volumeName)
	}

	// Remove the volume
	if err := vm.commander.RemoveVolume(volumeName); err != nil {
		return fmt.Errorf("failed to remove volume %s: %w", volumeName, err)
	}

	return nil
}

// InspectHomeVolume inspects a specific home volume by server name
// Requirements: 4.8, 4.13
func (vm *VolumeManager) InspectHomeVolume(serverName string) (*VolumeDetails, error) {
	if vm.commander == nil {
		return nil, fmt.Errorf("volume commander not initialized")
	}

	// Generate volume name from server name
	volumeName := sanitizeVolumeName(strings.Fields(serverName))

	// Inspect the volume
	details, err := vm.commander.InspectVolume(volumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume %s: %w", volumeName, err)
	}

	return details, nil
}
