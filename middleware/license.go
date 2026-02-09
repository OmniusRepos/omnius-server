package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"torrent-server/services"
)

// LicenseMiddleware enforces license validity on protected routes
type LicenseMiddleware struct {
	client *services.LicenseClient
}

func NewLicenseMiddleware(client *services.LicenseClient) *LicenseMiddleware {
	return &LicenseMiddleware{client: client}
}

// EnforceValid blocks requests if the license is not valid.
// Admin panel routes stay accessible for troubleshooting.
func (lm *LicenseMiddleware) EnforceValid(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always allow admin routes for troubleshooting
		if strings.HasPrefix(r.URL.Path, "/admin") {
			next.ServeHTTP(w, r)
			return
		}

		// Always allow health check
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Always allow license endpoints (so client can phone home)
		if strings.HasPrefix(r.URL.Path, "/api/v2/license/") {
			next.ServeHTTP(w, r)
			return
		}

		if !lm.client.IsValid() {
			status := lm.client.GetStatus()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":         "error",
				"status_message": "License invalid: " + status.Message,
				"license_status": status.Mode,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// DemoLimiter applies demo mode restrictions (max 10 movies, 2 series, no channels)
// This is applied as additional middleware when in demo mode
type DemoLimiter struct {
	client *services.LicenseClient
}

func NewDemoLimiter(client *services.LicenseClient) *DemoLimiter {
	return &DemoLimiter{client: client}
}

// InjectDemoFlag adds "demo": true to JSON responses when in demo mode.
// It also blocks channel endpoints in demo mode.
func (dl *DemoLimiter) InjectDemoFlag(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !dl.client.IsDemo() {
			next.ServeHTTP(w, r)
			return
		}

		// Block channel endpoints in demo mode
		if strings.Contains(r.URL.Path, "channel") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":         "error",
				"status_message": "Channels are not available in demo mode",
				"demo":           true,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
