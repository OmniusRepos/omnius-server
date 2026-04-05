# SQLite to GORM + PostgreSQL Migration

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace raw `database/sql` + SQLite with GORM supporting both SQLite (local dev) and PostgreSQL (production), fixing data loss on deploy.

**Architecture:** GORM wraps `*gorm.DB` with dual-driver init via `DATABASE_URL` env var. Empty/absent = SQLite at `./data/omnius.db`. Present = PostgreSQL. AutoMigrate replaces hand-written schema. All 9 database files rewritten to GORM query builder. Models already have GORM tags in `models/` package.

**Tech Stack:** `gorm.io/gorm`, `gorm.io/driver/postgres`, `gorm.io/driver/sqlite`

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Rewrite | `config/config.go` | Add `DatabaseURL` field |
| Delete | `database/sqlite.go` | Replaced by database.go |
| Create | `database/database.go` | GORM init, dual driver, AutoMigrate, seed |
| Rewrite | `database/movies.go` | Movie CRUD with GORM |
| Rewrite | `database/torrents.go` | Torrent CRUD with GORM |
| Rewrite | `database/series.go` | Series/Episode/SeasonPack CRUD with GORM |
| Rewrite | `database/analytics.go` | Views, streams, stats with GORM |
| Rewrite | `database/channel.go` | Channel CRUD, EPG, blocklist with GORM |
| Rewrite | `database/home.go` | HomeSection CRUD with GORM (remove local struct) |
| Rewrite | `database/curated.go` | CuratedList CRUD with GORM |
| Rewrite | `database/config.go` | ServiceConfig CRUD with GORM |
| Rewrite | `database/subtitles.go` | Subtitle CRUD with GORM |
| Modify | `main.go` | Update DB init call + handler wiring |
| Modify | `handlers/*.go` | Update type from `*database.DB` to accept GORM DB |
| Modify | `services/sync.go` | Update DB type |
| Modify | `services/subtitle.go` | Update DB type |
| Modify | `basepod.yaml` | Add DATABASE_URL env var |
| Modify | `go.mod` | Already has GORM deps |

**Parallelism:** Tasks 2-10 are independent (each rewrites one database file). Task 11 depends on all of 2-10. Tasks 12-13 depend on 11.

---

### Task 1: Foundation - Config + Database Init

**Files:**
- Modify: `config/config.go`
- Delete: `database/sqlite.go`
- Create: `database/database.go`

- [ ] **Step 1: Update config to add DatabaseURL**

```go
// config/config.go
package config

import "os"

type Config struct {
	Port          string
	AdminPassword string
	DatabasePath  string
	DatabaseURL   string
	DownloadDir   string
	OmdbAPIKey    string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin"),
		DatabasePath:  getEnv("DATABASE_PATH", "./data/omnius.db"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		DownloadDir:   getEnv("DOWNLOAD_DIR", "./data/downloads"),
		OmdbAPIKey:    getEnv("OMDB_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

- [ ] **Step 2: Create database/database.go with GORM dual-driver init**

```go
// database/database.go
package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"torrent-server/models"
)

type DB struct {
	*gorm.DB
}

