package crypto

import (
	"fmt"

	"github.com/nbd-wtf/go-nostr/nip04"
)

// EncryptNIP04 encrypts plaintext from senderSK to recipientPK using NIP-04.
//
// Deprecated: NIP-04 leaks message length, lacks authenticated encryption, and
// is superseded by NIP-44. Use EncryptNIP44 for all new code. This is provided
// only for interop with existing kind:4 direct messages.
func EncryptNIP04(plaintext, senderSK, recipientPK string) (string, error) {
	shared, err := nip04.ComputeSharedSecret(recipientPK, senderSK)
	if err != nil {
		return "", fmt.Errorf("nip04: compute shared secret: %w", err)
	}
	ciphertext, err := nip04.Encrypt(plaintext, shared)
	if err != nil {
		return "", fmt.Errorf("nip04: encrypt: %w", err)
	}
	return ciphertext, nil
}

// DecryptNIP04 decrypts a NIP-04 payload from senderPK, using recipientSK.
//
// Deprecated: see EncryptNIP04.
func DecryptNIP04(ciphertext, recipientSK, senderPK string) (string, error) {
	shared, err := nip04.ComputeSharedSecret(senderPK, recipientSK)
	if err != nil {
		return "", fmt.Errorf("nip04: compute shared secret: %w", err)
	}
	plaintext, err := nip04.Decrypt(ciphertext, shared)
	if err != nil {
		return "", fmt.Errorf("nip04: decrypt: %w", err)
	}
	return plaintext, nil
}
