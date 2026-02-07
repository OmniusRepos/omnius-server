package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	d := &DB{db}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return d, nil
}

func (d *DB) migrate() error {
	schema := `
	-- Movies table (mirrors YTS structure)
	CREATE TABLE IF NOT EXISTS movies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		imdb_code TEXT UNIQUE,
		title TEXT NOT NULL,
		title_english TEXT,
		title_long TEXT,
		slug TEXT,
		year INTEGER,
		rating REAL DEFAULT 0,
		runtime INTEGER DEFAULT 0,
		genres TEXT,
		summary TEXT,
		description_full TEXT,
		synopsis TEXT,
		yt_trailer_code TEXT,
		language TEXT DEFAULT 'en',
		background_image TEXT,
		small_cover_image TEXT,
		medium_cover_image TEXT,
		large_cover_image TEXT,
		date_uploaded TEXT,
		date_uploaded_unix INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Torrents table (multiple per movie)
	CREATE TABLE IF NOT EXISTS torrents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		movie_id INTEGER NOT NULL,
		url TEXT,
		hash TEXT NOT NULL,
		quality TEXT,
		type TEXT DEFAULT 'web',
		video_codec TEXT,
		seeds INTEGER DEFAULT 0,
		peers INTEGER DEFAULT 0,
		size TEXT,
		size_bytes INTEGER,
		date_uploaded TEXT,
		date_uploaded_unix INTEGER,
		FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
	);

	-- Series table
	CREATE TABLE IF NOT EXISTS series (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		imdb_code TEXT UNIQUE,
		title TEXT NOT NULL,
		title_slug TEXT,
		year INTEGER,
		rating REAL DEFAULT 0,
		genres TEXT,
		summary TEXT,
		poster_image TEXT,
		background_image TEXT,
		total_seasons INTEGER DEFAULT 0,
		status TEXT DEFAULT 'ongoing',
		date_added TEXT,
		date_added_unix INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Episodes table
	CREATE TABLE IF NOT EXISTS episodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		series_id INTEGER NOT NULL,
		season INTEGER NOT NULL,
		episode INTEGER NOT NULL,
		title TEXT,
		overview TEXT,
		air_date TEXT,
		imdb_code TEXT,
		FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE,
		UNIQUE(series_id, season, episode)
	);

	-- Episode torrents
	CREATE TABLE IF NOT EXISTS episode_torrents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		episode_id INTEGER NOT NULL,
		hash TEXT NOT NULL,
		quality TEXT,
		seeds INTEGER DEFAULT 0,
		peers INTEGER DEFAULT 0,
		size TEXT,
		size_bytes INTEGER,
		source TEXT,
		FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE CASCADE
	);

	-- Season packs
	CREATE TABLE IF NOT EXISTS season_packs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		series_id INTEGER NOT NULL,
		season INTEGER NOT NULL,
		hash TEXT NOT NULL,
		quality TEXT,
		seeds INTEGER DEFAULT 0,
		peers INTEGER DEFAULT 0,
		size TEXT,
		size_bytes INTEGER,
		source TEXT,
		FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
	);

	-- Curated lists table
	CREATE TABLE IF NOT EXISTS curated_lists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		slug TEXT UNIQUE NOT NULL,
		description TEXT,
		sort_by TEXT DEFAULT 'rating',
		order_by TEXT DEFAULT 'desc',
		minimum_rating REAL DEFAULT 0,
		maximum_rating REAL DEFAULT 10,
		minimum_year INTEGER,
		maximum_year INTEGER,
		genre TEXT,
		limit_count INTEGER DEFAULT 50,
		is_active INTEGER DEFAULT 1,
		display_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Curated list movie associations (for hand-picked lists)
	CREATE TABLE IF NOT EXISTS curated_list_movies (
		list_id INTEGER NOT NULL,
		movie_id INTEGER NOT NULL,
		display_order INTEGER DEFAULT 0,
		FOREIGN KEY (list_id) REFERENCES curated_lists(id) ON DELETE CASCADE,
		FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE,
		PRIMARY KEY (list_id, movie_id)
	);

	-- Home sections table
	CREATE TABLE IF NOT EXISTS home_sections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		section_id TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		section_type TEXT NOT NULL DEFAULT 'query',
		display_type TEXT NOT NULL DEFAULT 'carousel',
		query_type TEXT,
		genre TEXT,
		curated_list_id INTEGER,
		sort_by TEXT DEFAULT 'rating',
		order_by TEXT DEFAULT 'desc',
		minimum_rating REAL DEFAULT 0,
		limit_count INTEGER DEFAULT 10,
		is_active INTEGER DEFAULT 1,
		display_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (curated_list_id) REFERENCES curated_lists(id) ON DELETE SET NULL
	);

	-- Analytics/views tracking
	CREATE TABLE IF NOT EXISTS content_views (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content_type TEXT NOT NULL,          -- movie, series, episode
		content_id INTEGER NOT NULL,
		imdb_code TEXT,
		device_id TEXT,                      -- anonymous device identifier
		view_date DATE NOT NULL,             -- date only for daily aggregation
		view_count INTEGER DEFAULT 1,        -- views per day per device
		watch_duration INTEGER DEFAULT 0,    -- seconds watched
		completed INTEGER DEFAULT 0,         -- 1 if watched >90%
		quality TEXT,                        -- 720p, 1080p, 2160p, etc.
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(content_type, content_id, device_id, view_date)
	);

	-- Daily aggregated stats (for faster Top 10 queries)
	CREATE TABLE IF NOT EXISTS content_stats_daily (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content_type TEXT NOT NULL,
		content_id INTEGER NOT NULL,
		stat_date DATE NOT NULL,
		view_count INTEGER DEFAULT 0,
		unique_viewers INTEGER DEFAULT 0,
		total_watch_time INTEGER DEFAULT 0,
		completions INTEGER DEFAULT 0,
		UNIQUE(content_type, content_id, stat_date)
	);

	-- Active streams tracking (for real-time "Active Now" count)
	CREATE TABLE IF NOT EXISTS active_streams (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id TEXT NOT NULL UNIQUE,
		content_type TEXT NOT NULL,
		content_id INTEGER,
		imdb_code TEXT,
		quality TEXT,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_heartbeat DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_movies_imdb ON movies(imdb_code);
	CREATE INDEX IF NOT EXISTS idx_movies_year ON movies(year);
	CREATE INDEX IF NOT EXISTS idx_movies_rating ON movies(rating);
	CREATE INDEX IF NOT EXISTS idx_torrents_movie ON torrents(movie_id);
	CREATE INDEX IF NOT EXISTS idx_torrents_hash ON torrents(hash);
	CREATE INDEX IF NOT EXISTS idx_series_imdb ON series(imdb_code);
	CREATE INDEX IF NOT EXISTS idx_episodes_series ON episodes(series_id);
	CREATE INDEX IF NOT EXISTS idx_home_sections_order ON home_sections(display_order);
	CREATE INDEX IF NOT EXISTS idx_content_views_date ON content_views(view_date);
	CREATE INDEX IF NOT EXISTS idx_content_views_content ON content_views(content_type, content_id);
	CREATE INDEX IF NOT EXISTS idx_content_stats_date ON content_stats_daily(stat_date);
	CREATE INDEX IF NOT EXISTS idx_content_stats_content ON content_stats_daily(content_type, content_id);
	`

	_, err := d.Exec(schema)
	if err != nil {
		return err
	}

	// Add new columns to existing tables (ignore errors if columns already exist)
	migrations := []string{
		// Movie rating columns
		"ALTER TABLE movies ADD COLUMN imdb_rating REAL",
		"ALTER TABLE movies ADD COLUMN rotten_tomatoes INTEGER",
		"ALTER TABLE movies ADD COLUMN metacritic INTEGER",
		"ALTER TABLE movies ADD COLUMN mpa_rating TEXT",
		"ALTER TABLE movies ADD COLUMN url TEXT",
		"ALTER TABLE movies ADD COLUMN background_image_original TEXT",
		"ALTER TABLE movies ADD COLUMN like_count INTEGER DEFAULT 0",
		"ALTER TABLE movies ADD COLUMN download_count INTEGER DEFAULT 0",
		"ALTER TABLE movies ADD COLUMN ratings_updated_at TEXT",
		"ALTER TABLE movies ADD COLUMN state TEXT DEFAULT 'ok'",
		"ALTER TABLE movies ADD COLUMN franchise TEXT",
		"ALTER TABLE movies ADD COLUMN imdb_votes TEXT",
		"ALTER TABLE movies ADD COLUMN content_type TEXT DEFAULT 'movie'",
		"ALTER TABLE movies ADD COLUMN provider TEXT",
		// Series columns
		"ALTER TABLE series ADD COLUMN tvdb_id INTEGER",
		"ALTER TABLE series ADD COLUMN end_year INTEGER",
		"ALTER TABLE series ADD COLUMN runtime INTEGER DEFAULT 0",
		"ALTER TABLE series ADD COLUMN network TEXT",
		"ALTER TABLE series ADD COLUMN total_episodes INTEGER DEFAULT 0",
		"ALTER TABLE series ADD COLUMN imdb_rating REAL",
		"ALTER TABLE series ADD COLUMN rotten_tomatoes INTEGER",
		"ALTER TABLE series ADD COLUMN franchise TEXT",
		// Episode columns
		"ALTER TABLE episodes ADD COLUMN summary TEXT",
		"ALTER TABLE episodes ADD COLUMN runtime INTEGER",
		"ALTER TABLE episodes ADD COLUMN still_image TEXT",
		// Episode torrent columns
		"ALTER TABLE episode_torrents ADD COLUMN series_id INTEGER",
		"ALTER TABLE episode_torrents ADD COLUMN season_number INTEGER",
		"ALTER TABLE episode_torrents ADD COLUMN episode_number INTEGER",
		"ALTER TABLE episode_torrents ADD COLUMN video_codec TEXT",
		"ALTER TABLE episode_torrents ADD COLUMN release_group TEXT",
		"ALTER TABLE episode_torrents ADD COLUMN date_uploaded TEXT",
		"ALTER TABLE episode_torrents ADD COLUMN date_uploaded_unix INTEGER",
		// Seasons table
		`CREATE TABLE IF NOT EXISTS seasons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series_id INTEGER NOT NULL,
			season_number INTEGER NOT NULL,
			episode_count INTEGER DEFAULT 0,
			air_date TEXT,
			poster_image TEXT,
			FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE,
			UNIQUE(series_id, season_number)
		)`,
	}

	for _, m := range migrations {
		d.Exec(m) // Ignore errors (column may already exist)
	}

	// Home sections migrations
	homeMigrations := []string{
		"ALTER TABLE home_sections ADD COLUMN display_type TEXT DEFAULT 'carousel'",
		"ALTER TABLE home_sections ADD COLUMN content_id INTEGER",
		"ALTER TABLE home_sections ADD COLUMN content_type TEXT",
		"ALTER TABLE home_sections ADD COLUMN section_type TEXT DEFAULT 'query'",
		"ALTER TABLE home_sections ADD COLUMN sort_by TEXT DEFAULT 'rating'",
		"ALTER TABLE home_sections ADD COLUMN order_by TEXT DEFAULT 'desc'",
		"ALTER TABLE home_sections ADD COLUMN minimum_rating REAL DEFAULT 0",
		"ALTER TABLE home_sections ADD COLUMN limit_count INTEGER DEFAULT 10",
	}
	for _, m := range homeMigrations {
		d.Exec(m)
	}

	// Analytics migrations
	analyticsMigrations := []string{
		"ALTER TABLE content_views ADD COLUMN quality TEXT",
	}
	for _, m := range analyticsMigrations {
		d.Exec(m)
	}

	// Rich movie data migrations (from IMDB API)
	richMovieMigrations := []string{
		"ALTER TABLE movies ADD COLUMN director TEXT",
		"ALTER TABLE movies ADD COLUMN writers TEXT",        // JSON array
		"ALTER TABLE movies ADD COLUMN cast_json TEXT",      // JSON array with full cast info
		"ALTER TABLE movies ADD COLUMN budget TEXT",
		"ALTER TABLE movies ADD COLUMN box_office_gross TEXT",
		"ALTER TABLE movies ADD COLUMN country TEXT",
		"ALTER TABLE movies ADD COLUMN awards TEXT",
		"ALTER TABLE movies ADD COLUMN all_images TEXT",     // JSON array of image URLs
	}
	for _, m := range richMovieMigrations {
		d.Exec(m)
	}

	// Coming soon status migrations
	comingSoonMigrations := []string{
		"ALTER TABLE movies ADD COLUMN status TEXT DEFAULT 'available'",  // 'available' or 'coming_soon'
		"ALTER TABLE movies ADD COLUMN release_date TEXT",                 // YYYY-MM-DD format
	}
	for _, m := range comingSoonMigrations {
		d.Exec(m)
	}

	// Service config table
	serviceConfigMigrations := []string{
		`CREATE TABLE IF NOT EXISTS service_config (
			id TEXT PRIMARY KEY,
			label TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			icon TEXT,
			display_order INTEGER DEFAULT 0
		)`,
	}
	for _, m := range serviceConfigMigrations {
		d.Exec(m)
	}

	// Seed default services if empty
	var serviceCount int
	d.QueryRow("SELECT COUNT(*) FROM service_config").Scan(&serviceCount)
	if serviceCount == 0 {
		defaultServices := []string{
			`INSERT INTO service_config (id, label, enabled, icon, display_order) VALUES ('movies', 'Movies', 1, 'movie', 1)`,
			`INSERT INTO service_config (id, label, enabled, icon, display_order) VALUES ('series', 'TV Shows', 1, 'tv', 2)`,
			`INSERT INTO service_config (id, label, enabled, icon, display_order) VALUES ('channels', 'Live TV', 0, 'live', 3)`,
		}
		for _, s := range defaultServices {
			d.Exec(s)
		}
	}

	// Channels tables (IPTV)
	channelMigrations := []string{
		`CREATE TABLE IF NOT EXISTS channels (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			country TEXT,
			languages TEXT,
			categories TEXT,
			logo TEXT,
			stream_url TEXT,
			is_nsfw INTEGER DEFAULT 0,
			website TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		"CREATE INDEX IF NOT EXISTS idx_channels_country ON channels(country)",
		"CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name)",
		`CREATE TABLE IF NOT EXISTS channel_countries (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			flag TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS channel_categories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS channel_epg (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			start_time DATETIME NOT NULL,
			end_time DATETIME NOT NULL,
			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
		)`,
		"CREATE INDEX IF NOT EXISTS idx_epg_channel ON channel_epg(channel_id)",
		"CREATE INDEX IF NOT EXISTS idx_epg_time ON channel_epg(start_time, end_time)",
	}
	for _, m := range channelMigrations {
		d.Exec(m)
	}

	// Channel column additions (for existing tables)
	channelColumnMigrations := []string{
		"ALTER TABLE channels ADD COLUMN is_nsfw INTEGER DEFAULT 0",
		"ALTER TABLE channels ADD COLUMN website TEXT",
	}
	for _, m := range channelColumnMigrations {
		d.Exec(m)
	}

	// Subtitles table
	subtitleMigrations := []string{
		`CREATE TABLE IF NOT EXISTS subtitles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			imdb_code TEXT NOT NULL,
			language TEXT NOT NULL,
			language_name TEXT,
			release_name TEXT,
			hearing_impaired INTEGER DEFAULT 0,
			source TEXT,
			vtt_content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(imdb_code, language, release_name)
		)`,
		"CREATE INDEX IF NOT EXISTS idx_subtitles_imdb ON subtitles(imdb_code)",
		"CREATE INDEX IF NOT EXISTS idx_subtitles_imdb_lang ON subtitles(imdb_code, language)",
	}
	for _, m := range subtitleMigrations {
		d.Exec(m)
	}

	return nil
}
