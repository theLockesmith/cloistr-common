# cloistr-common

Shared Go library for Cloistr services.

## Packages

### relayprefs

Relay preference discovery for Cloistr services. Users store their relay preferences as signed Nostr events (kind:30078, d-tag "cloistr-relays"). This package handles discovering those preferences through a configurable query chain.

#### Installation

```bash
go get git.aegis-hq.xyz/coldforge/cloistr-common
```

#### Usage

```go
import "git.aegis-hq.xyz/coldforge/cloistr-common/relayprefs"

// Create client from environment variables
client := relayprefs.NewClientFromEnv()

// Get user's relay preferences
prefs, err := client.GetRelayPrefs(ctx, userPubkey)
if err != nil {
    return err
}

// Use the relays
for _, url := range prefs.WriteRelays() {
    // Publish to this relay
}
```

#### Configuration

Configure via environment variables:

| Variable | Purpose | Example |
|----------|---------|---------|
| `DISCOVERY_INTERNAL` | Self-hosted discovery URL | `http://my-discovery:8080` |
| `RELAY_LIST` | Comma-separated relays for direct query | `wss://my-relay.com,wss://backup.com` |
| `DISCOVERY_EXTERNAL` | Third-party discovery URL | `https://some-discovery.com` |
| `USE_CLOISTR_FALLBACK` | Use Cloistr services as fallback | `true` (default) |
| `RELAY_PREFS_CACHE_TTL` | Cache duration | `1h` (default) |

#### Query Chain

The library queries sources in this order:

1. Local cache
2. `DISCOVERY_INTERNAL` (if configured)
3. `RELAY_LIST` (if configured) - direct relay queries
4. `DISCOVERY_EXTERNAL` (if configured)
5. `discover.cloistr.xyz` (if `USE_CLOISTR_FALLBACK=true`)
6. `relay.cloistr.xyz` (if `USE_CLOISTR_FALLBACK=true`)

If no `cloistr-relays` event (kind:30078) is found, falls back to NIP-65 (kind:10002), then configured defaults.

#### Self-Hosting

Self-hosters can configure their own infrastructure:

```bash
# Use your own discovery service
export DISCOVERY_INTERNAL=http://my-discovery:8080

# Or query relays directly
export RELAY_LIST=wss://my-relay.com

# Optionally disable Cloistr fallback entirely
export USE_CLOISTR_FALLBACK=false
```

### platform

Unified access control, quota management, and usage tracking for Cloistr services. Supports both platform mode (shared PostgreSQL) and standalone mode (config-based).

#### Installation

```bash
go get git.aegis-hq.xyz/coldforge/cloistr-common
```

**Important:** In platform mode, you must import a PostgreSQL driver in your service:

```go
import _ "github.com/lib/pq" // or github.com/jackc/pgx/v5/stdlib
```

#### Usage

```go
import "git.aegis-hq.xyz/coldforge/cloistr-common/platform"

// Create client from environment variables
client, err := platform.NewClientFromEnv()
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Check access
if err := client.RequireAccess(ctx, pubkey); err != nil {
    return err // access denied
}

// Check quota before upload
if err := client.RequireQuota(ctx, pubkey, platform.QuotaTypeStorageBytes, fileSize); err != nil {
    return err // quota exceeded
}

// Record usage after successful operation
client.RecordUsage(ctx, pubkey, platform.QuotaTypeStorageBytes, fileSize)

// Release usage on delete
client.ReleaseUsage(ctx, pubkey, platform.QuotaTypeStorageBytes, fileSize)
```

#### Configuration

Configure via environment variables:

| Variable | Purpose | Mode |
|----------|---------|------|
| `CLOISTR_MODE` | `platform` or `standalone` | Both |
| `SERVICE_ID` | Service identifier (e.g., `blossom`) | Both |
| `DATABASE_URL` | PostgreSQL connection string | Platform |
| `WHITELIST_PUBKEYS` | Comma-separated allowed pubkeys | Standalone |
| `WHITELIST_ALLOW_ALL` | Allow all pubkeys (`true`/`false`) | Standalone |
| `ADMIN_PUBKEYS` | Comma-separated admin pubkeys | Standalone |
| `STORAGE_QUOTA_BYTES` | Per-user storage quota | Standalone |
| `SIGNING_REQUESTS_MAX` | Max signing requests per day | Standalone |

#### Modes

**Platform Mode:** Queries the shared PostgreSQL database for access control and quotas. Requires `DATABASE_URL` and uses the `has_service_access()` and `get_user_quota()` database functions.

**Standalone Mode:** Uses environment variables for configuration. Useful for self-hosted deployments without the full platform infrastructure.

#### Quota Types

| Constant | Description |
|----------|-------------|
| `QuotaTypeStorageBytes` | Storage space in bytes |
| `QuotaTypeSigningRequestsDaily` | Signing requests per day |
| `QuotaTypeEmailStorageBytes` | Email storage in bytes |

### errors

Standardized API error types following the potato-grade design format. All error responses include a machine-readable code, human-readable message, and optional retry guidance.

#### Usage

```go
import "git.aegis-hq.xyz/coldforge/cloistr-common/errors"

// Use pre-built errors
if !hasAccess {
    errors.ErrAccessDenied.WriteResponse(w)
    return
}

// Create custom errors with debug info
err := errors.BadRequest(errors.CodeInvalidInput, "invalid pubkey format").
    WithDebug("pubkey", pubkey)
err.WriteResponse(w)

// Rate limiting with retry guidance
err := errors.TooManyRequests(errors.CodeRateLimitExceeded, "rate limit exceeded", 60)
err.WriteResponse(w)
```

#### Response Format

```json
{
  "code": "STORAGE_TIMEOUT",
  "message": "Upload timed out after 30s",
  "retry_after": 60,
  "debug": {"timeout_at": "write_chunk_3"}
}
```

#### Standard Error Codes

- **Auth:** `AUTH_REQUIRED`, `AUTH_INVALID`, `ACCESS_DENIED`, `NOT_ADMIN`
- **Quota:** `QUOTA_EXCEEDED`, `STORAGE_FULL`, `RATE_LIMIT_EXCEEDED`
- **Resource:** `RESOURCE_NOT_FOUND`, `RESOURCE_EXISTS`, `RESOURCE_CONFLICT`
- **Storage:** `STORAGE_TIMEOUT`, `STORAGE_ERROR`, `UPLOAD_FAILED`
- **Validation:** `VALIDATION_FAILED`, `INVALID_INPUT`, `INVALID_PUBKEY`

## License

AGPL-3.0
