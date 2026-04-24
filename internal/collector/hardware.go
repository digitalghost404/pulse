package collector

import (
	"context"
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/domain"
	"github.com/xcoleman/pulse/internal/store"
)

var (
	lookPath           = exec.LookPath
	execCommandContext = exec.CommandContext
	sleep              = time.Sleep

	nvidiaSMIPath        = "nvidia-smi"
	thermalZoneTempPath  = "/sys/class/thermal/thermal_zone0/temp"
	cpuFreqPath          = "/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq"
	batteryBasePath      = "/sys/class/power_supply/BAT0"
	raplPackageEnergyPath = "/sys/class/powercap/intel-rapl:0/energy_uj"
)

type HardwareCollector struct{}

func (h *HardwareCollector) Name() string      { return "hardware" }
func (h *HardwareCollector) EnvVars() []string { return nil }

func (h *HardwareCollector) Enabled(cfg *config.Config) bool {
	if !cfg.AdapterEnabled("hardware") {
		return false
	}

	if _, err := lookPath(nvidiaSMIPath); err == nil {
		return true
	}

	return fileExists(thermalZoneTempPath) || fileExists(batteryBasePath) || fileExists(raplPackageEnergyPath)
}

func (h *HardwareCollector) Collect(ctx context.Context, s store.Store, _ *config.Config, syncID int64) error {
	snap := domain.HardwareSnapshot{}

	collectGPU(ctx, &snap)
	snap.CPUTempC = readIntValue(thermalZoneTempPath, 1000)
	snap.CPUFreqMHz = readIntValue(cpuFreqPath, 1000)
	collectBattery(&snap)
	snap.PackagePowerWatts = readRAPLWatts()

	return s.SaveHardwareSnapshot(ctx, syncID, snap)
}

func collectGPU(ctx context.Context, snap *domain.HardwareSnapshot) {
	if _, err := lookPath(nvidiaSMIPath); err != nil {
		return
	}

	cmd := execCommandContext(ctx, nvidiaSMIPath,
		"--query-gpu=name,temperature.gpu,utilization.gpu,memory.used,memory.free,memory.total,fan.speed,power.draw",
		"--format=csv,noheader,nounits",
	)
	out, err := cmd.Output()
	if err != nil {
		return
	}

	records, err := csv.NewReader(strings.NewReader(strings.TrimSpace(string(out)))).ReadAll()
	if err != nil || len(records) == 0 || len(records[0]) < 8 {
		return
	}

	fields := records[0]
	snap.GPUName = strings.TrimSpace(fields[0])
	snap.GPUTempC = parseInt(fields[1])
	snap.GPUUtilPct = parseFloat(fields[2])
	snap.GPUMemUsedMB = parseFloat(fields[3])
	snap.GPUMemTotalMB = parseFloat(fields[5])
	snap.GPUFanSpeedPct = parseInt(fields[6])
	snap.GPUPowerWatts = parseFloat(fields[7])
}

func collectBattery(snap *domain.HardwareSnapshot) {
	capacityPath := filepath.Join(batteryBasePath, "capacity")
	statusPath := filepath.Join(batteryBasePath, "status")
	powerNowPath := filepath.Join(batteryBasePath, "power_now")

	if fileExists(capacityPath) {
		snap.BatteryPct = readIntValue(capacityPath, 1)
	}
	if data, err := os.ReadFile(statusPath); err == nil {
		snap.BatteryStatus = strings.TrimSpace(string(data))
	}
	if powerNow, ok := readFloatFile(powerNowPath); ok {
		snap.BatteryWatts = powerNow / 1e6
	}
}

func readRAPLWatts() float64 {
	start, ok := readFloatFile(raplPackageEnergyPath)
	if !ok {
		return 0
	}
	const interval = 100 * time.Millisecond
	sleep(interval)
	end, ok := readFloatFile(raplPackageEnergyPath)
	if !ok || end < start {
		return 0
	}
	joules := (end - start) / 1e6
	return joules / interval.Seconds()
}

func readIntValue(path string, divisor int) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	if divisor > 1 {
		value /= divisor
	}
	return value
}

func readFloatFile(path string) (float64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseInt(s string) int {
	value, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return value
}

func parseFloat(s string) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return value
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func init() {
	Register(&HardwareCollector{})
}
