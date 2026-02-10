package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"torrent-server/config"
	"torrent-server/database"
	"torrent-server/models"
	"torrent-server/services"
)

// PaddleHandler handles Paddle webhook and license lookup endpoints
type PaddleHandler struct {
	db          *database.DB
	service     *services.LicenseService
	cfg         *config.Config
	emailSvc    *services.EmailService
	rateLimiter *rateLimiter
}

func NewPaddleHandler(db *database.DB, service *services.LicenseService, cfg *config.Config, emailSvc *services.EmailService) *PaddleHandler {
	return &PaddleHandler{
		db:          db,
		service:     service,
		cfg:         cfg,
		emailSvc:    emailSvc,
		rateLimiter: newRateLimiter(10, time.Minute),
	}
}

// HandleWebhook handles POST /api/v2/paddle/webhook
func (h *PaddleHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Paddle] Failed to read body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Verify signature
	signature := r.Header.Get("Paddle-Signature")
	if h.cfg.PaddleWebhookSecret != "" {
		if !verifyPaddleSignature(signature, body, h.cfg.PaddleWebhookSecret) {
			log.Printf("[Paddle] Invalid signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse event
	var event models.PaddleWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("[Paddle] Failed to parse webhook: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[Paddle] Received event: %s (id: %s, txn: %s)", event.EventType, event.EventID, event.Data.ID)

	// Only handle transaction.completed
	if event.EventType != "transaction.completed" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored"})
		return
	}

	// Idempotency check
	txnID := event.Data.ID
	existing, err := h.db.GetLicenseByPaddleTransaction(txnID)
	if err == nil && existing != nil {
		log.Printf("[Paddle] License already exists for txn %s (key: %s)", txnID, existing.LicenseKey)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "already_processed"})
		return
	}

	// Extract customer info
	email := event.Data.Customer.Email
	name := event.Data.Customer.Name

	// Map price ID to plan
	plan := models.PlanPersonal
	for _, item := range event.Data.Items {
		if item.Price.ID == h.cfg.PaddleBusinessPriceID {
			plan = models.PlanBusiness
			break
		}
	}

	// Create license
	license, err := h.service.CreateLicense(&models.AdminCreateLicenseRequest{
		Plan:       plan,
		OwnerEmail: email,
		OwnerName:  name,
		Notes:      fmt.Sprintf("Paddle transaction: %s", txnID),
	})
	if err != nil {
		log.Printf("[Paddle] Failed to create license: %v", err)
		http.Error(w, "Failed to create license", http.StatusInternalServerError)
		return
	}

	log.Printf("[Paddle] Created license %s for %s (plan: %s, txn: %s)", license.LicenseKey, email, plan, txnID)

	// Send email in background (optional)
	if h.emailSvc != nil && email != "" {
		go func() {
			if err := h.emailSvc.SendLicenseEmail(email, name, license.LicenseKey, plan); err != nil {
				log.Printf("[Paddle] Failed to send license email to %s: %v", email, err)
			} else {
				log.Printf("[Paddle] License email sent to %s", email)
			}
		}()
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// LicenseLookup handles GET /api/v2/license/lookup?email=xxx
func (h *PaddleHandler) LicenseLookup(w http.ResponseWriter, r *http.Request) {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}

	if !h.rateLimiter.allow(ip) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": "Rate limit exceeded. Try again later."})
		return
	}

	email := r.URL.Query().Get("email")
	if email == "" {
		jsonError(w, "email parameter is required", http.StatusBadRequest)
		return
	}

	license, err := h.db.GetLicenseByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "No license found for this email"})
			return
		}
		jsonError(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"license_key":     license.LicenseKey,
		"plan":            license.Plan,
		"max_deployments": license.MaxDeployments,
	})
}

// verifyPaddleSignature verifies the Paddle webhook signature
// Paddle-Signature format: ts=TIMESTAMP;h1=HASH
func verifyPaddleSignature(header string, body []byte, secret string) bool {
	if header == "" {
		return false
	}

	var ts, h1 string
	for _, part := range strings.Split(header, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "ts":
			ts = kv[1]
		case "h1":
			h1 = kv[1]
		}
	}

	if ts == "" || h1 == "" {
		return false
	}

	// Compute HMAC-SHA256 of "ts:body"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte(":"))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(h1))
}

// --- Rate limiter ---

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Filter old entries
	var recent []time.Time
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.limit {
		rl.requests[key] = recent
		return false
	}

	rl.requests[key] = append(recent, now)
	return true
}
