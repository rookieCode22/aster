package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type contextKey string

const userContextKey contextKey = "auth_user"

func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func GetUser(r interface{ Context() context.Context }) *User {
	if user, ok := r.Context().Value(userContextKey).(*User); ok {
		return user
	}
	return nil
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Simple password hashing (SHA256-based for zero-dependency demo).
// In production, replace with bcrypt or argon2.
func hashPassword(password string) string {
	salt := make([]byte, 8)
	rand.Read(salt)
	hash := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash[:])
}

func checkPassword(password, stored string) bool {
	parts := splitN(stored, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, _ := hex.DecodeString(parts[0])
	expectedHash, _ := hex.DecodeString(parts[1])
	hash := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(hash[:]) == hex.EncodeToString(expectedHash)
}

func splitN(s, sep string, n int) []string {
	result := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		if idx := indexOf(s, sep); idx >= 0 {
			result = append(result, s[:idx])
			s = s[idx+len(sep):]
		} else {
			result = append(result, s)
			return result
		}
	}
	result = append(result, s)
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// Ensure unused import doesn't cause issues
var _ = fmt.Sprintf
