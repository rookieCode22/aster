// Package auth provides JWT authentication and user management.
// In Phase 2, users are stored in-memory. Phase 3 will add database persistence.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// User represents a registered user.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// Service handles authentication.
type Service struct {
	secret   []byte
	mu       sync.RWMutex
	users    map[string]*userRecord // username → record
	sessions map[string]*User       // token → user (simplified session store)
}

type userRecord struct {
	ID           string
	Username     string
	PasswordHash string // bcrypt hash
}

func NewService(jwtSecret string) *Service {
	return &Service{
		secret:   []byte(jwtSecret),
		users:    make(map[string]*userRecord),
		sessions: make(map[string]*User),
	}
}

// --- JWT ---

type jwtClaims struct {
	UserID   string `json:"sub"`
	Username string `json:"name"`
	Exp      int64  `json:"exp"`
}

func (s *Service) createToken(user *User) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	claims := jwtClaims{
		UserID:   user.ID,
		Username: user.Username,
		Exp:      time.Now().Add(24 * time.Hour).Unix(),
	}
	claimsJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

func (s *Service) verifyToken(tokenStr string) (*User, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &User{ID: claims.UserID, Username: claims.Username}, nil
}

// --- HTTP Handlers ---

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func (s *Service) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	s.mu.Lock()
	if _, exists := s.users[req.Username]; exists {
		s.mu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]string{"error": "username taken"})
		return
	}
	id := generateID()
	s.users[req.Username] = &userRecord{
		ID:           id,
		Username:     req.Username,
		PasswordHash: hashPassword(req.Password),
	}
	s.mu.Unlock()

	user := &User{ID: id, Username: req.Username}
	token, err := s.createToken(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token creation failed"})
		return
	}

	s.mu.Lock()
	s.sessions[token] = user
	s.mu.Unlock()

	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: *user})
}

func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	s.mu.RLock()
	rec, exists := s.users[req.Username]
	s.mu.RUnlock()

	if !exists || !checkPassword(req.Password, rec.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	user := &User{ID: rec.ID, Username: rec.Username}
	token, err := s.createToken(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token creation failed"})
		return
	}

	s.mu.Lock()
	s.sessions[token] = user
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, authResponse{Token: token, User: *user})
}

func (s *Service) HandleMe(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// VerifyToken validates a JWT token string and returns the user.
func (s *Service) VerifyToken(tokenStr string) (*User, error) {
	return s.verifyToken(tokenStr)
}
func (s *Service) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing token"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		user, err := s.verifyToken(tokenStr)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		ctx := WithUser(r.Context(), user)
		next(w, r.WithContext(ctx))
	}
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
