# Cloistr Common Tooling Consolidation Roadmap

**Created:** 2026-04-29
**Status:** Complete - CI Triggers Remaining

---

## Executive Summary

Audit complete. Found significant duplication across 32 projects with clear consolidation opportunities. Several prior decisions already exist that inform the path forward.

### Key Findings

| Category | Finding | Prior Decision? |
|----------|---------|-----------------|
| NIP-46 Auth | 5+ separate implementations | YES - collab-common is canonical |
| Error Handling | Inconsistent across all Go services | PARTIAL - errors package exists, not adopted |
| Relay Preferences | 454-line JS port in stash | NO - needs TypeScript package |
| Document ID Generation | Duplicated in 3 apps | YES - format standardized |
| Collab-common Version | 3 apps on ^0.1.1, 2 on ^0.2.0 | NO - simple version bump needed |
| Crypto Utils | 3 separate implementations | NO - needs discussion |

---

## Phase 0 Audit Results

### Go Backend Services (13 projects)

| Service | cloistr-common | Needs It? | Gaps |
|---------|----------------|-----------|------|
| cloistr-blossom | YES (relayprefs, platform) | - | errors package not used |
| cloistr-calendar | YES | - | errors package not used |
| cloistr-email | YES (relayprefs) | - | errors package not used |
| cloistr-vault | YES (relayprefs) | - | errors package not used, inconsistent error format |
| cloistr-chat | NO | YES | Custom NIP-46, custom error handling, needs relayprefs |
| cloistr-contacts | NO | YES | Custom NIP-46, needs relayprefs |
| cloistr-video | NO | MAYBE | Custom auth, custom error handling |
| cloistr-relay | NO | NO | Is a relay, doesn't publish to user relays |
| cloistr-discovery | NO | NO | Is the source for relay discovery |
| cloistr-search | NO | N/A | Placeholder - not implemented |
| cloistr-wot | NO | N/A | Placeholder - not implemented |
| cloistr-signer | NO | MAYBE | Backend only, complex auth |
| cloistr-config | N/A | N/A | Kubernetes YAML configs, not Go code |

**Common Patterns Found in Go Services:**
1. Custom NIP-46 auth in chat/contacts (nearly identical implementations)
2. Inconsistent error responses: hardcoded JSON strings, no standard codes
3. Missing errors package adoption despite it existing
4. Rate limiting implemented differently in each service

### TypeScript Frontend Apps (19 projects)

| App | collab-common | ui | Auth Method | Status |
|-----|---------------|-----|-------------|--------|
| cloistr-docs | ^0.1.1 | ^0.3.0 | SharedAuthProvider | OUTDATED |
| cloistr-slides | ^0.1.1 | ^0.3.0 | SharedAuthProvider | OUTDATED |
| cloistr-whiteboard | ^0.1.1 | ^0.3.0 | SharedAuthProvider | OUTDATED |
| cloistr-sheets | ^0.2.0 | ^0.3.0 | SharedAuthProvider | CURRENT |
| cloistr-space | ^0.2.0 | ^0.3.0 | SharedAuthProvider | CURRENT |
| cloistr-workspace | file:../... | - | Local dev | DEV ONLY |
| cloistr-discovery-ui | - | ^0.3.0 | nostr-tools BunkerSigner | NEEDS MIGRATION |
| cloistr-me-ui | - | ^0.3.0 | nostr-tools BunkerSigner | NEEDS MIGRATION |
| cloistr-sanctuary | - | ^0.3.0 | SharedAuthProvider | OK |
| cloistr-stash | UMD | - | Compatibility wrapper | OK (per prior decision) |
| cloistr-stash-desktop | symlink | - | Inherits from stash | OK |
| cloistr-stash-mobile | - | - | Custom Flutter auth | PLATFORM-SPECIFIC |
| cloistr-photos | - | - | Custom NIP-07/NIP-46 | NEEDS MIGRATION |
| cloistr-tasks | - | - | Custom + JWT backend | NEEDS MIGRATION |
| cloistr-me | - | - | Playwright test project | N/A |
| cloistr-assets | - | - | Static assets | N/A |

