package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	// YTS-compatible API (public)
	r.Route("/api/v2", func(r chi.Router) {
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
	})

	// Stremio addon (public)
	r.Get("/manifest.json", stremioHandler.Manifest)
	r.Get("/catalog/{type}/{id}", stremioHandler.Catalog)
	r.Get("/stream/{type}/{id}", stremioHandler.Stream)

	// Video streaming (public)
	r.Get("/stream/{infoHash}/{fileIndex}", streamHandler.Stream)
	r.Get("/stats", streamHandler.Stats)

	// Admin routes (protected)
	r.Route("/admin", func(r chi.Router) {
		r.Get("/login", auth.Login)
		r.Post("/login", auth.Login)
		r.Get("/logout", auth.Logout)

		// Protected admin routes
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth)
			r.Get("/", adminHandler.Dashboard)
			r.Post("/movies", adminHandler.AddMovie)
			r.Post("/movies/{id}/torrent", adminHandler.AddTorrent)
			r.Delete("/movies/{id}", adminHandler.DeleteMovie)
			r.Post("/upload", adminHandler.UploadTorrent)
		})
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
