# CLAUDE.md - cloistr-common

**Shared Go library for Cloistr services.**

## Project Information

- **Company:** Coldforge
- **Type:** Go Library
- **Repository:** `git.aegis-hq.xyz/coldforge/cloistr-common`

**Parent Rules:** See [Coldforge CLAUDE.md](~/claude/coldforge/CLAUDE.md) and [global CLAUDE.md](~/claude/CLAUDE.md).

## Purpose

This library provides shared functionality for all Cloistr services. Currently includes:

- **relayprefs** - Relay preference discovery (kind:30078 cloistr-relays)
- **platform** - Unified access control, quota management, and usage tracking
- **errors** - Standardized API error types (potato-grade format)

## Design Philosophy

See the relay preferences design document for full context:
`~/claude/coldforge/cloistr/architecture/relay-preferences.md`

Key principles:
- **User freedom** - Users control where their data lives
- **Self-hoster friendly** - Works without Cloistr infrastructure if configured
- **Configurable query chain** - Discovery → relays → NIP-65 → defaults
- **Single source of truth** - Logic lives here, services just use it

## Package: relayprefs

### Files

| File | Purpose |
|------|---------|
| `prefs.go` | RelayPrefs type and methods (ReadRelays, WriteRelays) |
| `config.go` | Config struct, environment variable parsing |
| `cache.go` | In-memory TTL cache |
| `discovery.go` | Query discovery service API |
| `direct.go` | Query relays directly for kind:30078 and NIP-65 |
| `client.go` | Main Client type, query chain logic |

### Query Chain

```
1. Local cache
2. DISCOVERY_INTERNAL (self-hosted)
3. RELAY_LIST (direct relay queries)
4. DISCOVERY_EXTERNAL (third-party)
5. discover.cloistr.xyz (Cloistr discovery)
6. relay.cloistr.xyz (Cloistr relay)
```

Then preference chain:
```
cloistr-relays (kind:30078) → NIP-65 (kind:10002) → config defaults → Cloistr relay
```

### Testing

```bash
go test ./...
```

### Adding New Packages

When adding new shared functionality:

1. Create a new package directory
2. Follow the same pattern: types, config, implementation
3. Support environment variable configuration
4. Document in README.md
5. Add section to this CLAUDE.md

## Development

```bash
# Build
go build ./...

# Test
go test ./...

# Update dependencies
go mod tidy
```

## Integration

Services import this library:

```go
import "git.aegis-hq.xyz/coldforge/cloistr-common/relayprefs"

client := relayprefs.NewClientFromEnv()
prefs, _ := client.GetRelayPrefs(ctx, pubkey)
```

See `~/claude/coldforge/cloistr/architecture/relay-preferences.md` for full integration guide.

## Package: platform

Provides unified access control, quota management, and usage tracking for Cloistr services. Supports both platform mode (shared PostgreSQL) and standalone mode (config-based).

### Files

| File | Purpose |
|------|---------|
| `config.go` | Config struct, environment variable parsing, Client type |
| `errors.go` | Error definitions |
| `access.go` | HasAccess, IsAdmin functions |
| `quota.go` | GetQuota, CheckQuota, RecordUsage functions |

### Modes

**Platform Mode:** Services connect to the shared PostgreSQL database at `postgres-rw.db.coldforge.xyz`. Access control uses the `has_service_access()` function, quotas use the `user_quotas` table.

**Standalone Mode:** Services use environment variables and config files. No database required - useful for self-hosted deployments and open source users.

### Environment Variables

```bash
# Mode selection
CLOISTR_MODE=platform  # or "standalone"

# Platform mode
DATABASE_URL=postgres://user:pass@postgres-rw.db.coldforge.xyz:5432/cloistr
SERVICE_ID=blossom

# Standalone mode
WHITELIST_PUBKEYS=pubkey1,pubkey2
WHITELIST_ALLOW_ALL=false
ADMIN_PUBKEYS=adminpubkey1
STORAGE_QUOTA_BYTES=10737418240  # 10GB
SIGNING_REQUESTS_MAX=1000
```

### Important: PostgreSQL Driver

The platform package uses `database/sql` but does **not** import a driver. Services using platform mode must import a PostgreSQL driver:

```go
import _ "github.com/lib/pq"
// or
import _ "github.com/jackc/pgx/v5/stdlib"
```

