package crypto

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	enc := NewEncryptor("test-passphrase-12345")

	original := "this is a secret WireGuard private key"
	encrypted, err := enc.Encrypt(original)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if encrypted == original {
		t.Fatal("encrypted text should differ from original")
	}

	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if decrypted != original {
		t.Fatalf("round-trip failed: got %q, want %q", decrypted, original)
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	enc1 := NewEncryptor("passphrase-one")
	enc2 := NewEncryptor("passphrase-two")

	encrypted, err := enc1.Encrypt("secret data")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = enc2.Decrypt(encrypted)
	if err == nil {
		t.Fatal("expected error decrypting with wrong passphrase")
	}
}

func TestGenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	if priv == "" || pub == "" {
		t.Fatal("keys should not be empty")
	}
	if priv == pub {
		t.Fatal("private and public keys should differ")
	}

	// Derive public key and verify consistency
	derived, err := PublicKeyFromPrivate(priv)
	if err != nil {
		t.Fatalf("PublicKeyFromPrivate: %v", err)
	}
	if derived != pub {
		t.Fatalf("derived public key %q != generated %q", derived, pub)
	}
}

func TestGeneratePresharedKey(t *testing.T) {
	psk1, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey: %v", err)
	}

	psk2, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey: %v", err)
	}

	if psk1 == psk2 {
		t.Fatal("two generated PSKs should differ")
	}
}
