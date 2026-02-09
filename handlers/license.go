package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"torrent-server/database"
	"torrent-server/models"
	"torrent-server/services"
)

// LicenseHandler handles license-related HTTP endpoints
type LicenseHandler struct {
	db      *database.DB
	service *services.LicenseService
}

func NewLicenseHandler(db *database.DB, service *services.LicenseService) *LicenseHandler {
	return &LicenseHandler{db: db, service: service}
}

// --- Public API endpoints (called by customer binaries) ---

// Validate handles POST /api/v2/license/validate
func (h *LicenseHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var req models.LicenseValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.LicenseKey == "" {
		jsonError(w, "license_key is required", http.StatusBadRequest)
		return
	}

	resp := h.service.ValidateKey(req.LicenseKey, req.MachineFingerprint)
	w.Header().Set("Content-Type", "application/json")
	if !resp.Valid {
		w.WriteHeader(http.StatusForbidden)
	}
	json.NewEncoder(w).Encode(resp)
}

// Activate handles POST /api/v2/license/activate
func (h *LicenseHandler) Activate(w http.ResponseWriter, r *http.Request) {
	var req models.LicenseActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.LicenseKey == "" || req.MachineFingerprint == "" {
		jsonError(w, "license_key and machine_fingerprint are required", http.StatusBadRequest)
		return
	}

	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = fwd
	}

	resp := h.service.Activate(&req, ip)
	w.Header().Set("Content-Type", "application/json")
	if !resp.Valid {
		w.WriteHeader(http.StatusForbidden)
	}
	json.NewEncoder(w).Encode(resp)
}

// Heartbeat handles POST /api/v2/license/heartbeat
func (h *LicenseHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req models.LicenseHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.LicenseKey == "" || req.MachineFingerprint == "" {
		jsonError(w, "license_key and machine_fingerprint are required", http.StatusBadRequest)
		return
	}

	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = fwd
	}

	resp := h.service.Heartbeat(&req, ip)
	w.Header().Set("Content-Type", "application/json")
	if !resp.Valid {
		w.WriteHeader(http.StatusForbidden)
	}
	json.NewEncoder(w).Encode(resp)
}

// Deactivate handles POST /api/v2/license/deactivate
func (h *LicenseHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	var req models.LicenseDeactivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = fwd
	}

	resp := h.service.Deactivate(&req, ip)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- Admin API endpoints (license management, authority mode) ---

// AdminListLicenses handles GET /admin/api/licenses
func (h *LicenseHandler) AdminListLicenses(w http.ResponseWriter, r *http.Request) {
	licenses, err := h.db.ListLicenses()
	if err != nil {
		jsonError(w, "Failed to list licenses", http.StatusInternalServerError)
		return
	}
	if licenses == nil {
		licenses = []models.License{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(licenses)
}

// AdminCreateLicense handles POST /admin/api/licenses
func (h *LicenseHandler) AdminCreateLicense(w http.ResponseWriter, r *http.Request) {
	var req models.AdminCreateLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Plan == "" {
		req.Plan = models.PlanPersonal
	}

	license, err := h.service.CreateLicense(&req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(license)
}

// AdminGetLicense handles GET /admin/api/licenses/{id}
func (h *LicenseHandler) AdminGetLicense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid license ID", http.StatusBadRequest)
		return
	}

	license, err := h.db.GetLicenseByID(id)
	if err != nil {
		jsonError(w, "License not found", http.StatusNotFound)
		return
	}

	count, _ := h.db.CountActiveDeployments(license.ID)
	license.ActiveDeployments = count

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(license)
}

// AdminUpdateLicense handles PUT /admin/api/licenses/{id}
func (h *LicenseHandler) AdminUpdateLicense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid license ID", http.StatusBadRequest)
		return
	}

	var req models.AdminUpdateLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.db.UpdateLicense(id, &req); err != nil {
		jsonError(w, "Failed to update license", http.StatusInternalServerError)
		return
	}

	license, _ := h.db.GetLicenseByID(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(license)
}

// AdminDeleteLicense handles DELETE /admin/api/licenses/{id}
func (h *LicenseHandler) AdminDeleteLicense(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid license ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteLicense(id); err != nil {
		jsonError(w, "Failed to delete license", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// AdminGetDeployments handles GET /admin/api/licenses/{id}/deployments
func (h *LicenseHandler) AdminGetDeployments(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid license ID", http.StatusBadRequest)
		return
	}

	deps, err := h.db.GetDeploymentsByLicense(id)
	if err != nil {
		jsonError(w, "Failed to get deployments", http.StatusInternalServerError)
		return
	}
	if deps == nil {
		deps = []models.LicenseDeployment{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deps)
}

// AdminDeactivateDeployment handles DELETE /admin/api/licenses/{id}/deployments/{did}
func (h *LicenseHandler) AdminDeactivateDeployment(w http.ResponseWriter, r *http.Request) {
	did, err := strconv.ParseInt(chi.URLParam(r, "did"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid deployment ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeactivateDeploymentByID(did); err != nil {
		jsonError(w, "Failed to deactivate deployment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// AdminGetLicenseStatus handles GET /admin/api/license-status
// Returns the current instance's license status (works in all modes)
func (h *LicenseHandler) AdminGetLicenseStatus(w http.ResponseWriter, r *http.Request) {
	// This is populated by the license client in main.go and passed via context or a global
	// For now, return a basic status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
	})
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
