package relayprefs

import (
	"context"
	"fmt"
	"log"
)

// Client provides relay preference discovery for Cloistr services.
type Client struct {
	config Config
	cache  *cache
}

// NewClient creates a new relay preferences client with the given configuration.
func NewClient(cfg Config) *Client {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = DefaultCacheTTL
	}

	return &Client{
		config: cfg,
		cache:  newCache(cfg.CacheTTL),
	}
}

// NewClientFromEnv creates a new client configured from environment variables.
// See ConfigFromEnv for the list of supported environment variables.
func NewClientFromEnv() *Client {
	return NewClient(ConfigFromEnv())
}

// GetRelayPrefs retrieves relay preferences for a pubkey.
//
// The query chain is:
//  1. Local cache
//  2. DISCOVERY_INTERNAL (if configured)
//  3. RELAY_LIST (if configured) - direct relay query for kind:30078
//  4. DISCOVERY_EXTERNAL (if configured)
//  5. discover.cloistr.xyz (if UseCloistrFallback=true)
//  6. relay.cloistr.xyz (if UseCloistrFallback=true)
//
// If no cloistr-relays event is found, falls back to NIP-65, then config defaults.
func (c *Client) GetRelayPrefs(ctx context.Context, pubkey string) (*RelayPrefs, error) {
	// 1. Check cache first
	if prefs, ok := c.cache.Get(pubkey); ok {
		return prefs, nil
	}

	// Try the query chain for cloistr-relays
	prefs, err := c.queryChain(ctx, pubkey, true)
	if err != nil {
		return nil, err
	}

	if prefs != nil {
		c.cache.Set(pubkey, prefs)
		return prefs, nil
	}

	// No cloistr-relays found - try NIP-65 fallback
	prefs, err = c.queryChain(ctx, pubkey, false)
	if err != nil {
		return nil, err
	}

	if prefs != nil {
		c.cache.Set(pubkey, prefs)
		return prefs, nil
	}

	// Still nothing - use configured defaults or Cloistr relay
	prefs = c.defaultPrefs(pubkey)
	c.cache.Set(pubkey, prefs)
	return prefs, nil
}

// queryChain tries each source in order until preferences are found.
// If cloistrRelays is true, queries for kind:30078 d=cloistr-relays.
// If false, queries for kind:10002 (NIP-65).
func (c *Client) queryChain(ctx context.Context, pubkey string, cloistrRelays bool) (*RelayPrefs, error) {
	// 2. Internal discovery
	if c.config.InternalDiscovery != "" {
		prefs, err := queryDiscovery(ctx, c.config.InternalDiscovery, pubkey)
		if err != nil {
			log.Printf("relayprefs: internal discovery error: %v", err)
		} else if prefs != nil {
			return prefs, nil
		}
	}

	// 3. Direct relay queries
	for _, relayURL := range c.config.QueryRelays {
		var prefs *RelayPrefs
		var err error
		if cloistrRelays {
			prefs, err = queryRelayForCloistrPrefs(ctx, relayURL, pubkey)
		} else {
			prefs, err = queryRelayForNIP65(ctx, relayURL, pubkey)
		}
		if err != nil {
			log.Printf("relayprefs: relay query error (%s): %v", relayURL, err)
			continue
		}
		if prefs != nil {
			return prefs, nil
		}
	}

	// 4. External discovery
	if c.config.ExternalDiscovery != "" {
		prefs, err := queryDiscovery(ctx, c.config.ExternalDiscovery, pubkey)
		if err != nil {
			log.Printf("relayprefs: external discovery error: %v", err)
		} else if prefs != nil {
			return prefs, nil
		}
	}

	// 5 & 6. Cloistr fallback
	if c.config.UseCloistrFallback {
		// Try Cloistr discovery first (faster)
		prefs, err := queryDiscovery(ctx, DefaultCloistrDiscovery, pubkey)
		if err != nil {
			log.Printf("relayprefs: cloistr discovery error: %v", err)
		} else if prefs != nil {
			return prefs, nil
		}

		// Fall back to direct relay query
		var queryErr error
		if cloistrRelays {
			prefs, queryErr = queryRelayForCloistrPrefs(ctx, DefaultCloistrRelay, pubkey)
		} else {
			prefs, queryErr = queryRelayForNIP65(ctx, DefaultCloistrRelay, pubkey)
		}
		if queryErr != nil {
			log.Printf("relayprefs: cloistr relay error: %v", queryErr)
		} else if prefs != nil {
			return prefs, nil
		}
	}

	return nil, nil
}

// defaultPrefs returns default relay preferences when nothing is found.
func (c *Client) defaultPrefs(pubkey string) *RelayPrefs {
	prefs := &RelayPrefs{
		Pubkey: pubkey,
		Source: "default",
	}

	// Use configured query relays if available
	if len(c.config.QueryRelays) > 0 {
		prefs.Source = "config"
		for _, url := range c.config.QueryRelays {
			prefs.Relays = append(prefs.Relays, RelayConfig{
				URL:   url,
				Read:  true,
				Write: true,
			})
		}
		return prefs
	}

	// Otherwise use Cloistr relay if fallback enabled
	if c.config.UseCloistrFallback {
		prefs.Relays = []RelayConfig{
			{URL: DefaultCloistrRelay, Read: true, Write: true},
		}
		return prefs
	}

	// No relays available at all - this is a configuration error
	// but we return empty prefs rather than failing
	return prefs
}

// InvalidateCache removes cached preferences for a pubkey.
// Call this when a user updates their relay preferences.
func (c *Client) InvalidateCache(pubkey string) {
	c.cache.Invalidate(pubkey)
}

// InvalidateAllCache clears the entire cache.
func (c *Client) InvalidateAllCache() {
	c.cache.InvalidateAll()
}

// Config returns the client's configuration (read-only copy).
func (c *Client) Config() Config {
	return c.config
}

// Validate checks that the client can actually query for preferences.
// Returns an error if no query sources are configured and Cloistr fallback is disabled.
func (c *Client) Validate() error {
	if !c.config.HasQuerySources() && !c.config.UseCloistrFallback {
		return fmt.Errorf("no relay preference sources configured and Cloistr fallback disabled")
	}
	return nil
}
