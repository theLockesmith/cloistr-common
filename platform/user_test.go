package platform

import (
	"context"
	"testing"
)

func TestEnsureUserStandaloneIsNoop(t *testing.T) {
	// No db is set: a standalone client must never touch it.
	client := &Client{config: Config{Mode: ModeStandalone, ServiceID: "test"}}

	if err := client.EnsureUser(context.Background(), "anypubkey"); err != nil {
		t.Errorf("EnsureUser() in standalone mode returned error: %v", err)
	}
}
