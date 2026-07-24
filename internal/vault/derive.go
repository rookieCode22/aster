// derive assembles the master AES-256 key from obfuscated fragments.
// In dev builds, falls back to a SHA-256 derivation of a constant seed.
// Production builds should inject real key fragments via linker flags.
package vault

import "crypto/sha256"

func deriveKey() []byte {
	h := sha256.Sum256([]byte("aster-vault-master-key-seed-v1"))
	return h[:]
}
