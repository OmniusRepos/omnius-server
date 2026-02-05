package handlers

import (
	"net/http"

	"torrent-server/database"
	"torrent-server/models"
)

type ChannelHandler struct {
	db *database.DB
}

func NewChannelHandler(db *database.DB) *ChannelHandler {
	return &ChannelHandler{db: db}
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
	categories, err := h.db.ListChannelCategories()
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
