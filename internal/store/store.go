package store

import (
	"context"
	"time"

	"github.com/xcoleman/pulse/internal/domain"
)

// Store defines the data access interface. Concrete implementation uses SQLite.
// Tests can mock this interface.
type Store interface {
	// Sync run management
	CreateSyncRun(ctx context.Context) (int64, error)
	CompleteSyncRun(ctx context.Context, id int64, status string, syncErr string) error

	// Write methods (used by collectors)
	SaveGitSnapshot(ctx context.Context, syncID int64, snapshot domain.GitSnapshot) error
	SaveGitBranches(ctx context.Context, syncID int64, branches []domain.GitBranch) error
	SaveCostEntry(ctx context.Context, syncID int64, entry domain.CostEntry) error
	SaveDockerSnapshot(ctx context.Context, syncID int64, snapshot domain.DockerSnapshot) error
	SaveSystemSnapshot(ctx context.Context, syncID int64, snapshot domain.SystemSnapshot) error
	SaveHardwareSnapshot(ctx context.Context, syncID int64, snapshot domain.HardwareSnapshot) error
	SaveNetworkSnapshot(ctx context.Context, syncID int64, snapshot domain.NetworkSnapshot) error
	SaveStorageHealth(ctx context.Context, syncID int64, health domain.StorageHealth) error
	SaveJournalAlerts(ctx context.Context, syncID int64, alerts []domain.JournalAlert) error
	SaveGitHubNotifications(ctx context.Context, syncID int64, notifs []domain.Notification) error
	SaveBriefing(ctx context.Context, entry domain.BriefingEntry) error

	// Read methods (used by briefing engine)
	LatestSyncID(ctx context.Context) (int64, error)
	GetGitSnapshots(ctx context.Context, syncID int64) ([]domain.GitSnapshot, error)
	GetGitBranches(ctx context.Context, syncID int64, repoPath string) ([]domain.GitBranch, error)
	GetGitHubNotifications(ctx context.Context, syncID int64) ([]domain.Notification, error)
	GetCostEntries(ctx context.Context, since time.Time) ([]domain.CostEntry, error)
	GetLatestCostEntry(ctx context.Context, service string) (*domain.CostEntry, error)
	GetDockerSnapshots(ctx context.Context, syncID int64) ([]domain.DockerSnapshot, error)
	GetSystemSnapshot(ctx context.Context, syncID int64) (*domain.SystemSnapshot, error)
	GetHardwareSnapshot(ctx context.Context, syncID int64) (*domain.HardwareSnapshot, error)
	GetNetworkSnapshot(ctx context.Context, syncID int64) (*domain.NetworkSnapshot, error)
	GetStorageHealth(ctx context.Context, syncID int64) (*domain.StorageHealth, error)
	GetJournalAlerts(ctx context.Context, syncID int64) ([]domain.JournalAlert, error)
	GetLastBriefingTime(ctx context.Context) (time.Time, error)

	// Maintenance
	PruneBriefingHistory(ctx context.Context, olderThan time.Time) (int64, error)

	// Lifecycle
	Close() error
}
