package middleware

import (
	"net/http"
	"sync/atomic"
	"time"

	"aster/internal/store"
)

// LicenseRequired ensures an active, non-expired license exists.
// Returns 402 Payment Required if no license or expired.
func LicenseRequired(db *store.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec, err := db.GetActiveLicense()
			if err != nil {
				writeProblem(w, http.StatusInternalServerError, "license check failed")
				return
			}
			if rec == nil {
				writeProblem(w, http.StatusPaymentRequired, "no active license")
				return
			}
			if rec.ExpiresAt != nil && rec.ExpiresAt.Before(time.Now()) {
				writeProblem(w, http.StatusPaymentRequired, "license expired")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AgentLimit enforces max concurrent agent sessions via atomic counter.
// Pass nil for counter to skip tracking (unlimited).
func AgentLimit(db *store.DB, counter *int32) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if counter == nil {
				next.ServeHTTP(w, r)
				return
			}
			rec, err := db.GetActiveLicense()
			if err != nil || rec == nil || rec.MaxAgents <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			current := atomic.AddInt32(counter, 1)
			defer atomic.AddInt32(counter, -1)

			if current > int32(rec.MaxAgents) {
				writeProblem(w, http.StatusTooManyRequests, "agent limit reached")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + detail + `"}`))
}
