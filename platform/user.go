package platform

import (
	"context"
	"fmt"
)

// EnsureUser idempotently creates the platform users row for pubkey.
//
// has_service_access() checks users.enabled before the free-tier shortcut, and
// user_quota_usage has a foreign key on users, so a pubkey with no users row is
// denied everything and cannot record usage. Keys that never pass through
// cloistr-me (extension users, headless/fleet keys) have no row until something
// creates one.
//
// The insert goes through public.ensure_user(), a SECURITY DEFINER function
// owned by the platform schema (cloistr-me migrations). Service roles have no
// INSERT on public.users, and their per-role search_path puts their own schema
// first (several have their own "users" table), so a plain INSERT from a
// service fails or lands in the wrong table. The function pins
// search_path=public and leaves existing rows untouched, so a disabled user
// stays disabled. Safe to call on every authenticated request.
//
// In standalone mode there is no users table and this is a no-op.
func (c *Client) EnsureUser(ctx context.Context, pubkey string) error {
	if c.config.Mode == ModeStandalone {
		return nil
	}
	_, err := c.db.ExecContext(ctx, `SELECT public.ensure_user($1)`, pubkey)
	if err != nil {
		return fmt.Errorf("ensure user: %w", err)
	}
	return nil
}
