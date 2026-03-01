# CLAUDE.md - cloistr-common

**Shared Go library for Cloistr services.**

## Project Information

- **Company:** Coldforge
- **Type:** Go Library
- **Repository:** `git.coldforge.xyz/coldforge/cloistr-common`

**Parent Rules:** See [Coldforge CLAUDE.md](~/claude/coldforge/CLAUDE.md) and [global CLAUDE.md](~/claude/CLAUDE.md).

## Purpose

This library provides shared functionality for all Cloistr services. Currently includes:

- **relayprefs** - Relay preference discovery (kind:30078 cloistr-relays)

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
import "git.coldforge.xyz/coldforge/cloistr-common/relayprefs"

client := relayprefs.NewClientFromEnv()
prefs, _ := client.GetRelayPrefs(ctx, pubkey)
```

See `~/claude/coldforge/cloistr/architecture/relay-preferences.md` for full integration guide.

---

**Last Updated:** 2026-03-01
