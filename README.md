# cloistr-common

Shared Go library for Cloistr services.

## Packages

### relayprefs

Relay preference discovery for Cloistr services. Users store their relay preferences as signed Nostr events (kind:30078, d-tag "cloistr-relays"). This package handles discovering those preferences through a configurable query chain.

#### Installation

```bash
go get git.coldforge.xyz/coldforge/cloistr-common
```

#### Usage

```go
import "git.coldforge.xyz/coldforge/cloistr-common/relayprefs"

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

## License

AGPL-3.0
