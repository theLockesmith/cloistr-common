// Package username is the single source of truth for Cloistr address local-part
// (username) validation and for generating auto-assigned fallback addresses.
//
// The canonical rule mirrors the database CHECK constraint on addresses.username:
//
//	^[a-z0-9_-]{2,50}$
//
// Auto-assigned addresses (handed to nameless extension/NIP-07 identities so email and
// zaps work at all) use a reserved shape that humans are NOT allowed to claim, so an
// attacker cannot squat the auto-assign namespace:
//
//	^[a-z]+-[a-z]+-[0-9]{4}$   e.g. "happy-otter-1234"
//
// Both the Go backend (cloistr-me, cloistr-signer) and the TypeScript frontend
// (@cloistr/ui) enforce the same rules; keep this package and @cloistr/ui/src/lib/username.ts
// in lockstep. The DB CHECK remains authoritative.
package username

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
)

// Canonical patterns. Keep in sync with the addresses.username DB CHECK and
// @cloistr/ui/src/lib/username.ts.
var (
	// validPattern is the canonical username format (matches the DB CHECK).
	validPattern = regexp.MustCompile(`^[a-z0-9_-]{2,50}$`)
	// autoAssignedPattern is the reserved shape for auto-assigned addresses.
	autoAssignedPattern = regexp.MustCompile(`^[a-z]+-[a-z]+-[0-9]{4}$`)
)

// ValidPatternString / AutoAssignedPatternString expose the raw regex source so
// other layers (or tests) can assert they match.
const (
	ValidPatternString        = `^[a-z0-9_-]{2,50}$`
	AutoAssignedPatternString = `^[a-z]+-[a-z]+-[0-9]{4}$`
)

// IsValid reports whether name satisfies the canonical username format.
func IsValid(name string) bool {
	return validPattern.MatchString(name)
}

// IsAutoAssigned reports whether name has the reserved auto-assigned shape
// (adjective-noun-NNNN). Auto-assigned addresses never confer the named tier.
func IsAutoAssigned(name string) bool {
	return autoAssignedPattern.MatchString(name)
}

// IsValidHumanName reports whether name is a valid username that a human may claim.
// It is valid format AND not the reserved auto-assigned shape — this is what blocks
// a user from squatting the auto-assign namespace.
func IsValidHumanName(name string) bool {
	return IsValid(name) && !IsAutoAssigned(name)
}

// AvailFunc reports whether a candidate username is free to claim. It returns an
// error only on an underlying failure (e.g. a database error), which aborts Generate.
type AvailFunc func(candidate string) (bool, error)

// maxGenerateAttempts bounds collision retries so Generate can't loop forever.
const maxGenerateAttempts = 50

// Generate returns an available auto-assigned username of the form
// adjective-noun-NNNN, using avail to skip collisions. The result always satisfies
// IsAutoAssigned and IsValid, and never IsValidHumanName. It returns an error if no
// free candidate is found within maxGenerateAttempts or if avail errors.
func Generate(avail AvailFunc) (string, error) {
	for i := 0; i < maxGenerateAttempts; i++ {
		adj, err := pick(adjectives)
		if err != nil {
			return "", err
		}
		noun, err := pick(nouns)
		if err != nil {
			return "", err
		}
		n, err := rand.Int(rand.Reader, big.NewInt(9000))
		if err != nil {
			return "", err
		}
		candidate := fmt.Sprintf("%s-%s-%04d", adj, noun, n.Int64()+1000) // 1000..9999

		free, err := avail(candidate)
		if err != nil {
			return "", fmt.Errorf("username generate: availability check failed: %w", err)
		}
		if free {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("username generate: no available candidate after %d attempts", maxGenerateAttempts)
}

// pick returns a cryptographically-random element of list.
func pick(list []string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(list))))
	if err != nil {
		return "", err
	}
	return list[n.Int64()], nil
}
