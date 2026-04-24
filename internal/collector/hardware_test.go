package collector

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xcoleman/pulse/internal/config"
	"github.com/xcoleman/pulse/internal/store"
)

func TestHardwareCollector_Enabled(t *testing.T) {
	t.Run("enabled when gpu command exists", func(t *testing.T) {
		collector := &HardwareCollector{}
		restore := stubHardwareEnv(t)
		defer restore()

		lookPath = func(file string) (string, error) {
			if file == nvidiaSMIPath {
				return "/usr/bin/nvidia-smi", nil
			}
			return "", exec.ErrNotFound
		}

		if !collector.Enabled(&config.Config{}) {
			t.Fatal("expected hardware collector enabled when nvidia-smi exists")
		}
	})

	t.Run("enabled when sysfs source exists", func(t *testing.T) {
		collector := &HardwareCollector{}
		restore := stubHardwareEnv(t)
		defer restore()

		thermalPath := filepath.Join(t.TempDir(), "thermal_zone0", "temp")
		if err := os.MkdirAll(filepath.Dir(thermalPath), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(thermalPath, []byte("42000\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		thermalZoneTempPath = thermalPath

		if !collector.Enabled(&config.Config{}) {
			t.Fatal("expected hardware collector enabled when sysfs source exists")
		}
	})

	t.Run("disabled when adapter disabled", func(t *testing.T) {
		collector := &HardwareCollector{}
		restore := stubHardwareEnv(t)
		defer restore()

		lookPath = func(file string) (string, error) {
			return "/usr/bin/nvidia-smi", nil
		}

		cfg := &config.Config{Adapters: map[string]bool{"hardware": false}}
		if collector.Enabled(cfg) {
			t.Fatal("expected hardware collector disabled by config")
		}
	})

	t.Run("disabled when no sources available", func(t *testing.T) {
		collector := &HardwareCollector{}
		restore := stubHardwareEnv(t)
		defer restore()

		if collector.Enabled(&config.Config{}) {
			t.Fatal("expected hardware collector disabled when no sources exist")
		}
	})
}

func TestHardwareCollector_Collect_NoGPU(t *testing.T) {
	restore := stubHardwareEnv(t)
	defer restore()

	thermalZoneTempPath = writeHardwareFile(t, "thermal_zone0/temp", "57000\n")
	cpuFreqPath = writeHardwareFile(t, "cpu0/cpufreq/scaling_cur_freq", "4200000\n")
	batteryBasePath = filepath.Dir(writeHardwareFile(t, "BAT0/capacity", "88\n"))
	writeHardwareFileInDir(t, batteryBasePath, "status", "Discharging\n")
	writeHardwareFileInDir(t, batteryBasePath, "power_now", "24500000\n")

	lookPath = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		t.Fatalf("unexpected exec call: %s %v", name, args)
		return nil
	}

	s := newHardwareTestStore(t)
	ctx := context.Background()
	syncID, err := s.CreateSyncRun(ctx)
	if err != nil {
		t.Fatalf("CreateSyncRun: %v", err)
	}

	collector := &HardwareCollector{}
	if err := collector.Collect(ctx, s, &config.Config{}, syncID); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	snap, err := s.GetHardwareSnapshot(ctx, syncID)
	if err != nil {
		t.Fatalf("GetHardwareSnapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("expected hardware snapshot to be saved")
	}
	if snap.GPUName != "" || snap.GPUUtilPct != 0 || snap.GPUPowerWatts != 0 {
		t.Fatalf("expected zero GPU fields when nvidia-smi is unavailable, got %+v", *snap)
	}
	if snap.CPUTempC != 57 {
		t.Fatalf("expected cpu temp 57C, got %d", snap.CPUTempC)
	}
	if snap.CPUFreqMHz != 4200 {
		t.Fatalf("expected cpu freq 4200MHz, got %d", snap.CPUFreqMHz)
	}
	if snap.BatteryPct != 88 || snap.BatteryStatus != "Discharging" {
		t.Fatalf("expected battery data to be collected, got %+v", *snap)
	}
	if snap.BatteryWatts != 24.5 {
		t.Fatalf("expected battery watts 24.5, got %f", snap.BatteryWatts)
	}
}

func TestHardwareCollector_Collect_Integration(t *testing.T) {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		t.Skip("nvidia-smi not available")
	}
	if out, err := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader,nounits").Output(); err != nil || len(out) == 0 {
		t.Skip("nvidia-smi available but no GPU data returned")
	}

	s := newHardwareTestStore(t)
	ctx := context.Background()
	syncID, err := s.CreateSyncRun(ctx)
	if err != nil {
		t.Fatalf("CreateSyncRun: %v", err)
	}

	collector := &HardwareCollector{}
	if err := collector.Collect(ctx, s, &config.Config{}, syncID); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	snap, err := s.GetHardwareSnapshot(ctx, syncID)
	if err != nil {
		t.Fatalf("GetHardwareSnapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("expected hardware snapshot to be saved")
	}
	if snap.GPUName == "" {
		t.Fatalf("expected GPU name from nvidia-smi, got %+v", *snap)
	}
}

func stubHardwareEnv(t *testing.T) func() {
	t.Helper()
	origLookPath := lookPath
	origExecCommandContext := execCommandContext
	origSleep := sleep
	origThermalPath := thermalZoneTempPath
	origCPUFreqPath := cpuFreqPath
	origBatteryBasePath := batteryBasePath
	origRAPLEnergyPath := raplPackageEnergyPath

	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	execCommandContext = exec.CommandContext
	sleep = func(duration time.Duration) {}
	thermalZoneTempPath = filepath.Join(t.TempDir(), "missing-thermal")
	cpuFreqPath = filepath.Join(t.TempDir(), "missing-cpufreq")
	batteryBasePath = filepath.Join(t.TempDir(), "missing-battery")
	raplPackageEnergyPath = filepath.Join(t.TempDir(), "missing-rapl")

	return func() {
		lookPath = origLookPath
		execCommandContext = origExecCommandContext
		sleep = origSleep
		thermalZoneTempPath = origThermalPath
		cpuFreqPath = origCPUFreqPath
		batteryBasePath = origBatteryBasePath
		raplPackageEnergyPath = origRAPLEnergyPath
	}
}

func newHardwareTestStore(t *testing.T) store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "hardware.db")
	s, err := store.NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("NewSQLite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func writeHardwareFile(t *testing.T, rel string, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func writeHardwareFileInDir(t *testing.T, dir string, name string, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}
