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

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_movies_imdb ON movies(imdb_code);
	CREATE INDEX IF NOT EXISTS idx_movies_year ON movies(year);
	CREATE INDEX IF NOT EXISTS idx_movies_rating ON movies(rating);
	CREATE INDEX IF NOT EXISTS idx_torrents_movie ON torrents(movie_id);
	CREATE INDEX IF NOT EXISTS idx_torrents_hash ON torrents(hash);
	CREATE INDEX IF NOT EXISTS idx_series_imdb ON series(imdb_code);
	CREATE INDEX IF NOT EXISTS idx_episodes_series ON episodes(series_id);
	CREATE INDEX IF NOT EXISTS idx_home_sections_order ON home_sections(display_order);
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

	return nil
}
