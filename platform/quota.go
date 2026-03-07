package platform

import (
	"context"
	"database/sql"
	"sync"
)

// QuotaInfo contains information about a user's quota.
type QuotaInfo struct {
	Limit         int64 // Maximum allowed (0 = unlimited)
	CurrentUsage  int64 // Current usage
	Remaining     int64 // Remaining capacity
	IsTenantQuota bool  // True if this is a tenant-level quota
}

// Unlimited returns true if the quota has no limit.
func (q QuotaInfo) Unlimited() bool {
	return q.Limit == 0
}

// Standard quota type IDs (must match database).
const (
	QuotaTypeStorageBytes         = "storage_bytes"
	QuotaTypeSigningRequestsDaily = "signing_requests_daily"
	QuotaTypeEmailStorageBytes    = "email_storage_bytes"
)

// standaloneUsage tracks usage in standalone mode (in-memory).
var (
	standaloneUsage   = make(map[string]map[string]int64) // pubkey -> quotaType -> usage
	standaloneUsageMu sync.RWMutex
)

// GetQuota retrieves quota information for a user.
func (c *Client) GetQuota(ctx context.Context, pubkey string, quotaType string) (*QuotaInfo, error) {
	if c.config.Mode == ModeStandalone {
		return c.getQuotaStandalone(pubkey, quotaType)
	}
	return c.getQuotaPlatform(ctx, pubkey, quotaType)
}

// getQuotaPlatform retrieves quota from the database.
func (c *Client) getQuotaPlatform(ctx context.Context, pubkey string, quotaType string) (*QuotaInfo, error) {
	var limit, usage, remaining int64
	var isTenant bool

	err := c.db.QueryRowContext(ctx,
		"SELECT quota_limit, current_usage, remaining, is_tenant_quota FROM get_user_quota($1, $2)",
		pubkey, quotaType,
	).Scan(&limit, &usage, &remaining, &isTenant)

	if err == sql.ErrNoRows {
		// No quota record means unlimited
		return &QuotaInfo{Limit: 0, CurrentUsage: 0, Remaining: 0, IsTenantQuota: false}, nil
	}
	if err != nil {
		return nil, err
	}

	return &QuotaInfo{
		Limit:         limit,
		CurrentUsage:  usage,
		Remaining:     remaining,
		IsTenantQuota: isTenant,
	}, nil
}

// getQuotaStandalone retrieves quota from in-memory tracking.
func (c *Client) getQuotaStandalone(pubkey string, quotaType string) (*QuotaInfo, error) {
	standaloneUsageMu.RLock()
	defer standaloneUsageMu.RUnlock()

	// Get configured limit based on quota type
	var limit int64
	switch quotaType {
	case QuotaTypeStorageBytes:
		limit = c.config.StorageQuotaBytes
	case QuotaTypeSigningRequestsDaily:
		limit = c.config.SigningRequestsMax
	default:
		limit = 0 // Unlimited by default
	}

	// Get current usage
	var usage int64
	if userUsage, ok := standaloneUsage[pubkey]; ok {
		usage = userUsage[quotaType]
	}

	var remaining int64
	if limit > 0 {
		remaining = limit - usage
		if remaining < 0 {
			remaining = 0
		}
	}

	return &QuotaInfo{
		Limit:         limit,
		CurrentUsage:  usage,
		Remaining:     remaining,
		IsTenantQuota: false,
	}, nil
}

// CheckQuota checks if a user has enough quota for an operation.
// Returns true if the operation can proceed, false otherwise.
func (c *Client) CheckQuota(ctx context.Context, pubkey string, quotaType string, requiredAmount int64) (bool, error) {
	quota, err := c.GetQuota(ctx, pubkey, quotaType)
	if err != nil {
		return false, err
	}

	// Unlimited quota
	if quota.Unlimited() {
		return true, nil
	}

	// Check if there's enough remaining
	return quota.Remaining >= requiredAmount, nil
}

// RequireQuota is a convenience method that returns an error if quota is insufficient.
func (c *Client) RequireQuota(ctx context.Context, pubkey string, quotaType string, requiredAmount int64) error {
	ok, err := c.CheckQuota(ctx, pubkey, quotaType, requiredAmount)
	if err != nil {
		return err
	}
	if !ok {
		return ErrQuotaExceeded
	}
	return nil
}

// RecordUsage records usage and updates the quota.
// This should be called after a successful operation.
func (c *Client) RecordUsage(ctx context.Context, pubkey string, quotaType string, amount int64) error {
	if c.config.Mode == ModeStandalone {
		return c.recordUsageStandalone(pubkey, quotaType, amount)
	}
	return c.recordUsagePlatform(ctx, pubkey, quotaType, amount)
}

// recordUsagePlatform records usage in the database.
func (c *Client) recordUsagePlatform(ctx context.Context, pubkey string, quotaType string, amount int64) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert usage record
	_, err = tx.ExecContext(ctx, `
		INSERT INTO usage_records (pubkey, quota_type_id, service_id, amount)
		VALUES ($1, $2, $3, $4)
	`, pubkey, quotaType, c.config.ServiceID, amount)
	if err != nil {
		return err
	}

	// Update current usage in user_quotas
	// First ensure the quota record exists
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_quotas (pubkey, quota_type_id, quota_limit, current_usage)
		SELECT $1, $2, COALESCE(
			(SELECT default_limit FROM quota_types WHERE id = $2), 0
		), $3
		ON CONFLICT (pubkey, quota_type_id) DO UPDATE
		SET current_usage = user_quotas.current_usage + $3,
		    last_updated = NOW()
	`, pubkey, quotaType, amount)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// recordUsageStandalone records usage in memory.
func (c *Client) recordUsageStandalone(pubkey string, quotaType string, amount int64) error {
	standaloneUsageMu.Lock()
	defer standaloneUsageMu.Unlock()

	if _, ok := standaloneUsage[pubkey]; !ok {
		standaloneUsage[pubkey] = make(map[string]int64)
	}
	standaloneUsage[pubkey][quotaType] += amount
	return nil
}

// ReleaseUsage decreases usage (e.g., when a file is deleted).
func (c *Client) ReleaseUsage(ctx context.Context, pubkey string, quotaType string, amount int64) error {
	// Record negative usage
	return c.RecordUsage(ctx, pubkey, quotaType, -amount)
}

// ResetStandaloneUsage resets all standalone mode usage tracking (useful for testing).
func ResetStandaloneUsage() {
	standaloneUsageMu.Lock()
	defer standaloneUsageMu.Unlock()
	standaloneUsage = make(map[string]map[string]int64)
}
