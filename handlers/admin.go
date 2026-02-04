package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"torrent-server/database"
	"torrent-server/models"
	"torrent-server/services"
)

type AdminHandler struct {
	db             *database.DB
	torrentService *services.TorrentService
	templates      *template.Template
}

func NewAdminHandler(db *database.DB, ts *services.TorrentService) *AdminHandler {
	return &AdminHandler{
		db:             db,
		torrentService: ts,
	}
}

func (h *AdminHandler) SetTemplates(t *template.Template) {
	h.templates = t
}

// Dashboard handles GET /admin
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	movies, movieCount, _ := h.db.ListMovies(database.MovieFilter{Limit: 50, Page: 1})
	series, seriesCount, _ := h.db.ListSeries(50, 1)

	data := map[string]interface{}{
		"Movies":      movies,
		"MovieCount":  movieCount,
		"Series":      series,
		"SeriesCount": seriesCount,
		"Stats":       h.torrentService.GetStats(),
	}

	if h.templates != nil {
		h.templates.ExecuteTemplate(w, "admin.html", data)
	} else {
		// Fallback simple HTML
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(adminFallbackHTML(data)))
	}
}

// AddMovie handles POST /admin/movies
func (h *AdminHandler) AddMovie(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	year, _ := strconv.Atoi(r.FormValue("year"))
	rating, _ := strconv.ParseFloat(r.FormValue("rating"), 32)
	runtime, _ := strconv.Atoi(r.FormValue("runtime"))

	movie := &models.Movie{
		ImdbCode:         r.FormValue("imdb_code"),
		Title:            r.FormValue("title"),
		TitleEnglish:     r.FormValue("title_english"),
		TitleLong:        r.FormValue("title_long"),
		Slug:             strings.ToLower(strings.ReplaceAll(r.FormValue("title"), " ", "-")),
		Year:             uint(year),
		Rating:           float32(rating),
		Runtime:          uint(runtime),
		Genres:           strings.Split(r.FormValue("genres"), ","),
		Summary:          r.FormValue("summary"),
		DescriptionFull:  r.FormValue("description_full"),
		Synopsis:         r.FormValue("synopsis"),
		YtTrailerCode:    r.FormValue("yt_trailer_code"),
		Language:         r.FormValue("language"),
		BackgroundImage:  r.FormValue("background_image"),
		SmallCoverImage:  r.FormValue("small_cover_image"),
		MediumCoverImage: r.FormValue("medium_cover_image"),
		LargeCoverImage:  r.FormValue("large_cover_image"),
	}

	if movie.Language == "" {
		movie.Language = "en"
	}

	if err := h.db.CreateMovie(movie); err != nil {
		http.Error(w, "Failed to create movie: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// If JSON request, return JSON
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(movie)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// AddTorrent handles POST /admin/movies/{id}/torrent
func (h *AdminHandler) AddTorrent(w http.ResponseWriter, r *http.Request) {
	movieID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		r.ParseForm()
	}

	// Get torrent hash - either from magnet link or direct hash
	hash := r.FormValue("hash")
	magnetLink := r.FormValue("magnet")

	if hash == "" && magnetLink != "" {
		// Extract hash from magnet link
		hash = extractHashFromMagnet(magnetLink)
	}

	if hash == "" {
		http.Error(w, "Torrent hash is required", http.StatusBadRequest)
		return
	}

	sizeBytes, _ := strconv.ParseUint(r.FormValue("size_bytes"), 10, 64)
	seeds, _ := strconv.Atoi(r.FormValue("seeds"))
	peers, _ := strconv.Atoi(r.FormValue("peers"))

	torrent := &models.Torrent{
		MovieID:    uint(movieID),
		Hash:       strings.ToUpper(hash),
		URL:        magnetLink,
		Quality:    r.FormValue("quality"),
		Type:       r.FormValue("type"),
		VideoCodec: r.FormValue("video_codec"),
		Size:       r.FormValue("size"),
		SizeBytes:  sizeBytes,
		Seeds:      uint(seeds),
		Peers:      uint(peers),
	}

	if torrent.Quality == "" {
		torrent.Quality = "1080p"
	}
	if torrent.Type == "" {
		torrent.Type = "web"
	}

	if err := h.db.CreateTorrent(torrent); err != nil {
		http.Error(w, "Failed to add torrent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(torrent)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// DeleteMovie handles DELETE /admin/movies/{id}
func (h *AdminHandler) DeleteMovie(w http.ResponseWriter, r *http.Request) {
	movieID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteMovie(uint(movieID)); err != nil {
		http.Error(w, "Failed to delete movie: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// UpdateMovie handles PUT /admin/api/movies/{id}
func (h *AdminHandler) UpdateMovie(w http.ResponseWriter, r *http.Request) {
	movieID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	// Get existing movie
	movie, err := h.db.GetMovie(uint(movieID))
	if err != nil {
		http.Error(w, "Movie not found", http.StatusNotFound)
		return
	}

	// Parse JSON body
	var update struct {
		ImdbCode         string  `json:"imdb_code"`
		Title            string  `json:"title"`
		Year             int     `json:"year"`
		Rating           float64 `json:"rating"`
		Runtime          int     `json:"runtime"`
		Genres           string  `json:"genres"`
		Language         string  `json:"language"`
		Summary          string  `json:"summary"`
		YtTrailerCode    string  `json:"yt_trailer_code"`
		MediumCoverImage string  `json:"medium_cover_image"`
		BackgroundImage  string  `json:"background_image"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update fields
	if update.ImdbCode != "" {
		movie.ImdbCode = update.ImdbCode
	}
	if update.Title != "" {
		movie.Title = update.Title
		movie.Slug = strings.ToLower(strings.ReplaceAll(update.Title, " ", "-"))
	}
	if update.Year > 0 {
		movie.Year = uint(update.Year)
	}
	movie.Rating = float32(update.Rating)
	movie.Runtime = uint(update.Runtime)
	if update.Genres != "" {
		movie.Genres = strings.Split(update.Genres, ",")
		for i := range movie.Genres {
			movie.Genres[i] = strings.TrimSpace(movie.Genres[i])
		}
	}
	if update.Language != "" {
		movie.Language = update.Language
	}
	movie.Summary = update.Summary
	movie.YtTrailerCode = update.YtTrailerCode
	movie.MediumCoverImage = update.MediumCoverImage
	movie.BackgroundImage = update.BackgroundImage

	if err := h.db.UpdateMovie(movie); err != nil {
		http.Error(w, "Failed to update movie: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movie)
}

// UploadTorrent handles POST /admin/upload
func (h *AdminHandler) UploadTorrent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("torrent")
	if err != nil {
		http.Error(w, "Failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read the torrent file
	var buf []byte
	buf = make([]byte, 32<<20) // 32MB max
	n, _ := file.Read(buf)
	buf = buf[:n]

	// Add to torrent client
	t, err := h.torrentService.AddTorrentFile(buf)
	if err != nil {
		http.Error(w, "Failed to add torrent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"hash": t.InfoHash().HexString(),
		"name": t.Name(),
		"size": t.Length(),
	})
}

func extractHashFromMagnet(magnet string) string {
	// Extract hash from magnet:?xt=urn:btih:HASH&...
	magnet = strings.ToLower(magnet)
	start := strings.Index(magnet, "btih:")
	if start == -1 {
		return ""
	}
	start += 5
	end := strings.IndexAny(magnet[start:], "&?")
	if end == -1 {
		return magnet[start:]
	}
	return magnet[start : start+end]
}

func adminFallbackHTML(data map[string]interface{}) string {
	return `<!DOCTYPE html>
<html>
<head>
	<title>Torrent Server Admin</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 20px; background: #1a1a2e; color: #eee; }
		h1, h2 { color: #e94560; }
		.container { max-width: 1200px; margin: 0 auto; }
		.card { background: #16213e; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
		table { width: 100%; border-collapse: collapse; }
		th, td { padding: 10px; text-align: left; border-bottom: 1px solid #333; }
		th { color: #e94560; }
		input, select, textarea { width: 100%; padding: 8px; margin: 5px 0 15px; border: 1px solid #333; border-radius: 4px; background: #0f3460; color: #eee; }
		button { background: #e94560; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; }
		button:hover { background: #ff6b6b; }
		.btn-delete { background: #dc3545; }
		.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 20px; }
	</style>
</head>
<body>
	<div class="container">
		<h1>Torrent Server Admin</h1>

		<div class="card">
			<h2>Add Movie</h2>
			<form method="POST" action="/admin/movies">
				<input name="imdb_code" placeholder="IMDB Code (e.g., tt1234567)" required>
				<input name="title" placeholder="Title" required>
				<input name="year" type="number" placeholder="Year">
				<input name="rating" type="number" step="0.1" placeholder="Rating">
				<input name="genres" placeholder="Genres (comma separated)">
				<textarea name="summary" placeholder="Summary"></textarea>
				<input name="medium_cover_image" placeholder="Cover Image URL">
				<button type="submit">Add Movie</button>
			</form>
		</div>

		<div class="card">
			<h2>Movies</h2>
			<p>Total: ` + strconv.Itoa(data["MovieCount"].(int)) + `</p>
			<table>
				<tr><th>ID</th><th>Title</th><th>Year</th><th>IMDB</th><th>Actions</th></tr>
			</table>
		</div>
	</div>
</body>
</html>`
}
