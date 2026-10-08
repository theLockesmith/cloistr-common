package relayprefs

import (
	"strings"
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

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

// setRequired sets every variable ConfigFromEnv requires, with fallback on.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv(EnvUseCloistrFallback, "true")
	t.Setenv(EnvCloistrDiscovery, "https://discover.example")
	t.Setenv(EnvCloistrRelay, "wss://relay.example")
}

func TestConfigFromEnv_Defaults(t *testing.T) {
	// Empty is treated the same as unset; t.Setenv restores the original after the test.
	t.Setenv("DISCOVERY_INTERNAL", "")
	t.Setenv("RELAY_LIST", "")
	t.Setenv("DISCOVERY_EXTERNAL", "")
	t.Setenv("RELAY_PREFS_CACHE_TTL", "")
	setRequired(t)

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InternalDiscovery != "" {
		t.Errorf("expected empty internal discovery, got %s", cfg.InternalDiscovery)
	}

	if len(cfg.QueryRelays) != 0 {
		t.Errorf("expected 0 query relays, got %d", len(cfg.QueryRelays))
	}

	if !cfg.UseCloistrFallback {
		t.Error("expected UseCloistrFallback to be true")
	}
	if cfg.CloistrDiscovery != "https://discover.example" || cfg.CloistrRelay != "wss://relay.example" {
		t.Errorf("expected Cloistr URLs from env, got %q and %q", cfg.CloistrDiscovery, cfg.CloistrRelay)
	}

	if cfg.CacheTTL != DefaultCacheTTL {
		t.Errorf("expected default cache TTL, got %v", cfg.CacheTTL)
	}
}

func TestConfigFromEnv_RequiredVariables(t *testing.T) {
	tests := []struct {
		name    string
		unset   string
		value   string
		wantVar string
	}{
		{"fallback unset", EnvUseCloistrFallback, "", EnvUseCloistrFallback},
		{"fallback invalid", EnvUseCloistrFallback, "yes", EnvUseCloistrFallback},
		{"discovery unset", EnvCloistrDiscovery, "", EnvCloistrDiscovery},
		{"relay unset", EnvCloistrRelay, "", EnvCloistrRelay},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequired(t)
			t.Setenv(tt.unset, tt.value)

			if _, err := ConfigFromEnv(); err == nil || !strings.Contains(err.Error(), tt.wantVar) {
				t.Fatalf("expected an error naming %s, got %v", tt.wantVar, err)
			}
			if _, err := NewClientFromEnv(); err == nil || !strings.Contains(err.Error(), tt.wantVar) {
				t.Fatalf("NewClientFromEnv: expected an error naming %s, got %v", tt.wantVar, err)
			}
		})
	}
}

func TestConfigFromEnv_FallbackOffNeedsNoURLs(t *testing.T) {
	t.Setenv(EnvUseCloistrFallback, "false")
	t.Setenv(EnvCloistrDiscovery, "")
	t.Setenv(EnvCloistrRelay, "")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UseCloistrFallback {
		t.Error("expected UseCloistrFallback to be false")
	}
}

func TestDefaultPrefs_NoHardcodedHost(t *testing.T) {
	// A Config built in code with fallback on but no URLs must not invent one.
	c := NewClient(Config{UseCloistrFallback: true})
	if prefs := c.defaultPrefs("pk"); len(prefs.Relays) != 0 {
		t.Errorf("expected no default relays, got %v", prefs.Relays)
	}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), EnvCloistrDiscovery) {
		t.Errorf("expected Validate to name %s, got %v", EnvCloistrDiscovery, err)
	}

	c = NewClient(Config{UseCloistrFallback: true, CloistrDiscovery: "https://d", CloistrRelay: "wss://r"})
	if prefs := c.defaultPrefs("pk"); len(prefs.Relays) != 1 || prefs.Relays[0].URL != "wss://r" {
		t.Errorf("expected the configured relay as default, got %v", prefs.Relays)
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
