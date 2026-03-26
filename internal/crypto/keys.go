package crypto

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// GenerateKeyPair generates a new WireGuard private/public key pair.
// Returns (privateKey, publicKey, error) as base64-encoded strings.
func GenerateKeyPair() (string, string, error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", fmt.Errorf("generating private key: %w", err)
	}

	return key.String(), key.PublicKey().String(), nil
}

// GeneratePresharedKey generates a random preshared key.
func GeneratePresharedKey() (string, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return "", fmt.Errorf("generating preshared key: %w", err)
	}

	return key.String(), nil
}

// PublicKeyFromPrivate derives the public key from a base64-encoded private key.
func PublicKeyFromPrivate(privateKeyBase64 string) (string, error) {
	key, err := wgtypes.ParseKey(privateKeyBase64)
	if err != nil {
		return "", fmt.Errorf("parsing private key: %w", err)
	}

	// Derive public key via Curve25519 scalar base multiplication
	var pub, priv [32]byte
	copy(priv[:], key[:])
	curve25519.ScalarBaseMult(&pub, &priv)

	pubKey, err := wgtypes.NewKey(pub[:])
	if err != nil {
		return "", fmt.Errorf("creating public key: %w", err)
	}

	return pubKey.String(), nil
}

// GenerateRandomBytes generates n random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("reading random bytes: %w", err)
	}
	return b, nil
}
