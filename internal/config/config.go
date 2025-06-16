package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config represents the complete application configuration
type Config struct {
	Server ServerConfig `json:"server"`
	LLM    LLMConfig    `json:"llm"`
	Cache  CacheConfig  `json:"cache"`
	Log    LogConfig    `json:"log"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port         int           `json:"port"`
	Host         string        `json:"host"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// LLMConfig contains LLM client configuration
type LLMConfig struct {
	ServerURL     string        `json:"server_url"`
	Timeout       time.Duration `json:"timeout"`
	RetryAttempts int           `json:"retry_attempts"`
	RetryDelay    time.Duration `json:"retry_delay"`
	MaxRetryDelay time.Duration `json:"max_retry_delay"`
}

// CacheConfig contains schema cache configuration
type CacheConfig struct {
	MaxSize int           `json:"max_size"`
	TTL     time.Duration `json:"ttl"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnvInt("PORT", 8081),
			Host:         getEnvString("HOST", ""),
			ReadTimeout:  getEnvDuration("READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvDuration("IDLE_TIMEOUT", 120*time.Second),
		},
		LLM: LLMConfig{
			ServerURL:     getEnvString("LLM_SERVER_URL", "http://localhost:8080"),
			Timeout:       getEnvDuration("LLM_TIMEOUT", 30*time.Second),
			RetryAttempts: getEnvInt("LLM_RETRY_ATTEMPTS", 3),
			RetryDelay:    getEnvDuration("LLM_RETRY_DELAY", 1*time.Second),
			MaxRetryDelay: getEnvDuration("LLM_MAX_RETRY_DELAY", 10*time.Second),
		},
		Cache: CacheConfig{
			MaxSize: getEnvInt("SCHEMA_CACHE_SIZE", 100),
			TTL:     getEnvDuration("SCHEMA_CACHE_TTL", 1*time.Hour),
		},
		Log: LogConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate ensures configuration values are valid
func (c *Config) Validate() error {
	// Server validation
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server read timeout must be positive, got %v", c.Server.ReadTimeout)
	}
	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server write timeout must be positive, got %v", c.Server.WriteTimeout)
	}
	if c.Server.IdleTimeout <= 0 {
		return fmt.Errorf("server idle timeout must be positive, got %v", c.Server.IdleTimeout)
	}

	// LLM validation
	if c.LLM.ServerURL == "" {
		return fmt.Errorf("LLM server URL cannot be empty")
	}
	if c.LLM.Timeout <= 0 {
		return fmt.Errorf("LLM timeout must be positive, got %v", c.LLM.Timeout)
	}
	if c.LLM.RetryAttempts < 0 {
		return fmt.Errorf("LLM retry attempts must be non-negative, got %d", c.LLM.RetryAttempts)
	}
	if c.LLM.RetryDelay <= 0 {
		return fmt.Errorf("LLM retry delay must be positive, got %v", c.LLM.RetryDelay)
	}
	if c.LLM.MaxRetryDelay < c.LLM.RetryDelay {
		return fmt.Errorf("LLM max retry delay must be >= retry delay, got %v < %v", c.LLM.MaxRetryDelay, c.LLM.RetryDelay)
	}

	// Cache validation
	if c.Cache.MaxSize <= 0 {
		return fmt.Errorf("cache max size must be positive, got %d", c.Cache.MaxSize)
	}
	if c.Cache.TTL <= 0 {
		return fmt.Errorf("cache TTL must be positive, got %v", c.Cache.TTL)
	}

	// Log validation
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLevels, strings.ToLower(c.Log.Level)) {
		return fmt.Errorf("log level must be one of %v, got %s", validLevels, c.Log.Level)
	}
	validFormats := []string{"json", "text"}
	if !contains(validFormats, strings.ToLower(c.Log.Format)) {
		return fmt.Errorf("log format must be one of %v, got %s", validFormats, c.Log.Format)
	}

	return nil
}

// Address returns the server address in host:port format
func (c *Config) Address() string {
	if c.Server.Host == "" {
		return fmt.Sprintf(":%d", c.Server.Port)
	}
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
