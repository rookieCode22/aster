// license-gen is a CLI tool for generating signed Aster license files.
//
// Usage:
//
//	license-gen --id LIC-001 --customer "Acme Corp" --email acme@example.com \
//	            --plan yearly --modules "recon,exploit,web" --duration 365d \
//	            --max-agents 5 --output acme.lic
//
// The signing secret is read from the ASTER_LICENSE_SECRET env var or derived
// from the same key fragments used by the vault.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"aster/internal/license"
)

func main() {
	var (
		id        = flag.String("id", "", "License ID (e.g., LIC-001)")
		customer  = flag.String("customer", "", "Customer name")
		email     = flag.String("email", "", "Customer email")
		plan      = flag.String("plan", "monthly", "Plan: monthly, yearly, trial, perpetual")
		modules   = flag.String("modules", "*", "Comma-separated module names (use * for all)")
		duration  = flag.String("duration", "30d", "Duration: 7d, 30d, 365d, etc. (ignored for perpetual)")
		maxAgents = flag.Int("max-agents", 0, "Max concurrent agents (0=unlimited)")
		hwid      = flag.String("hwid", "", "Hardware ID for binding (empty=no binding)")
		output    = flag.String("output", "license.lic", "Output file path")
		secret    = flag.String("secret", "", "Signing secret (or set ASTER_LICENSE_SECRET env)")
	)
	flag.Parse()

	if *id == "" || *customer == "" || *email == "" {
		fmt.Fprintln(os.Stderr, "Error: --id, --customer, and --email are required")
		flag.Usage()
		os.Exit(1)
	}

	// Determine secret
	signingSecret := *secret
	if signingSecret == "" {
		signingSecret = os.Getenv("ASTER_LICENSE_SECRET")
	}
	if signingSecret == "" {
		// Fallback: derive from same key material as vault
		signingSecret = deriveKey()
	}
	fmt.Fprintf(os.Stderr, "Using signing secret (first 8 chars): %s...\n", signingSecret[:min(8, len(signingSecret))])

	// Parse plan
	var p license.Plan
	switch *plan {
	case "monthly":
		p = license.PlanMonthly
	case "yearly":
		p = license.PlanYearly
	case "trial":
		p = license.PlanTrial
	case "perpetual":
		p = license.PlanPerpetual
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown plan '%s'\n", *plan)
		os.Exit(1)
	}

	// Parse duration
	dur, err := time.ParseDuration(*duration)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid duration '%s': %v\n", *duration, err)
		os.Exit(1)
	}

	// Parse modules
	modList := strings.Split(*modules, ",")
	for i := range modList {
		modList[i] = strings.TrimSpace(modList[i])
	}

	// Generate license
	gen := license.NewGenerator([]byte(signingSecret))
	lic, err := gen.Issue(*id, *customer, *email, p, modList, dur, *hwid, *maxAgents)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating license: %v\n", err)
		os.Exit(1)
	}

	// Write output
	data, err := json.MarshalIndent(lic, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling license: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing license file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("License generated: %s\n", *output)
	fmt.Printf("  ID:       %s\n", lic.ID)
	fmt.Printf("  Customer: %s\n", lic.Customer)
	fmt.Printf("  Plan:     %s\n", lic.Plan)
	fmt.Printf("  Modules:  %s\n", strings.Join(lic.Modules, ", "))
	fmt.Printf("  Expires:  %s\n", lic.ExpiresAt)
	fmt.Printf("  Agents:   %d\n", lic.MaxAgents)
	if lic.HWID != "" {
		fmt.Printf("  HWID:     %s\n", lic.HWID)
	}
}

// deriveKey creates a fallback signing key from the same key fragments as vault.
// In production, use ASTER_LICENSE_SECRET env var instead.
func deriveKey() string {
	// This is intentionally a different derivation than the vault key.
	// In a real build, these would be injected at compile time.
	h := sha256.New()
	h.Write([]byte("aster-license-signing-key-v1"))
	return fmt.Sprintf("%x", h.Sum(nil))
}
