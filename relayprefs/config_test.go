package relayprefs

import (
	"testing"
	"time"
)

func TestConfigFromEnv(t *testing.T) {
	// Test with all env vars set
	t.Setenv("DISCOVERY_INTERNAL", "http://internal:8080")
	t.Setenv("RELAY_LIST", "wss://relay1.com, wss://relay2.com")
	t.Setenv("DISCOVERY_EXTERNAL", "http://external:8080")
	t.Setenv("USE_CLOISTR_FALLBACK", "false")
	t.Setenv("RELAY_PREFS_CACHE_TTL", "30m")

	cfg := ConfigFromEnv()

	if cfg.InternalDiscovery != "http://internal:8080" {
		t.Errorf("expected internal discovery http://internal:8080, got %s", cfg.InternalDiscovery)
	}

	if len(cfg.QueryRelays) != 2 {
		t.Errorf("expected 2 query relays, got %d", len(cfg.QueryRelays))
	}
	if cfg.QueryRelays[0] != "wss://relay1.com" {
		t.Errorf("expected first relay wss://relay1.com, got %s", cfg.QueryRelays[0])
	}

	if cfg.ExternalDiscovery != "http://external:8080" {
		t.Errorf("expected external discovery http://external:8080, got %s", cfg.ExternalDiscovery)
	}

	if cfg.UseCloistrFallback {
		t.Error("expected UseCloistrFallback to be false")
	}

	if cfg.CacheTTL != 30*time.Minute {
		t.Errorf("expected cache TTL 30m, got %v", cfg.CacheTTL)
	}
}

func TestConfigFromEnv_Defaults(t *testing.T) {
	// Empty is treated the same as unset; t.Setenv restores the original after the test.
	t.Setenv("DISCOVERY_INTERNAL", "")
	t.Setenv("RELAY_LIST", "")
	t.Setenv("DISCOVERY_EXTERNAL", "")
	t.Setenv("USE_CLOISTR_FALLBACK", "")
	t.Setenv("RELAY_PREFS_CACHE_TTL", "")

	cfg := ConfigFromEnv()

	if cfg.InternalDiscovery != "" {
		t.Errorf("expected empty internal discovery, got %s", cfg.InternalDiscovery)
	}

	if len(cfg.QueryRelays) != 0 {
		t.Errorf("expected 0 query relays, got %d", len(cfg.QueryRelays))
	}

	if !cfg.UseCloistrFallback {
		t.Error("expected UseCloistrFallback to default to true")
	}

	if cfg.CacheTTL != DefaultCacheTTL {
		t.Errorf("expected default cache TTL, got %v", cfg.CacheTTL)
	}
}

func TestConfig_HasQuerySources(t *testing.T) {
	// Empty config
	cfg := Config{}
	if cfg.HasQuerySources() {
		t.Error("empty config should not have query sources")
	}

	// With internal discovery
	cfg = Config{InternalDiscovery: "http://test"}
	if !cfg.HasQuerySources() {
		t.Error("config with internal discovery should have query sources")
	}

	// With relay list
	cfg = Config{QueryRelays: []string{"wss://test"}}
	if !cfg.HasQuerySources() {
		t.Error("config with query relays should have query sources")
	}

	// With external discovery
	cfg = Config{ExternalDiscovery: "http://test"}
	if !cfg.HasQuerySources() {
		t.Error("config with external discovery should have query sources")
	}
}
