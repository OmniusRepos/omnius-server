package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"torrent-server/config"
	"torrent-server/database"
	"torrent-server/handlers"
	authMiddleware "torrent-server/middleware"
	"torrent-server/services"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Version is set at build time via -ldflags "-X main.Version=..."
var Version = "dev"

const imdbAPIBaseURL = "https://api.imdbapi.dev"
// YTS API URL - auto-detected from working mirrors
var ytsAPIBaseURL string
var ytsMirrors = []string{
	"https://yts.bz/api/v2",
	"https://yts.gg/api/v2",
	"https://yts.lt/api/v2",
	"https://yts.am/api/v2",
	"https://yts.ag/api/v2",
	"https://yts.mx/api/v2",
	"https://yts.torrentbay.st/api/v2",
}

func init() {
	// Allow override via env var
	if url := os.Getenv("YTS_API_URL"); url != "" {
		ytsAPIBaseURL = url
		log.Printf("Using YTS API from env: %s", ytsAPIBaseURL)
		return
	}

	// Auto-detect working mirror
	go detectWorkingYTSMirror()
	// Set default while detecting
	ytsAPIBaseURL = ytsMirrors[0]
}

func detectWorkingYTSMirror() {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, mirror := range ytsMirrors {
		testURL := mirror + "/list_movies.json?limit=1"
		resp, err := client.Get(testURL)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			ytsAPIBaseURL = mirror
			log.Printf("YTS mirror detected: %s", mirror)
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	log.Printf("Warning: No working YTS mirror found, using default: %s", ytsAPIBaseURL)
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize torrent service
	torrentService, err := services.NewTorrentService(cfg.DownloadDir)
	if err != nil {
		log.Fatalf("Failed to initialize torrent service: %v", err)
	}
	defer torrentService.Close()

	// Initialize OMDB service
	omdbService := services.NewOMDBService(cfg.OmdbAPIKey)

	// --- License System ---
	// Always runs as client — omnius.stream is the sole license authority
	fingerprint, err := services.GetMachineFingerprint()
	if err != nil {
		log.Printf("[License] Warning: failed to get machine fingerprint: %v", err)
		fingerprint = "unknown"
	}
	// Use env var, fallback to persisted key file
	licenseKey := cfg.LicenseKey
	if licenseKey == "" {
		if saved, err := os.ReadFile("data/.license-key"); err == nil {
			licenseKey = strings.TrimSpace(string(saved))
		}
	}
	licenseClient := services.NewLicenseClient(licenseKey, cfg.LicenseServerURL, fingerprint, "1.0.0", cfg.ServerDomain)
	if err := licenseClient.Start(); err != nil {
		log.Fatalf("[License] %v", err)
	}
	defer licenseClient.Stop()
	log.Printf("[License] Status: %s", licenseClient.GetStatus().Message)

	// Create subtitles directory
	subtitlesDir := "data/subtitles"
	if err := os.MkdirAll(subtitlesDir, 0755); err != nil {
		log.Fatalf("Failed to create subtitles directory: %v", err)
	}

	// Migrate existing subtitle VTT content from DB to disk files
	migrateSubtitlesToDisk(db, subtitlesDir)

	// Parse templates
	templates, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Printf("Warning: Failed to parse templates: %v", err)
		templates = nil
	}

	// Initialize services
	subtitleService := services.NewSubtitleServiceWithDB(db, subtitlesDir)
	imdbService := services.NewIMDBService()

	// Initialize handlers
	apiHandler := handlers.NewAPIHandler(db)
	streamHandler := handlers.NewStreamHandler(torrentService, subtitleService, db)
	adminHandler := handlers.NewAdminHandler(db, torrentService)
	adminHandler.SetTemplates(templates)
	subtitleHandler := handlers.NewSubtitleHandler(subtitleService, db)
	imdbHandler := handlers.NewIMDBHandler(imdbService)
	configHandler := handlers.NewConfigHandler(db)

	// Initialize auth middleware
	auth := authMiddleware.NewAuthMiddleware(cfg.AdminPassword)

	// Start session cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			auth.CleanupExpiredSessions()
		}
	}()

	// Create router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(corsMiddleware)

	// License enforcement middleware (client mode only)
	if licenseClient != nil {
		licenseMw := authMiddleware.NewLicenseMiddleware(licenseClient)
		r.Use(licenseMw.EnforceValid)
		r.Use(licenseMw.EnforceLiveChannels)
		demoLimiter := authMiddleware.NewDemoLimiter(licenseClient)
		r.Use(demoLimiter.InjectDemoFlag)
	}

	// Health check
	r.Get("/health", streamHandler.Health)

	// Initialize sync service (for syncing movies from external sources)
	syncService := services.NewSyncService(db, subtitlesDir)

	// Initialize ratings handler
	ratingsHandler := handlers.NewRatingsHandler(db, syncService)

	// Initialize series handler
	seriesHandler := handlers.NewSeriesHandler(db)

	// Initialize curated handler
	curatedHandler := handlers.NewCuratedHandler(db)

	// Initialize home handler
	homeHandler := handlers.NewHomeHandler(db)

	// Initialize channel handler
	channelHandler := handlers.NewChannelHandler(db)

	// Initialize analytics handler
	analyticsHandler := handlers.NewAnalyticsHandler(db, torrentService)

	// YTS-compatible API (public)
	r.Route("/api/v2", func(r chi.Router) {
		// Server config (client reads this to build sidebar)
		r.Get("/config.json", configHandler.GetConfig)

		// Home
		r.Get("/home.json", apiHandler.Home)

		// Search (unified)
		r.Get("/search.json", apiHandler.UnifiedSearch)

		// Movies
		r.Get("/list_movies.json", apiHandler.ListMovies)
		r.Get("/movie_details.json", apiHandler.MovieDetails)
		r.Get("/movie_suggestions.json", apiHandler.MovieSuggestions)
		r.Get("/franchise_movies.json", apiHandler.FranchiseMovies)
		r.Get("/check_availability", apiHandler.CheckAvailability)

		// Series
		r.Get("/list_series.json", seriesHandler.ListSeries)
		r.Get("/series_details.json", seriesHandler.SeriesDetails)
		r.Get("/season_episodes.json", seriesHandler.SeasonEpisodes)

		// Channels (IPTV)
		r.Get("/list_channels.json", channelHandler.ListChannels)
		r.Get("/channel_details.json", channelHandler.GetChannel)
		r.Get("/channel_countries.json", channelHandler.ListCountries)
		r.Get("/channel_categories.json", channelHandler.ListCategories)
		r.Get("/channels_by_country.json", channelHandler.GetChannelsByCountry)
		r.Get("/channel_epg.json", channelHandler.GetEPG)

		// Ratings & Sync
		r.Post("/get_ratings", ratingsHandler.GetRatings)
		r.Post("/sync_movie", ratingsHandler.SyncMovie)
		r.Post("/sync_movies", ratingsHandler.SyncMovies)
		r.Post("/sync_series", ratingsHandler.SyncSeries)
		r.Post("/refresh_movie", ratingsHandler.RefreshMovie)
		r.Post("/refresh_series", ratingsHandler.RefreshSeries)

		// Curated Lists
		r.Get("/curated_lists.json", curatedHandler.ListCuratedLists)
		r.Get("/curated_list.json", curatedHandler.GetCuratedList)

		// Analytics (public - unified event endpoint + read endpoints)
		r.Post("/analytics", analyticsHandler.HandleEvent)
		r.Get("/analytics/top-movies", analyticsHandler.GetTopMoviesAPI)

		// Subtitles
		r.Get("/subtitles/search", subtitleHandler.Search)
		r.Get("/subtitles/search_by_filename", subtitleHandler.SearchByFilename)
		r.Get("/subtitles/download", subtitleHandler.Download)
		r.Get("/subtitles/stored/{id}", subtitleHandler.ServeStored)
		r.Get("/subtitle_languages", subtitleHandler.Languages)

		// Stream management
		r.Post("/stream/start", streamHandler.StartStream)
		r.Get("/stream/status", streamHandler.StreamStatus)
		r.Post("/stream/stop", streamHandler.StopStream)
		r.Get("/torrent_files", streamHandler.ListFiles)

		// IMDB proxy (public)
		r.Get("/imdb/images/{imdbCode}", imdbHandler.Images)
		r.Get("/imdb/search", imdbHandler.Search)
		r.Get("/imdb/title/{imdbCode}", imdbHandler.Title)
	})

	// Video streaming (public)
	r.Get("/stream/{infoHash}/{fileIndex}", streamHandler.Stream)
	r.Get("/stats", streamHandler.Stats)

	// Admin routes
	r.Route("/admin", func(r chi.Router) {
		// Public auth endpoints
		r.Get("/login", auth.Login)
		r.Post("/login", auth.Login)
		r.Get("/logout", auth.Logout)

		// Auth API endpoints for SPA
		r.Get("/api/auth/check", auth.CheckAuth)
		r.Post("/api/login", auth.LoginAPI)

		// YTS API proxy (to search for torrents)
		r.Get("/api/yts/search", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("query")
			imdb := r.URL.Query().Get("imdb")

			var apiURL string
			if imdb != "" {
				apiURL = fmt.Sprintf("%s/list_movies.json?query_term=%s&limit=1", ytsAPIBaseURL, url.QueryEscape(imdb))
			} else if query != "" {
				apiURL = fmt.Sprintf("%s/list_movies.json?query_term=%s&limit=10", ytsAPIBaseURL, url.QueryEscape(query))
			} else {
				http.Error(w, "query or imdb required", http.StatusBadRequest)
				return
			}

			// Use client with timeout
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get(apiURL)
			if err != nil {
				// Return empty result instead of error
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":         "error",
					"status_message": "YTS API unavailable: " + err.Error(),
					"data":           map[string]interface{}{"movies": []interface{}{}},
				})
				return
			}
			defer resp.Body.Close()
			w.Header().Set("Content-Type", "application/json")
			io.Copy(w, resp.Body)
		})

		// IMDB API proxy (to avoid CORS issues)
		r.Get("/api/imdb/search", func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("query")
			if query == "" {
				http.Error(w, "query required", http.StatusBadRequest)
				return
			}
			// URL-encode the query parameter
			apiURL := fmt.Sprintf("%s/search/titles?query=%s", imdbAPIBaseURL, url.QueryEscape(query))
			resp, err := http.Get(apiURL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			w.Header().Set("Content-Type", "application/json")
			io.Copy(w, resp.Body)
		})

		r.Get("/api/imdb/title/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			resp, err := http.Get(imdbAPIBaseURL + "/titles/" + id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			w.Header().Set("Content-Type", "application/json")
			io.Copy(w, resp.Body)
		})

		// Protected admin API routes
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth)

			// Legacy admin routes (form-based)
			r.Post("/movies", adminHandler.AddMovie)
			r.Post("/movies/{id}/torrent", adminHandler.AddTorrent)
			r.Delete("/movies/{id}", adminHandler.DeleteMovie)
			r.Post("/upload", adminHandler.UploadTorrent)

			// Curated lists admin API
			r.Get("/api/curated", curatedHandler.AdminListCuratedLists)
			r.Post("/api/curated", curatedHandler.AdminCreateCuratedList)
			r.Put("/api/curated/{id}", curatedHandler.AdminUpdateCuratedList)
			r.Delete("/api/curated/{id}", curatedHandler.AdminDeleteCuratedList)
			r.Post("/api/curated/{id}/movies", curatedHandler.AdminAddMovieToList)
			r.Delete("/api/curated/{id}/movies/{movieId}", curatedHandler.AdminRemoveMovieFromList)

			// Home sections admin API
			r.Get("/api/home/sections", homeHandler.AdminListHomeSections)
			r.Post("/api/home/sections", homeHandler.AdminCreateHomeSection)
			r.Put("/api/home/sections/{id}", homeHandler.AdminUpdateHomeSection)
			r.Delete("/api/home/sections/{id}", homeHandler.AdminDeleteHomeSection)
			r.Post("/api/home/sections/reorder", homeHandler.AdminReorderHomeSections)

			// Analytics admin API
			r.Get("/api/analytics", analyticsHandler.GetAnalytics)

			// Subtitles admin API
			r.Post("/api/subtitles/sync", subtitleHandler.SyncSubtitles)
			r.Get("/api/subtitles", subtitleHandler.ListStored)
			r.Delete("/api/subtitles/{id}", subtitleHandler.DeleteStored)
			r.Get("/api/subtitles/{id}/preview", subtitleHandler.PreviewStored)

			// Services config admin API
			r.Get("/api/services", configHandler.AdminListServices)
			r.Put("/api/services", configHandler.AdminUpdateServices)

			// Channels admin API (IPTV sync)
			r.Post("/api/channels/sync", channelHandler.SyncIPTV)
			r.Get("/api/channels/sync/status", channelHandler.SyncStatus)
			r.Get("/api/channels/stats", channelHandler.ChannelStats)
			r.Get("/api/channels/settings", channelHandler.GetChannelSettings)
			r.Put("/api/channels/settings", channelHandler.UpdateM3UURL)
			r.Post("/api/channels/health-check", channelHandler.StartHealthCheck)
			r.Get("/api/channels/health-check/status", channelHandler.GetHealthCheckStatus)
			r.Delete("/api/channels/blocklist", channelHandler.ClearBlocklist)
			r.Delete("/api/channels/{id}", channelHandler.DeleteChannel)

			// Sync/Refresh admin API
			r.Post("/api/refresh_all_movies", ratingsHandler.RefreshAllMovies)
			r.Post("/api/refresh_all_series", ratingsHandler.RefreshAllSeries)

			// Move movie to series (when IMDB type is actually a series)
			r.Post("/api/move-to-series", func(w http.ResponseWriter, rq *http.Request) {
				var req struct {
					ImdbCode string `json:"imdb_code"`
					MovieID  uint   `json:"movie_id"`
				}
				if err := json.NewDecoder(rq.Body).Decode(&req); err != nil || req.ImdbCode == "" || req.MovieID == 0 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "imdb_code and movie_id are required"})
					return
				}

				// Create series entry
				series, err := syncService.SyncSeries(req.ImdbCode)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create series: " + err.Error()})
					return
				}

				// Populate with IMDB data (best-effort — may fail for unusual types)
				refreshed, refreshErr := syncService.RefreshSeries(series)
				if refreshErr != nil {
					log.Printf("[MoveToSeries] RefreshSeries failed for %s: %v (series created with ID %d)", req.ImdbCode, refreshErr, series.ID)
				} else {
					series = refreshed
				}

				// Delete the movie
				if err := db.DeleteMovie(req.MovieID); err != nil {
					log.Printf("[MoveToSeries] Warning: failed to delete movie %d: %v", req.MovieID, err)
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":    "ok",
					"series_id": series.ID,
					"title":     series.Title,
				})
			})

			// Movies API
			r.Get("/api/movies", func(w http.ResponseWriter, r *http.Request) {
				movies, count, _ := db.ListMovies(database.MovieFilter{Limit: 50, Page: 1})
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"movies": movies,
					"count":  count,
				})
			})
			r.Get("/api/movies/by-imdb/{imdbCode}", func(w http.ResponseWriter, r *http.Request) {
				imdbCode := chi.URLParam(r, "imdbCode")
				movie, err := db.GetMovieByIMDB(imdbCode)
				w.Header().Set("Content-Type", "application/json")
				if err != nil || movie == nil {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"exists": false,
					})
					return
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"exists": true,
					"movie":  movie,
				})
			})
			r.Delete("/api/movies/{id}", adminHandler.DeleteMovie)
			r.Put("/api/movies/{id}", adminHandler.UpdateMovie)

			// Series API
			r.Post("/series", seriesHandler.AddSeries)
			r.Get("/api/series/by-imdb/{imdbCode}", func(w http.ResponseWriter, r *http.Request) {
				imdbCode := chi.URLParam(r, "imdbCode")
				series, err := db.GetSeriesByIMDB(imdbCode)
				w.Header().Set("Content-Type", "application/json")
				if err != nil || series == nil {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"exists": false,
					})
					return
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"exists": true,
					"series": series,
				})
			})
			r.Delete("/api/series/{id}", seriesHandler.DeleteSeries)
			r.Put("/api/series/{id}", seriesHandler.UpdateSeries)
			r.Post("/api/season-packs/{id}/expand", seriesHandler.ExpandSeasonPack)
			r.Post("/episodes/{id}/torrent", seriesHandler.AddEpisodeTorrent)

			// Stats API
			r.Get("/api/stats", func(w http.ResponseWriter, r *http.Request) {
				_, movieCount, _ := db.ListMovies(database.MovieFilter{Limit: 1, Page: 1})
				_, seriesCount, _ := db.ListSeries(1, 1)
				torrentStats := torrentService.GetStats()
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"movies":        movieCount,
					"series":        seriesCount,
					"torrents":      torrentStats,
					"activeStreams": torrentStats["active_torrents"],
				})
			})

			// License management — single license for this server
			r.Get("/api/license-status", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				hostname, _ := os.Hostname()
				status := licenseClient.GetStatus()
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":      status,
					"fingerprint": licenseClient.GetFingerprint(),
					"hostname":    hostname,
					"domain":      r.Host,
					"server_url":  cfg.LicenseServerURL,
				})
			})

			r.Post("/api/license-activate", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					LicenseKey string `json:"license_key"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.LicenseKey == "" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"error": "license_key is required"})
					return
				}

				key := strings.TrimSpace(body.LicenseKey)
				if err := licenseClient.Restart(key); err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnprocessableEntity)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":  err.Error(),
						"status": licenseClient.GetStatus(),
					})
					return
				}

				hostname, _ := os.Hostname()
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":      licenseClient.GetStatus(),
					"fingerprint": licenseClient.GetFingerprint(),
					"hostname":    hostname,
					"domain":      r.Host,
				})
			})

			// YTS Mirror settings
			r.Get("/api/settings/yts", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"current_mirror": ytsAPIBaseURL,
					"mirrors":        ytsMirrors,
				})
			})
			r.Put("/api/settings/yts", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Mirror string `json:"mirror"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid request body", http.StatusBadRequest)
					return
				}

				// Validate the mirror URL is in our list or allow custom
				valid := false
				for _, m := range ytsMirrors {
					if m == req.Mirror {
						valid = true
						break
					}
				}

				// Also allow any URL that ends with /api/v2
				if !valid && strings.HasSuffix(req.Mirror, "/api/v2") {
					valid = true
				}

				if !valid {
					http.Error(w, "Invalid mirror URL", http.StatusBadRequest)
					return
				}

				ytsAPIBaseURL = req.Mirror
				log.Printf("YTS mirror changed to: %s", ytsAPIBaseURL)

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "ok",
					"mirror":  ytsAPIBaseURL,
				})
			})
			r.Post("/api/settings/yts/test", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Mirror string `json:"mirror"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, "Invalid request body", http.StatusBadRequest)
					return
				}

				// Test the mirror
				client := &http.Client{Timeout: 10 * time.Second}
				testURL := req.Mirror + "/list_movies.json?limit=1"
				resp, err := client.Get(testURL)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"status":  "error",
						"message": err.Error(),
					})
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"status":  "error",
						"message": fmt.Sprintf("HTTP %d", resp.StatusCode),
					})
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "ok",
					"message": "Mirror is working",
				})
			})
		})

		// Check for updates via GitHub releases API
		r.Get("/api/check-update", func(w http.ResponseWriter, rq *http.Request) {
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get("https://api.github.com/repos/OmniusRepos/omnius-releases/releases/latest")
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to check for updates"})
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"update_available": false,
					"current_version":  Version,
					"message":          "Could not check for updates",
				})
				return
			}

			var release struct {
				TagName     string `json:"tag_name"`
				PublishedAt string `json:"published_at"`
				Body        string `json:"body"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"update_available": false,
					"current_version":  Version,
				})
				return
			}

			latestVersion := strings.TrimPrefix(release.TagName, "v")
			hasUpdate := Version == "dev" || latestVersion != Version

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"update_available": hasUpdate,
				"current_version":  Version,
				"latest_version":   latestVersion,
				"published_at":     release.PublishedAt,
				"release_notes":    release.Body,
			})
		})

		// Auto-update: download latest binary from GitHub releases and restart
			r.Post("/api/update", func(w http.ResponseWriter, r *http.Request) {
				execPath, err := os.Executable()
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Cannot determine executable path"})
					return
				}

				goos := runtime.GOOS
				goarch := runtime.GOARCH

				downloadURL := fmt.Sprintf("https://github.com/OmniusRepos/omnius-releases/releases/latest/download/omnius-%s-%s", goos, goarch)

				client := &http.Client{Timeout: 120 * time.Second}
				resp, err := client.Get(downloadURL)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Failed to download update: " + err.Error()})
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Download failed with status: %d", resp.StatusCode)})
					return
				}

				tmpFile, err := os.CreateTemp("", "omnius-update-*")
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create temp file"})
					return
				}
				tmpPath := tmpFile.Name()

				if _, err := io.Copy(tmpFile, resp.Body); err != nil {
					tmpFile.Close()
					os.Remove(tmpPath)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Failed to write update"})
					return
				}
				tmpFile.Close()

				if err := os.Chmod(tmpPath, 0755); err != nil {
					os.Remove(tmpPath)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"error": "Failed to set permissions"})
					return
				}

				// Replace binary
				if err := os.Rename(tmpPath, execPath); err != nil {
					// Cross-device fallback: copy instead
					srcFile, openErr := os.Open(tmpPath)
					if openErr != nil {
						os.Remove(tmpPath)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(map[string]string{"error": "Failed to open temp file"})
						return
					}
					defer srcFile.Close()

					dstFile, createErr := os.Create(execPath)
					if createErr != nil {
						os.Remove(tmpPath)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(map[string]string{"error": "Failed to replace binary"})
						return
					}
					defer dstFile.Close()

					if _, copyErr := io.Copy(dstFile, srcFile); copyErr != nil {
						os.Remove(tmpPath)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(map[string]string{"error": "Failed to write binary"})
						return
					}
					os.Remove(tmpPath)
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "updated",
					"message": "Update complete. Restarting...",
				})

				// Flush response before exiting
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}

				// Exit after response — Docker/systemd will restart with new binary
				go func() {
					time.Sleep(2 * time.Second)
					os.Exit(0)
				}()
			})

		// Serve SPA - must be last to catch all other routes
		r.Get("/*", serveSPA)
		r.Get("/", serveSPA)
	})

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Start server
	serverAddr := ":" + cfg.Port
	log.Printf("Starting Torrent Server on http://localhost%s", serverAddr)
	log.Printf("Admin UI: http://localhost%s/admin", serverAddr)
	log.Printf("API: http://localhost%s/api/v2/list_movies.json", serverAddr)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		if licenseClient != nil {
			licenseClient.Stop()
		}
		torrentService.Close()
		db.Close()
		os.Exit(0)
	}()

	if err := http.ListenAndServe(serverAddr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

	// Keep reference to avoid unused import error
	_ = omdbService
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// migrateSubtitlesToDisk moves existing VTT content from DB to disk files.
func migrateSubtitlesToDisk(db *database.DB, subtitlesDir string) {
	subs, err := db.GetSubtitlesWithContent()
	if err != nil {
		log.Printf("[Migration] Failed to get subtitles for migration: %v", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	log.Printf("[Migration] Migrating %d subtitles from DB to disk...", len(subs))
	migrated := 0
	for _, sub := range subs {
		dir := filepath.Join(subtitlesDir, sub.ImdbCode)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("[Migration] Failed to create dir %s: %v", dir, err)
			continue
		}
		vttPath := filepath.Join(dir, fmt.Sprintf("%d.vtt", sub.ID))
		if err := os.WriteFile(vttPath, []byte(sub.VTTContent), 0644); err != nil {
			log.Printf("[Migration] Failed to write %s: %v", vttPath, err)
			continue
		}
		if err := db.UpdateSubtitlePath(sub.ID, vttPath); err != nil {
			log.Printf("[Migration] Failed to update path for ID %d: %v", sub.ID, err)
			continue
		}
		migrated++
	}
	log.Printf("[Migration] Migrated %d/%d subtitles to disk", migrated, len(subs))
}

// serveSPA serves the Svelte SPA from the embedded static/admin directory
func serveSPA(w http.ResponseWriter, r *http.Request) {
	// Get the path
	urlPath := r.URL.Path

	// Remove /admin prefix if present
	if strings.HasPrefix(urlPath, "/admin") {
		urlPath = strings.TrimPrefix(urlPath, "/admin")
	}
	if urlPath == "" || urlPath == "/" {
		urlPath = "/index.html"
	}

	// Try to get the file from embedded FS
	subFS, err := fs.Sub(staticFS, "static/admin")
	if err != nil {
		http.Error(w, "Admin UI not available", http.StatusInternalServerError)
		return
	}

	// Check if the requested file exists
	filePath := strings.TrimPrefix(urlPath, "/")
	file, err := subFS.Open(filePath)
	if err != nil {
		// File doesn't exist, serve index.html for SPA routing
		filePath = "index.html"
	} else {
		file.Close()
	}

	// Get the file info for content type
	content, err := fs.ReadFile(subFS, filePath)
	if err != nil {
		http.Error(w, "Admin UI not available", http.StatusNotFound)
		return
	}

	// Set content type based on extension
	ext := path.Ext(filePath)
	contentType := "text/html"
	switch ext {
	case ".js":
		contentType = "application/javascript"
	case ".css":
		contentType = "text/css"
	case ".svg":
		contentType = "image/svg+xml"
	case ".png":
		contentType = "image/png"
	case ".ico":
		contentType = "image/x-icon"
	case ".json":
		contentType = "application/json"
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(content)
}
