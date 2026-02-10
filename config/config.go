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
	ServerDomain      string

	// Paddle
	PaddleWebhookSecret  string
	PaddlePersonalPriceID string
	PaddleBusinessPriceID string

	// SMTP (optional)
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string
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
		ServerDomain:      getEnv("SERVER_DOMAIN", ""),

		PaddleWebhookSecret:   getEnv("PADDLE_WEBHOOK_SECRET", ""),
		PaddlePersonalPriceID: getEnv("PADDLE_PERSONAL_PRICE_ID", ""),
		PaddleBusinessPriceID: getEnv("PADDLE_BUSINESS_PRICE_ID", ""),

		SMTPHost: getEnv("SMTP_HOST", ""),
		SMTPPort: getEnv("SMTP_PORT", "587"),
		SMTPUser: getEnv("SMTP_USER", ""),
		SMTPPass: getEnv("SMTP_PASS", ""),
		SMTPFrom: getEnv("SMTP_FROM", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
