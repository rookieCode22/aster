# Phase 5 — License System Integration

## Files Added

```
internal/license/license.go          — License struct, Generator, Verifier (HMAC-SHA256)
internal/store/licenses.go          — License storage (ActivateLicense, GetActiveLicense, ListLicenses)
internal/server/api/license.go      — License API handler (activate, status, history, check)
internal/server/middleware/license.go — LicenseRequired & AgentLimit middleware
internal/server/server_license.go   — RegisterLicenseRoutes helper
cmd/license-gen/main.go             — CLI tool to generate signed .lic files
```

## server.go Integration

Add these imports:
```go
import (
    "github.com/Q16G/aster/internal/license"
    "github.com/Q16G/aster/internal/server/api"
)
```

In your Server struct or New() function, add after store initialization:
```go
// License verifier (uses same secret as vault or ASTER_LICENSE_SECRET env)
licenseSecret := os.Getenv("ASTER_LICENSE_SECRET")
if licenseSecret == "" {
    licenseSecret = vault.DeriveLicenseKey() // or sha256 of vault key
}
verifier := license.NewVerifier([]byte(licenseSecret))
licHandler := api.NewLicenseHandler(s.store, verifier)
RegisterLicenseRoutes(s.mux, s.store, licHandler)
```

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/license/activate` | None | Activate a license (body = .lic JSON) |
| GET | `/api/v1/license/status` | None | Current license status |
| GET | `/api/v1/license/history` | License | All past licenses |
| GET | `/api/v1/license/check/{module}` | License | Check if module is authorized |

## Generating Licenses

```bash
export ASTER_LICENSE_SECRET="your-64-char-hex-secret"
go run ./cmd/license-gen \
  --id LIC-001 \
  --customer "Acme Corp" \
  --email acme@example.com \
  --plan yearly \
  --modules "recon,exploit,web,cloud" \
  --duration 365d \
  --max-agents 10 \
  --output acme.lic
```

## Optional: Agent Limit on WebSocket

In server.go, wrap the WebSocket handler:
```go
var agentCounter int32
wsHandler := ws.NewHub(s.store)
s.mux.Handle("/ws/chat/", middleware.AgentLimit(s.store, &agentCounter)(wsHandler))
```
