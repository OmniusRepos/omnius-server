package handlers

import (
	"encoding/json"
	"net/http"

	"torrent-server/database"
	"torrent-server/models"
)

type ConfigHandler struct {
	db *database.DB
}

func NewConfigHandler(db *database.DB) *ConfigHandler {
	return &ConfigHandler{db: db}
}

// GetConfig handles GET /api/v2/config.json
// Public endpoint — client uses this to know which services are available
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	services, err := h.db.ListServices()
	if err != nil {
		services = []models.ServiceConfig{}
	}

	// Only return enabled services to the client
	var enabled []models.ServiceConfig
	for _, s := range services {
		if s.Enabled {
			enabled = append(enabled, s)
		}
	}
	if enabled == nil {
		enabled = []models.ServiceConfig{}
	}

	writeSuccess(w, map[string]interface{}{
		"services": enabled,
	})
}

// AdminListServices handles GET /admin/api/services
// Returns ALL services (enabled and disabled) for admin management
func (h *ConfigHandler) AdminListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.db.ListServices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if services == nil {
		services = []models.ServiceConfig{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

// AdminUpdateServices handles PUT /admin/api/services
// Accepts array of services to update
func (h *ConfigHandler) AdminUpdateServices(w http.ResponseWriter, r *http.Request) {
	var services []models.ServiceConfig
	if err := json.NewDecoder(r.Body).Decode(&services); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	for _, s := range services {
		if err := h.db.CreateService(&s); err != nil {
			http.Error(w, "failed to update service: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