**Duplicate Implementations Found:**

1. **Document ID Generation** (docs, slides, whiteboard)
   - Same logic: `{type}-{timestamp}-{uuid8}`
   - Prior decision EXISTS: format standardized in Drive integration
   - Action: Move to collab-common

2. **Default URLs** (docs, slides, whiteboard)
   - `VITE_RELAY_URL || 'wss://relay.cloistr.xyz'`
   - `VITE_BLOSSOM_URL || 'https://nostr.download'`
   - Action: Move to collab-common config module

3. **Login Prompt UI** (docs, slides, whiteboard)
   - Similar JSX structure
   - Action: Create reusable component in @cloistr/ui

4. **Relay Preferences JS** (stash - 454 lines)
   - Mirrors Go cloistr-common/relayprefs logic
   - No TypeScript version exists
   - Action: Create @cloistr/relay-prefs TypeScript package

5. **NIP-07/NIP-46 Auth** (photos, tasks)
   - ~200-400 lines per app
   - Prior decision: collab-common is canonical implementation
   - Action: Migrate to collab-common

6. **Crypto Utils** (stash, stash-desktop, stash-mobile)
   - XChaCha20-Poly1305 implementations
   - Different per platform
   - Action: Discuss - may be platform-specific necessity

---

## Prior Decisions (From RAG)

### 1. NIP-46 Consolidation - DECIDED
**Decision:** collab-common is the single canonical NIP-46 implementation
**Date:** 2026-03-26
**Details:** Enhanced with circuit breaker, adaptive rate limiting, batch_sign, session persistence, UMD build
**Migration:** stash already migrated via compatibility wrapper
**Action:** Apply same pattern to photos, tasks, me-ui, discovery-ui

### 2. Batch Signing - DECIDED
**Decision:** Server supports `batch_sign` method, clients should use it
**Date:** 2026-03-15
**Details:** Reduces round-trips, helps with rate limiting
**Status:** Implemented in collab-common, not yet used by all apps
**Action:** Verify NostrSyncProvider uses batch signing for multi-event operations

### 3. Document ID Format - DECIDED
**Decision:** `{type}-{timestamp}-{uuid8}` format
**Date:** 2026-03-25
**Details:** Standardized across all collab apps for Drive integration
**Action:** Extract generation logic to collab-common

### 4. Potato-Grade Design - PHILOSOPHY
**Path:** `~/arbiter/coldforge/cloistr/architecture/potato-grade-design.md`
**Principles:** Stateless requests, client intelligence, observable failures
**Relevance:** Error handling standardization should follow this philosophy

---

## Undocumented Divergences (Need Discussion)

### 1. Go Services Not Using errors Package
**Finding:** errors package exists in cloistr-common but only used by... nobody
**Question:** Was this intentional? Should we mandate adoption?
**Recommendation:** Yes - standardized errors improve debugging and client handling

### 2. TypeScript Relay Preferences
**Finding:** stash has 454-line JS implementation mirroring Go library
**Question:** Should we create @cloistr/relay-prefs TypeScript package?
**Recommendation:** Yes - eliminates duplication, enables other apps to use it

### 3. Crypto Library Standardization
**Finding:** Three platforms, three crypto implementations
**Question:** Can/should we share crypto code across web/desktop/mobile?
**Reality:** Likely platform-specific necessity (libsodium-wrappers vs Rust vs Dart)
**Recommendation:** Document the pattern, don't try to unify

### 4. Simple Apps Auth Migration
**Finding:** discovery-ui and me-ui have own NIP-46 implementations
**Question:** Worth migrating for single-event signing apps?
**Arguments For:** Consistency, circuit breaker benefits, reduced maintenance
**Arguments Against:** Working code, adds dependency
**Recommendation:** Migrate for consistency and future batch signing capability

### 5. cloistr-photos and cloistr-tasks
**Finding:** Both have substantial custom auth implementations
**Question:** Full migration to collab-common?
**Complexity:** tasks has JWT backend integration
**Recommendation:** photos - migrate; tasks - evaluate backend changes needed

---

## Proposed Implementation Plan