This is intentional - it allows services to choose their preferred driver and avoids pulling unnecessary dependencies in standalone mode.

### Usage

```go
import "git.aegis-hq.xyz/coldforge/cloistr-common/platform"

// Create client from environment
client, err := platform.NewClientFromEnv()
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Check access
hasAccess, err := client.HasAccess(ctx, pubkey)

// Check quota before upload
if err := client.RequireQuota(ctx, pubkey, platform.QuotaTypeStorageBytes, fileSize); err != nil {
    return err // quota exceeded
}

// Record usage after successful operation
client.RecordUsage(ctx, pubkey, platform.QuotaTypeStorageBytes, fileSize)

// Check admin status
if isAdmin, _ := client.IsAdmin(ctx, pubkey); isAdmin {
    // allow admin operation
}
```

### Key Functions

| Function | Description |
|----------|-------------|
| `HasAccess(ctx, pubkey)` | Check if user can access current service |
| `HasAccessToService(ctx, pubkey, serviceID)` | Check access to any service |
| `RequireAccess(ctx, pubkey)` | Returns error if access denied |
| `IsAdmin(ctx, pubkey)` | Check if user is admin |
| `RequireAdmin(ctx, pubkey)` | Returns error if not admin |
| `GetQuota(ctx, pubkey, quotaType)` | Get quota info |
| `CheckQuota(ctx, pubkey, quotaType, amount)` | Check if quota available |
| `RequireQuota(ctx, pubkey, quotaType, amount)` | Returns error if quota exceeded |
| `RecordUsage(ctx, pubkey, quotaType, amount)` | Record usage |
| `ReleaseUsage(ctx, pubkey, quotaType, amount)` | Release usage (e.g., on delete) |

See `~/claude/coldforge/cloistr/schemas/SERVICE-INTEGRATION.md` for full integration patterns.

## Package: errors

Standardized API error types following the potato-grade design format. **All Cloistr services must use this package for API error responses.**

### Error Format

```json
{
  "code": "STORAGE_TIMEOUT",
  "message": "Upload timed out after 30s",
  "retry_after": 60,
  "debug": {"timeout_at": "write_chunk_3"}
}
```

### Files

| File | Purpose |
|------|---------|
| `errors.go` | APIError type and constructors |
| `codes.go` | Standard error codes |

### Usage

```go
import "git.aegis-hq.xyz/coldforge/cloistr-common/errors"

// Use pre-built errors
if !hasAccess {
    errors.ErrAccessDenied.WriteResponse(w)
    return
}

// Create custom errors
err := errors.BadRequest(errors.CodeInvalidInput, "invalid pubkey format").
    WithDebug("pubkey", pubkey)
err.WriteResponse(w)

// With retry guidance
err := errors.TooManyRequests(errors.CodeRateLimitExceeded, "rate limit exceeded", 60)
err.WriteResponse(w)
```

### Standard Error Codes

| Category | Codes |
|----------|-------|
| Auth | `AUTH_REQUIRED`, `AUTH_INVALID`, `ACCESS_DENIED`, `NOT_ADMIN` |
| Quota | `QUOTA_EXCEEDED`, `STORAGE_FULL`, `RATE_LIMIT_EXCEEDED` |
| Resource | `RESOURCE_NOT_FOUND`, `RESOURCE_EXISTS`, `RESOURCE_CONFLICT` |
| Storage | `STORAGE_TIMEOUT`, `STORAGE_ERROR`, `UPLOAD_FAILED` |
| Validation | `VALIDATION_FAILED`, `INVALID_INPUT`, `INVALID_PUBKEY` |
| Service | `SERVICE_UNAVAILABLE`, `INTERNAL_ERROR` |

### Pre-built Errors

| Variable | Code | HTTP Status |
|----------|------|-------------|
| `ErrAuthRequired` | AUTH_REQUIRED | 401 |
| `ErrAccessDenied` | ACCESS_DENIED | 403 |
| `ErrQuotaExceeded` | QUOTA_EXCEEDED | 507 |
| `ErrResourceNotFound` | RESOURCE_NOT_FOUND | 404 |
| `ErrInternalError` | INTERNAL_ERROR | 500 |

---

**Last Updated:** 2026-03-22
