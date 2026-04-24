package collector

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type InterfaceStats = domain.InterfaceStats

type NetworkCollector struct{}

func (n *NetworkCollector) Name() string      { return "network" }
func (n *NetworkCollector) EnvVars() []string { return nil }

func (n *NetworkCollector) Enabled(cfg *config.Config) bool {
	return runtime.GOOS == "linux" && cfg.AdapterEnabled("network")
}

func (n *NetworkCollector) Collect(ctx context.Context, s store.Store, _ *config.Config, syncID int64) error {
	_ = ctx
	if runtime.GOOS != "linux" {
		return nil
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	snap := domain.NetworkSnapshot{}
	snap.Interfaces = make([]domain.InterfaceStats, 0, len(ifaces))
	for _, iface := range ifaces {
		stats := domain.InterfaceStats{
			Name:      iface.Name,
			State:     interfaceState(iface),
			RxBytes:   readUint64(filepath.Join("/sys/class/net", iface.Name, "statistics", "rx_bytes")),
			TxBytes:   readUint64(filepath.Join("/sys/class/net", iface.Name, "statistics", "tx_bytes")),
			RxErrors:  readUint64(filepath.Join("/sys/class/net", iface.Name, "statistics", "rx_errors")),
			TxErrors:  readUint64(filepath.Join("/sys/class/net", iface.Name, "statistics", "tx_errors")),
			RxDropped: readUint64(filepath.Join("/sys/class/net", iface.Name, "statistics", "rx_dropped")),
		}
		snap.Interfaces = append(snap.Interfaces, stats)
		if snap.ActiveInterface == "" && stats.State == "up" && iface.Flags&net.FlagLoopback == 0 {
			snap.ActiveInterface = iface.Name
		}
	}

	snap.ConnectionType = connectionType(snap.Interfaces)
	return s.SaveNetworkSnapshot(ctx, syncID, snap)
}

func interfaceState(iface net.Interface) string {
	if iface.Flags&net.FlagUp != 0 {
		return "up"
	}
	return "down"
}

func connectionType(interfaces []InterfaceStats) string {
	for _, iface := range interfaces {
		if iface.State != "up" {
			continue
		}
		name := strings.ToLower(iface.Name)
		if strings.HasPrefix(name, "wl") || strings.HasPrefix(name, "wlan") {
			return "wifi"
		}
	}
	for _, iface := range interfaces {
		if iface.State != "up" {
			continue
		}
		name := strings.ToLower(iface.Name)
		if strings.HasPrefix(name, "en") || strings.HasPrefix(name, "eth") {
			return "ethernet"
		}
	}
	return "unknown"
}

func readUint64(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func init() {
	Register(&NetworkCollector{})
}
