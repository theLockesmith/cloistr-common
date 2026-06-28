package crypto

import (
	"context"
	"errors"
	"testing"

	"github.com/nbd-wtf/go-nostr"
)

func mustKeypair(t *testing.T) (sk, pk string) {
	t.Helper()
	sk, pk, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	return sk, pk
}

func TestNIP44_RoundTrip(t *testing.T) {
	aliceSK, alicePK := mustKeypair(t)
	bobSK, bobPK := mustKeypair(t)

	const msg = "the duck swims at midnight 🦆"

	ct, err := EncryptNIP44(msg, aliceSK, bobPK)
	if err != nil {
		t.Fatalf("EncryptNIP44: %v", err)
	}
	if ct == msg {
		t.Fatal("ciphertext equals plaintext")
	}

	// Bob decrypts with his SK and Alice's PK (symmetric conversation key).
	got, err := DecryptNIP44(ct, bobSK, alicePK)
	if err != nil {
		t.Fatalf("DecryptNIP44: %v", err)
	}
	if got != msg {
		t.Errorf("round-trip = %q, want %q", got, msg)
	}
}

func TestNIP04_RoundTrip(t *testing.T) {
	aliceSK, alicePK := mustKeypair(t)
	bobSK, bobPK := mustKeypair(t)

	const msg = "legacy but supported"

	ct, err := EncryptNIP04(msg, aliceSK, bobPK)
	if err != nil {
		t.Fatalf("EncryptNIP04: %v", err)
	}
	got, err := DecryptNIP04(ct, bobSK, alicePK)
	if err != nil {
		t.Fatalf("DecryptNIP04: %v", err)
	}
	if got != msg {
		t.Errorf("round-trip = %q, want %q", got, msg)
	}
}

func TestLocalSigner_RoundTrip(t *testing.T) {
	alice, err := GenerateLocalSigner()
	if err != nil {
		t.Fatalf("GenerateLocalSigner: %v", err)
	}
	bob, err := GenerateLocalSigner()
	if err != nil {
		t.Fatalf("GenerateLocalSigner: %v", err)
	}
	ctx := context.Background()

	for _, scheme := range []Scheme{SchemeNIP44, SchemeNIP04} {
		const msg = "via the Signer interface"
		ct, err := alice.Encrypt(ctx, scheme, msg, bob.PublicKey())
		if err != nil {
			t.Fatalf("[%s] Encrypt: %v", scheme, err)
		}
		got, err := bob.Decrypt(ctx, scheme, ct, alice.PublicKey())
		if err != nil {
			t.Fatalf("[%s] Decrypt: %v", scheme, err)
		}
		if got != msg {
			t.Errorf("[%s] round-trip = %q, want %q", scheme, got, msg)
		}
	}
}

func TestLocalSigner_SignEvent(t *testing.T) {
	s, err := GenerateLocalSigner()
	if err != nil {
		t.Fatalf("GenerateLocalSigner: %v", err)
	}
	e := &nostr.Event{
		Kind:      nostr.KindTextNote,
		CreatedAt: nostr.Timestamp(1000),
		Content:   "hello",
	}
	if err := s.SignEvent(context.Background(), e); err != nil {
		t.Fatalf("SignEvent: %v", err)
	}
	if e.PubKey != s.PublicKey() {
		t.Errorf("event PubKey = %q, want %q", e.PubKey, s.PublicKey())
	}
	ok, err := e.CheckSignature()
	if err != nil || !ok {
		t.Errorf("CheckSignature ok=%v err=%v, want valid signature", ok, err)
	}
}

func TestUnknownScheme(t *testing.T) {
	s, _ := GenerateLocalSigner()
	if _, err := s.Encrypt(context.Background(), Scheme("nip-bogus"), "x", s.PublicKey()); err == nil {
		t.Error("expected error for unknown scheme, got nil")
	}
}

func TestClientSideSigner_DefersToClient(t *testing.T) {
	_, pk := mustKeypair(t)
	s := NewClientSideSigner(pk)
	ctx := context.Background()

	if s.CanEncrypt() {
		t.Error("ClientSideSigner.CanEncrypt() = true, want false")
	}
	if s.PublicKey() != pk {
		t.Errorf("PublicKey() = %q, want %q", s.PublicKey(), pk)
	}
	if _, err := s.Encrypt(ctx, SchemeNIP44, "x", pk); !errors.Is(err, ErrClientSide) {
		t.Errorf("Encrypt err = %v, want ErrClientSide", err)
	}
	if _, err := s.Decrypt(ctx, SchemeNIP44, "x", pk); !errors.Is(err, ErrClientSide) {
		t.Errorf("Decrypt err = %v, want ErrClientSide", err)
	}
	if err := s.SignEvent(ctx, &nostr.Event{}); !errors.Is(err, ErrClientSide) {
		t.Errorf("SignEvent err = %v, want ErrClientSide", err)
	}
}

func TestValidateKeys(t *testing.T) {
	sk, pk := mustKeypair(t)
	if err := ValidateSecretKey(sk); err != nil {
		t.Errorf("ValidateSecretKey(valid) = %v", err)
	}
	if err := ValidatePublicKey(pk); err != nil {
		t.Errorf("ValidatePublicKey(valid) = %v", err)
	}
	if err := ValidateSecretKey("xyz"); err == nil {
		t.Error("ValidateSecretKey(non-hex) = nil, want error")
	}
	if err := ValidateSecretKey("abcd"); err == nil {
		t.Error("ValidateSecretKey(short) = nil, want error")
	}
}
