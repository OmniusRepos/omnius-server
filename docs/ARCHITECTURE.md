# Torrent Server Architecture

## Overview

Torrent Server (Omnius) is a Go-based backend service that provides:
- Movie/Series metadata management
- Torrent information aggregation
- IPTV channel directory
- Analytics tracking
- Admin dashboard

---

## Project Structure

```
torrent-server/
├── main.go                 # Application entry point, router setup
├── config/
│   └── config.go           # Configuration management
├── database/
│   ├── sqlite.go           # Database connection, migrations
│   ├── movies.go           # Movie CRUD operations
│   ├── series.go           # Series CRUD operations
│   ├── channel.go          # Channel CRUD operations
│   ├── home.go             # Home sections management
│   ├── curated.go          # Curated lists management
│   └── analytics.go        # Analytics data operations
├── handlers/
│   ├── api.go              # Public API endpoints
│   ├── admin.go            # Admin dashboard handlers
│   ├── series.go           # Series API handlers
│   ├── channel.go          # Channel API handlers
│   ├── ratings.go          # Ratings & sync handlers
│   ├── analytics.go        # Analytics handlers
│   ├── curated.go          # Curated list handlers
│   ├── home.go             # Home page handlers
│   ├── stream.go           # Video streaming handlers
│   └── stremio.go          # Stremio addon handlers
├── models/
│   ├── movie.go            # Movie data structures
│   ├── series.go           # Series data structures
│   └── channel.go          # Channel data structures
├── providers/
│   ├── yts.go              # YTS torrent provider
│   └── eztv.go             # EZTV torrent provider
├── services/
│   ├── sync.go             # Movie sync service
│   ├── omdb.go             # OMDB API client
│   ├── imdb.go             # IMDB API client
│   └── torrent.go          # Torrent service
├── frontend/               # Svelte 5 admin dashboard
│   ├── src/
│   │   ├── routes/         # Page components
│   │   ├── lib/            # Shared components & utilities
│   │   └── App.svelte      # Main app component
│   └── package.json
├── static/
│   └── admin/              # Built admin frontend (embedded)
├── templates/
│   └── admin.html          # Admin HTML template
├── data/
│   └── omnius.db           # SQLite database
└── docs/                   # Documentation
```

---

## Core Components

### 1. Router (Chi)

Uses `go-chi/chi` for HTTP routing with middleware:

```go
r := chi.NewRouter()
r.Use(middleware.Logger)
r.Use(middleware.RealIP)
r.Use(cors.Handler(cors.Options{...}))
```

### 2. Database Layer

Pure Go SQLite driver (`modernc.org/sqlite`) for portability:

```go
type DB struct {
    *sql.DB
}

func New(dbPath string) (*DB, error) {
    db, err := sql.Open("sqlite", dbPath)
    // Enable foreign keys, run migrations
    return &DB{db}, nil
}
```

### 3. Handlers

Stateless HTTP handlers that receive database connection:

```go
type APIHandler struct {
    db *database.DB
}

func (h *APIHandler) ListMovies(w http.ResponseWriter, r *http.Request) {
    // Parse query params, call db, return JSON
}
```

### 4. Providers

External torrent source integrations:

```go
type Provider interface {
    Name() string
    SearchMovie(title string, year int) ([]TorrentResult, error)
    SearchSeries(title string, season, episode int) ([]TorrentResult, error)
}
```

Current providers:
- **YTS**: Movies (https://yts.mx/api)
- **EZTV**: TV Series (https://eztvx.to/api)

### 5. Services

Business logic and external API clients:

- **SyncService**: Coordinates movie syncing from multiple sources
- **OMDBClient**: Fetches ratings from OMDB API
- **IMDBClient**: Fetches metadata from IMDB API
- **TorrentService**: Manages torrent operations

---

## Data Flow

### Movie Sync Flow

```
User Request → SyncMovie Handler
    ↓
Check if exists in DB
    ↓ (not found)
YTS Provider → Fetch movie data
    ↓
OMDB API → Fetch ratings
    ↓
IMDB API → Fetch images/cast
    ↓
Save to SQLite
    ↓
Return to User
```

### Search Flow

```
GET /api/v2/search.json?query=term
    ↓
APIHandler.UnifiedSearch()
    ↓
┌──────────────┬──────────────┬──────────────┐
│ Movies DB    │ Series DB    │ Channels DB  │
│ (query_term) │ (title LIKE) │ (query_term) │
└──────────────┴──────────────┴──────────────┘
    ↓
Combine Results
    ↓
Return JSON
```

---

## Embedded Static Files

Admin frontend is embedded into the Go binary using `//go:embed`:

```go
//go:embed static/admin
var adminFS embed.FS

// Serve static files
r.Handle("/admin/*", http.FileServer(http.FS(adminFS)))
```

**Important**: After frontend changes, must rebuild Go binary:
```bash
cd frontend && npm run build
cd .. && go build -o torrent-server .
```

---

## API Response Format

All API responses follow YTS-compatible format:

```json
{
  "status": "ok",
  "status_message": "Query was successful",
  "data": {
    // Response data here
  }
}
```

Error response:
```json
{
  "status": "error",
  "status_message": "Error description"
}
```

---

## Configuration

Environment variables and config file support:

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | HTTP server port |
| DB_PATH | data/omnius.db | SQLite database path |
| OMDB_API_KEY | - | OMDB API key for ratings |

---

## Deployment

### Local Development
```bash
go run .
# or
go build -o torrent-server . && ./torrent-server
```

### Production (Basepod)
```bash
bp deploy
```

Creates container with:
- Compiled Go binary
- SQLite database (persistent volume)
- Exposed port 8080

---

## Stremio Addon

Implements Stremio addon protocol for streaming integration:

```
GET /manifest.json         # Addon manifest
GET /catalog/{type}/{id}   # Content catalog
GET /stream/{type}/{id}    # Stream sources
```

---

## Security Considerations

1. **CORS**: Configured to allow all origins for API access
2. **No Auth on Public API**: Read-only endpoints don't require auth
3. **Admin Auth**: Dashboard requires authentication (basic auth/session)
4. **SQL Injection**: Uses parameterized queries throughout
5. **Input Validation**: Query params are validated and sanitized
