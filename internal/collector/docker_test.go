package collector_test

import (
	"testing"

	"github.com/xcoleman/pulse/internal/collector"
	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
)

func TestDockerCollector_ParseDockerPS(t *testing.T) {
	// Table-driven: mock docker ps JSON output → expected snapshots
	tests := []struct {
		name     string
		jsonLine string
		want     domain.DockerSnapshot
	}{
		{
			name:     "running_container",
			jsonLine: `{"Names":"redis","Image":"redis:7","Status":"Up 2 hours","Ports":"0.0.0.0:6379->6379/tcp"}`,
			want: domain.DockerSnapshot{
				ContainerName: "redis",
				Image:         "redis:7",
				Status:        "Up 2 hours",
				Ports:         "0.0.0.0:6379->6379/tcp",
			},
		},
		{
			name:     "stopped_container",
			jsonLine: `{"Names":"postgres","Image":"postgres:16","Status":"Exited (0) 3 hours ago","Ports":""}`,
			want: domain.DockerSnapshot{
				ContainerName: "postgres",
				Image:         "postgres:16",
				Status:        "Exited (0) 3 hours ago",
				Ports:         "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collector.ParseDockerPSLine(tt.jsonLine)
			if err != nil {
				t.Fatalf("ParseDockerPSLine: %v", err)
			}
			if got.ContainerName != tt.want.ContainerName {
				t.Errorf("name: got %s, want %s", got.ContainerName, tt.want.ContainerName)
			}
			if got.Image != tt.want.Image {
				t.Errorf("image: got %s, want %s", got.Image, tt.want.Image)
			}
			if got.Status != tt.want.Status {
				t.Errorf("status: got %s, want %s", got.Status, tt.want.Status)
			}
		})
	}
}

func TestParsePercent(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"12.34%", 12.34},
		{"0.00%", 0},
		{"100.00%", 100},
		{"", 0},
	}
	for _, tt := range tests {
		got := collector.ParsePercent(tt.input)
		if got != tt.want {
			t.Errorf("parsePercent(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestParseMemoryMB(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"123.4MiB / 1.5GiB", 123.4},
		{"1.5GiB / 4GiB", 1536},
		{"512KiB / 1GiB", 0.5},
		{"", 0},
	}
	for _, tt := range tests {
		got := collector.ParseMemoryMB(tt.input)
		if got != tt.want {
			t.Errorf("parseMemoryMB(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestDockerCollector_EnabledCheck(t *testing.T) {
	dc := &collector.DockerCollector{}

	cfg := &config.Config{}
	if !dc.Enabled(cfg) {
		t.Error("expected docker collector enabled by default")
	}

	cfg = &config.Config{Adapters: map[string]bool{"docker": false}}
	if dc.Enabled(cfg) {
		t.Error("expected docker collector disabled")
	}
}
