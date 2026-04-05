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
	var serviceCount int64
	d.Model(&models.ServiceConfig{}).Count(&serviceCount)
	if serviceCount == 0 {
		d.Create(&models.ServiceConfig{ID: "movies", Label: "Movies", Enabled: true, Icon: "movie", DisplayOrder: 1})
		d.Create(&models.ServiceConfig{ID: "series", Label: "TV Shows", Enabled: true, Icon: "tv", DisplayOrder: 2})
		d.Create(&models.ServiceConfig{ID: "channels", Label: "Live TV", Enabled: false, Icon: "live", DisplayOrder: 3})
	}

	var top10Count int64
	d.Model(&models.HomeSection{}).Where("display_type = ?", "top10").Count(&top10Count)
	if top10Count == 0 {
		d.Where("1 = 1").Delete(&models.HomeSection{})
		heroContentID := uint(243)
		sections := []models.HomeSection{
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
		}
		for i := range sections {
			d.Create(&sections[i])
		}
	}
}
