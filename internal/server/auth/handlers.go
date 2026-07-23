package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aster/internal/store"
)

type jwtClaims struct {
	UserID   string `json:"sub"`
	Username string `json:"name"`
	Exp      int64  `json:"exp"`
}

func (s *Service) createToken(user *store.UserRow) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := jwtClaims{UserID: user.ID, Username: user.Username, Exp: time.Now().Add(24 * time.Hour).Unix()}
	claimsJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// VerifyToken validates a JWT and returns the user.
func (s *Service) VerifyToken(tokenStr string) (*store.UserRow, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token")
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signingInput))
	if !hmac.Equal([]byte(parts[2]), []byte(base64.RawURLEncoding.EncodeToString(mac.Sum(nil)))) {
		return nil, fmt.Errorf("invalid signature")
	}
	claimsJSON, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil || time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}
	return s.store.GetUserByID(claims.UserID)
}

// --- HTTP Handlers ---

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type authResponse struct {
	Token string         `json:"token"`
	User  *store.UserRow `json:"user"`
}

func (s *Service) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	json.NewDecoder(r.Body).Decode(&req)
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, 400, map[string]string{"error": "username and password required"})
		return
	}
	user, err := s.store.CreateUser(req.Username, req.Password)
	if err != nil {
		writeJSON(w, 409, map[string]string{"error": err.Error()})
		return
	}
	token, _ := s.createToken(user)
	writeJSON(w, 201, authResponse{Token: token, User: user})
}

func (s *Service) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	json.NewDecoder(r.Body).Decode(&req)
	user, err := s.store.GetUserByUsername(req.Username)
	if err != nil || user == nil || !user.VerifyPassword(req.Password) {
		writeJSON(w, 401, map[string]string{"error": "invalid credentials"})
		return
	}
	token, _ := s.createToken(user)
	writeJSON(w, 200, authResponse{Token: token, User: user})
}

func (s *Service) HandleMe(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	if user == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, 200, user)
}

func (s *Service) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, 401, map[string]string{"error": "missing token"})
			return
		}
		user, err := s.VerifyToken(strings.TrimPrefix(authHeader, "Bearer "))
		if err != nil {
			writeJSON(w, 401, map[string]string{"error": err.Error()})
			return
		}
		next(w, r.WithContext(WithUser(r.Context(), user)))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
