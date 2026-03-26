package crypto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// Encryptor encrypts and decrypts data using age with a scrypt passphrase.
type Encryptor struct {
	passphrase string
}

// NewEncryptor creates an Encryptor with the given passphrase.
func NewEncryptor(passphrase string) *Encryptor {
	return &Encryptor{passphrase: passphrase}
}

// Encrypt encrypts plaintext and returns a base64-encoded ciphertext.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	recipient, err := age.NewScryptRecipient(e.passphrase)
	if err != nil {
		return "", fmt.Errorf("creating scrypt recipient: %w", err)
	}
	// Use lower work factor for encrypted-at-rest keys (fast encrypt/decrypt).
	recipient.SetWorkFactor(15)

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return "", fmt.Errorf("creating encrypt writer: %w", err)
	}
	if _, err := io.WriteString(w, plaintext); err != nil {
		return "", fmt.Errorf("writing plaintext: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("closing encrypt writer: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext.
func (e *Encryptor) Decrypt(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decoding base64: %w", err)
	}

	identity, err := age.NewScryptIdentity(e.passphrase)
	if err != nil {
		return "", fmt.Errorf("creating scrypt identity: %w", err)
	}

	r, err := age.Decrypt(bytes.NewReader(ciphertext), identity)
	if err != nil {
		return "", fmt.Errorf("decrypting: %w", err)
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("reading plaintext: %w", err)
	}

	return string(plaintext), nil
}

// EncryptArmored encrypts plaintext and returns age-armored (PEM-like) output.
func (e *Encryptor) EncryptArmored(plaintext string) (string, error) {
	recipient, err := age.NewScryptRecipient(e.passphrase)
	if err != nil {
		return "", fmt.Errorf("creating scrypt recipient: %w", err)
	}
	recipient.SetWorkFactor(15)

	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)
	w, err := age.Encrypt(armorWriter, recipient)
	if err != nil {
		return "", fmt.Errorf("creating encrypt writer: %w", err)
	}
	if _, err := io.WriteString(w, plaintext); err != nil {
		return "", fmt.Errorf("writing plaintext: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("closing encrypt writer: %w", err)
	}
	if err := armorWriter.Close(); err != nil {
		return "", fmt.Errorf("closing armor writer: %w", err)
	}

	return buf.String(), nil
}
