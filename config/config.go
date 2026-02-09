package config

import (
	"os"
)

type Config struct {
	Port              string
	AdminPassword     string
	DatabasePath      string
	DownloadDir       string
	OmdbAPIKey        string
	LicenseKey        string
	LicenseServerURL  string
	LicenseServerMode bool
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8080"),
		AdminPassword:     getEnv("ADMIN_PASSWORD", "admin"),
		DatabasePath:      getEnv("DATABASE_PATH", "./data/omnius.db"),
		DownloadDir:       getEnv("DOWNLOAD_DIR", "./data/downloads"),
		OmdbAPIKey:        getEnv("OMDB_API_KEY", ""),
		LicenseKey:        getEnv("LICENSE_KEY", ""),
		LicenseServerURL:  getEnv("LICENSE_SERVER_URL", "https://license.omnius.lol"),
		LicenseServerMode: getEnv("LICENSE_SERVER_MODE", "false") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
