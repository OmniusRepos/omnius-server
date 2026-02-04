package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"torrent-server/database"
	"torrent-server/models"
	"torrent-server/services"
)

type StremioHandler struct {
	db             *database.DB
	torrentService *services.TorrentService
	baseURL        string
}

func NewStremioHandler(db *database.DB, ts *services.TorrentService, baseURL string) *StremioHandler {
	return &StremioHandler{
		db:             db,
		torrentService: ts,
		baseURL:        baseURL,
	}
}

// Manifest handles GET /manifest.json
func (h *StremioHandler) Manifest(w http.ResponseWriter, r *http.Request) {
	manifest := models.StremioManifest{
		ID:          "com.torrent-server",
		Version:     "1.0.0",
		Name:        "Torrent Server",
		Description: "Self-hosted torrent streaming server",
		Resources:   []string{"catalog", "stream"},
		Types:       []string{"movie", "series"},
		Catalogs: []models.StremioCatalog{
			{Type: "movie", ID: "torrent-movies", Name: "Torrent Movies"},
			{Type: "series", ID: "torrent-series", Name: "Torrent Series"},
		},
		IDPrefixes: []string{"tt"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(manifest)
}

// Catalog handles GET /catalog/{type}/{id}.json
func (h *StremioHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	contentType := chi.URLParam(r, "type")
	catalogID := chi.URLParam(r, "id")
	catalogID = strings.TrimSuffix(catalogID, ".json")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var metas []models.StremioMeta

	switch contentType {
	case "movie":
		movies, _, _ := h.db.ListMovies(database.MovieFilter{Limit: 100, Page: 1})
		for _, m := range movies {
			metas = append(metas, models.StremioMeta{
				ID:     m.ImdbCode,
				Type:   "movie",
				Name:   m.Title,
				Poster: m.MediumCoverImage,
				Year:   m.Year,
			})
		}
	case "series":
		seriesList, _, _ := h.db.ListSeries(100, 1)
		for _, s := range seriesList {
			metas = append(metas, models.StremioMeta{
				ID:     s.ImdbCode,
				Type:   "series",
				Name:   s.Title,
				Poster: s.PosterImage,
				Year:   s.Year,
			})
		}
	}

	if metas == nil {
		metas = []models.StremioMeta{}
	}

	response := models.StremioCatalogResponse{Metas: metas}
	json.NewEncoder(w).Encode(response)
}

// Stream handles GET /stream/{type}/{id}.json
func (h *StremioHandler) Stream(w http.ResponseWriter, r *http.Request) {
	contentType := chi.URLParam(r, "type")
	id := chi.URLParam(r, "id")
	id = strings.TrimSuffix(id, ".json")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var streams []models.StremioStream

	switch contentType {
	case "movie":
		movie, err := h.db.GetMovieByIMDB(id)
		if err == nil && movie != nil {
			for _, t := range movie.Torrents {
				stream := models.StremioStream{
					InfoHash: t.Hash,
					Title:    fmt.Sprintf("%s %s", t.Quality, t.Type),
					Name:     fmt.Sprintf("Torrent Server\n%s", t.Size),
				}

				// Add direct URL if base URL is configured
				if h.baseURL != "" {
					stream.URL = fmt.Sprintf("%s/stream/%s/0", h.baseURL, t.Hash)
				}

				streams = append(streams, stream)
			}
		}

	case "series":
		// Parse series ID format: tt1234567:1:5 (imdb:season:episode)
		parts := strings.Split(id, ":")
		if len(parts) >= 3 {
			imdbCode := parts[0]
			season := parseInt(parts[1], 0)
			episode := parseInt(parts[2], 0)

			series, err := h.db.GetSeriesByIMDB(imdbCode)
			if err == nil && series != nil {
				episodes, _ := h.db.GetEpisodes(series.ID, season)
				for _, ep := range episodes {
					if ep.Episode == uint(episode) {
						for _, t := range ep.Torrents {
							streams = append(streams, models.StremioStream{
								InfoHash: t.Hash,
								Title:    fmt.Sprintf("%s - S%02dE%02d %s", series.Title, season, episode, t.Quality),
								Name:     fmt.Sprintf("Torrent Server\n%s", t.Size),
							})
						}
					}
				}

				// Also include season packs
				packs, _ := h.db.GetSeasonPacks(series.ID)
				for _, p := range packs {
					if p.Season == uint(season) {
						streams = append(streams, models.StremioStream{
							InfoHash: p.Hash,
							Title:    fmt.Sprintf("Season %d Pack - %s", p.Season, p.Quality),
							Name:     fmt.Sprintf("Torrent Server\n%s (Full Season)", p.Size),
						})
					}
				}
			}
		}
	}

	if streams == nil {
		streams = []models.StremioStream{}
	}

	response := models.StremioStreamResponse{Streams: streams}
	json.NewEncoder(w).Encode(response)
}
