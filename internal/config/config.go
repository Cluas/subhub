package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Port            string
	DBPath          string
	CacheTTL        time.Duration
	CacheMaxEntries int
	APIToken        string
	CORSOrigins     string
	LogLevel        string
	BaseURL         string
}

// Load reads configuration from environment variables, using defaults when unset.
func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "9000"),
		DBPath:          getEnv("DB_PATH", "subhub.db"),
		CacheTTL:        getDurationEnv("CACHE_TTL_SECONDS", 5*time.Minute),
		CacheMaxEntries: getIntEnv("CACHE_MAX_ENTRIES", 1000),
		APIToken:        getEnv("API_TOKEN", ""),
		CORSOrigins:     getEnv("CORS_ORIGINS", "*"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		BaseURL:         getEnv("BASE_URL", "http://localhost:"+getEnv("PORT", "9000")),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	secs, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return time.Duration(secs) * time.Second
}

func getIntEnv(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