// New creates a GORM database connection.
// If databaseURL is non-empty, connects to PostgreSQL.
// Otherwise falls back to SQLite at dbPath.
func New(dbPath, databaseURL string) (*DB, error) {
	var dialector gorm.Dialector

	if databaseURL != "" {
		dialector = postgres.Open(databaseURL)
		log.Printf("Connecting to PostgreSQL...")
	} else {
		// Ensure directory exists for SQLite
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
		dialector = sqlite.Open(dbPath)
		log.Printf("Using SQLite at %s", dbPath)
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable foreign keys for SQLite
	if databaseURL == "" {
		db.Exec("PRAGMA foreign_keys = ON")
	}

	d := &DB{db}

	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	d.seed()

	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *DB) migrate() error {
	return d.AutoMigrate(
		&models.Movie{},
		&models.Torrent{},
		&models.Series{},
		&models.Season{},
		&models.Episode{},
		&models.EpisodeTorrent{},
		&models.SeasonPack{},
		&models.CuratedList{},
		&models.CuratedListMovie{},
		&models.HomeSection{},
		&models.ContentView{},
		&models.ContentStatsDaily{},
		&models.ActiveStream{},
		&models.Channel{},
		&models.ChannelCountry{},
		&models.ChannelCategory{},
		&models.ChannelEPG{},
		&models.ChannelBlocklist{},
		&models.ServiceConfig{},
		&models.StoredSubtitle{},
	)
}

func (d *DB) seed() {
	// Seed default services if empty
	var serviceCount int64
	d.Model(&models.ServiceConfig{}).Count(&serviceCount)
	if serviceCount == 0 {
		d.Create(&models.ServiceConfig{ID: "movies", Label: "Movies", Enabled: true, Icon: "movie", DisplayOrder: 1})
		d.Create(&models.ServiceConfig{ID: "series", Label: "TV Shows", Enabled: true, Icon: "tv", DisplayOrder: 2})
		d.Create(&models.ServiceConfig{ID: "channels", Label: "Live TV", Enabled: false, Icon: "live", DisplayOrder: 3})
	}

	// Seed default home sections if missing Netflix-style sections
	var top10Count int64
	d.Model(&models.HomeSection{}).Where("display_type = ?", "top10").Count(&top10Count)
	if top10Count == 0 {
		d.Where("1 = 1").Delete(&models.HomeSection{})
		heroContentID := uint(243)
		d.Create(&[]models.HomeSection{
			{SectionID: "hero_featured", Title: "Featured", DisplayType: "hero", ContentType: "movie", ContentID: &heroContentID, SortBy: "rating", OrderBy: "desc", LimitCount: 1, IsActive: true, DisplayOrder: 0},
			{SectionID: "top_10", Title: "Top 10 on Omnius Today", DisplayType: "top10", SectionType: "top_rated", SortBy: "rating", OrderBy: "desc", MinimumRating: 7.0, LimitCount: 10, IsActive: true, DisplayOrder: 1},
			{SectionID: "trending", Title: "Trending Now", DisplayType: "carousel", SectionType: "top_viewed", SortBy: "download_count", OrderBy: "desc", LimitCount: 20, IsActive: true, DisplayOrder: 2},
			{SectionID: "recently_added", Title: "New on Omnius", DisplayType: "carousel", SectionType: "recent", SortBy: "date_uploaded", OrderBy: "desc", LimitCount: 20, IsActive: true, DisplayOrder: 3},
			{SectionID: "top_rated", Title: "Critically Acclaimed", DisplayType: "top10", SectionType: "top_rated", SortBy: "rating", OrderBy: "desc", MinimumRating: 8.0, LimitCount: 10, IsActive: true, DisplayOrder: 4},
			{SectionID: "action", Title: "Action & Adventure", DisplayType: "carousel", SectionType: "genre", Genre: "Action", SortBy: "rating", OrderBy: "desc", MinimumRating: 5.0, LimitCount: 20, IsActive: true, DisplayOrder: 5},
			{SectionID: "comedy", Title: "Comedy", DisplayType: "carousel", SectionType: "genre", Genre: "Comedy", SortBy: "rating", OrderBy: "desc", MinimumRating: 5.0, LimitCount: 20, IsActive: true, DisplayOrder: 6},
			{SectionID: "scifi", Title: "Sci-Fi & Fantasy", DisplayType: "carousel", SectionType: "genre", Genre: "Sci-Fi", SortBy: "rating", OrderBy: "desc", MinimumRating: 5.0, LimitCount: 20, IsActive: true, DisplayOrder: 7},
			{SectionID: "horror", Title: "Horror", DisplayType: "carousel", SectionType: "genre", Genre: "Horror", SortBy: "rating", OrderBy: "desc", MinimumRating: 4.0, LimitCount: 20, IsActive: true, DisplayOrder: 8},
			{SectionID: "drama", Title: "Drama", DisplayType: "carousel", SectionType: "genre", Genre: "Drama", SortBy: "rating", OrderBy: "desc", MinimumRating: 6.0, LimitCount: 20, IsActive: true, DisplayOrder: 9},
			{SectionID: "thriller", Title: "Thrillers", DisplayType: "carousel", SectionType: "genre", Genre: "Thriller", SortBy: "rating", OrderBy: "desc", MinimumRating: 5.0, LimitCount: 20, IsActive: true, DisplayOrder: 10},
		})
	}
}
```

- [ ] **Step 3: Delete old database/sqlite.go**

```bash
rm database/sqlite.go
```

- [ ] **Step 4: Verify it compiles (will fail on database method calls - expected)**

```bash
go build ./database/...
```

Expected: Compile errors in other database files referencing old `*sql.DB` methods. This is expected - Tasks 2-10 fix these.

- [ ] **Step 5: Commit**

```bash
git add config/config.go database/database.go
git add -u database/sqlite.go
git commit -m "feat: add GORM foundation with dual SQLite/PostgreSQL driver support"
```

---

### Task 2: Rewrite database/movies.go

**Files:**
- Rewrite: `database/movies.go`

**Depends on:** Task 1

- [ ] **Step 1: Rewrite movies.go with GORM**

```go
// database/movies.go
package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"torrent-server/models"
)

type MovieFilter struct {
	Limit         int
	Page          int
	Quality       string
	MinimumRating float32
	QueryTerm     string
	Genre         string
	SortBy        string
	OrderBy       string
	Year          int
	MaximumYear   int
	MinimumYear   int
	Status        string
}

func (d *DB) ListMovies(filter MovieFilter) ([]models.Movie, int, error) {
	if filter.Limit <= 0 || filter.Limit > 50 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.SortBy == "" {
		filter.SortBy = "date_uploaded"
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "desc"
	}

	query := d.Model(&models.Movie{})

	if filter.MinimumRating > 0 {
		query = query.Where("rating >= ?", filter.MinimumRating)
	}
	if filter.QueryTerm != "" {
		query = query.Where("title LIKE ? OR imdb_code = ?", "%"+filter.QueryTerm+"%", filter.QueryTerm)
	}
	if filter.Genre != "" {
		query = query.Where("genres LIKE ?", "%"+filter.Genre+"%")
	}
	if filter.Year > 0 {
		query = query.Where("year = ?", filter.Year)
	}
	if filter.MinimumYear > 0 {
		query = query.Where("year >= ?", filter.MinimumYear)
	}
	if filter.MaximumYear > 0 {
		query = query.Where("year <= ?", filter.MaximumYear)
	}
	if filter.Status != "" {
		query = query.Where("COALESCE(status, 'available') = ?", filter.Status)
	}

	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Validate and map sort column
	validSortColumns := map[string]string{
		"title":          "title",
		"year":           "year",
		"rating":         "rating",
		"date_uploaded":  "date_uploaded_unix",
		"date_added":     "date_uploaded_unix",
		"seeds":          "seeds",
		"download_count": "download_count",
	}
	sortCol, ok := validSortColumns[filter.SortBy]
	if !ok {
		sortCol = "date_uploaded_unix"
	}

	orderDir := "DESC"
	if strings.ToLower(filter.OrderBy) == "asc" {
		orderDir = "ASC"
	}

	offset := (filter.Page - 1) * filter.Limit

	var movies []models.Movie
	err := query.
		Order(fmt.Sprintf("%s %s", sortCol, orderDir)).
		Limit(filter.Limit).
		Offset(offset).
		Find(&movies).Error
	if err != nil {
		return nil, 0, err
	}

	// Load torrents for each movie
	for i := range movies {
		d.DB.Where("movie_id = ?", movies[i].ID).Find(&movies[i].Torrents)
	}

	return movies, int(totalCount), nil
}

