package relayprefs

import (
	"context"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

const (
	// KindAppData is NIP-78 application-specific data
	KindAppData = 30078

	// KindRelayList is NIP-65 relay list metadata
	KindRelayList = 10002

	// DTagCloistrRelays is our d-tag for relay preferences
	DTagCloistrRelays = "cloistr-relays"
)

// queryRelayForCloistrPrefs queries a relay directly for kind:30078 d=cloistr-relays
func queryRelayForCloistrPrefs(ctx context.Context, relayURL, pubkey string) (*RelayPrefs, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	relay, err := nostr.RelayConnect(ctx, relayURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to relay: %w", err)
	}
	defer relay.Close()

	// Query for kind:30078 with d-tag "cloistr-relays"
	filter := nostr.Filter{
		Authors: []string{pubkey},
		Kinds:   []int{KindAppData},
		Tags: nostr.TagMap{
			"d": []string{DTagCloistrRelays},
		},
		Limit: 1,
	}

	events, err := relay.QuerySync(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("querying relay: %w", err)
	}

	if len(events) == 0 {
		return nil, nil // Not found
	}

	return parseCloistrRelaysEvent(events[0])
}

// queryRelayForNIP65 queries a relay for kind:10002 (NIP-65 relay list)
func queryRelayForNIP65(ctx context.Context, relayURL, pubkey string) (*RelayPrefs, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	relay, err := nostr.RelayConnect(ctx, relayURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to relay: %w", err)
	}
	defer relay.Close()

	filter := nostr.Filter{
		Authors: []string{pubkey},
		Kinds:   []int{KindRelayList},
		Limit:   1,
	}

	events, err := relay.QuerySync(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("querying relay: %w", err)
	}

	if len(events) == 0 {
		return nil, nil // Not found
	}

	return parseNIP65Event(events[0])
}

// parseCloistrRelaysEvent parses a kind:30078 event into RelayPrefs
func parseCloistrRelaysEvent(event *nostr.Event) (*RelayPrefs, error) {
	prefs := &RelayPrefs{
		Pubkey: event.PubKey,
		Source: "cloistr-relays",
	}

	// Parse "r" tags following NIP-65 conventions
	for _, tag := range event.Tags {
		if len(tag) < 2 || tag[0] != "r" {
			continue
		}

		url := tag[1]
		rc := RelayConfig{URL: url}

		if len(tag) == 2 {
			// No marker = read + write
			rc.Read = true
			rc.Write = true
		} else {
			// Has marker
			marker := tag[2]
			switch marker {
			case "read":
				rc.Read = true
			case "write":
				rc.Write = true
			default:
				// Unknown marker - treat as read+write
				rc.Read = true
				rc.Write = true
			}
		}

		prefs.Relays = append(prefs.Relays, rc)
	}

	if len(prefs.Relays) == 0 {
		return nil, nil // No relays in event
	}

	return prefs, nil
}

// parseNIP65Event parses a kind:10002 event into RelayPrefs
func parseNIP65Event(event *nostr.Event) (*RelayPrefs, error) {
	prefs := &RelayPrefs{
		Pubkey: event.PubKey,
		Source: "nip65",
	}

	// NIP-65 uses "r" tags with optional read/write markers
	for _, tag := range event.Tags {
		if len(tag) < 2 || tag[0] != "r" {
			continue
		}

		url := tag[1]
		rc := RelayConfig{URL: url}

		if len(tag) == 2 {
			// No marker = read + write
			rc.Read = true
			rc.Write = true
		} else {
			marker := tag[2]
			switch marker {
			case "read":
				rc.Read = true
			case "write":
				rc.Write = true
			default:
				rc.Read = true
				rc.Write = true
			}
		}

		prefs.Relays = append(prefs.Relays, rc)
	}

	if len(prefs.Relays) == 0 {
		return nil, nil
	}

	return prefs, nil
}
