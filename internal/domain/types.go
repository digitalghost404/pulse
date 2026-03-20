package domain

import "time"

// GitSnapshot represents the state of a git repo at a point in time.
type GitSnapshot struct {
	RepoPath       string
	RepoName       string
	Branch         string
	DirtyFiles     int
	Ahead          int
	Behind         int
	LastCommitHash string
	LastCommitMsg  string
	LastCommitAt   time.Time
}

// GitBranch represents a branch in a git repo.
type GitBranch struct {
	RepoPath     string
	BranchName   string
	LastCommitAt time.Time
	IsMerged     bool
	IsCurrent    bool
}

// Notification represents a GitHub notification.
type Notification struct {
	RepoName  string
	Type      string // pr, issue, ci
	Title     string
	URL       string
	State     string
	UpdatedAt time.Time
}

// CostEntry represents a normalized cost record from any service.
type CostEntry struct {
	Service       string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	AmountCents   int
	Currency      string
	UsageQuantity float64
	UsageUnit     string
	RawData       string // JSON
}

// DockerSnapshot represents the state of a Docker container.
type DockerSnapshot struct {
	ContainerName string
	Image         string
	Status        string
	Ports         string // JSON
	CPUPct        float64
	MemoryMB      float64
}

// SystemSnapshot represents system resource usage.
type SystemSnapshot struct {
	CPUPct        float64
	MemoryUsedMB  float64
	MemoryTotalMB float64
	DiskUsedGB    float64
	DiskTotalGB   float64
}

// SyncRun represents a single sync execution.
type SyncRun struct {
	ID          int64
	StartedAt   time.Time
	CompletedAt time.Time
	Status      string // success, partial, failed
	Error       string
}

// BriefingEntry represents a rendered briefing stored in history.
type BriefingEntry struct {
	ID        int64
	CreatedAt time.Time
	Content   string
	Writer    string
}

// Briefing is the intermediate representation between the DB and Writers.
type Briefing struct {
	GeneratedAt   time.Time
	Projects      []ProjectSummary
	Notifications []Notification
	CostSummary   CostSummary
	Docker        []DockerSnapshot
	System        SystemSnapshot
}

// ProjectSummary combines git snapshot with branch info for display.
type ProjectSummary struct {
	GitSnapshot
	Branches []GitBranch
}

// CostSummary aggregates cost data for the briefing.
type CostSummary struct {
	TotalCents    int
	Currency      string
	ByService     []ServiceCost
	Period        string
	BurnRateCents int // daily average
}

// ServiceCost represents cost for a single service.
type ServiceCost struct {
	Service       string
	AmountCents   int
	UsageQuantity float64
	UsageUnit     string
}
