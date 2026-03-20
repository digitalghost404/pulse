package collector

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type DockerCollector struct{}

func (d *DockerCollector) Name() string      { return "docker" }
func (d *DockerCollector) EnvVars() []string { return nil }

func (d *DockerCollector) Enabled(cfg *config.Config) bool {
	return cfg.AdapterEnabled("docker")
}

func (d *DockerCollector) Collect(ctx context.Context, s store.Store, cfg *config.Config, syncID int64) error {
	// Check if docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		return nil // docker not installed, skip silently
	}

	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", `{"Names":"{{.Names}}","Image":"{{.Image}}","Status":"{{.Status}}","Ports":"{{.Ports}}"}`)
	out, err := cmd.Output()
	if err != nil {
		return nil // docker not running, skip
	}

	// Parse container list
	snapshots := make(map[string]domain.DockerSnapshot)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		snap, err := ParseDockerPSLine(line)
		if err != nil {
			continue
		}
		snapshots[snap.ContainerName] = snap
	}

	// Collect CPU/memory stats for running containers
	statsCmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", `{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}`)
	statsOut, err := statsCmd.Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(statsOut)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) < 3 {
				continue
			}
			name := parts[0]
			if snap, ok := snapshots[name]; ok {
				snap.CPUPct = ParsePercent(parts[1])
				snap.MemoryMB = ParseMemoryMB(parts[2])
				snapshots[name] = snap
			}
		}
	}

	for _, snap := range snapshots {
		if err := s.SaveDockerSnapshot(ctx, syncID, snap); err != nil {
			return err
		}
	}

	return nil
}

type dockerPSOutput struct {
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
}

// ParseDockerPSLine parses a single JSON line from docker ps output.
func ParseDockerPSLine(line string) (domain.DockerSnapshot, error) {
	var ps dockerPSOutput
	if err := json.Unmarshal([]byte(line), &ps); err != nil {
		return domain.DockerSnapshot{}, err
	}
	return domain.DockerSnapshot{
		ContainerName: ps.Names,
		Image:         ps.Image,
		Status:        ps.Status,
		Ports:         ps.Ports,
	}, nil
}

// parsePercent parses "12.34%" to 12.34
func ParsePercent(s string) float64 {
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseMemoryMB parses "123.4MiB / 1.5GiB" to 123.4
func ParseMemoryMB(s string) float64 {
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return 0
	}
	used := strings.TrimSpace(parts[0])
	if strings.HasSuffix(used, "GiB") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(used, "GiB"), 64)
		return v * 1024
	}
	if strings.HasSuffix(used, "MiB") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(used, "MiB"), 64)
		return v
	}
	if strings.HasSuffix(used, "KiB") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(used, "KiB"), 64)
		return v / 1024
	}
	return 0
}

func init() {
	Register(&DockerCollector{})
}
