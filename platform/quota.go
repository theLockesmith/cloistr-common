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
//
// Storage is a single pool shared across every storage-consuming service (blossom,
// drive, photos, documents, vault, tasks, calendar, and email) — there is no separate
// email storage quota; email usage is recorded against storage_bytes with service='email'.
const (
	QuotaTypeStorageBytes         = "storage_bytes"
	QuotaTypeSigningRequestsDaily = "signing_requests_daily"
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

// getQuotaPlatform retrieves quota from the database via effective_quota(), which
// resolves the identity-scaled limit (anonymous vs named vs per-user override) plus
// any non-expired top-up grants, minus the SUM of per-service usage in user_quota_usage.
// A Limit of 0 means unlimited (see QuotaInfo.Unlimited()).
func (c *Client) getQuotaPlatform(ctx context.Context, pubkey string, quotaType string) (*QuotaInfo, error) {
	var limit, usage, remaining int64

	err := c.db.QueryRowContext(ctx,
		"SELECT quota_limit, current_usage, remaining FROM effective_quota($1, $2)",
		pubkey, quotaType,
	).Scan(&limit, &usage, &remaining)

	if err == sql.ErrNoRows {
		// effective_quota always returns exactly one row; treat an empty result as unlimited.
		return &QuotaInfo{}, nil
	}
	if err != nil {
		return nil, err
	}

	return &QuotaInfo{
		Limit:        limit,
		CurrentUsage: usage,
		Remaining:    remaining,
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

// recordUsagePlatform records this service's usage component in user_quota_usage.
// Usage is tracked per (pubkey, quota_type, service); the user's total usage is the
// SUM across services (see effective_quota()). The delta is additive and clamped at 0
// so a decrement (release) never drives a component negative. A single UPSERT — no
// transaction needed.
//
// NOTE: the pubkey must already have a users row (FK on user_quota_usage). Services
// route auth through cloistr-me, which auto-provisions the users row on first touch.
func (c *Client) recordUsagePlatform(ctx context.Context, pubkey string, quotaType string, amount int64) error {
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO user_quota_usage (pubkey, quota_type_id, service, bytes, updated_at)
		VALUES ($1, $2, $3, GREATEST(0, $4::BIGINT), NOW())
		ON CONFLICT (pubkey, quota_type_id, service) DO UPDATE
		SET bytes = GREATEST(0, user_quota_usage.bytes + $4::BIGINT),
		    updated_at = NOW()
	`, pubkey, quotaType, c.config.ServiceID, amount)
	return err
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
