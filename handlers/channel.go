package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"torrent-server/database"
	"torrent-server/models"
	"torrent-server/services"
)

type ChannelHandler struct {
	db            *database.DB
	iptvService   *services.IPTVSyncService
	healthService *services.ChannelHealthService
}

func NewChannelHandler(db *database.DB) *ChannelHandler {
	return &ChannelHandler{
		db:            db,
		iptvService:   services.NewIPTVSyncService(db),
		healthService: services.NewChannelHealthService(db),
	}
}

// ListChannels handles GET /api/v2/list_channels.json
func (h *ChannelHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := database.ChannelFilter{
		Limit:     parseInt(q.Get("limit"), 50),
		Page:      parseInt(q.Get("page"), 1),
		Country:   q.Get("country"),
		Category:  q.Get("category"),
		QueryTerm: q.Get("query_term"),
	}

	channels, totalCount, err := h.db.ListChannels(filter)
	if err != nil {
		writeError(w, "Failed to fetch channels: "+err.Error())
		return
	}

	if channels == nil {
		channels = []models.Channel{}
	}

	data := map[string]interface{}{
		"channel_count": totalCount,
		"limit":         filter.Limit,
		"page_number":   filter.Page,
		"channels":      channels,
	}

	writeSuccess(w, data)
}

// GetChannel handles GET /api/v2/channel_details.json
func (h *ChannelHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		writeError(w, "channel_id is required")
		return
	}

	channel, err := h.db.GetChannel(channelID)
	if err != nil {
		writeError(w, "Channel not found")
		return
	}

	writeSuccess(w, map[string]interface{}{
		"channel": channel,
	})
}

// ListCountries handles GET /api/v2/channel_countries.json
func (h *ChannelHandler) ListCountries(w http.ResponseWriter, r *http.Request) {
	countries, err := h.db.ListChannelCountries()
	if err != nil {
		writeError(w, "Failed to fetch countries: "+err.Error())
		return
	}

	if countries == nil {
		countries = []models.ChannelCountry{}
	}

	writeSuccess(w, map[string]interface{}{
		"countries": countries,
	})
}

// ListCategories handles GET /api/v2/channel_categories.json
func (h *ChannelHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.db.GetChannelCountByCategory()
	if err != nil {
		writeError(w, "Failed to fetch categories: "+err.Error())
		return
	}

	if categories == nil {
		categories = []models.ChannelCategory{}
	}

	writeSuccess(w, map[string]interface{}{
		"categories": categories,
	})
}

// GetChannelsByCountry handles GET /api/v2/channels_by_country.json
func (h *ChannelHandler) GetChannelsByCountry(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	country := q.Get("country")
	if country == "" {
		writeError(w, "country is required")
		return
	}

	limit := parseInt(q.Get("limit"), 50)

	channels, err := h.db.GetChannelsByCountry(country, limit)
	if err != nil {
		writeError(w, "Failed to fetch channels: "+err.Error())
		return
	}

	if channels == nil {
		channels = []models.Channel{}
	}

	writeSuccess(w, map[string]interface{}{
		"channels": channels,
	})
}

// GetEPG handles GET /api/v2/channel_epg.json?channel_id=...
func (h *ChannelHandler) GetEPG(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		writeError(w, "channel_id is required")
		return
	}

	epg, err := h.db.GetEPG(channelID)
	if err != nil {
		writeError(w, "Failed to fetch EPG: "+err.Error())
		return
	}

	if epg == nil {
		epg = []models.ChannelEPG{}
	}

	writeSuccess(w, map[string]interface{}{
		"epg": epg,
	})
}

// --- Admin endpoints ---

// SyncIPTV handles POST /admin/api/channels/sync
// Accepts optional JSON body: { "m3u_url": "https://..." }
func (h *ChannelHandler) SyncIPTV(w http.ResponseWriter, r *http.Request) {
	var req struct {
		M3UURL string `json:"m3u_url"`
	}
	json.NewDecoder(r.Body).Decode(&req) // optional body, ignore errors

	if err := h.iptvService.SyncFromM3U(req.M3UURL); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "IPTV sync started from M3U",
		"m3u_url": services.IPTVM3UURL,
	})
}

// UpdateM3UURL handles PUT /admin/api/channels/settings
func (h *ChannelHandler) UpdateM3UURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		M3UURL string `json:"m3u_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.M3UURL == "" {
		http.Error(w, "m3u_url required", http.StatusBadRequest)
		return
	}

	services.IPTVM3UURL = req.M3UURL

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"m3u_url": services.IPTVM3UURL,
	})
}

// GetChannelSettings handles GET /admin/api/channels/settings
func (h *ChannelHandler) GetChannelSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"m3u_url": services.IPTVM3UURL,
	})
}

// SyncStatus handles GET /admin/api/channels/sync/status
func (h *ChannelHandler) SyncStatus(w http.ResponseWriter, r *http.Request) {
	status := h.iptvService.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ChannelStats handles GET /admin/api/channels/stats
func (h *ChannelHandler) ChannelStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetChannelStats()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// StartHealthCheck handles POST /admin/api/channels/health-check
func (h *ChannelHandler) StartHealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.healthService.RunHealthCheck(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Health check started",
	})
}

// GetHealthCheckStatus handles GET /admin/api/channels/health-check/status
func (h *ChannelHandler) GetHealthCheckStatus(w http.ResponseWriter, r *http.Request) {
	status := h.healthService.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ClearBlocklist handles DELETE /admin/api/channels/blocklist
func (h *ChannelHandler) ClearBlocklist(w http.ResponseWriter, r *http.Request) {
	if err := h.db.ClearBlocklist(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Blocklist cleared",
	})
}

// DeleteChannel handles DELETE /admin/api/channels/{id}
func (h *ChannelHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteChannel(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
