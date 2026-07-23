// Package license provides license generation, signing, and verification.
// Licenses are JSON files signed with HMAC-SHA256, supporting time-based
// subscriptions (monthly/yearly) and optional hardware binding.
package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Plan represents a subscription tier.
type Plan string

const (
	PlanMonthly Plan = "monthly"
	PlanYearly  Plan = "yearly"
	PlanTrial   Plan = "trial"
	PlanPerpetual Plan = "perpetual"
)

// License is the serialized license payload.
type License struct {
	ID        string   `json:"id"`
	Customer  string   `json:"customer"`
	Email     string   `json:"email"`
	Plan      Plan     `json:"plan"`
	Modules   []string `json:"modules"`    // authorized module names
	IssuedAt  string   `json:"issued_at"`  // RFC3339
	ExpiresAt string   `json:"expires_at"` // RFC3339, empty for perpetual
	MaxAgents int      `json:"max_agents"` // max concurrent agent sessions, 0 = unlimited
	HWID      string   `json:"hwid,omitempty"` // hardware fingerprint (optional)
	Signature string   `json:"signature"`       // HMAC-SHA256(base64url)
}

// Generator creates signed licenses.
type Generator struct {
	secret []byte
}

// NewGenerator creates a license generator with the given signing secret.
func NewGenerator(secret []byte) *Generator {
	return &Generator{secret: secret}
}

// Issue creates a new signed license.
func (g *Generator) Issue(id, customer, email string, plan Plan, modules []string, duration time.Duration, hwid string, maxAgents int) (*License, error) {
	now := time.Now().UTC()
	lic := &License{
		ID:        id,
		Customer:  customer,
		Email:     email,
		Plan:      plan,
		Modules:   modules,
		IssuedAt:  now.Format(time.RFC3339),
		MaxAgents: maxAgents,
		HWID:      hwid,
	}
	if plan != PlanPerpetual {
		lic.ExpiresAt = now.Add(duration).Format(time.RFC3339)
	}
	if err := g.sign(lic); err != nil {
		return nil, err
	}
	return lic, nil
}

// sign computes and sets the HMAC-SHA256 signature on the license.
func (g *Generator) sign(lic *License) error {
	payload, err := lic.payloadBytes()
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, g.secret)
	mac.Write(payload)
	lic.Signature = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return nil
}

// payloadBytes serializes the license without the signature field for signing.
func (lic *License) payloadBytes() ([]byte, error) {
	// Marshal without Signature
	type payload License
	p := payload(*lic)
	p.Signature = ""
	return json.Marshal(p)
}

// Verifier checks license validity.
type Verifier struct {
	secret    []byte
	hwidFunc  func() string // optional: returns current machine HWID
}

// NewVerifier creates a license verifier.
func NewVerifier(secret []byte) *Verifier {
	return &Verifier{secret: secret}
}

// WithHWID sets a hardware ID function for binding checks.
func (v *Verifier) WithHWID(fn func() string) *Verifier {
	v.hwidFunc = fn
	return v
}

// Verify checks the license signature, expiration, and optional HWID binding.
// Returns the parsed License and any error.
func (v *Verifier) Verify(data []byte) (*License, error) {
	var lic License
	if err := json.Unmarshal(data, &lic); err != nil {
		return nil, fmt.Errorf("invalid license format: %w", err)
	}

	// 1. Verify signature
	if err := v.verifySignature(&lic); err != nil {
		return nil, err
	}

	// 2. Check expiration
	if lic.ExpiresAt != "" && lic.Plan != PlanPerpetual {
		exp, err := time.Parse(time.RFC3339, lic.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("invalid expiration date: %w", err)
		}
		if time.Now().UTC().After(exp) {
			return nil, errors.New("license has expired")
		}
	}

	// 3. Check HWID binding (if set)
	if lic.HWID != "" && v.hwidFunc != nil {
		currentHWID := v.hwidFunc()
		if currentHWID != "" && currentHWID != lic.HWID {
			return nil, fmt.Errorf("license is bound to a different machine (hwid mismatch)")
		}
	}

	return &lic, nil
}

// verifySignature checks the HMAC signature.
func (v *Verifier) verifySignature(lic *License) error {
	if lic.Signature == "" {
		return errors.New("license has no signature")
	}
	payload, err := lic.payloadBytes()
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(payload)
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(lic.Signature), []byte(expected)) {
		return errors.New("license signature is invalid (tampered or corrupted)")
	}
	return nil
}

// HasModule checks if the license authorizes a specific module.
func (lic *License) HasModule(name string) bool {
	for _, m := range lic.Modules {
		if m == name || m == "*" {
			return true
		}
	}
	return false
}

// IsExpired checks if the license is past its expiration.
func (lic *License) IsExpired() bool {
	if lic.ExpiresAt == "" || lic.Plan == PlanPerpetual {
		return false
	}
	exp, err := time.Parse(time.RFC3339, lic.ExpiresAt)
	if err != nil {
		return true // unparseable = expired
	}
	return time.Now().UTC().After(exp)
}

// DaysRemaining returns the number of days until expiration.
func (lic *License) DaysRemaining() int {
	if lic.ExpiresAt == "" || lic.Plan == PlanPerpetual {
		return -1 // unlimited
	}
	exp, err := time.Parse(time.RFC3339, lic.ExpiresAt)
	if err != nil {
		return 0
	}
	days := int(time.Until(exp).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
