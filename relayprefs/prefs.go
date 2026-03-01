// Package relayprefs provides relay preference discovery for Cloistr services.
//
// Users store their relay preferences as signed Nostr events (kind:30078, d-tag "cloistr-relays").
// This package handles discovering those preferences through a configurable query chain,
// falling back through discovery services, direct relay queries, NIP-65, and defaults.
package relayprefs

// RelayPrefs represents a user's relay preferences.
type RelayPrefs struct {
	// Pubkey is the user's public key these preferences belong to.
	Pubkey string `json:"pubkey"`

	// Relays is the list of relay configurations.
	Relays []RelayConfig `json:"relays"`

	// Source indicates where these preferences came from.
	// Possible values: "cloistr-relays", "nip65", "config", "default"
	Source string `json:"source"`
}

// RelayConfig represents a single relay's read/write configuration.
type RelayConfig struct {
	URL   string `json:"url"`
	Read  bool   `json:"read"`
	Write bool   `json:"write"`
}

// ReadRelays returns URLs of relays configured for reading.
func (p *RelayPrefs) ReadRelays() []string {
	var relays []string
	for _, r := range p.Relays {
		if r.Read {
			relays = append(relays, r.URL)
		}
	}
	return relays
}

// WriteRelays returns URLs of relays configured for writing.
func (p *RelayPrefs) WriteRelays() []string {
	var relays []string
	for _, r := range p.Relays {
		if r.Write {
			relays = append(relays, r.URL)
		}
	}
	return relays
}

// AllRelays returns all unique relay URLs regardless of read/write config.
func (p *RelayPrefs) AllRelays() []string {
	seen := make(map[string]bool)
	var relays []string
	for _, r := range p.Relays {
		if !seen[r.URL] {
			seen[r.URL] = true
			relays = append(relays, r.URL)
		}
	}
	return relays
}

// HasRelays returns true if there is at least one relay configured.
func (p *RelayPrefs) HasRelays() bool {
	return len(p.Relays) > 0
}