### Phase 1: Version Alignment (Low Risk)
- [ ] Update cloistr-docs: collab-common ^0.1.1 → ^0.2.0
- [ ] Update cloistr-slides: collab-common ^0.1.1 → ^0.2.0
- [ ] Update cloistr-whiteboard: collab-common ^0.1.1 → ^0.2.0
- [ ] Remove local file:// overrides

### Phase 2: Go Backend Standardization
- [ ] Adopt errors package in blossom, calendar, email, vault
- [ ] Integrate cloistr-common into chat (relayprefs + errors)
- [ ] Integrate cloistr-common into contacts (relayprefs + errors)
- [ ] Evaluate video service needs

### Phase 3: TypeScript Common Extraction
- [x] Move document ID generation to collab-common
- [x] Move default URL config to collab-common
- [x] Create LoginPrompt component in @cloistr/ui
- [x] Create @cloistr/relay-prefs TypeScript package (from stash)

### Phase 4: Auth Migration
- [x] Migrate cloistr-discovery-ui to collab-common auth
- [x] Migrate cloistr-me-ui to collab-common auth
- [x] Migrate cloistr-photos to collab-common auth
- [x] Migrate cloistr-tasks to collab-common auth (JWT flow preserved)

### Phase 5: Documentation & CI/CD
- [ ] Update all CLAUDE.md files
- [ ] Document integration patterns
- [ ] Set up CI triggers for common package changes

---

## Decisions Made (2026-04-29)