func (d *DB) GetMovie(id uint) (*models.Movie, error) {
	var m models.Movie
	if err := d.Preload("Torrents").First(&m, id).Error; err != nil {
		return nil, err
	}
	if m.Status == "" {
		m.Status = "available"
	}
	return &m, nil
}

func (d *DB) GetMovieByIMDB(imdbCode string) (*models.Movie, error) {
	var m models.Movie
	if err := d.Preload("Torrents").Where("imdb_code = ?", imdbCode).First(&m).Error; err != nil {
		return nil, err
	}
	if m.Status == "" {
		m.Status = "available"
	}
	return &m, nil
}

func (d *DB) CreateMovie(m *models.Movie) error {
	now := time.Now()
	if m.DateUploaded == "" {
		m.DateUploaded = now.Format("2006-01-02 15:04:05")
	}
	if m.DateUploadedUnix == 0 {
		m.DateUploadedUnix = now.Unix()
	}
	if m.Status == "" {
		m.Status = "available"
	}
	return d.Create(m).Error
}

func (d *DB) UpdateMovie(m *models.Movie) error {
	return d.Save(m).Error
}

func (d *DB) DeleteMovie(id uint) error {
	return d.Delete(&models.Movie{}, id).Error
}

func (d *DB) GetMovieSuggestions(movieID uint, limit int) ([]models.Movie, error) {
	if limit <= 0 {
		limit = 4
	}

	movie, err := d.GetMovie(movieID)
	if err != nil {
		return nil, err
	}

	var movies []models.Movie
	err = d.Preload("Torrents").
		Where("id != ?", movieID).
		Order(gorm.Expr("ABS(year - ?) ASC, rating DESC", movie.Year)).
		Limit(limit).
		Find(&movies).Error

	return movies, err
}

func (d *DB) GetFranchiseMovies(movieID uint, franchise string) ([]models.Movie, error) {
	if franchise == "" {
		return nil, nil
	}

	var movies []models.Movie
	err := d.Preload("Torrents").
		Where("franchise = ? AND id != ?", franchise, movieID).
		Order("year ASC").
		Find(&movies).Error

	return movies, err
}

func (d *DB) GetMovieRating(imdbCode string) (*models.LocalRating, error) {
	var m models.Movie
	err := d.Select("rating, imdb_rating, rotten_tomatoes, metacritic").
		Where("imdb_code = ?", imdbCode).
		First(&m).Error
	if err != nil {
		return nil, err
	}

	result := &models.LocalRating{}
	if m.ImdbRating != nil && *m.ImdbRating > 0 {
		result.ImdbRating = m.ImdbRating
	} else if m.Rating > 0 {
		r := m.Rating
		result.ImdbRating = &r
	}
	if m.RottenTomatoes != nil && *m.RottenTomatoes > 0 {
		result.RottenTomatoes = m.RottenTomatoes
	}
	if m.Metacritic != nil && *m.Metacritic > 0 {
		result.Metacritic = m.Metacritic
	}

	return result, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add database/movies.go
git commit -m "feat: rewrite movies.go to GORM"
```

---

### Task 3: Rewrite database/torrents.go

**Files:**
- Rewrite: `database/torrents.go`

**Depends on:** Task 1

- [ ] **Step 1: Rewrite torrents.go with GORM**

```go
// database/torrents.go
package database

import (
	"time"

	"torrent-server/models"
)

func (d *DB) GetTorrentsForMovie(movieID uint) ([]models.Torrent, error) {
	var torrents []models.Torrent
	err := d.Where("movie_id = ?", movieID).Find(&torrents).Error
	return torrents, err
}

func (d *DB) GetTorrentByHash(hash string) (*models.Torrent, error) {
	var t models.Torrent
	if err := d.Where("hash = ?", hash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (d *DB) CreateTorrent(t *models.Torrent) error {
	now := time.Now()
	if t.DateUploaded == "" {
		t.DateUploaded = now.Format("2006-01-02 15:04:05")
	}
	if t.DateUploadedUnix == 0 {
		t.DateUploadedUnix = now.Unix()
	}
	return d.Create(t).Error
}

func (d *DB) DeleteTorrent(id uint) error {
	return d.Delete(&models.Torrent{}, id).Error
}

func (d *DB) GetIMDBByHash(hash string) (string, error) {
	var result struct{ ImdbCode string }
	err := d.Model(&models.Torrent{}).
		Select("movies.imdb_code").
		Joins("JOIN movies ON movies.id = torrents.movie_id").
		Where("torrents.hash = ?", hash).
		Scan(&result).Error
	return result.ImdbCode, err
}

func (d *DB) DeleteTorrentsByMovie(movieID uint) error {
	return d.Where("movie_id = ?", movieID).Delete(&models.Torrent{}).Error
}
```

- [ ] **Step 2: Commit**

```bash
git add database/torrents.go
git commit -m "feat: rewrite torrents.go to GORM"
```

---

### Task 4: Rewrite database/series.go

**Files:**
- Rewrite: `database/series.go`

**Depends on:** Task 1

- [ ] **Step 1: Rewrite series.go with GORM**

```go
// database/series.go
package database

import (
	"time"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

func (d *DB) ListSeries(limit, page int) ([]models.Series, int, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	var totalCount int64
	d.Model(&models.Series{}).Count(&totalCount)

	offset := (page - 1) * limit
	var seriesList []models.Series
	err := d.Order("date_added_unix DESC").
		Limit(limit).
		Offset(offset).
		Find(&seriesList).Error

	return seriesList, int(totalCount), err
}

func (d *DB) GetSeries(id uint) (*models.Series, error) {
	var s models.Series
	if err := d.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) GetSeriesByIMDB(imdbCode string) (*models.Series, error) {
	var s models.Series
	if err := d.Where("imdb_code = ?", imdbCode).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) CreateSeries(s *models.Series) error {
	now := time.Now()
	if s.DateAdded == "" {
		s.DateAdded = now.Format("2006-01-02 15:04:05")
	}
	if s.DateAddedUnix == 0 {
		s.DateAddedUnix = now.Unix()
	}
	return d.Create(s).Error
}

func (d *DB) UpdateSeries(s *models.Series) error {
	return d.Save(s).Error
}

func (d *DB) DeleteSeries(id uint) error {
	// Delete episode torrents for all episodes in this series
	d.Where("episode_id IN (?)",
		d.DB.Model(&models.Episode{}).Select("id").Where("series_id = ?", id),
	).Delete(&models.EpisodeTorrent{})

	// Delete episodes
	d.Where("series_id = ?", id).Delete(&models.Episode{})

	// Delete season packs
	d.Where("series_id = ?", id).Delete(&models.SeasonPack{})

	// Delete the series
	return d.Delete(&models.Series{}, id).Error
}

func (d *DB) GetEpisodes(seriesID uint, season int) ([]models.Episode, error) {
	query := d.Where("series_id = ?", seriesID)
	if season > 0 {
		query = query.Where("season_number = ?", season)
	}

	var episodes []models.Episode
	err := query.Order("season_number, episode_number").Find(&episodes).Error
	if err != nil {
		return nil, err
	}

	// Load torrents for each episode
	for i := range episodes {
		d.DB.Where("episode_id = ?", episodes[i].ID).Find(&episodes[i].Torrents)
	}

	return episodes, nil
}

func (d *DB) CreateEpisode(e *models.Episode) error {
	result := d.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "series_id"},
			{Name: "season_number"},
			{Name: "episode_number"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "summary", "air_date", "runtime", "still_image",
		}),
	}).Create(e)
	return result.Error
}

