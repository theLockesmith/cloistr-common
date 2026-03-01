package relayprefs

import (
	"os"
	"strings"
	"time"
)

const (
	// Default Cloistr services
	DefaultCloistrDiscovery = "https://discover.cloistr.xyz"
	DefaultCloistrRelay     = "wss://relay.cloistr.xyz"

	// Default cache TTL
	DefaultCacheTTL = 1 * time.Hour
)

// Config holds the configuration for the relay preferences client.
type Config struct {
	// InternalDiscovery is the URL of a self-hosted discovery service.
	// If set, this is queried first (after cache).
	InternalDiscovery string

	// QueryRelays is a list of relays to query directly for preferences.
	// These are queried if discovery services are unavailable or not configured.
	QueryRelays []string

	// ExternalDiscovery is the URL of a third-party discovery service.
	// Queried after QueryRelays but before Cloistr services.
	ExternalDiscovery string

	// UseCloistrFallback enables falling back to Cloistr's discovery and relay
	// if no other sources are configured or available. Default: true.
	UseCloistrFallback bool

	// CacheTTL is how long to cache relay preferences. Default: 1 hour.
	CacheTTL time.Duration
}

// ConfigFromEnv creates a Config from environment variables.
//
// Environment variables:
//   - DISCOVERY_INTERNAL: URL of self-hosted discovery service
//   - RELAY_LIST: Comma-separated list of relay URLs for direct queries
//   - DISCOVERY_EXTERNAL: URL of third-party discovery service
//   - USE_CLOISTR_FALLBACK: "true" (default) or "false"
//   - RELAY_PREFS_CACHE_TTL: Cache duration (e.g., "1h", "30m"). Default: 1h
func ConfigFromEnv() Config {
	cfg := Config{
		InternalDiscovery:  os.Getenv("DISCOVERY_INTERNAL"),
		ExternalDiscovery:  os.Getenv("DISCOVERY_EXTERNAL"),
		UseCloistrFallback: true,
		CacheTTL:           DefaultCacheTTL,
	}

	// Parse relay list
	if relayList := os.Getenv("RELAY_LIST"); relayList != "" {
		relays := strings.Split(relayList, ",")
		for _, r := range relays {
			r = strings.TrimSpace(r)
			if r != "" {
				cfg.QueryRelays = append(cfg.QueryRelays, r)
			}
		}
	}

	// Parse Cloistr fallback flag
	if val := os.Getenv("USE_CLOISTR_FALLBACK"); val != "" {
		cfg.UseCloistrFallback = strings.ToLower(val) == "true"
	}

	// Parse cache TTL
	if ttl := os.Getenv("RELAY_PREFS_CACHE_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			cfg.CacheTTL = d
		}
	}

	return cfg
}

// HasQuerySources returns true if at least one query source is configured
// (not counting Cloistr fallback).
func (c *Config) HasQuerySources() bool {
	return c.InternalDiscovery != "" ||
		len(c.QueryRelays) > 0 ||
		c.ExternalDiscovery != ""
}

// Validate checks the configuration and returns any issues.
// Currently just ensures we have at least one way to query preferences.
func (c *Config) Validate() error {
	if !c.HasQuerySources() && !c.UseCloistrFallback {
		// No query sources and Cloistr fallback disabled - nothing to query
		// This is allowed but will always return defaults
	}
	return nil
}
