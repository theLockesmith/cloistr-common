// Package platform provides unified access control, quota management, and usage
// tracking for Cloistr services. It supports both platform mode (using the shared
// PostgreSQL database) and standalone mode (using config files/env vars).
package platform

import (
	"database/sql"
	"os"
	"strconv"
	"strings"
)

// Mode determines whether the service runs in platform or standalone mode.
type Mode string

const (
	ModePlatform   Mode = "platform"
	ModeStandalone Mode = "standalone"
)

// Config holds the configuration for the platform package.
type Config struct {
	// Mode determines whether to use the database or config-based access control.
	Mode Mode

	// DatabaseURL is the PostgreSQL connection string (platform mode only).
	DatabaseURL string

	// ServiceID is the identifier for this service (e.g., "blossom", "drive").
	ServiceID string

	// Standalone mode settings
	WhitelistPubkeys   []string // Pubkeys allowed to access the service
	WhitelistAllowAll  bool     // If true, all pubkeys are allowed
	AdminPubkeys       []string // Pubkeys with admin access
	StorageQuotaBytes  int64    // Default quota per user (0 = unlimited)
	SigningRequestsMax int64    // Max signing requests per day (0 = unlimited)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Mode:      ModeStandalone,
		ServiceID: "unknown",
	}
}

// ConfigFromEnv creates a Config from environment variables.
//
// Environment variables:
//   - CLOISTR_MODE: "platform" or "standalone" (default: "standalone")
//   - DATABASE_URL: PostgreSQL connection string (platform mode)
//   - SERVICE_ID: Service identifier (e.g., "blossom")
//   - WHITELIST_PUBKEYS: Comma-separated list of allowed pubkeys (standalone)
//   - WHITELIST_ALLOW_ALL: "true" to allow all pubkeys (standalone)
//   - ADMIN_PUBKEYS: Comma-separated list of admin pubkeys (standalone)
//   - STORAGE_QUOTA_BYTES: Per-user storage quota in bytes (standalone)
//   - SIGNING_REQUESTS_MAX: Max signing requests per day (standalone)
func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	// Mode
	if mode := os.Getenv("CLOISTR_MODE"); mode != "" {
		if mode == "platform" {
			cfg.Mode = ModePlatform
		} else {
			cfg.Mode = ModeStandalone
		}
	}

	// Database URL
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")

	// Service ID
	if serviceID := os.Getenv("SERVICE_ID"); serviceID != "" {
		cfg.ServiceID = serviceID
	}

	// Whitelist
	if pubkeys := os.Getenv("WHITELIST_PUBKEYS"); pubkeys != "" {
		cfg.WhitelistPubkeys = strings.Split(pubkeys, ",")
		for i := range cfg.WhitelistPubkeys {
			cfg.WhitelistPubkeys[i] = strings.TrimSpace(cfg.WhitelistPubkeys[i])
		}
	}

	// Allow all
	if allowAll := os.Getenv("WHITELIST_ALLOW_ALL"); allowAll == "true" || allowAll == "1" {
		cfg.WhitelistAllowAll = true
	}

	// Admin pubkeys
	if admins := os.Getenv("ADMIN_PUBKEYS"); admins != "" {
		cfg.AdminPubkeys = strings.Split(admins, ",")
		for i := range cfg.AdminPubkeys {
			cfg.AdminPubkeys[i] = strings.TrimSpace(cfg.AdminPubkeys[i])
		}
	}

	// Quotas
	if quota := os.Getenv("STORAGE_QUOTA_BYTES"); quota != "" {
		if v, err := strconv.ParseInt(quota, 10, 64); err == nil {
			cfg.StorageQuotaBytes = v
		}
	}

	if max := os.Getenv("SIGNING_REQUESTS_MAX"); max != "" {
		if v, err := strconv.ParseInt(max, 10, 64); err == nil {
			cfg.SigningRequestsMax = v
		}
	}

	return cfg
}

// Validate checks that the config is valid.
func (c *Config) Validate() error {
	if c.Mode == ModePlatform && c.DatabaseURL == "" {
		return ErrNoDatabaseURL
	}
	if c.ServiceID == "" {
		return ErrNoServiceID
	}
	return nil
}

// Client provides access to platform services (access control, quotas, etc.).
type Client struct {
	config Config
	db     *sql.DB
}

// NewClient creates a new platform client from the given config.
func NewClient(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	c := &Client{config: cfg}

	if cfg.Mode == ModePlatform {
		db, err := sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}
		if err := db.Ping(); err != nil {
			return nil, err
		}
		c.db = db
	}

	return c, nil
}

// NewClientFromEnv creates a new platform client from environment variables.
func NewClientFromEnv() (*Client, error) {
	return NewClient(ConfigFromEnv())
}

// Close closes the database connection (if any).
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Mode returns the current mode (platform or standalone).
func (c *Client) Mode() Mode {
	return c.config.Mode
}

// ServiceID returns the configured service ID.
func (c *Client) ServiceID() string {
	return c.config.ServiceID
}