func (d *DB) GetEpisodeTorrents(episodeID uint) ([]models.EpisodeTorrent, error) {
	var torrents []models.EpisodeTorrent
	err := d.Where("episode_id = ?", episodeID).Find(&torrents).Error
	return torrents, err
}

func (d *DB) CreateEpisodeTorrent(t *models.EpisodeTorrent) error {
	return d.Create(t).Error
}

func (d *DB) GetSeasonPacks(seriesID uint) ([]models.SeasonPack, error) {
	var packs []models.SeasonPack
	err := d.Where("series_id = ?", seriesID).
		Order("season_number").
		Find(&packs).Error
	return packs, err
}

func (d *DB) GetSeasonPack(id uint) (*models.SeasonPack, error) {
	var p models.SeasonPack
	if err := d.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *DB) CreateSeasonPack(p *models.SeasonPack) error {
	// Check if hash already exists
	var count int64
	d.Model(&models.SeasonPack{}).Where("hash = ?", p.Hash).Count(&count)
	if count > 0 {
		return nil
	}
	return d.Create(p).Error
}
```

- [ ] **Step 2: Commit**

```bash
git add database/series.go
git commit -m "feat: rewrite series.go to GORM"
```

---

### Task 5: Rewrite database/analytics.go

**Files:**
- Rewrite: `database/analytics.go`

**Depends on:** Task 1

- [ ] **Step 1: Rewrite analytics.go with GORM**

```go
// database/analytics.go
package database

import (
	"time"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

func (d *DB) RecordView(contentType string, contentID uint, imdbCode string, deviceID string, duration int, completed bool, quality string) error {
	today := time.Now().Format("2006-01-02")

	// Upsert content view
	view := models.ContentView{
		ContentType:   contentType,
		ContentID:     contentID,
		ImdbCode:      imdbCode,
		DeviceID:      deviceID,
		ViewDate:      today,
		ViewCount:     1,
		WatchDuration: duration,
		Completed:     completed,
		Quality:       quality,
	}
	d.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "content_type"}, {Name: "content_id"}, {Name: "device_id"}, {Name: "view_date"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"view_count":     clause.Expr{SQL: "content_views.view_count + 1"},
			"watch_duration": clause.Expr{SQL: "content_views.watch_duration + ?", Vars: []interface{}{duration}},
			"quality":        quality,
		}),
	}).Create(&view)

	// Upsert daily stats
	completedInt := 0
	if completed {
		completedInt = 1
	}
	stat := models.ContentStatsDaily{
		ContentType:    contentType,
		ContentID:      contentID,
		StatDate:       today,
		ViewCount:      1,
		UniqueViewers:  1,
		TotalWatchTime: duration,
		Completions:    completedInt,
	}
	d.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "content_type"}, {Name: "content_id"}, {Name: "stat_date"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"view_count":       clause.Expr{SQL: "content_stats_daily.view_count + 1"},
			"total_watch_time": clause.Expr{SQL: "content_stats_daily.total_watch_time + ?", Vars: []interface{}{duration}},
			"completions":      clause.Expr{SQL: "content_stats_daily.completions + ?", Vars: []interface{}{completedInt}},
		}),
	}).Create(&stat)

	return nil
}

