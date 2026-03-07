package platform

import (
	"context"
	"testing"
)

func TestGetQuotaStandalone(t *testing.T) {
	// Reset usage before tests
	ResetStandaloneUsage()

	client := &Client{
		config: Config{
			Mode:              ModeStandalone,
			ServiceID:         "test",
			StorageQuotaBytes: 1024 * 1024 * 100, // 100MB
		},
	}

	// Get quota for user with no usage
	quota, err := client.GetQuota(context.Background(), "user1", QuotaTypeStorageBytes)
	if err != nil {
		t.Fatalf("GetQuota() error = %v", err)
	}

	if quota.Limit != 1024*1024*100 {
		t.Errorf("Limit = %d, want %d", quota.Limit, 1024*1024*100)
	}
	if quota.CurrentUsage != 0 {
		t.Errorf("CurrentUsage = %d, want 0", quota.CurrentUsage)
	}
	if quota.Remaining != 1024*1024*100 {
		t.Errorf("Remaining = %d, want %d", quota.Remaining, 1024*1024*100)
	}
}

func TestCheckQuotaStandalone(t *testing.T) {
	// Reset usage before tests
	ResetStandaloneUsage()

	client := &Client{
		config: Config{
			Mode:              ModeStandalone,
			ServiceID:         "test",
			StorageQuotaBytes: 100, // 100 bytes
		},
	}

	// Should have enough quota
	ok, err := client.CheckQuota(context.Background(), "user1", QuotaTypeStorageBytes, 50)
	if err != nil {
		t.Fatalf("CheckQuota() error = %v", err)
	}
	if !ok {
		t.Error("CheckQuota(50) = false, want true")
	}

	// Record some usage
	if err := client.RecordUsage(context.Background(), "user1", QuotaTypeStorageBytes, 60); err != nil {
		t.Fatalf("RecordUsage() error = %v", err)
	}

	// Now should not have enough quota for 50 more bytes
	ok, err = client.CheckQuota(context.Background(), "user1", QuotaTypeStorageBytes, 50)
	if err != nil {
		t.Fatalf("CheckQuota() error = %v", err)
	}
	if ok {
		t.Error("CheckQuota(50) after 60 used = true, want false")
	}

	// But should have enough for 40 bytes
	ok, err = client.CheckQuota(context.Background(), "user1", QuotaTypeStorageBytes, 40)
	if err != nil {
		t.Fatalf("CheckQuota() error = %v", err)
	}
	if !ok {
		t.Error("CheckQuota(40) after 60 used = false, want true")
	}
}

func TestRecordUsageStandalone(t *testing.T) {
	// Reset usage before tests
	ResetStandaloneUsage()

	client := &Client{
		config: Config{
			Mode:              ModeStandalone,
			ServiceID:         "test",
			StorageQuotaBytes: 1000,
		},
	}

	// Record usage for user1
	if err := client.RecordUsage(context.Background(), "user1", QuotaTypeStorageBytes, 100); err != nil {
		t.Fatalf("RecordUsage() error = %v", err)
	}

	// Check that usage is tracked
	quota, err := client.GetQuota(context.Background(), "user1", QuotaTypeStorageBytes)
	if err != nil {
		t.Fatalf("GetQuota() error = %v", err)
	}
	if quota.CurrentUsage != 100 {
		t.Errorf("CurrentUsage = %d, want 100", quota.CurrentUsage)
	}

	// Record more usage
	if err := client.RecordUsage(context.Background(), "user1", QuotaTypeStorageBytes, 50); err != nil {
		t.Fatalf("RecordUsage() error = %v", err)
	}

	// Check cumulative usage
	quota, err = client.GetQuota(context.Background(), "user1", QuotaTypeStorageBytes)
	if err != nil {
		t.Fatalf("GetQuota() error = %v", err)
	}
	if quota.CurrentUsage != 150 {
		t.Errorf("CurrentUsage = %d, want 150", quota.CurrentUsage)
	}
}

func TestReleaseUsageStandalone(t *testing.T) {
	// Reset usage before tests
	ResetStandaloneUsage()

	client := &Client{
		config: Config{
			Mode:              ModeStandalone,
			ServiceID:         "test",
			StorageQuotaBytes: 1000,
		},
	}

	// Record initial usage
	if err := client.RecordUsage(context.Background(), "user1", QuotaTypeStorageBytes, 200); err != nil {
		t.Fatalf("RecordUsage() error = %v", err)
	}

	// Release some
	if err := client.ReleaseUsage(context.Background(), "user1", QuotaTypeStorageBytes, 50); err != nil {
		t.Fatalf("ReleaseUsage() error = %v", err)
	}

	// Check remaining usage
	quota, err := client.GetQuota(context.Background(), "user1", QuotaTypeStorageBytes)
	if err != nil {
		t.Fatalf("GetQuota() error = %v", err)
	}
	if quota.CurrentUsage != 150 {
		t.Errorf("CurrentUsage after release = %d, want 150", quota.CurrentUsage)
	}
}

func TestUnlimitedQuota(t *testing.T) {
	// Reset usage before tests
	ResetStandaloneUsage()

	client := &Client{
		config: Config{
			Mode:              ModeStandalone,
			ServiceID:         "test",
			StorageQuotaBytes: 0, // Unlimited
		},
	}

	quota, err := client.GetQuota(context.Background(), "user1", QuotaTypeStorageBytes)
	if err != nil {
		t.Fatalf("GetQuota() error = %v", err)
	}
	if !quota.Unlimited() {
		t.Error("Unlimited() = false, want true")
	}

	// Should always have enough quota when unlimited
	ok, err := client.CheckQuota(context.Background(), "user1", QuotaTypeStorageBytes, 999999999999)
	if err != nil {
		t.Fatalf("CheckQuota() error = %v", err)
	}
	if !ok {
		t.Error("CheckQuota() with unlimited quota = false, want true")
	}
}

func TestRequireQuota(t *testing.T) {
	// Reset usage before tests
	ResetStandaloneUsage()

	client := &Client{
		config: Config{
			Mode:              ModeStandalone,
			ServiceID:         "test",
			StorageQuotaBytes: 100,
		},
	}

	// Use up most of the quota
	if err := client.RecordUsage(context.Background(), "user1", QuotaTypeStorageBytes, 80); err != nil {
		t.Fatalf("RecordUsage() error = %v", err)
	}

	// Should succeed for small amount
	if err := client.RequireQuota(context.Background(), "user1", QuotaTypeStorageBytes, 20); err != nil {
		t.Errorf("RequireQuota(20) returned error: %v", err)
	}

	// Should fail for too much
	if err := client.RequireQuota(context.Background(), "user1", QuotaTypeStorageBytes, 30); err != ErrQuotaExceeded {
		t.Errorf("RequireQuota(30) = %v, want %v", err, ErrQuotaExceeded)
	}
}
