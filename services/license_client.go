package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"torrent-server/models"
)

const (
	licenseCacheFile  = "data/.license-cache.json"
	cacheValidDays    = 7
	cacheGraceDays    = 30
	heartbeatInterval = 6 * time.Hour
)

// LicenseClient handles phone-home, caching, and heartbeat for customer binaries
type LicenseClient struct {
	licenseKey    string
	serverURL     string
	fingerprint   string
	serverVersion string
	domain        string
	httpClient    *http.Client

	mu     sync.RWMutex
	status LicenseStatus
	cache  *models.LicenseCacheData

	stopCh chan struct{}
}

// LicenseStatus represents the current license state of this instance
type LicenseStatus struct {
	Mode       string   `json:"mode"`        // "licensed", "demo", "grace", "invalid"
	Plan       string   `json:"plan"`        // plan name if licensed
	Message    string   `json:"message"`     // human-readable status
	GraceEnd   string   `json:"grace_end"`   // when grace period expires
	Valid      bool     `json:"valid"`       // whether the server should operate normally
	DemoMode   bool     `json:"demo_mode"`   // true if running in demo mode
	LicenseKey string   `json:"license_key"` // masked key for display
	Features   []string `json:"features"`    // enabled features for this license
}

func NewLicenseClient(licenseKey, serverURL, fingerprint, version, domain string) *LicenseClient {
	return &LicenseClient{
		licenseKey:    licenseKey,
		serverURL:     serverURL,
		fingerprint:   fingerprint,
		serverVersion: version,
		domain:        domain,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
		stopCh:        make(chan struct{}),
	}
}

// Start performs initial validation and starts the heartbeat goroutine.
// Returns an error only if startup should be blocked.
func (c *LicenseClient) Start() error {
	if c.licenseKey == "" {
		c.mu.Lock()
		c.status = LicenseStatus{
			Mode:     "demo",
			Message:  "No license key set - running in demo mode",
			Valid:    true,
			DemoMode: true,
		}
		c.mu.Unlock()
		log.Println("[License] No LICENSE_KEY set - running in DEMO mode (limited features)")
		return nil
	}

	maskedKey := maskKey(c.licenseKey)

	// Try loading cache first
	c.cache = c.loadCache()

	if c.cache != nil && c.cache.Valid && c.cache.LicenseKey == c.licenseKey {
		age := time.Since(c.cache.ValidatedAt)

		if age < cacheValidDays*24*time.Hour {
			// Cache is fresh — use it, revalidate in background
			log.Printf("[License] Using cached validation (age: %s) for %s", age.Round(time.Minute), maskedKey)
			c.mu.Lock()
			c.status = LicenseStatus{
				Mode:       "licensed",
				Plan:       c.cache.Plan,
				Message:    "License active (cached)",
				Valid:      true,
				LicenseKey: maskedKey,
				Features:   c.cache.Features,
			}
			c.mu.Unlock()

			go c.revalidate()
			go c.heartbeatLoop()
			return nil
		}

		if age < cacheGraceDays*24*time.Hour {
			// Stale cache — try to revalidate, fall back to grace period
			resp, err := c.activate()
			if err == nil && resp.Valid {
				c.applyResponse(resp, maskedKey)
				go c.heartbeatLoop()
				return nil
			}

			// Network error with stale cache — grace period
			graceEnd := c.cache.ValidatedAt.Add(cacheGraceDays * 24 * time.Hour)
			log.Printf("[License] Cannot reach license server, grace period until %s", graceEnd.Format("2006-01-02"))
			c.mu.Lock()
			c.status = LicenseStatus{
				Mode:       "grace",
				Plan:       c.cache.Plan,
				Message:    fmt.Sprintf("License server unreachable - grace period until %s", graceEnd.Format("2006-01-02")),
				GraceEnd:   graceEnd.Format(time.RFC3339),
				Valid:      true,
				LicenseKey: maskedKey,
				Features:   c.cache.Features,
			}
			c.mu.Unlock()
			go c.heartbeatLoop()
			return nil
		}
	}

	// No cache or expired — must activate
	resp, err := c.activate()
	if err != nil {
		log.Printf("[License] Failed to activate: %v", err)
		return fmt.Errorf("license validation failed: cannot reach license server at %s - %w", c.serverURL, err)
	}

	if !resp.Valid {
		log.Printf("[License] License invalid: %s", resp.Message)
		c.mu.Lock()
		c.status = LicenseStatus{
			Mode:       "invalid",
			Message:    resp.Message,
			Valid:      false,
			LicenseKey: maskedKey,
		}
		c.mu.Unlock()
		return fmt.Errorf("license invalid: %s", resp.Message)
	}

	c.applyResponse(resp, maskedKey)
	go c.heartbeatLoop()
	return nil
}

// Stop sends a deactivation request and stops the heartbeat
func (c *LicenseClient) Stop() {
	close(c.stopCh)

	if c.licenseKey == "" {
		return
	}

	req := &models.LicenseDeactivateRequest{
		LicenseKey:         c.licenseKey,
		MachineFingerprint: c.fingerprint,
	}
	body, _ := json.Marshal(req)
	resp, err := c.httpClient.Post(c.serverURL+"/api/v2/license/deactivate", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[License] Deactivation request failed: %v", err)
		return
	}
	resp.Body.Close()
	log.Println("[License] Deployment deactivated")
}

