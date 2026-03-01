package relayprefs

import (
	"os"
	"testing"
	"time"
)

func TestConfigFromEnv(t *testing.T) {
	// Save and restore env
	origInternal := os.Getenv("DISCOVERY_INTERNAL")
	origRelayList := os.Getenv("RELAY_LIST")
	origExternal := os.Getenv("DISCOVERY_EXTERNAL")
	origFallback := os.Getenv("USE_CLOISTR_FALLBACK")
	origTTL := os.Getenv("RELAY_PREFS_CACHE_TTL")
	defer func() {
		os.Setenv("DISCOVERY_INTERNAL", origInternal)
		os.Setenv("RELAY_LIST", origRelayList)
		os.Setenv("DISCOVERY_EXTERNAL", origExternal)
		os.Setenv("USE_CLOISTR_FALLBACK", origFallback)
		os.Setenv("RELAY_PREFS_CACHE_TTL", origTTL)
	}()

	// Test with all env vars set
	os.Setenv("DISCOVERY_INTERNAL", "http://internal:8080")
	os.Setenv("RELAY_LIST", "wss://relay1.com, wss://relay2.com")
	os.Setenv("DISCOVERY_EXTERNAL", "http://external:8080")
	os.Setenv("USE_CLOISTR_FALLBACK", "false")
	os.Setenv("RELAY_PREFS_CACHE_TTL", "30m")

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
	// Clear env vars
	os.Unsetenv("DISCOVERY_INTERNAL")
	os.Unsetenv("RELAY_LIST")
	os.Unsetenv("DISCOVERY_EXTERNAL")
	os.Unsetenv("USE_CLOISTR_FALLBACK")
	os.Unsetenv("RELAY_PREFS_CACHE_TTL")

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
