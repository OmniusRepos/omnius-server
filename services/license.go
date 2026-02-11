package services

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"torrent-server/database"
	"torrent-server/models"
)

// LicenseService handles license authority operations (server-side)
type LicenseService struct {
	db *database.DB
}

func NewLicenseService(db *database.DB) *LicenseService {
	return &LicenseService{db: db}
}

// GenerateKey creates a new license key in the format OMNI-XXXX-XXXX-XXXX-XXXX
func GenerateKey() string {
	return generateKeyWithPrefix("OMNI")
}

// GenerateLiveKey creates a license key in the format OMNI-LIVE-XXXX-XXXX
// for enterprise licenses that include live channel support
func GenerateLiveKey() string {
	return generateKeyWithPrefix("OMNI-LIVE")
}

func generateKeyWithPrefix(prefix string) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous: 0/O, 1/I/L
	segCount := 4
	if prefix == "OMNI-LIVE" {
		segCount = 2 // OMNI-LIVE-XXXX-XXXX
	}
	segments := make([]string, segCount)
	for i := range segments {
		b := make([]byte, 4)
		rand.Read(b)
		seg := make([]byte, 4)
		for j := range seg {
			seg[j] = chars[int(b[j])%len(chars)]
		}
		segments[i] = string(seg)
	}
	return prefix + "-" + strings.Join(segments, "-")
}

// IsLiveKey checks if a license key is a live-enabled enterprise key
func IsLiveKey(key string) bool {
	return strings.HasPrefix(key, "OMNI-LIVE-")
}

// CreateLicense generates a key and persists a new license
func (s *LicenseService) CreateLicense(req *models.AdminCreateLicenseRequest) (*models.License, error) {
	maxDep := req.MaxDeployments
	if maxDep <= 0 {
		switch req.Plan {
		case models.PlanPersonal:
			maxDep = 1
		case models.PlanBusiness:
			maxDep = 5
		case models.PlanEnterprise:
			maxDep = 50
		default:
			maxDep = 1
		}
	}

	// Enterprise plan gets a live key with live_channels feature
	features := models.PlanFeatures(req.Plan)
	var key string
	if req.Plan == models.PlanEnterprise {
		key = GenerateLiveKey()
	} else {
		key = GenerateKey()
	}

	l := &models.License{
		LicenseKey:     key,
		Plan:           req.Plan,
		OwnerEmail:     req.OwnerEmail,
		OwnerName:      req.OwnerName,
		MaxDeployments: maxDep,
		IsActive:       true,
		Notes:          req.Notes,
		Features:       strings.Join(features, ","),
	}

	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			l.ExpiresAt = &t
		}
	}

	id, err := s.db.CreateLicense(l)
	if err != nil {
		return nil, fmt.Errorf("failed to create license: %w", err)
	}
	l.ID = id

	s.db.LogLicenseEvent(id, models.EventCreated, "", "", fmt.Sprintf("Plan: %s, MaxDep: %d", l.Plan, l.MaxDeployments))

	return l, nil
}

// ValidateKey checks if a license key is valid for a given fingerprint
func (s *LicenseService) ValidateKey(key, fingerprint string) *models.LicenseResponse {
	l, err := s.db.GetLicenseByKey(key)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.LicenseResponse{Valid: false, Status: "invalid", Message: "License key not found"}
		}
		return &models.LicenseResponse{Valid: false, Status: "error", Message: "Internal error"}
	}

	if !l.IsActive {
		return &models.LicenseResponse{Valid: false, Status: "revoked", Message: "License has been revoked"}
	}

	if l.RevokedAt != nil {
		return &models.LicenseResponse{Valid: false, Status: "revoked", Message: "License has been revoked"}
	}

	if l.ExpiresAt != nil && time.Now().After(*l.ExpiresAt) {
		return &models.LicenseResponse{Valid: false, Status: "expired", Message: "License has expired"}
	}

	features := parseFeaturesFromLicense(l)
	resp := &models.LicenseResponse{
		Valid:          true,
		Plan:           l.Plan,
		Status:         "active",
		MaxDeployments: l.MaxDeployments,
		GraceDays:      7,
		Features:       features,
	}
	if l.ExpiresAt != nil {
		resp.ExpiresAt = l.ExpiresAt.Format(time.RFC3339)
	}
	return resp
}

