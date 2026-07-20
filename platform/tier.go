package platform

import "context"

// Tier is a user's identity state, which scales quotas and send rights.
//
//   - TierAnonymous: extension/NIP-07 login or an auto-assigned address only. 100 MB
//     storage floor, receive-only email. Closes the sybil hole (infinite keys would
//     otherwise mean infinite free 1 GiB pools).
//   - TierNamed: claimed a real (non-auto-assigned) address. 1 GiB storage, send+receive.
//   - TierPaid: holds a non-expired quota grant (top-up) or an active subscription.
type Tier string

const (
	TierAnonymous Tier = "anonymous"
	TierNamed     Tier = "named"
	TierPaid      Tier = "paid"
)

// GetTier resolves a user's identity tier.
//
// In platform mode this is backed by the get_user_tier() DB function. In standalone
// (self-host) mode there is no shared identity database, so every user is treated as
// TierNamed — self-hosters get the full (non-anonymous) limits by default and gate
// abuse controls via their own config instead.
func (c *Client) GetTier(ctx context.Context, pubkey string) (Tier, error) {
	if c.config.Mode == ModeStandalone {
		return TierNamed, nil
	}

	var tier string
	err := c.db.QueryRowContext(ctx, "SELECT get_user_tier($1)", pubkey).Scan(&tier)
	if err != nil {
		return TierAnonymous, err
	}
	switch Tier(tier) {
	case TierAnonymous, TierNamed, TierPaid:
		return Tier(tier), nil
	default:
		// Unknown value from the DB: fail closed to the most restrictive tier.
		return TierAnonymous, nil
	}
}
