package admin

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr             string
	AdminToken       string
	DataPath         string
	DefaultCoinGrant int64
	Clock            func() time.Time
	Emby             EmbyConfig
}

type EmbyConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

func LoadConfigFromEnv() Config {
	cfg := Config{
		Addr:             envOrDefault("EDEN_ADMIN_ADDR", ":8080"),
		AdminToken:       os.Getenv("EDEN_ADMIN_TOKEN"),
		DataPath:         envOrDefault("EDEN_ADMIN_DATA", "data/eden-admin.json"),
		DefaultCoinGrant: parseInt64Env("EDEN_DEFAULT_COINS", 0),
		Clock:            time.Now,
		Emby: EmbyConfig{
			BaseURL: os.Getenv("EMBY_BASE_URL"),
			APIKey:  os.Getenv("EMBY_API_KEY"),
			Timeout: 10 * time.Second,
		},
	}
	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseInt64Env(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
