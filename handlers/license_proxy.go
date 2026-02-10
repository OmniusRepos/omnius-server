package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// LicenseProxyHandler proxies license admin requests to the central
// license authority (omnius.stream) when running in client mode.
type LicenseProxyHandler struct {
	serverURL   string // e.g. "https://omnius.stream"
	adminSecret string // x-admin-secret header value
	client      *http.Client
}

func NewLicenseProxyHandler(serverURL, adminSecret string) *LicenseProxyHandler {
	return &LicenseProxyHandler{
		serverURL:   serverURL,
		adminSecret: adminSecret,
		client:      &http.Client{},
	}
}

// proxyRequest makes a request to the license authority with the admin secret.
func (h *LicenseProxyHandler) proxyRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := h.serverURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-admin-secret", h.adminSecret)
	return h.client.Do(req)
}

// mapLicense transforms a Dart license JSON (from omnius.stream) into the Go frontend format.
func mapLicense(src map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":                 src["id"],
		"license_key":        src["license_key"],
		"plan":               src["plan"],
		"owner_email":        src["user_email"],
		"owner_name":         src["user_name"],
		"max_deployments":    src["max_deployments"],
		"is_active":          src["is_active"],
		"notes":              src["notes"],
		"created_at":         src["createdAt"],
		"expires_at":         src["expires_at"],
		"active_deployments": 0,
	}
}

// mapDeployment transforms a Dart deployment JSON into the Go frontend format.
func mapDeployment(src map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":                  src["id"],
		"license_id":          src["license_id"],
		"machine_fingerprint": src["machine_fingerprint"],
		"machine_label":       src["machine_label"],
		"ip_address":          src["ip_address"],
		"server_version":      src["server_version"],
		"first_seen":          src["createdAt"],
		"last_heartbeat":      src["last_seen_at"],
		"is_active":           src["is_active"],
	}
}

// ListLicenses proxies GET /admin/api/licenses
func (h *LicenseProxyHandler) ListLicenses(w http.ResponseWriter, r *http.Request) {
	resp, err := h.proxyRequest("GET", "/api/admin/licenses", nil)
	if err != nil {
		jsonError(w, "Failed to reach license authority", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, "Invalid response from license authority", http.StatusBadGateway)
		return
	}

	// omnius.stream returns {"licenses": [...]}; Go frontend expects [...]
	rawLicenses, ok := result["licenses"].([]interface{})
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	mapped := make([]interface{}, 0, len(rawLicenses))
	for _, item := range rawLicenses {
		if m, ok := item.(map[string]interface{}); ok {
			mapped = append(mapped, mapLicense(m))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mapped)
}

// CreateLicense proxies POST /admin/api/licenses
func (h *LicenseProxyHandler) CreateLicense(w http.ResponseWriter, r *http.Request) {
	var goReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&goReq); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Map Go frontend fields → omnius.stream fields
	dartReq := map[string]interface{}{
		"plan":            goReq["plan"],
		"max_deployments": goReq["max_deployments"],
		"user_email":      goReq["owner_email"],
	}
	if notes, ok := goReq["notes"]; ok {
		dartReq["notes"] = notes
	}
	if expiresAt, ok := goReq["expires_at"]; ok {
		dartReq["expires_at"] = expiresAt
	}

	body, _ := json.Marshal(dartReq)
	resp, err := h.proxyRequest("POST", "/api/admin/licenses", bytes.NewReader(body))
	if err != nil {
		jsonError(w, "Failed to reach license authority", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, "Invalid response from license authority", http.StatusBadGateway)
		return
	}

	if resp.StatusCode >= 400 {
		msg := "License creation failed"
		if e, ok := result["error"].(string); ok {
			msg = e
		}
		jsonError(w, msg, resp.StatusCode)
		return
	}

	// omnius.stream returns {"license": {...}}; Go frontend expects {...}
	if licData, ok := result["license"].(map[string]interface{}); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(mapLicense(licData))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	json.NewEncoder(w).Encode(result)
}