func (d *DB) GetTopMovies(days int, genre string, limit int) ([]models.Movie, error) {
	if limit <= 0 {
		limit = 10
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	query := d.Model(&models.Movie{}).
		Select("movies.*, COALESCE(SUM(s.view_count), 0) as download_count").
		Joins("LEFT JOIN content_stats_daily s ON s.content_type = 'movie' AND s.content_id = movies.id AND s.stat_date >= ?", startDate)

	if genre != "" {
		query = query.Where("movies.genres LIKE ?", "%"+genre+"%")
	}

	var movies []models.Movie
	err := query.
		Group("movies.id").
		Having("COALESCE(SUM(s.view_count), 0) > 0").
		Order("download_count DESC").
		Limit(limit).
		Find(&movies).Error
	if err != nil {
		return nil, err
	}

	for i := range movies {
		d.DB.Where("movie_id = ?", movies[i].ID).Find(&movies[i].Torrents)
	}

	return movies, nil
}

func (d *DB) GetTopSeries(days int, limit int) ([]models.Series, error) {
	if limit <= 0 {
		limit = 10
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	var series []models.Series
	err := d.Model(&models.Series{}).
		Select("series.*, COALESCE(SUM(cs.view_count), 0) as total_views").
		Joins("LEFT JOIN content_stats_daily cs ON cs.content_type = 'series' AND cs.content_id = series.id AND cs.stat_date >= ?", startDate).
		Group("series.id").
		Having("COALESCE(SUM(cs.view_count), 0) > 0").
		Order("total_views DESC").
		Limit(limit).
		Find(&series).Error

	return series, err
}

func (d *DB) StartStream(deviceID string, contentType string, contentID uint, imdbCode string, quality string) error {
	cid := contentID
	stream := models.ActiveStream{
		DeviceID:    deviceID,
		ContentType: contentType,
		ContentID:   &cid,
		ImdbCode:    imdbCode,
		Quality:     quality,
	}
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"content_type", "content_id", "imdb_code", "quality", "started_at", "last_heartbeat"}),
	}).Create(&stream).Error
}

func (d *DB) HeartbeatStream(deviceID string) error {
	return d.Model(&models.ActiveStream{}).
		Where("device_id = ?", deviceID).
		Update("last_heartbeat", time.Now()).Error
}

func (d *DB) EndStream(deviceID string) error {
	return d.Where("device_id = ?", deviceID).Delete(&models.ActiveStream{}).Error
}

func (d *DB) GetActiveStreamCount() int {
	var count int64
	cutoff := time.Now().Add(-2 * time.Minute)
	d.Model(&models.ActiveStream{}).Where("last_heartbeat > ?", cutoff).Count(&count)
	return int(count)
}

func (d *DB) CleanupStaleStreams() {
	cutoff := time.Now().Add(-5 * time.Minute)
	d.Where("last_heartbeat < ?", cutoff).Delete(&models.ActiveStream{})
}

func (d *DB) GetViewStats(days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	stats := make(map[string]interface{})

	var totalViews int64
	d.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Select("COALESCE(SUM(view_count), 0)").Scan(&totalViews)
	stats["total_views"] = totalViews

	var uniqueContent int64
	d.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Distinct("content_id", "content_type").Count(&uniqueContent)
	stats["unique_content"] = uniqueContent

	var totalWatchTime int64
	d.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Select("COALESCE(SUM(total_watch_time), 0)").Scan(&totalWatchTime)
	stats["total_watch_hours"] = totalWatchTime / 3600

	var completions int64
	d.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Select("COALESCE(SUM(completions), 0)").Scan(&completions)
	stats["completions"] = completions

	return stats, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add database/analytics.go
git commit -m "feat: rewrite analytics.go to GORM"
```

---

### Task 6: Rewrite database/channel.go

**Files:**
- Rewrite: `database/channel.go`

**Depends on:** Task 1

- [ ] **Step 1: Rewrite channel.go with GORM**

```go
// database/channel.go
package database

import (
	"fmt"
	"time"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

type ChannelFilter struct {
	Limit     int
	Page      int
	Country   string
	Category  string
	QueryTerm string
}

func (d *DB) ListChannels(filter ChannelFilter) ([]models.Channel, int, error) {
	if filter.Limit <= 0 || filter.Limit > 50000 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	query := d.Model(&models.Channel{})

	if filter.Country != "" {
		query = query.Where("country = ?", filter.Country)
	}
	if filter.Category != "" {
		query = query.Where("categories LIKE ?", "%"+filter.Category+"%")
	}
	if filter.QueryTerm != "" {
		query = query.Where("name LIKE ?", "%"+filter.QueryTerm+"%")
	}

	var totalCount int64
	query.Count(&totalCount)

	offset := (filter.Page - 1) * filter.Limit
	var channels []models.Channel
	err := query.Order("name ASC").Limit(filter.Limit).Offset(offset).Find(&channels).Error

	return channels, int(totalCount), err
}

func (d *DB) GetChannel(id string) (*models.Channel, error) {
	var c models.Channel
	if err := d.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (d *DB) ListChannelCountries() ([]models.ChannelCountry, error) {
	var countries []models.ChannelCountry
	err := d.Model(&models.ChannelCountry{}).
		Select("channel_countries.code, channel_countries.name, COALESCE(channel_countries.flag, '') as flag, COUNT(channels.id) as channel_count").
		Joins("LEFT JOIN channels ON channels.country = channel_countries.code").
		Group("channel_countries.code, channel_countries.name, channel_countries.flag").
		Having("COUNT(channels.id) > 0").
		Order("channel_countries.name").
		Find(&countries).Error
	return countries, err
}

func (d *DB) ListChannelCategories() ([]models.ChannelCategory, error) {
	var categories []models.ChannelCategory
	err := d.Order("name").Find(&categories).Error
	return categories, err
}

func (d *DB) GetChannelsByCountry(countryCode string, limit int) ([]models.Channel, error) {
	if limit <= 0 {
		limit = 50
	}
	var channels []models.Channel
	err := d.Where("country = ?", countryCode).Order("name").Limit(limit).Find(&channels).Error
	return channels, err
}

func (d *DB) UpsertChannel(ch *models.Channel) error {
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "country", "languages", "categories", "logo", "stream_url", "is_nsfw", "website", "updated_at"}),
	}).Create(ch).Error
}

func (d *DB) UpsertChannelCountry(c *models.ChannelCountry) error {
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "flag"}),
	}).Create(c).Error
}

