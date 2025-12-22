package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Server ServerConfig
	Data   DataConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host string
	Port int
}

// DataConfig holds data-related configuration
type DataConfig struct {
	DashboardFile string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnvAsInt("SERVER_PORT", 3000),
		},
		Data: DataConfig{
			DashboardFile: getEnv("DASHBOARD_DATA_FILE", "data/dashboard.json"),
		},
	}
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return c.Server.Host + ":" + strconv.Itoa(c.Server.Port)
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvAsInt gets an environment variable as integer with a fallback value
func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
