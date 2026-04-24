package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Projects ProjectsConfig  `mapstructure:"projects"`
	GitHub   GitHubConfig    `mapstructure:"github"`
	Obsidian ObsidianConfig  `mapstructure:"obsidian"`
	Claude   ClaudeConfig    `mapstructure:"claude"`
	Adapters map[string]bool `mapstructure:"adapters"`
	Sync     SyncConfig      `mapstructure:"sync"`
	Costs    CostsConfig     `mapstructure:"costs"`
	Journal  JournalConfig   `mapstructure:"journal"`
}

type ProjectsConfig struct {
	Scan   []string `mapstructure:"scan"`
	Ignore []string `mapstructure:"ignore"`
}

type GitHubConfig struct {
	Username string `mapstructure:"username"`
}

type ObsidianConfig struct {
	VaultPath      string `mapstructure:"vault_path"`
	DailyNotePath  string `mapstructure:"daily_note_path"`
	SectionHeading string `mapstructure:"section_heading"`
}

type ClaudeConfig struct {
	Subscription     string `mapstructure:"subscription"`       // "max", "api", or "" (auto-detect)
	MonthlyCostCents int    `mapstructure:"monthly_cost_cents"` // fixed monthly cost in cents (e.g., 10000 = $100)
}

type SyncConfig struct {
	Timeout string `mapstructure:"timeout"`
	LogFile string `mapstructure:"log_file"`
}

type JournalConfig struct {
	WatchUnits  []string `mapstructure:"watch_units"`
	MinPriority int      `mapstructure:"min_priority"`
}

type CostsConfig struct {
	DefaultPeriod string               `mapstructure:"default_period"`
	Currency      string               `mapstructure:"currency"`
	Subscriptions []SubscriptionConfig `mapstructure:"subscriptions"`
	Pricing       PricingConfig        `mapstructure:"pricing"`
}

type PricingConfig struct {
	TavilyCentsPerRequest     int `mapstructure:"tavily_cents_per_request"`      // e.g., 1 = $0.01/request
	ElevenLabsCentsPer1KChars int `mapstructure:"elevenlabs_cents_per_1k_chars"` // e.g., 30 = $0.30/1k chars
}

type SubscriptionConfig struct {
	Name             string `mapstructure:"name"`               // display name (e.g., "Google One Premium")
	Service          string `mapstructure:"service"`            // service key (e.g., "google-one")
	MonthlyCostCents int    `mapstructure:"monthly_cost_cents"` // cost in cents (e.g., 2499 = $24.99)
	Notes            string `mapstructure:"notes"`              // optional description
}

// AdapterEnabled returns whether an adapter is enabled. Unlisted adapters default to enabled.
func (c *Config) AdapterEnabled(name string) bool {
	if enabled, ok := c.Adapters[name]; ok {
		return enabled
	}
	return true
}

// DefaultConfigDir returns ~/.config/pulse/
func DefaultConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pulse")
}

// DefaultConfigPath returns ~/.config/pulse/config.yaml
func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config.yaml")
}

// ObsidianDailyNotePath resolves the daily note path with date tokens.
// Translates Obsidian-style tokens (YYYY, MM, DD) to Go time format.
func (c *Config) ObsidianDailyNotePath(t interface{ Format(string) string }) string {
	path := c.Obsidian.DailyNotePath
	// Replace Obsidian tokens with Go time format placeholders, then format
	path = strings.ReplaceAll(path, "YYYY", "2006")
	path = strings.ReplaceAll(path, "MM", "01")
	path = strings.ReplaceAll(path, "DD", "02")
	return filepath.Join(c.Obsidian.VaultPath, t.Format(path))
}

func Load(path string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("sync.timeout", "30s")
	v.SetDefault("costs.default_period", "30d")
	v.SetDefault("costs.currency", "USD")
	v.SetDefault("obsidian.section_heading", "## Pulse Briefing")

	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				if !os.IsNotExist(err) {
					return nil, fmt.Errorf("reading config: %w", err)
				}
			}
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

const defaultConfigTemplate = `# Pulse configuration
projects:
  scan:
    - ~/projects
  ignore: []

github:
  username: ""

obsidian:
  vault_path: ""
  daily_note_path: "Daily Notes/YYYY-MM-DD.md"
  section_heading: "## Pulse Briefing"

# Claude Code cost tracking (reads local logs, no API key needed)
# claude:
#   subscription: ""          # "max" for fixed monthly, "" for API pricing
#   monthly_cost_cents: 0     # e.g., 10000 = $100/mo for Max plan

adapters:
  git: true
  github: true
  claude: true
  tavily: true
  elevenlabs: true
  docker: true
  system: true

sync:
  timeout: 30s

costs:
  default_period: 30d
  currency: USD
  pricing:
    tavily_cents_per_request: 1        # $0.01/request (pay-as-you-go)
    elevenlabs_cents_per_1k_chars: 30  # $0.30/1k characters
  # subscriptions:
  #   - name: "Example Service"
  #     service: "example"
  #     monthly_cost_cents: 999       # $9.99/mo
`

func GenerateDefault(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	return os.WriteFile(path, []byte(defaultConfigTemplate), 0600)
}