func (d *DB) UpsertChannelCategory(c *models.ChannelCategory) error {
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name"}),
	}).Create(c).Error
}

func (d *DB) ClearChannels() error {
	return d.Where("1 = 1").Delete(&models.Channel{}).Error
}

func (d *DB) CountChannels() (int, error) {
	var count int64
	err := d.Model(&models.Channel{}).Count(&count).Error
	return int(count), err
}

func (d *DB) DeleteChannel(id string) error {
	return d.Where("id = ?", id).Delete(&models.Channel{}).Error
}

func (d *DB) UpsertEPG(epg *models.ChannelEPG) error {
	return d.Create(epg).Error
}

func (d *DB) GetEPG(channelID string) ([]models.ChannelEPG, error) {
	var epgs []models.ChannelEPG
	err := d.Where("channel_id = ? AND end_time >= ?", channelID, time.Now().Format("2006-01-02 15:04:05")).
		Order("start_time ASC").
		Limit(50).
		Find(&epgs).Error
	return epgs, err
}

func (d *DB) ClearEPG() error {
	return d.Where("1 = 1").Delete(&models.ChannelEPG{}).Error
}

func (d *DB) GetChannelStats() (map[string]int, error) {
	stats := make(map[string]int)

	var count int64
	d.Model(&models.Channel{}).Count(&count)
	stats["channels"] = int(count)

	d.Model(&models.Channel{}).Where("country != ''").Distinct("country").Count(&count)
	stats["countries"] = int(count)

	d.Model(&models.ChannelCategory{}).Count(&count)
	stats["categories"] = int(count)

	d.Model(&models.Channel{}).Where("stream_url != '' AND stream_url IS NOT NULL").Count(&count)
	stats["with_streams"] = int(count)

	d.Model(&models.ChannelBlocklist{}).Count(&count)
	stats["blocklisted"] = int(count)

	return stats, nil
}

func (d *DB) GetAllChannelsWithStreams() ([]models.Channel, error) {
	var channels []models.Channel
	err := d.Where("stream_url IS NOT NULL AND stream_url != ''").Find(&channels).Error
	return channels, err
}

func (d *DB) UpdateChannelStream(channelID, streamURL string) error {
	return d.Model(&models.Channel{}).Where("id = ?", channelID).
		Updates(map[string]interface{}{"stream_url": streamURL, "updated_at": time.Now()}).Error
}

// Blocklist methods

func (d *DB) AddToBlocklist(channelID, reason string) error {
	bl := models.ChannelBlocklist{ChannelID: channelID, Reason: reason}
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"reason", "blocked_at"}),
	}).Create(&bl).Error
}

func (d *DB) IsBlocklisted(channelID string) bool {
	var count int64
	d.Model(&models.ChannelBlocklist{}).Where("channel_id = ?", channelID).Count(&count)
	return count > 0
}

func (d *DB) GetBlocklistCount() int {
	var count int64
	d.Model(&models.ChannelBlocklist{}).Count(&count)
	return int(count)
}

func (d *DB) ClearBlocklist() error {
	return d.Where("1 = 1").Delete(&models.ChannelBlocklist{}).Error
}

func (d *DB) GetBlocklistedIDs() map[string]bool {
	result := make(map[string]bool)
	var ids []string
	d.Model(&models.ChannelBlocklist{}).Pluck("channel_id", &ids)
	for _, id := range ids {
		result[id] = true
	}
	return result
}

