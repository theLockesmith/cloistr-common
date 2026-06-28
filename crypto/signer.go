package crypto

import (
	"context"
	"errors"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

// Scheme identifies a Nostr encryption scheme.
type Scheme string

const (
	// SchemeNIP44 is NIP-44 v2 (authenticated, the default for new code).
	SchemeNIP44 Scheme = "nip44"
	// SchemeNIP04 is legacy NIP-04. Deprecated: interop only.
	SchemeNIP04 Scheme = "nip04"
)

// ErrClientSide indicates the operation must be performed by the client. A
// ClientSideSigner (NIP-07 mode) returns this: the browser holds the key, so the
// server stores ciphertext it cannot itself produce or read.
var ErrClientSide = errors.New("crypto: operation must be performed client-side (NIP-07)")

// Signer abstracts custody of a Nostr identity's secret key. Implementations
// either hold the key in process (LocalSigner), delegate to a NIP-46 remote
// signer (NIP46Signer), or signal that a browser/client performs the operation
// (ClientSideSigner).
//
// This is the canonical signer shape across Cloistr Go services, consolidating
// the previously divergent cloistr-video/internal/crypto.Encryptor and
// cloistr-email/internal/encryption.Signer interfaces. It mirrors
// @cloistr/collab-common's SignerInterface on the TypeScript side.
type Signer interface {
	// PublicKey returns the signer's hex x-only public key.
	PublicKey() string

	// Encrypt encrypts plaintext to recipientPK using the given scheme.
	Encrypt(ctx context.Context, scheme Scheme, plaintext, recipientPK string) (string, error)

	// Decrypt decrypts ciphertext from senderPK using the given scheme.
	Decrypt(ctx context.Context, scheme Scheme, ciphertext, senderPK string) (string, error)

	// SignEvent signs e in place, setting its ID, PubKey, and Sig.
	SignEvent(ctx context.Context, e *nostr.Event) error

	// CanEncrypt reports whether this signer performs crypto itself. A
	// ClientSideSigner returns false — callers must defer to the client.
	CanEncrypt() bool
}

// encryptScheme dispatches a raw encrypt by scheme. Shared by signers that hold
// a local secret key.
func encryptScheme(scheme Scheme, plaintext, sk, recipientPK string) (string, error) {
	switch scheme {
	case SchemeNIP44:
		return EncryptNIP44(plaintext, sk, recipientPK)
	case SchemeNIP04:
		return EncryptNIP04(plaintext, sk, recipientPK)
	default:
		return "", fmt.Errorf("crypto: unknown scheme %q", scheme)
	}
}

func decryptScheme(scheme Scheme, ciphertext, sk, senderPK string) (string, error) {
	switch scheme {
	case SchemeNIP44:
		return DecryptNIP44(ciphertext, sk, senderPK)
	case SchemeNIP04:
		return DecryptNIP04(ciphertext, sk, senderPK)
	default:
		return "", fmt.Errorf("crypto: unknown scheme %q", scheme)
	}
}

// LocalSigner holds a secret key in process and performs every operation locally.
// Use for backend services that legitimately custody a key (and for tests).
type LocalSigner struct {
	sk string
	pk string
}

// NewLocalSigner builds a LocalSigner from a hex secret key.
func NewLocalSigner(skHex string) (*LocalSigner, error) {
	if err := ValidateSecretKey(skHex); err != nil {
		return nil, err
	}
	pk, err := nostr.GetPublicKey(skHex)
	if err != nil {
		return nil, fmt.Errorf("derive public key: %w", err)
	}
	return &LocalSigner{sk: skHex, pk: pk}, nil
}

// GenerateLocalSigner builds a LocalSigner from a fresh random keypair.
func GenerateLocalSigner() (*LocalSigner, error) {
	sk, pk, err := GenerateKeypair()
	if err != nil {
		return nil, err
	}
	return &LocalSigner{sk: sk, pk: pk}, nil
}

func (s *LocalSigner) PublicKey() string { return s.pk }

func (s *LocalSigner) Encrypt(_ context.Context, scheme Scheme, plaintext, recipientPK string) (string, error) {
	return encryptScheme(scheme, plaintext, s.sk, recipientPK)
}

func (s *LocalSigner) Decrypt(_ context.Context, scheme Scheme, ciphertext, senderPK string) (string, error) {
	return decryptScheme(scheme, ciphertext, s.sk, senderPK)
}

func (s *LocalSigner) SignEvent(_ context.Context, e *nostr.Event) error {
	e.PubKey = s.pk
	return e.Sign(s.sk)
}

func (s *LocalSigner) CanEncrypt() bool { return true }

// ClientSideSigner is a marker for NIP-07 mode. It performs no cryptography; it
// records the user's public key and reports that the client is responsible for
// encryption, decryption, and signing. Every crypto method returns ErrClientSide.
type ClientSideSigner struct {
	pk string
}

// NewClientSideSigner builds a marker signer for the given hex public key.
func NewClientSideSigner(pk string) *ClientSideSigner { return &ClientSideSigner{pk: pk} }

func (s *ClientSideSigner) PublicKey() string { return s.pk }

func (s *ClientSideSigner) Encrypt(context.Context, Scheme, string, string) (string, error) {
	return "", ErrClientSide
}

func (s *ClientSideSigner) Decrypt(context.Context, Scheme, string, string) (string, error) {
	return "", ErrClientSide
}

func (s *ClientSideSigner) SignEvent(context.Context, *nostr.Event) error { return ErrClientSide }

func (s *ClientSideSigner) CanEncrypt() bool { return false }

// Compile-time assertions that the concrete signers satisfy Signer.
var (
	_ Signer = (*LocalSigner)(nil)
	_ Signer = (*ClientSideSigner)(nil)
)