// parseFeaturesFromLicense returns the features list from a license.
// Uses stored features field, falls back to key prefix detection + plan defaults.
func parseFeaturesFromLicense(l *models.License) []string {
	if l.Features != "" {
		return strings.Split(l.Features, ",")
	}
	// Fallback: detect from key format or plan
	if IsLiveKey(l.LicenseKey) {
		return []string{models.FeatureLiveChannels}
	}
	return models.PlanFeatures(l.Plan)
}

// Activate registers or reactivates a deployment for a license
func (s *LicenseService) Activate(req *models.LicenseActivateRequest, ip string) *models.LicenseResponse {
	l, err := s.db.GetLicenseByKey(req.LicenseKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.LicenseResponse{Valid: false, Status: "invalid", Message: "License key not found"}
		}
		return &models.LicenseResponse{Valid: false, Status: "error", Message: "Internal error"}
	}

	validation := s.ValidateKey(req.LicenseKey, req.MachineFingerprint)
	if !validation.Valid {
		return validation
	}

	// Check if this fingerprint already has an active deployment
	active, _ := s.db.IsDeploymentActive(l.ID, req.MachineFingerprint)
	if !active {
		// Check deployment limit
		count, _ := s.db.CountActiveDeployments(l.ID)
		if count >= l.MaxDeployments {
			return &models.LicenseResponse{
				Valid:   false,
				Status:  "over_limit",
				Message: fmt.Sprintf("Maximum deployments reached (%d/%d)", count, l.MaxDeployments),
			}
		}
	}

	_, err = s.db.UpsertDeployment(l.ID, req.MachineFingerprint, req.MachineLabel, ip, req.ServerVersion)
	if err != nil {
		return &models.LicenseResponse{Valid: false, Status: "error", Message: "Failed to register deployment"}
	}

	s.db.LogLicenseEvent(l.ID, models.EventActivated, req.MachineFingerprint, ip, req.MachineLabel)

	features := parseFeaturesFromLicense(l)
	resp := &models.LicenseResponse{
		Valid:          true,
		Plan:           l.Plan,
		Status:         "active",
		MaxDeployments: l.MaxDeployments,
		GraceDays:      7,
		Features:       features,
	}
	if l.ExpiresAt != nil {
		resp.ExpiresAt = l.ExpiresAt.Format(time.RFC3339)
	}
	return resp
}

// Heartbeat updates the last-seen timestamp for a deployment
func (s *LicenseService) Heartbeat(req *models.LicenseHeartbeatRequest, ip string) *models.LicenseResponse {
	l, err := s.db.GetLicenseByKey(req.LicenseKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.LicenseResponse{Valid: false, Status: "invalid", Message: "License key not found"}
		}
		return &models.LicenseResponse{Valid: false, Status: "error", Message: "Internal error"}
	}

	validation := s.ValidateKey(req.LicenseKey, req.MachineFingerprint)
	if !validation.Valid {
		return validation
	}

	if err := s.db.UpdateDeploymentHeartbeat(l.ID, req.MachineFingerprint, ip, req.ServerVersion); err != nil {
		return &models.LicenseResponse{Valid: false, Status: "error", Message: "Failed to update heartbeat"}
	}

	return validation
}

// Deactivate removes a deployment slot
func (s *LicenseService) Deactivate(req *models.LicenseDeactivateRequest, ip string) *models.LicenseResponse {
	l, err := s.db.GetLicenseByKey(req.LicenseKey)
	if err != nil {
		return &models.LicenseResponse{Valid: false, Status: "invalid", Message: "License key not found"}
	}

	s.db.DeactivateDeployment(l.ID, req.MachineFingerprint)
	s.db.LogLicenseEvent(l.ID, models.EventDeactivated, req.MachineFingerprint, ip, "clean shutdown")

	return &models.LicenseResponse{Valid: true, Status: "deactivated", Message: "Deployment deactivated"}
}

// CleanupStaleDeployments marks inactive any deployment with no heartbeat in 24h
func (s *LicenseService) CleanupStaleDeployments() {
	affected, err := s.db.MarkStaleDeployments(24 * time.Hour)
	if err != nil {
		log.Printf("[License] Stale cleanup error: %v", err)
		return
	}
	if affected > 0 {
		log.Printf("[License] Marked %d stale deployments as inactive", affected)
	}
}

// StartStaleCleanupLoop runs the stale deployment cleanup every hour
func (s *LicenseService) StartStaleCleanupLoop(stop chan struct{}) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.CleanupStaleDeployments()
		case <-stop:
			return
		}
	}
}
