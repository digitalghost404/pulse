package collector

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

func TestNetworkCollector_Enabled(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("network collector is linux-specific")
	}

	nc := &NetworkCollector{}
	if !nc.Enabled(&config.Config{}) {
		t.Fatal("expected network collector enabled on linux")
	}
}

func TestNetworkCollector_Collect(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("network collector is linux-specific")
	}

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	syncID, err := s.CreateSyncRun(ctx)
	if err != nil {
		t.Fatalf("CreateSyncRun: %v", err)
	}

	nc := &NetworkCollector{}
	if err := nc.Collect(ctx, s, &config.Config{}, syncID); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	snap, err := s.GetNetworkSnapshot(ctx, syncID)
	if err != nil {
		t.Fatalf("GetNetworkSnapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("expected non-nil network snapshot")
	}
	if len(snap.Interfaces) == 0 {
		t.Fatal("expected at least one network interface")
	}

	foundLoopback := false
	for _, iface := range snap.Interfaces {
		if iface.Name == "lo" {
			foundLoopback = true
			break
		}
	}
	if !foundLoopback {
		t.Fatal("expected loopback interface in network snapshot")
	}
}

func TestConnectionType(t *testing.T) {
	tests := []struct {
		name       string
		interfaces []domain.InterfaceStats
		want       string
	}{
		{
			name: "wifi preferred when up",
			interfaces: []domain.InterfaceStats{
				{Name: "wlp2s0", State: "up"},
				{Name: "enp0s31f6", State: "down"},
			},
			want: "wifi",
		},
		{
			name: "ethernet when ethernet up",
			interfaces: []domain.InterfaceStats{
				{Name: "enp0s31f6", State: "up"},
			},
			want: "ethernet",
		},
		{
			name: "unknown when no known interface up",
			interfaces: []domain.InterfaceStats{
				{Name: "lo", State: "up"},
			},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := connectionType(tt.interfaces)
			if got != tt.want {
				t.Fatalf("connectionType() = %q, want %q", got, tt.want)
			}
		})
	}
}
