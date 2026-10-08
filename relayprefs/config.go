package relayprefs

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Default cache TTL
const DefaultCacheTTL = 1 * time.Hour

// Environment variables that ConfigFromEnv requires. There are deliberately no
// defaults: a default pointing at Cloistr's production services meant any new
// environment that forgot them silently talked to production.
const (
	EnvUseCloistrFallback = "USE_CLOISTR_FALLBACK"
	EnvCloistrDiscovery   = "RELAYPREFS_CLOISTR_DISCOVERY"
	EnvCloistrRelay       = "RELAYPREFS_CLOISTR_RELAY"
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

	// UseCloistrFallback enables falling back to CloistrDiscovery and
	// CloistrRelay if no other sources are configured or available.
	UseCloistrFallback bool

	// CloistrDiscovery is the URL of the Cloistr discovery service used as a
	// fallback. Required when UseCloistrFallback is true.
	CloistrDiscovery string

	// CloistrRelay is the URL of the Cloistr relay used as a fallback and as
	// the default relay. Required when UseCloistrFallback is true.
	CloistrRelay string

	// CacheTTL is how long to cache relay preferences. Default: 1 hour.
	CacheTTL time.Duration
}

// ConfigFromEnv creates a Config from environment variables.
//
// Environment variables:
//   - DISCOVERY_INTERNAL: URL of self-hosted discovery service
//   - RELAY_LIST: Comma-separated list of relay URLs for direct queries
//   - DISCOVERY_EXTERNAL: URL of third-party discovery service
//   - USE_CLOISTR_FALLBACK: "true" or "false". Required.
//   - RELAYPREFS_CLOISTR_DISCOVERY: Cloistr discovery URL. Required if fallback is true.
//   - RELAYPREFS_CLOISTR_RELAY: Cloistr relay URL. Required if fallback is true.
//   - RELAY_PREFS_CACHE_TTL: Cache duration (e.g., "1h", "30m"). Default: 1h
//
// It returns an error naming the variable when a required one is unset or
// invalid. Callers should refuse to start on that error.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		InternalDiscovery: os.Getenv("DISCOVERY_INTERNAL"),
		ExternalDiscovery: os.Getenv("DISCOVERY_EXTERNAL"),
		CloistrDiscovery:  os.Getenv(EnvCloistrDiscovery),
		CloistrRelay:      os.Getenv(EnvCloistrRelay),
		CacheTTL:          DefaultCacheTTL,
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
	switch val := strings.ToLower(strings.TrimSpace(os.Getenv(EnvUseCloistrFallback))); val {
	case "true":
		cfg.UseCloistrFallback = true
	case "false":
		cfg.UseCloistrFallback = false
	case "":
		return Config{}, fmt.Errorf("relayprefs: %s is not set; set it to \"true\" or \"false\"", EnvUseCloistrFallback)
	default:
		return Config{}, fmt.Errorf("relayprefs: %s=%q is invalid; set it to \"true\" or \"false\"", EnvUseCloistrFallback, val)
	}

	// Parse cache TTL
	if ttl := os.Getenv("RELAY_PREFS_CACHE_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			cfg.CacheTTL = d
		}
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// HasQuerySources returns true if at least one query source is configured
// (not counting Cloistr fallback).
func (c *Config) HasQuerySources() bool {
	return c.InternalDiscovery != "" ||
		len(c.QueryRelays) > 0 ||
		c.ExternalDiscovery != ""
}

// Validate checks the configuration and returns any issues.
// With Cloistr fallback enabled, both Cloistr URLs must be set.
func (c *Config) Validate() error {
	// No query sources with Cloistr fallback disabled is allowed: there is
	// nothing to query, so lookups always return the config defaults.
	if !c.UseCloistrFallback {
		return nil
	}
	if c.CloistrDiscovery == "" {
		return fmt.Errorf("relayprefs: %s is not set; it is required when %s=true", EnvCloistrDiscovery, EnvUseCloistrFallback)
	}
	if c.CloistrRelay == "" {
		return fmt.Errorf("relayprefs: %s is not set; it is required when %s=true", EnvCloistrRelay, EnvUseCloistrFallback)
	}
	return nil
}
