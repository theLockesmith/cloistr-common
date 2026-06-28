// Package crypto provides Cloistr's canonical Nostr encryption primitives and
// signer abstractions, wrapping github.com/nbd-wtf/go-nostr.
//
// It exists to replace the per-service crypto wrappers that had drifted apart
// (e.g. cloistr-video/internal/crypto, cloistr-email/internal/encryption) with a
// single, test-vector-backed implementation. The package is layered:
//
//   - Raw functions — EncryptNIP44/DecryptNIP44, EncryptNIP04/DecryptNIP04 —
//     operate directly on hex-encoded keys. Use them when the secret key is held
//     in process.
//   - The Signer interface delegates crypto to a key custodian (LocalSigner,
//     NIP46Signer, or the ClientSideSigner marker), so backend services and the
//     web apps share one shape. It mirrors @cloistr/collab-common's
//     SignerInterface on the TypeScript side.
//
// All public keys are hex-encoded 32-byte x-only keys; all secret keys are
// hex-encoded 32 bytes — the same representation go-nostr uses throughout.
package crypto

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr/nip44"
)

// EncryptNIP44 encrypts plaintext from senderSK to recipientPK using NIP-44 v2.
//
// The NIP-44 conversation key is symmetric, so decryption derives the same key
// from (senderPK, recipientSK). Keys are hex-encoded.
func EncryptNIP44(plaintext, senderSK, recipientPK string) (string, error) {
	ck, err := nip44.GenerateConversationKey(recipientPK, senderSK)
	if err != nil {
		return "", fmt.Errorf("nip44: derive conversation key: %w", err)
	}
	ciphertext, err := nip44.Encrypt(plaintext, ck)
	if err != nil {
		return "", fmt.Errorf("nip44: encrypt: %w", err)
	}
	return ciphertext, nil
}

// DecryptNIP44 decrypts a NIP-44 v2 payload sent from senderPK, using recipientSK.
// Keys are hex-encoded.
func DecryptNIP44(ciphertext, recipientSK, senderPK string) (string, error) {
	ck, err := nip44.GenerateConversationKey(senderPK, recipientSK)
	if err != nil {
		return "", fmt.Errorf("nip44: derive conversation key: %w", err)
	}
	plaintext, err := nip44.Decrypt(ciphertext, ck)
	if err != nil {
		return "", fmt.Errorf("nip44: decrypt: %w", err)
	}
	return plaintext, nil
}
