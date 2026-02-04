package config

import (
	"os"
)

type Config struct {
	Port          string
	AdminPassword string
	DatabasePath  string
	DownloadDir   string
	OmdbAPIKey    string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin"),
		DatabasePath:  getEnv("DATABASE_PATH", "./torrents.db"),
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
