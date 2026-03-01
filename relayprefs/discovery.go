package relayprefs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// discoveryResponse is the expected response from a discovery service.
type discoveryResponse struct {
	Pubkey   string        `json:"pubkey"`
	Relays   []RelayConfig `json:"relays"`
	Source   string        `json:"source"`
	CachedAt string        `json:"cached_at,omitempty"`
}

// queryDiscovery queries a discovery service for relay preferences.
func queryDiscovery(ctx context.Context, baseURL, pubkey string) (*RelayPrefs, error) {
	url := fmt.Sprintf("%s/api/v1/relay-prefs/%s", baseURL, pubkey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No preferences found - not an error, just no data
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var dr discoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(dr.Relays) == 0 {
		// Empty relay list - treat as not found
		return nil, nil
	}

	return &RelayPrefs{
		Pubkey: dr.Pubkey,
		Relays: dr.Relays,
		Source: dr.Source,
	}, nil
}