// Restart stops the current heartbeat, sets a new license key, and re-activates.
func (c *LicenseClient) Restart(newKey string) error {
	// Stop current heartbeat loop
	select {
	case <-c.stopCh:
		// Already closed
	default:
		close(c.stopCh)
	}

	c.mu.Lock()
	c.licenseKey = newKey
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	// Persist the new key so it survives restarts
	os.WriteFile("data/.license-key", []byte(newKey), 0600)

	return c.Start()
}

// GetKey returns the current (unmasked) license key.
func (c *LicenseClient) GetKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.licenseKey
}

// GetFingerprint returns this machine's fingerprint.
func (c *LicenseClient) GetFingerprint() string {
	return c.fingerprint
}

// GetStatus returns the current license status
func (c *LicenseClient) GetStatus() LicenseStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

// IsDemo returns true if running in demo mode
func (c *LicenseClient) IsDemo() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status.DemoMode
}

// HasFeature returns true if the current license includes the specified feature
func (c *LicenseClient) HasFeature(feature string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, f := range c.status.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// IsValid returns true if the license allows normal operation
func (c *LicenseClient) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.status.DemoMode {
		return true
	}

	if c.status.Mode == "grace" {
		// Check if grace period has expired
		if c.status.GraceEnd != "" {
			graceEnd, err := time.Parse(time.RFC3339, c.status.GraceEnd)
			if err == nil && time.Now().After(graceEnd) {
				return false
			}
		}
		return true
	}

	return c.status.Valid
}

func (c *LicenseClient) activate() (*models.LicenseResponse, error) {
	hostname, _ := os.Hostname()
	req := &models.LicenseActivateRequest{
		LicenseKey:         c.licenseKey,
		MachineFingerprint: c.fingerprint,
		MachineLabel:       hostname,
		ServerVersion:      c.serverVersion,
		Domain:             c.domain,
	}
	body, _ := json.Marshal(req)
	resp, err := c.httpClient.Post(c.serverURL+"/api/v2/license/activate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lr models.LicenseResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, fmt.Errorf("invalid response from license server: %w", err)
	}
	return &lr, nil
}

func (c *LicenseClient) revalidate() {
	resp, err := c.activate()
	if err != nil {
		log.Printf("[License] Background revalidation failed: %v", err)
		return
	}
	if resp.Valid {
		maskedKey := maskKey(c.licenseKey)
		c.applyResponse(resp, maskedKey)
		log.Println("[License] Background revalidation successful")
	} else {
		log.Printf("[License] Background revalidation: license no longer valid: %s", resp.Message)
		c.mu.Lock()
		c.status.Mode = "invalid"
		c.status.Message = resp.Message
		// Don't immediately invalidate — let the cache grace period handle it
		c.mu.Unlock()
	}
}

func (c *LicenseClient) applyResponse(resp *models.LicenseResponse, maskedKey string) {
	c.mu.Lock()
	c.status = LicenseStatus{
		Mode:       "licensed",
		Plan:       resp.Plan,
		Message:    "License active",
		Valid:      true,
		LicenseKey: maskedKey,
		Features:   resp.Features,
	}
	c.mu.Unlock()

	c.saveCache(&models.LicenseCacheData{
		LicenseKey:    c.licenseKey,
		Plan:          resp.Plan,
		Valid:         true,
		Status:        resp.Status,
		GraceDays:     resp.GraceDays,
		ValidatedAt:   time.Now(),
		ExpiresAt:     resp.ExpiresAt,
		Fingerprint:   c.fingerprint,
		ServerVersion: c.serverVersion,
		Features:      resp.Features,
	})

	log.Printf("[License] Activated: plan=%s, key=%s", resp.Plan, maskedKey)
}

func (c *LicenseClient) heartbeatLoop() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.sendHeartbeat()
		case <-c.stopCh:
			return
		}
	}
}

func (c *LicenseClient) sendHeartbeat() {
	req := &models.LicenseHeartbeatRequest{
		LicenseKey:         c.licenseKey,
		MachineFingerprint: c.fingerprint,
		ServerVersion:      c.serverVersion,
		Domain:             c.domain,
	}
	body, _ := json.Marshal(req)
	resp, err := c.httpClient.Post(c.serverURL+"/api/v2/license/heartbeat", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[License] Heartbeat failed: %v", err)
		return
	}
	defer resp.Body.Close()

	var lr models.LicenseResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return
	}

	if lr.Valid {
		// Refresh cache on successful heartbeat
		c.saveCache(&models.LicenseCacheData{
			LicenseKey:    c.licenseKey,
			Plan:          lr.Plan,
			Valid:         true,
			Status:        lr.Status,
			GraceDays:     lr.GraceDays,
			ValidatedAt:   time.Now(),
			ExpiresAt:     lr.ExpiresAt,
			Fingerprint:   c.fingerprint,
			ServerVersion: c.serverVersion,
		})
	} else {
		log.Printf("[License] Heartbeat: server says license invalid: %s", lr.Message)
	}
}

func (c *LicenseClient) loadCache() *models.LicenseCacheData {
	data, err := os.ReadFile(licenseCacheFile)
	if err != nil {
		return nil
	}
	var cache models.LicenseCacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}
	return &cache
}

func (c *LicenseClient) saveCache(cache *models.LicenseCacheData) {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(licenseCacheFile, data, 0600)
}

func maskKey(key string) string {
	if len(key) <= 10 {
		return "OMNI-****-****-****-****"
	}
	return key[:10] + "****-****"
}
