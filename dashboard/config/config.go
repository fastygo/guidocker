package config

import (
	"os"
	"path/filepath"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Server ServerConfig
	Data   DataConfig
	Auth   AuthConfig
	Stacks StacksConfig
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

// AuthConfig holds HTTP Basic Auth credentials.
type AuthConfig struct {
	AdminUser string
	AdminPass string
	Disabled  bool
}

// StacksConfig holds storage settings for managed stacks.
type StacksConfig struct {
	Dir    string
	DBFile string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	stacksDir := getEnv("STACKS_DIR", "/opt/stacks")

	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnvAsIntAny([]string{"PAAS_PORT", "SERVER_PORT"}, 3000),
		},
		Data: DataConfig{
			DashboardFile: getEnv("DASHBOARD_DATA_FILE", "data/dashboard.json"),
		},
		Auth: AuthConfig{
			AdminUser: getEnv("PAAS_ADMIN_USER", "admin"),
			AdminPass: getEnv("PAAS_ADMIN_PASS", "admin@123"),
			Disabled:  getEnvAsBoolAny([]string{"DASHBOARD_AUTH_DISABLED", "PAAS_AUTH_DISABLED"}, false),
		},
		Stacks: StacksConfig{
			Dir:    stacksDir,
			DBFile: getEnv("BOLT_DB_FILE", filepath.Join(stacksDir, ".paas.db")),
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

// getEnvAsIntAny gets the first available environment variable as integer.
func getEnvAsIntAny(keys []string, fallback int) int {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			if intVal, err := strconv.Atoi(value); err == nil {
				return intVal
			}
		}
	}

	return fallback
}

func getEnvAsBoolAny(keys []string, fallback bool) bool {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			if boolVal, err := strconv.ParseBool(value); err == nil {
				return boolVal
			}
		}
	}

	return fallback
}
