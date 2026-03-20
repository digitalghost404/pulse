package collector

import (
	"context"
	"encoding/json"
	"os/exec"
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

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		snap, err := ParseDockerPSLine(line)
		if err != nil {
			continue
		}
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

func init() {
	Register(&DockerCollector{})
}
