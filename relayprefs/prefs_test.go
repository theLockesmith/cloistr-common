package relayprefs

import (
	"testing"
)

func TestRelayPrefs_ReadRelays(t *testing.T) {
	prefs := &RelayPrefs{
		Relays: []RelayConfig{
			{URL: "wss://read-write.com", Read: true, Write: true},
			{URL: "wss://read-only.com", Read: true, Write: false},
			{URL: "wss://write-only.com", Read: false, Write: true},
		},
	}

	readRelays := prefs.ReadRelays()
	if len(readRelays) != 2 {
		t.Errorf("expected 2 read relays, got %d", len(readRelays))
	}

	expected := map[string]bool{
		"wss://read-write.com": true,
		"wss://read-only.com":  true,
	}
	for _, r := range readRelays {
		if !expected[r] {
			t.Errorf("unexpected read relay: %s", r)
		}
	}
}

func TestRelayPrefs_WriteRelays(t *testing.T) {
	prefs := &RelayPrefs{
		Relays: []RelayConfig{
			{URL: "wss://read-write.com", Read: true, Write: true},
			{URL: "wss://read-only.com", Read: true, Write: false},
			{URL: "wss://write-only.com", Read: false, Write: true},
		},
	}

	writeRelays := prefs.WriteRelays()
	if len(writeRelays) != 2 {
		t.Errorf("expected 2 write relays, got %d", len(writeRelays))
	}

	expected := map[string]bool{
		"wss://read-write.com":  true,
		"wss://write-only.com": true,
	}
	for _, r := range writeRelays {
		if !expected[r] {
			t.Errorf("unexpected write relay: %s", r)
		}
	}
}

func TestRelayPrefs_AllRelays(t *testing.T) {
	prefs := &RelayPrefs{
		Relays: []RelayConfig{
			{URL: "wss://relay1.com", Read: true, Write: true},
			{URL: "wss://relay2.com", Read: true, Write: false},
			{URL: "wss://relay1.com", Read: false, Write: true}, // Duplicate
		},
	}

	allRelays := prefs.AllRelays()
	if len(allRelays) != 2 {
		t.Errorf("expected 2 unique relays, got %d", len(allRelays))
	}
}

func TestRelayPrefs_HasRelays(t *testing.T) {
	empty := &RelayPrefs{}
	if empty.HasRelays() {
		t.Error("expected empty prefs to return false for HasRelays")
	}

	withRelays := &RelayPrefs{
		Relays: []RelayConfig{{URL: "wss://test.com", Read: true, Write: true}},
	}
	if !withRelays.HasRelays() {
		t.Error("expected prefs with relays to return true for HasRelays")
	}
}