func (d *DB) GetChannelCountByCategory() ([]models.ChannelCategory, error) {
	cats, err := d.ListChannelCategories()
	if err != nil {
		return nil, err
	}

	for i, cat := range cats {
		var count int64
		d.Model(&models.Channel{}).Where("categories LIKE ?", fmt.Sprintf("%%%s%%", cat.ID)).Count(&count)
		cats[i].ChannelCount = int(count)
	}

	return cats, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add database/channel.go
git commit -m "feat: rewrite channel.go to GORM"
```

---

### Task 7: Rewrite database/home.go

**Files:**
- Rewrite: `database/home.go`

**Depends on:** Task 1

- [ ] **Step 1: Rewrite home.go with GORM (remove local HomeSection struct, use models.HomeSection)**

```go
// database/home.go
package database

import (
	"torrent-server/models"
)

func (d *DB) ListHomeSections(includeInactive bool) ([]models.HomeSection, error) {
	query := d.Model(&models.HomeSection{})
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	var sections []models.HomeSection
	err := query.Order("display_order ASC, id ASC").Find(&sections).Error
	return sections, err
}

func (d *DB) GetHomeSection(id uint) (*models.HomeSection, error) {
	var s models.HomeSection
	if err := d.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) CreateHomeSection(s *models.HomeSection) error {
	if s.DisplayType == "" {
		s.DisplayType = "carousel"
	}
	return d.Create(s).Error
}

func (d *DB) UpdateHomeSection(s *models.HomeSection) error {
	if s.DisplayType == "" {
		s.DisplayType = "carousel"
	}
	return d.Save(s).Error
}

func (d *DB) DeleteHomeSection(id uint) error {
	return d.Delete(&models.HomeSection{}, id).Error
}

func (d *DB) ReorderHomeSections(ids []uint) error {
	return d.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&models.HomeSection{}).Where("id = ?", id).Update("display_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
```

Note: Add `"gorm.io/gorm"` to imports for `gorm.DB` in Transaction callback.

- [ ] **Step 2: Commit**

```bash
git add database/home.go
git commit -m "feat: rewrite home.go to GORM"
```

---

### Task 8: Rewrite database/curated.go

**Files:**
- Rewrite: `database/curated.go`

**Depends on:** Task 1

- [ ] **Step 1: Rewrite curated.go with GORM**

```go
// database/curated.go
package database

import (
	"strings"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

func (d *DB) ListCuratedLists(includeInactive bool) ([]models.CuratedList, error) {
	query := d.Model(&models.CuratedList{})
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	var lists []models.CuratedList
	err := query.Order("display_order ASC, name ASC").Find(&lists).Error
	return lists, err
}

func (d *DB) GetCuratedListByID(id uint) (*models.CuratedList, error) {
	var l models.CuratedList
	if err := d.First(&l, id).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (d *DB) GetCuratedList(idOrSlug string) (*models.CuratedList, error) {
	var l models.CuratedList
	if err := d.Where("id = ? OR slug = ?", idOrSlug, idOrSlug).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (d *DB) GetCuratedListMovies(list *models.CuratedList) ([]models.Movie, error) {
	// Check hand-picked movies first
	var movieIDs []uint
	d.Model(&models.CuratedListMovie{}).
		Where("list_id = ?", list.ID).
		Order("display_order ASC").
		Pluck("movie_id", &movieIDs)

	if len(movieIDs) > 0 {
		var movies []models.Movie
		for _, id := range movieIDs {
			if m, err := d.GetMovie(id); err == nil {
				movies = append(movies, *m)
			}
		}
		return movies, nil
	}

	// Otherwise use filter-based selection
	filter := MovieFilter{
		Limit:         list.LimitCount,
		SortBy:        list.SortBy,
		OrderBy:       list.OrderBy,
		MinimumRating: list.MinimumRating,
		MinimumYear:   list.MinimumYear,
		MaximumYear:   list.MaximumYear,
		Genre:         list.Genre,
	}

	movies, _, err := d.ListMovies(filter)
	return movies, err
}

func (d *DB) CreateCuratedList(list *models.CuratedList) error {
	if list.Slug == "" {
		list.Slug = strings.ToLower(strings.ReplaceAll(list.Name, " ", "-"))
	}
	return d.Create(list).Error
}

func (d *DB) UpdateCuratedList(list *models.CuratedList) error {
	return d.Save(list).Error
}

func (d *DB) DeleteCuratedList(id uint) error {
	return d.Delete(&models.CuratedList{}, id).Error
}

func (d *DB) AddMovieToCuratedList(listID, movieID uint, order int) error {
	clm := models.CuratedListMovie{ListID: listID, MovieID: movieID, DisplayOrder: order}
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "list_id"}, {Name: "movie_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"display_order"}),
	}).Create(&clm).Error
}

func (d *DB) RemoveMovieFromCuratedList(listID, movieID uint) error {
	return d.Where("list_id = ? AND movie_id = ?", listID, movieID).Delete(&models.CuratedListMovie{}).Error
}
```

- [ ] **Step 2: Commit**

```bash
git add database/curated.go
git commit -m "feat: rewrite curated.go to GORM"
```

---

### Task 9: Rewrite database/config.go

**Files:**
- Rewrite: `database/config.go`

**Depends on:** Task 1

- [ ] **Step 1: Rewrite config.go with GORM**

```go
// database/config.go
package database

import (
	"gorm.io/gorm/clause"

	"torrent-server/models"
)

func (d *DB) ListServices() ([]models.ServiceConfig, error) {
	var services []models.ServiceConfig
	err := d.Order("display_order").Find(&services).Error
	return services, err
}

func (d *DB) UpdateService(s *models.ServiceConfig) error {
	return d.Save(s).Error
}

func (d *DB) CreateService(s *models.ServiceConfig) error {
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"label", "enabled", "icon", "display_order"}),
	}).Create(s).Error
}

func (d *DB) DeleteService(id string) error {
	return d.Where("id = ?", id).Delete(&models.ServiceConfig{}).Error
}
```

- [ ] **Step 2: Commit**

```bash
git add database/config.go
git commit -m "feat: rewrite config.go to GORM"
```

---

### Task 10: Rewrite database/subtitles.go

**Files:**
- Rewrite: `database/subtitles.go`

**Depends on:** Task 1

- [ ] **Step 1: Rewrite subtitles.go with GORM**

```go
// database/subtitles.go
package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

func (d *DB) GetSubtitlesByIMDB(imdbCode, language string) ([]models.StoredSubtitle, error) {
	return d.GetSubtitlesByIMDBEpisode(imdbCode, language, 0, 0)
}

func (d *DB) GetSubtitlesByIMDBEpisode(imdbCode, language string, season, episode int) ([]models.StoredSubtitle, error) {
	query := d.Model(&models.StoredSubtitle{}).Where("imdb_code = ?", imdbCode)

	if season > 0 {
		query = query.Where("season_number = ?", season)
	}
	if episode > 0 {
		query = query.Where("episode_number = ?", episode)
	}
	if language != "" {
		langs := strings.Split(language, ",")
		trimmed := make([]string, len(langs))
		for i, l := range langs {
			trimmed[i] = strings.TrimSpace(l)
		}
		query = query.Where("language IN ?", trimmed)
	}

	var subtitles []models.StoredSubtitle
	err := query.
		Select("id, imdb_code, language, language_name, release_name, hearing_impaired, source, season_number, episode_number, created_at").
		Order("season_number, episode_number, created_at DESC").
		Find(&subtitles).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query subtitles: %w", err)
	}

	return subtitles, nil
}

func (d *DB) GetSubtitleByID(id uint) (*models.StoredSubtitle, error) {
	var sub models.StoredSubtitle
	if err := d.First(&sub, id).Error; err != nil {
		return nil, fmt.Errorf("subtitle not found")
	}
	return &sub, nil
}

func (d *DB) CreateSubtitle(sub *models.StoredSubtitle) error {
	result := d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "imdb_code"}, {Name: "language"}, {Name: "release_name"}},
		DoNothing: true,
	}).Create(sub)
	if result.Error != nil {
		return fmt.Errorf("failed to create subtitle: %w", result.Error)
	}
	return nil
}