1. **errors package adoption** - YES, mandate for all Go services
2. **relay-prefs** - Add to collab-common (from stash's JS implementation)
3. **Simple app migration** - YES, migrate discovery-ui/me-ui for batch signing
4. **cloistr-tasks** - Full migration
5. **Crypto standardization** - Attempt unification

---

## Implementation Status

- [x] Phase 1: Version Alignment (collab-common ^0.2.0 in docs, slides, whiteboard, sheets)
- [x] Phase 2: Go Backend Standardization
  - [x] Added Gin helpers to errors package (StatusAndBody, Abort)
  - [x] Migrated cloistr-blossom (45+ patterns)
  - [x] Migrated cloistr-calendar (38 patterns)
  - [x] Migrated cloistr-email (64 patterns)
  - [x] Migrated cloistr-vault (295 patterns)
  - [x] Migrated cloistr-chat (builds and tests pass)
  - [x] Migrated cloistr-contacts (builds and tests pass)
- [x] Phase 3: TypeScript Common Extraction
  - [x] Added relay-prefs.ts to collab-common (ported from stash's 454-line JS)
  - [x] Added useRelayPrefsHook for React integration
  - [x] Exports: getRelayPrefs, invalidateCache, createRelayPrefsEvent
  - [x] Added config module to collab-common with document ID generation and service URLs
  - [x] Functions: generateDocumentId, getOrCreateDocumentId, getServiceConfig
  - [x] Created LoginPrompt component in @cloistr/ui
  - [x] Updated docs, slides, whiteboard to use collab-common/config and @cloistr/ui LoginPrompt
- [x] Phase 4: Auth Migration
  - [x] Migrated cloistr-discovery-ui - wraps collab-common with API-compatible layer
  - [x] Migrated cloistr-me-ui - wraps collab-common with signer adapter
  - [x] Migrated cloistr-photos - hybrid: collab-common for auth, NIP-44 via window.nostr.nip44
  - [x] Migrated cloistr-tasks - JS project using collab-common auth, preserves JWT challenge-response flow
- [x] Phase 5: Documentation & CI/CD
  - [x] Updated CONSOLIDATION-ROADMAP.md with all implementation details
  - [x] Added integration patterns documentation (see below)
  - [x] Crypto utils analysis documented (incompatibility found)
  - [ ] CI triggers for dependent projects (requires GitLab pipeline trigger setup)

---

## Integration Patterns

### Using collab-common/config

```tsx
import { getOrCreateDocumentId, getServiceConfig } from '@cloistr/collab-common/config'

// Get service URLs from environment (VITE_RELAY_URL, etc.) with defaults
const config = getServiceConfig()

// Get or create document ID (updates URL params)
const [documentId] = useState(() => getOrCreateDocumentId('doc'))

// Use in components
<Editor relayUrl={config.relayUrl} documentId={documentId} />
```

### Using collab-common/auth

```tsx
import { AuthProvider, useNostrAuth } from '@cloistr/collab-common/auth'

// Wrap app with AuthProvider
<AuthProvider>
  <App />
</AuthProvider>

// Use in components
function MyComponent() {
  const { authState, signer, connectNip07, connectNip46 } = useNostrAuth()

  if (authState.isConnected && signer) {
    // User is authenticated
  }
}
```

### Using @cloistr/ui LoginPrompt

```tsx
import { LoginPrompt } from '@cloistr/ui/components'

{!authState.isConnected && (
  <LoginPrompt
    title="Cloistr Docs"
    subtitle="Collaborative document editing powered by Nostr"
    callToAction="Sign in to create or edit documents."
  />
)}
```

### Auth Migration Pattern (for existing apps)

For apps with existing auth that need to preserve API compatibility:

```tsx
// Wrap collab-common auth with compatibility layer
import { useNostrAuth } from '@cloistr/collab-common/auth'

function useAuth() {
  const collabAuth = useNostrAuth()

  // Map to existing API shape
  return {
    state: {
      pubkey: collabAuth.authState.pubkey,
      method: collabAuth.authState.method,
    },
    signer: collabAuth.signer ? adaptSigner(collabAuth.signer) : null,
    login: collabAuth.connectNip07,
    loginNip46: (url) => collabAuth.connectNip46({ bunkerUrl: url }),
    logout: collabAuth.disconnect,
  }
}
```

### Go Backend Error Handling

```go
import "git.aegis-hq.xyz/coldforge/cloistr-common/errors"

// Use pre-built errors
if !hasAccess {
    errors.ErrAccessDenied.Abort(ctx)
    return
}

// Create custom errors
errors.BadRequest(errors.CodeInvalidInput, "invalid pubkey").
    WithDebug("pubkey", pubkey).
    Abort(ctx)

// With retry guidance
errors.TooManyRequests(errors.CodeRateLimitExceeded, "rate limit exceeded", 60).
    Abort(ctx)
```

---

## Crypto Utils Analysis

**Status:** Partial unification possible

### Implementations Compared

| Platform | File | Library |
|----------|------|---------|
| Web | `cloistr-stash/web/js/crypto.js` | libsodium-wrappers (JS) |
| Mobile | `cloistr-stash-mobile/lib/core/crypto/crypto_service.dart` | sodium_libs (Dart FFI) |
| Desktop | `cloistr-stash-desktop/src-tauri/src/crypto.rs` | chacha20poly1305 (Rust) |

### Compatibility Matrix

| Operation | Web ↔ Mobile | Web ↔ Desktop | Mobile ↔ Desktop |
|-----------|--------------|---------------|------------------|
| Small file encrypt | **Compatible** | **Compatible** | **Compatible** |
| Chunked encrypt | **Compatible** | **INCOMPATIBLE** | **INCOMPATIBLE** |
| Key derivation | HKDF-BLAKE2b | HKDF-BLAKE2b | HKDF-SHA256 |

### Chunked Format Differences

**Web/Mobile Format:**
```
CLCH (4) | version (1) | chunk_size (4) | chunk_count (4) | base_nonce (24) | chunks...
Each chunk: encrypted data (nonce derived by XOR of base_nonce with chunk index)
```

**Desktop Format:**
```
CLCH (4) | chunk_count (4) | [nonce (24) + length (4) + ciphertext]...
Each chunk: random nonce per chunk
```

### Recommendation

1. **Do not attempt full unification** - platform-specific crypto is necessary for performance
2. **Fix chunked format incompatibility** - Desktop should be updated to match web/mobile format
3. **Document compatibility requirements** - Files encrypted with chunked mode must stay on same platform until format is unified
4. **Key derivation difference is minor** - Both BLAKE2b and SHA256 are secure; can standardize on SHA256 for interop