// GetLicense proxies GET /admin/api/licenses/{id}
func (h *LicenseProxyHandler) GetLicense(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// omnius.stream doesn't have a single-license endpoint, so fetch all and filter
	resp, err := h.proxyRequest("GET", "/api/admin/licenses", nil)
	if err != nil {
		jsonError(w, "Failed to reach license authority", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, "Invalid response from license authority", http.StatusBadGateway)
		return
	}

	rawLicenses, _ := result["licenses"].([]interface{})
	idNum, _ := strconv.ParseFloat(id, 64)

	for _, item := range rawLicenses {
		if m, ok := item.(map[string]interface{}); ok {
			if licID, ok := m["id"].(float64); ok && licID == idNum {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(mapLicense(m))
				return
			}
		}
	}

	jsonError(w, "License not found", http.StatusNotFound)
}

// UpdateLicense proxies PUT /admin/api/licenses/{id}
func (h *LicenseProxyHandler) UpdateLicense(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var goReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&goReq); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Map fields
	dartReq := make(map[string]interface{})
	if v, ok := goReq["plan"]; ok {
		dartReq["plan"] = v
	}
	if v, ok := goReq["max_deployments"]; ok {
		dartReq["max_deployments"] = v
	}
	if v, ok := goReq["is_active"]; ok {
		dartReq["is_active"] = v
	}
	if v, ok := goReq["notes"]; ok {
		dartReq["notes"] = v
	}
	if v, ok := goReq["expires_at"]; ok {
		dartReq["expires_at"] = v
	}

	body, _ := json.Marshal(dartReq)
	path := fmt.Sprintf("/api/admin/licenses?id=%s", id)
	resp, err := h.proxyRequest("PUT", path, bytes.NewReader(body))
	if err != nil {
		jsonError(w, "Failed to reach license authority", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, "Invalid response from license authority", http.StatusBadGateway)
		return
	}

	if resp.StatusCode >= 400 {
		msg := "License update failed"
		if e, ok := result["error"].(string); ok {
			msg = e
		}
		jsonError(w, msg, resp.StatusCode)
		return
	}

	if licData, ok := result["license"].(map[string]interface{}); ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mapLicense(licData))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DeleteLicense — not directly supported by omnius.stream, revoke instead
func (h *LicenseProxyHandler) DeleteLicense(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Revoke by setting is_active=false
	body, _ := json.Marshal(map[string]interface{}{"is_active": false})
	path := fmt.Sprintf("/api/admin/licenses?id=%s", id)
	resp, err := h.proxyRequest("PUT", path, bytes.NewReader(body))
	if err != nil {
		jsonError(w, "Failed to reach license authority", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetDeployments proxies GET /admin/api/licenses/{id}/deployments
func (h *LicenseProxyHandler) GetDeployments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	path := fmt.Sprintf("/api/admin/deployments?license_id=%s", id)
	resp, err := h.proxyRequest("GET", path, nil)
	if err != nil {
		jsonError(w, "Failed to reach license authority", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, "Invalid response from license authority", http.StatusBadGateway)
		return
	}

	// omnius.stream returns {"deployments": [...]}; Go frontend expects [...]
	rawDeps, ok := result["deployments"].([]interface{})
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	mapped := make([]interface{}, 0, len(rawDeps))
	for _, item := range rawDeps {
		if m, ok := item.(map[string]interface{}); ok {
			mapped = append(mapped, mapDeployment(m))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mapped)
}

// DeactivateDeployment proxies DELETE /admin/api/licenses/{id}/deployments/{did}
func (h *LicenseProxyHandler) DeactivateDeployment(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")

	path := fmt.Sprintf("/api/admin/deployments?id=%s", did)
	resp, err := h.proxyRequest("DELETE", path, nil)
	if err != nil {
		jsonError(w, "Failed to reach license authority", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
