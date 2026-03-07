package platform

import (
	"context"
	"testing"
)

func TestHasAccessStandalone(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		pubkey     string
		wantAccess bool
	}{
		{
			name: "allowed pubkey",
			config: Config{
				Mode:             ModeStandalone,
				ServiceID:        "test",
				WhitelistPubkeys: []string{"abc123", "def456"},
			},
			pubkey:     "abc123",
			wantAccess: true,
		},
		{
			name: "denied pubkey",
			config: Config{
				Mode:             ModeStandalone,
				ServiceID:        "test",
				WhitelistPubkeys: []string{"abc123", "def456"},
			},
			pubkey:     "xyz789",
			wantAccess: false,
		},
		{
			name: "allow all",
			config: Config{
				Mode:              ModeStandalone,
				ServiceID:         "test",
				WhitelistAllowAll: true,
			},
			pubkey:     "anypubkey",
			wantAccess: true,
		},
		{
			name: "empty whitelist",
			config: Config{
				Mode:             ModeStandalone,
				ServiceID:        "test",
				WhitelistPubkeys: []string{},
			},
			pubkey:     "abc123",
			wantAccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{config: tt.config}
			got, err := client.HasAccess(context.Background(), tt.pubkey)
			if err != nil {
				t.Fatalf("HasAccess() error = %v", err)
			}
			if got != tt.wantAccess {
				t.Errorf("HasAccess() = %v, want %v", got, tt.wantAccess)
			}
		})
	}
}

func TestIsAdminStandalone(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		pubkey    string
		wantAdmin bool
	}{
		{
			name: "is admin",
			config: Config{
				Mode:         ModeStandalone,
				ServiceID:    "test",
				AdminPubkeys: []string{"admin1", "admin2"},
			},
			pubkey:    "admin1",
			wantAdmin: true,
		},
		{
			name: "not admin",
			config: Config{
				Mode:         ModeStandalone,
				ServiceID:    "test",
				AdminPubkeys: []string{"admin1", "admin2"},
			},
			pubkey:    "user1",
			wantAdmin: false,
		},
		{
			name: "no admins configured",
			config: Config{
				Mode:         ModeStandalone,
				ServiceID:    "test",
				AdminPubkeys: []string{},
			},
			pubkey:    "anyone",
			wantAdmin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{config: tt.config}
			got, err := client.IsAdmin(context.Background(), tt.pubkey)
			if err != nil {
				t.Fatalf("IsAdmin() error = %v", err)
			}
			if got != tt.wantAdmin {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.wantAdmin)
			}
		})
	}
}

func TestRequireAccess(t *testing.T) {
	client := &Client{
		config: Config{
			Mode:             ModeStandalone,
			ServiceID:        "test",
			WhitelistPubkeys: []string{"allowed"},
		},
	}

	// Should succeed for allowed pubkey
	if err := client.RequireAccess(context.Background(), "allowed"); err != nil {
		t.Errorf("RequireAccess() for allowed pubkey returned error: %v", err)
	}

	// Should fail for denied pubkey
	if err := client.RequireAccess(context.Background(), "denied"); err != ErrAccessDenied {
		t.Errorf("RequireAccess() for denied pubkey = %v, want %v", err, ErrAccessDenied)
	}
}

func TestRequireAdmin(t *testing.T) {
	client := &Client{
		config: Config{
			Mode:         ModeStandalone,
			ServiceID:    "test",
			AdminPubkeys: []string{"admin"},
		},
	}

	// Should succeed for admin
	if err := client.RequireAdmin(context.Background(), "admin"); err != nil {
		t.Errorf("RequireAdmin() for admin returned error: %v", err)
	}

	// Should fail for non-admin
	if err := client.RequireAdmin(context.Background(), "user"); err != ErrNotAdmin {
		t.Errorf("RequireAdmin() for non-admin = %v, want %v", err, ErrNotAdmin)
	}
}
