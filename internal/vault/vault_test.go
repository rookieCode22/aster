package vault

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := Key()
	if len(key) != 32 {
		t.Fatalf("key length = %d, want 32", len(key))
	}

	plaintext := []byte("this is a secret skill content that should be encrypted")

	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("roundtrip mismatch:\n got  %q\n want %q", decrypted, plaintext)
	}

	t.Logf("✅ encrypt/decrypt roundtrip OK (encrypted %d bytes → %d)", len(plaintext), len(encrypted))
}

func TestDecryptWrongKey(t *testing.T) {
	plaintext := []byte("secret")
	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt the encrypted data
	corrupted := make([]byte, len(encrypted))
	copy(corrupted, encrypted)
	corrupted[15] ^= 0xFF // flip bits in ciphertext

	_, err = Decrypt(corrupted)
	if err == nil {
		t.Fatal("expected decryption failure on corrupted data")
	}
	t.Log("✅ corrupted data correctly rejected")
}
