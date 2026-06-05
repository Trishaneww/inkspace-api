package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func newTestKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, keyLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func TestCipherRoundTrip(t *testing.T) {
	c, err := NewCipher(newTestKey(t))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	plaintext := "ya29.a0AfH6SMexample-refresh-token"
	encoded, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if encoded == plaintext {
		t.Fatal("ciphertext equals plaintext")
	}

	got, err := c.Decrypt(encoded)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestEncryptUsesRandomNonce(t *testing.T) {
	c, err := NewCipher(newTestKey(t))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	a, _ := c.Encrypt("same")
	b, _ := c.Encrypt("same")
	if a == b {
		t.Fatal("expected distinct ciphertexts for repeated plaintext")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	enc, err := NewCipher(newTestKey(t))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	encoded, err := enc.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	other, err := NewCipher(newTestKey(t))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	if _, err := other.Decrypt(encoded); err == nil {
		t.Fatal("expected decryption with wrong key to fail")
	}
}

func TestNewCipherRejectsBadKey(t *testing.T) {
	if _, err := NewCipher(""); err == nil {
		t.Fatal("expected error for empty key")
	}
	if _, err := NewCipher("not-base64!!!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
	short := base64.StdEncoding.EncodeToString([]byte("too-short"))
	if _, err := NewCipher(short); err == nil {
		t.Fatal("expected error for wrong key length")
	}
}