func (d *DB) UpdateSubtitlePath(id uint, vttPath string) error {
	return d.Model(&models.StoredSubtitle{}).Where("id = ?", id).
		Updates(map[string]interface{}{"vtt_path": vttPath, "vtt_content": ""}).Error
}

func (d *DB) GetSubtitlesWithContent() ([]models.StoredSubtitle, error) {
	var subs []models.StoredSubtitle
	err := d.Select("id, imdb_code, vtt_content").
		Where("vtt_content != '' AND (vtt_path = '' OR vtt_path IS NULL)").
		Find(&subs).Error
	return subs, err
}

func (d *DB) DeleteSubtitle(id uint) error {
	return d.Delete(&models.StoredSubtitle{}, id).Error
}

func (d *DB) DeleteSubtitlesByIMDB(imdbCode string) error {
	return d.Where("imdb_code = ?", imdbCode).Delete(&models.StoredSubtitle{}).Error
}

func (d *DB) CountSubtitlesByIMDB(imdbCode string) (int, error) {
	var count int64
	err := d.Model(&models.StoredSubtitle{}).Where("imdb_code = ?", imdbCode).Count(&count).Error
	return int(count), err
}

func (d *DB) CountSubtitlesByIMDBEpisode(imdbCode string, season, episode int) (int, error) {
	var count int64
	err := d.Model(&models.StoredSubtitle{}).
		Where("imdb_code = ? AND season_number = ? AND episode_number = ?", imdbCode, season, episode).
		Count(&count).Error
	return int(count), err
}
```

- [ ] **Step 2: Commit**

```bash
git add database/subtitles.go
git commit -m "feat: rewrite subtitles.go to GORM"
```

---

### Task 11: Wire up handlers, services, and main.go

**Files:**
- Modify: `main.go` (lines 92-96: DB init, line 115: migration, lines 129-167: handler constructors)
- Modify: All `handlers/*.go` (type `*database.DB` stays the same - no change needed since DB struct name is preserved)
- Modify: `handlers/home.go` (change `database.HomeSection` references to `models.HomeSection`)
- Modify: `services/sync.go` (type stays `*database.DB`)
- Modify: `services/subtitle.go` (type stays `*database.DB`)

**Depends on:** Tasks 1-10

- [ ] **Step 1: Update main.go DB initialization**

In `main.go`, change the DB init from:
```go
db, err := database.New(cfg.DatabasePath)
```
to:
```go
db, err := database.New(cfg.DatabasePath, cfg.DatabaseURL)
```

- [ ] **Step 2: Update handlers/home.go to use models.HomeSection**

Find all references to `database.HomeSection` in `handlers/home.go` and replace with `models.HomeSection`. Add `"torrent-server/models"` to imports if not already present.

- [ ] **Step 3: Update any handler that references database.HomeSection**

Search all handlers for `database.HomeSection` and replace with `models.HomeSection`.

- [ ] **Step 4: Remove unused imports**

Run through all modified files and clean up imports. Remove `"database/sql"` from any database file that no longer uses it. Remove `"encoding/json"` from database files that no longer do manual JSON parsing.

- [ ] **Step 5: Build and fix compilation errors**

```bash
go build ./...
```

Fix any remaining compilation errors. Common issues:
- Missing imports
- Type mismatches (HomeSection moved to models package)
- Method signature changes

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: wire up GORM database layer to handlers and services"
```

---

### Task 12: Deployment Configuration

**Files:**
- Modify: `basepod.yaml`

**Depends on:** Task 11

- [ ] **Step 1: Update basepod.yaml with DATABASE_URL**

```yaml
name: omnius-api
server: bp.common.al
port: 8080
domains:
  - api.omnius.lol
build:
  dockerfile: Dockerfile
  context: .
env:
  LICENSE_SERVER_URL: "https://omnius.stream"
  LICENSE_KEY: "OMNI-LIVE-EBME-AVUZ"
  ADMIN_PASSWORD: "demo"
  LICENSE_ADMIN_SECRET: "omnius-admin-2026"
  SERVER_DOMAIN: "api.omnius.lol"
  DATABASE_URL: "postgres://basepod:617d9005e7ee42ddfcfe3b60@basepod-postgres:5432/omnius?sslmode=disable"
volumes:
  - omnius-data:/app/data
```

Note: The volume is kept for subtitle files stored on disk, even though the DB is now in PostgreSQL.

- [ ] **Step 2: Create the omnius database on the server**

SSH into the server and create the database:
```bash
ssh base@common.al
# Then on the server:
docker exec -it basepod-postgres psql -U basepod -d app -c "CREATE DATABASE omnius;"
```

- [ ] **Step 3: Commit**

```bash
git add basepod.yaml
git commit -m "feat: add PostgreSQL DATABASE_URL to basepod deployment"
```

---

### Task 13: Build and Verify

**Depends on:** Task 12

- [ ] **Step 1: Full build**

```bash
go build -o torrent-server .
```

Expected: Clean build with no errors.

- [ ] **Step 2: Test locally with SQLite (no DATABASE_URL)**

```bash
./torrent-server
```

Expected: Server starts, logs "Using SQLite at ./data/omnius.db", creates tables via AutoMigrate, seeds default data.

- [ ] **Step 3: Test a few API endpoints**

```bash
curl http://localhost:8080/api/v2/list_movies.json?limit=1
curl http://localhost:8080/api/v2/home_sections.json
```

Expected: Valid JSON responses (empty data is fine for fresh DB).

- [ ] **Step 4: Stop server, clean up**

```bash
# Ctrl+C to stop
rm -rf data/  # Clean test data
```

- [ ] **Step 5: Final commit and push**

```bash
git add -A
git commit -m "feat: complete SQLite to GORM + PostgreSQL migration"
git push
```
