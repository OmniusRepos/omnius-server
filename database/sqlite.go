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

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_movies_imdb ON movies(imdb_code);
	CREATE INDEX IF NOT EXISTS idx_movies_year ON movies(year);
	CREATE INDEX IF NOT EXISTS idx_movies_rating ON movies(rating);
	CREATE INDEX IF NOT EXISTS idx_torrents_movie ON torrents(movie_id);
	CREATE INDEX IF NOT EXISTS idx_torrents_hash ON torrents(hash);
	CREATE INDEX IF NOT EXISTS idx_series_imdb ON series(imdb_code);
	CREATE INDEX IF NOT EXISTS idx_episodes_series ON episodes(series_id);
	`

	_, err := d.Exec(schema)
	return err
}
