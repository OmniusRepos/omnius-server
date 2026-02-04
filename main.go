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

	// Parse templates
	templates, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Printf("Warning: Failed to parse templates: %v", err)
		templates = nil
	}

	// Initialize handlers
	apiHandler := handlers.NewAPIHandler(db)
	streamHandler := handlers.NewStreamHandler(torrentService)
	stremioHandler := handlers.NewStremioHandler(db, torrentService, getBaseURL(cfg.Port))
	adminHandler := handlers.NewAdminHandler(db, torrentService)
	adminHandler.SetTemplates(templates)

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

	// Health check
	r.Get("/health", streamHandler.Health)

	// Initialize ratings handler
	ratingsHandler := handlers.NewRatingsHandler(db)

	// Initialize series handler
	seriesHandler := handlers.NewSeriesHandler(db)

	// Initialize curated handler
	curatedHandler := handlers.NewCuratedHandler(db)

	// Initialize home handler
	homeHandler := handlers.NewHomeHandler(db)

	// YTS-compatible API (public)
	r.Route("/api/v2", func(r chi.Router) {
		// Home
		r.Get("/home.json", apiHandler.Home)

		// Movies
		r.Get("/list_movies.json", apiHandler.ListMovies)
		r.Get("/movie_details.json", apiHandler.MovieDetails)
		r.Get("/movie_suggestions.json", apiHandler.MovieSuggestions)

		// Series
		r.Get("/list_series.json", seriesHandler.ListSeries)
		r.Get("/series_details.json", seriesHandler.SeriesDetails)
		r.Get("/season_episodes.json", seriesHandler.SeasonEpisodes)

		// Ratings & Sync
		r.Post("/get_ratings", ratingsHandler.GetRatings)
		r.Post("/sync_movie", ratingsHandler.SyncMovie)
		r.Post("/sync_movies", ratingsHandler.SyncMovies)
		r.Get("/torrent_stats", ratingsHandler.TorrentStats)

		// Curated Lists
		r.Get("/curated_lists.json", curatedHandler.ListCuratedLists)
		r.Get("/curated_list.json", curatedHandler.GetCuratedList)
	})

	// Stremio addon (public)
	r.Get("/manifest.json", stremioHandler.Manifest)
	r.Get("/catalog/{type}/{id}", stremioHandler.Catalog)
	r.Get("/stream/{type}/{id}", stremioHandler.Stream)

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
	log.Printf("Stremio Manifest: http://localhost%s/manifest.json", serverAddr)
	log.Printf("YTS API: http://localhost%s/api/v2/list_movies.json", serverAddr)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
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

func getBaseURL(port string) string {
	// In production, this should be configured via environment variable
	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		return baseURL
	}
	return fmt.Sprintf("http://localhost:%s", port)
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
