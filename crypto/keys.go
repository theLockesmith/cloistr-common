package crypto

import (
	"encoding/hex"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// GenerateKeypair returns a new random secp256k1 keypair as hex strings.
func GenerateKeypair() (sk string, pk string, err error) {
	sk = nostr.GeneratePrivateKey()
	pk, err = nostr.GetPublicKey(sk)
	if err != nil {
		return "", "", fmt.Errorf("derive public key: %w", err)
	}
	return sk, pk, nil
}

// PublicKey derives the hex x-only public key from a hex secret key.
func PublicKey(sk string) (string, error) {
	return nostr.GetPublicKey(sk)
}

// ValidateSecretKey reports an error unless sk is a 32-byte hex secret key.
func ValidateSecretKey(sk string) error {
	b, err := hex.DecodeString(sk)
	if err != nil {
		return fmt.Errorf("secret key is not valid hex: %w", err)
	}
	if len(b) != 32 {
		return fmt.Errorf("secret key must be 32 bytes, got %d", len(b))
	}
	return nil
}

// ValidatePublicKey reports an error unless pk is a 32-byte hex x-only key.
func ValidatePublicKey(pk string) error {
	b, err := hex.DecodeString(pk)
	if err != nil {
		return fmt.Errorf("public key is not valid hex: %w", err)
	}
	if len(b) != 32 {
		return fmt.Errorf("public key must be 32 bytes, got %d", len(b))
	}
	return nil
}
