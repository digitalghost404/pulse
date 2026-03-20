package collector

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

type SystemCollector struct{}

func (s *SystemCollector) Name() string      { return "system" }
func (s *SystemCollector) EnvVars() []string { return nil }

func (s *SystemCollector) Enabled(cfg *config.Config) bool {
	return cfg.AdapterEnabled("system")
}

func (s *SystemCollector) Collect(ctx context.Context, st store.Store, cfg *config.Config, syncID int64) error {
	snap := domain.SystemSnapshot{}

	// CPU from /proc/stat (snapshot — not a delta, but good enough for a point-in-time view)
	snap.CPUPct = readCPUPercent()

	// Memory from /proc/meminfo
	snap.MemoryTotalMB, snap.MemoryUsedMB = readMemory()

	// Disk from syscall.Statfs
	snap.DiskTotalGB, snap.DiskUsedGB = readDisk("/")

	return st.SaveSystemSnapshot(ctx, syncID, snap)
}

func readCPUPercent() float64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				user, _ := strconv.ParseFloat(fields[1], 64)
				nice, _ := strconv.ParseFloat(fields[2], 64)
				system, _ := strconv.ParseFloat(fields[3], 64)
				idle, _ := strconv.ParseFloat(fields[4], 64)
				total := user + nice + system + idle
				if total > 0 {
					return (total - idle) / total * 100
				}
			}
		}
	}
	return 0
}

func readMemory() (totalMB, usedMB float64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	var totalKB, availKB float64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				totalKB, _ = strconv.ParseFloat(fields[1], 64)
			}
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				availKB, _ = strconv.ParseFloat(fields[1], 64)
			}
		}
	}
	totalMB = totalKB / 1024
	usedMB = (totalKB - availKB) / 1024
	return
}

func readDisk(path string) (totalGB, usedGB float64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	totalBytes := float64(stat.Blocks) * float64(stat.Bsize)
	freeBytes := float64(stat.Bfree) * float64(stat.Bsize)
	totalGB = totalBytes / (1024 * 1024 * 1024)
	usedGB = (totalBytes - freeBytes) / (1024 * 1024 * 1024)
	return
}

func init() {
	Register(&SystemCollector{})
}
