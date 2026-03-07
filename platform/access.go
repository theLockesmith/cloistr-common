package platform

import (
	"context"
	"database/sql"
)

// HasAccess checks if a user has access to a specific service.
// In platform mode, this queries the database using the has_service_access() function.
// In standalone mode, this checks against the whitelist configuration.
func (c *Client) HasAccess(ctx context.Context, pubkey string) (bool, error) {
	return c.HasAccessToService(ctx, pubkey, c.config.ServiceID)
}

// HasAccessToService checks if a user has access to any service.
// Use this when you need to check access to a service other than the current one.
func (c *Client) HasAccessToService(ctx context.Context, pubkey string, serviceID string) (bool, error) {
	if c.config.Mode == ModeStandalone {
		return c.hasAccessStandalone(pubkey)
	}
	return c.hasAccessPlatform(ctx, pubkey, serviceID)
}

// hasAccessPlatform checks access using the database.
func (c *Client) hasAccessPlatform(ctx context.Context, pubkey string, serviceID string) (bool, error) {
	var hasAccess bool
	err := c.db.QueryRowContext(ctx,
		"SELECT has_service_access($1, $2)",
		pubkey, serviceID,
	).Scan(&hasAccess)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return hasAccess, nil
}

// hasAccessStandalone checks access using the whitelist configuration.
func (c *Client) hasAccessStandalone(pubkey string) (bool, error) {
	// Allow all if configured
	if c.config.WhitelistAllowAll {
		return true, nil
	}

	// Check whitelist
	for _, allowed := range c.config.WhitelistPubkeys {
		if allowed == pubkey {
			return true, nil
		}
	}

	return false, nil
}

// RequireAccess is a convenience method that returns an error if access is denied.
func (c *Client) RequireAccess(ctx context.Context, pubkey string) error {
	hasAccess, err := c.HasAccess(ctx, pubkey)
	if err != nil {
		return err
	}
	if !hasAccess {
		return ErrAccessDenied
	}
	return nil
}

// IsAdmin checks if a user is an admin for the current service.
func (c *Client) IsAdmin(ctx context.Context, pubkey string) (bool, error) {
	return c.IsAdminForService(ctx, pubkey, c.config.ServiceID)
}

// IsAdminForService checks if a user is an admin for a specific service.
func (c *Client) IsAdminForService(ctx context.Context, pubkey string, serviceID string) (bool, error) {
	if c.config.Mode == ModeStandalone {
		return c.isAdminStandalone(pubkey)
	}
	return c.isAdminPlatform(ctx, pubkey, serviceID)
}

// isAdminPlatform checks admin status using the database.
func (c *Client) isAdminPlatform(ctx context.Context, pubkey string, serviceID string) (bool, error) {
	var isAdmin bool
	err := c.db.QueryRowContext(ctx,
		"SELECT is_service_admin($1, $2)",
		pubkey, serviceID,
	).Scan(&isAdmin)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return isAdmin, nil
}

// isAdminStandalone checks admin status using the configuration.
func (c *Client) isAdminStandalone(pubkey string) (bool, error) {
	for _, admin := range c.config.AdminPubkeys {
		if admin == pubkey {
			return true, nil
		}
	}
	return false, nil
}

// RequireAdmin is a convenience method that returns an error if the user is not an admin.
func (c *Client) RequireAdmin(ctx context.Context, pubkey string) error {
	isAdmin, err := c.IsAdmin(ctx, pubkey)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrNotAdmin
	}
	return nil
}
