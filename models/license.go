package models

import "time"

// License plans
const (
	PlanPersonal   = "personal"
	PlanBusiness   = "business"
	PlanEnterprise = "enterprise"
)

// License features
const (
	FeatureLiveChannels = "live_channels"
)

// PlanFeatures returns the default features for each plan
func PlanFeatures(plan string) []string {
	switch plan {
	case PlanEnterprise:
		return []string{FeatureLiveChannels}
	default:
		return []string{}
	}
}

// License represents a license key and its metadata
type License struct {
	ID             int64      `json:"id"`
	LicenseKey     string     `json:"license_key"`
	Plan           string     `json:"plan"`
	OwnerEmail     string     `json:"owner_email"`
	OwnerName      string     `json:"owner_name"`
	MaxDeployments int        `json:"max_deployments"`
	IsActive       bool       `json:"is_active"`
	Notes          string     `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	Features string `json:"features,omitempty"` // comma-separated feature flags
	// Computed fields
	ActiveDeployments int `json:"active_deployments,omitempty"`
}

// LicenseDeployment represents an activated deployment
type LicenseDeployment struct {
	ID                 int64     `json:"id"`
	LicenseID          int64     `json:"license_id"`
	MachineFingerprint string    `json:"machine_fingerprint"`
	MachineLabel       string    `json:"machine_label"`
	IPAddress          string    `json:"ip_address"`
	ServerVersion      string    `json:"server_version"`
	FirstSeen          time.Time `json:"first_seen"`
	LastHeartbeat      time.Time `json:"last_heartbeat"`
	IsActive           bool      `json:"is_active"`
}

// LicenseEvent represents an audit log entry
type LicenseEvent struct {
	ID                 int64     `json:"id"`
	LicenseID          int64     `json:"license_id"`
	EventType          string    `json:"event_type"`
	MachineFingerprint string    `json:"machine_fingerprint,omitempty"`
	IPAddress          string    `json:"ip_address,omitempty"`
	Details            string    `json:"details,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// Event types
const (
	EventActivated   = "activated"
	EventDeactivated = "deactivated"
	EventHeartbeat   = "heartbeat"
	EventValidated   = "validated"
	EventRevoked     = "revoked"
	EventExpired     = "expired"
	EventCreated     = "created"
)

// --- Request/Response types ---

// LicenseValidateRequest is sent by customer binary to validate a key
type LicenseValidateRequest struct {
	LicenseKey         string `json:"license_key"`
	MachineFingerprint string `json:"machine_fingerprint"`
}

// LicenseActivateRequest is sent by customer binary to register a deployment
type LicenseActivateRequest struct {
	LicenseKey         string `json:"license_key"`
	MachineFingerprint string `json:"machine_fingerprint"`
	MachineLabel       string `json:"machine_label"`
	ServerVersion      string `json:"server_version"`
	Domain             string `json:"domain,omitempty"`
}

// LicenseHeartbeatRequest is sent periodically by customer binary
type LicenseHeartbeatRequest struct {
	LicenseKey         string `json:"license_key"`
	MachineFingerprint string `json:"machine_fingerprint"`
	ServerVersion      string `json:"server_version"`
	Domain             string `json:"domain,omitempty"`
}

// LicenseDeactivateRequest is sent on clean shutdown
type LicenseDeactivateRequest struct {
	LicenseKey         string `json:"license_key"`
	MachineFingerprint string `json:"machine_fingerprint"`
}

// LicenseResponse is returned by the authority to client binaries
type LicenseResponse struct {
	Valid          bool     `json:"valid"`
	Plan           string   `json:"plan,omitempty"`
	Status         string   `json:"status"` // "active", "expired", "revoked", "over_limit", "invalid"
	Message        string   `json:"message,omitempty"`
	MaxDeployments int      `json:"max_deployments,omitempty"`
	GraceDays      int      `json:"grace_days,omitempty"`
	ExpiresAt      string   `json:"expires_at,omitempty"`
	Features       []string `json:"features,omitempty"`
}

// AdminCreateLicenseRequest is used by the admin to create a new license
type AdminCreateLicenseRequest struct {
	Plan           string `json:"plan"`
	OwnerEmail     string `json:"owner_email"`
	OwnerName      string `json:"owner_name"`
	MaxDeployments int    `json:"max_deployments"`
	Notes          string `json:"notes,omitempty"`
	ExpiresAt      string `json:"expires_at,omitempty"` // RFC3339
}

// AdminUpdateLicenseRequest is used by the admin to update a license
type AdminUpdateLicenseRequest struct {
	Plan           *string `json:"plan,omitempty"`
	OwnerEmail     *string `json:"owner_email,omitempty"`
	OwnerName      *string `json:"owner_name,omitempty"`
	MaxDeployments *int    `json:"max_deployments,omitempty"`
	IsActive       *bool   `json:"is_active,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
}

// LicenseCacheData is persisted locally by the client
type LicenseCacheData struct {
	LicenseKey    string    `json:"license_key"`
	Plan          string    `json:"plan"`
	Valid         bool      `json:"valid"`
	Status        string    `json:"status"`
	GraceDays     int       `json:"grace_days"`
	ValidatedAt   time.Time `json:"validated_at"`
	ExpiresAt     string    `json:"expires_at,omitempty"`
	Fingerprint   string    `json:"fingerprint"`
	ServerVersion string    `json:"server_version"`
	Features      []string  `json:"features,omitempty"`
}
