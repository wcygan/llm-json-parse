package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("default_configuration", func(t *testing.T) {
		// Clear environment
		clearEnv()

		config, err := LoadConfig()
		require.NoError(t, err)

		// Verify defaults
		assert.Equal(t, 8081, config.Server.Port)
		assert.Equal(t, "", config.Server.Host)
		assert.Equal(t, 30*time.Second, config.Server.ReadTimeout)
		assert.Equal(t, 30*time.Second, config.Server.WriteTimeout)
		assert.Equal(t, 120*time.Second, config.Server.IdleTimeout)

		assert.Equal(t, "http://localhost:8080", config.LLM.ServerURL)
		assert.Equal(t, 30*time.Second, config.LLM.Timeout)
		assert.Equal(t, 3, config.LLM.RetryAttempts)
		assert.Equal(t, 1*time.Second, config.LLM.RetryDelay)
		assert.Equal(t, 10*time.Second, config.LLM.MaxRetryDelay)

		assert.Equal(t, 100, config.Cache.MaxSize)
		assert.Equal(t, 1*time.Hour, config.Cache.TTL)

		assert.Equal(t, "info", config.Log.Level)
		assert.Equal(t, "json", config.Log.Format)
	})

	t.Run("environment_overrides", func(t *testing.T) {
		clearEnv()

		// Set environment variables
		os.Setenv("PORT", "9090")
		os.Setenv("HOST", "0.0.0.0")
		os.Setenv("LLM_SERVER_URL", "http://llm.example.com:8000")
		os.Setenv("LLM_TIMEOUT", "45s")
		os.Setenv("SCHEMA_CACHE_SIZE", "500")
		os.Setenv("LOG_LEVEL", "debug")
		defer clearEnv()

		config, err := LoadConfig()
		require.NoError(t, err)

		assert.Equal(t, 9090, config.Server.Port)
		assert.Equal(t, "0.0.0.0", config.Server.Host)
		assert.Equal(t, "http://llm.example.com:8000", config.LLM.ServerURL)
		assert.Equal(t, 45*time.Second, config.LLM.Timeout)
		assert.Equal(t, 500, config.Cache.MaxSize)
		assert.Equal(t, "debug", config.Log.Level)
	})

	t.Run("invalid_port", func(t *testing.T) {
		clearEnv()
		os.Setenv("PORT", "99999")
		defer clearEnv()

		_, err := LoadConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server port must be between 1 and 65535")
	})

	t.Run("invalid_log_level", func(t *testing.T) {
		clearEnv()
		os.Setenv("LOG_LEVEL", "invalid")
		defer clearEnv()

		_, err := LoadConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "log level must be one of")
	})

	t.Run("invalid_duration", func(t *testing.T) {
		clearEnv()

		// Test that invalid duration falls back to default
		os.Setenv("LLM_TIMEOUT", "invalid")
		defer clearEnv()

		config, err := LoadConfig()
		require.NoError(t, err)

		// Should use default timeout when parsing fails
		assert.Equal(t, 30*time.Second, config.LLM.Timeout)
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("valid_config", func(t *testing.T) {
		config := &Config{
			Server: ServerConfig{
				Port:         8080,
				Host:         "localhost",
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			LLM: LLMConfig{
				ServerURL:     "http://localhost:8080",
				Timeout:       30 * time.Second,
				RetryAttempts: 3,
				RetryDelay:    1 * time.Second,
				MaxRetryDelay: 10 * time.Second,
			},
			Cache: CacheConfig{
				MaxSize: 100,
				TTL:     1 * time.Hour,
			},
			Log: LogConfig{
				Level:  "info",
				Format: "json",
			},
		}

		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid_server_port", func(t *testing.T) {
		config := createValidConfig()
		config.Server.Port = 0

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server port must be between 1 and 65535")
	})

	t.Run("invalid_timeouts", func(t *testing.T) {
		config := createValidConfig()
		config.Server.ReadTimeout = 0

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server read timeout must be positive")
	})

	t.Run("empty_llm_url", func(t *testing.T) {
		config := createValidConfig()
		config.LLM.ServerURL = ""

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LLM server URL cannot be empty")
	})

	t.Run("invalid_retry_config", func(t *testing.T) {
		config := createValidConfig()
		config.LLM.MaxRetryDelay = 500 * time.Millisecond
		config.LLM.RetryDelay = 1 * time.Second

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LLM max retry delay must be >= retry delay")
	})

	t.Run("negative_cache_size", func(t *testing.T) {
		config := createValidConfig()
		config.Cache.MaxSize = -1

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache max size must be positive")
	})

	t.Run("invalid_log_format", func(t *testing.T) {
		config := createValidConfig()
		config.Log.Format = "xml"

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "log format must be one of")
	})
}

func TestConfigAddress(t *testing.T) {
	t.Run("with_host", func(t *testing.T) {
		config := &Config{
			Server: ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
		}

		assert.Equal(t, "localhost:8080", config.Address())
	})

	t.Run("without_host", func(t *testing.T) {
		config := &Config{
			Server: ServerConfig{
				Host: "",
				Port: 8080,
			},
		}

		assert.Equal(t, ":8080", config.Address())
	})
}

func TestEnvHelpers(t *testing.T) {
	t.Run("getEnvString", func(t *testing.T) {
		clearEnv()

		// Test default
		assert.Equal(t, "default", getEnvString("TEST_STRING", "default"))

		// Test override
		os.Setenv("TEST_STRING", "custom")
		assert.Equal(t, "custom", getEnvString("TEST_STRING", "default"))

		clearEnv()
	})

	t.Run("getEnvInt", func(t *testing.T) {
		clearEnv()

		// Test default
		assert.Equal(t, 42, getEnvInt("TEST_INT", 42))

		// Test valid override
		os.Setenv("TEST_INT", "100")
		assert.Equal(t, 100, getEnvInt("TEST_INT", 42))

		// Test invalid override (should use default)
		os.Setenv("TEST_INT", "invalid")
		assert.Equal(t, 42, getEnvInt("TEST_INT", 42))

		clearEnv()
	})

	t.Run("getEnvDuration", func(t *testing.T) {
		clearEnv()

		defaultDur := 30 * time.Second

		// Test default
		assert.Equal(t, defaultDur, getEnvDuration("TEST_DURATION", defaultDur))

		// Test valid override
		os.Setenv("TEST_DURATION", "1m")
		assert.Equal(t, 1*time.Minute, getEnvDuration("TEST_DURATION", defaultDur))

		// Test invalid override (should use default)
		os.Setenv("TEST_DURATION", "invalid")
		assert.Equal(t, defaultDur, getEnvDuration("TEST_DURATION", defaultDur))

		clearEnv()
	})
}

// Helper functions

func clearEnv() {
	vars := []string{
		"PORT", "HOST", "READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT",
		"LLM_SERVER_URL", "LLM_TIMEOUT", "LLM_RETRY_ATTEMPTS", "LLM_RETRY_DELAY", "LLM_MAX_RETRY_DELAY",
		"SCHEMA_CACHE_SIZE", "SCHEMA_CACHE_TTL",
		"LOG_LEVEL", "LOG_FORMAT",
		"TEST_STRING", "TEST_INT", "TEST_DURATION",
	}

	for _, v := range vars {
		os.Unsetenv(v)
	}
}

func createValidConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			Host:         "localhost",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		LLM: LLMConfig{
			ServerURL:     "http://localhost:8080",
			Timeout:       30 * time.Second,
			RetryAttempts: 3,
			RetryDelay:    1 * time.Second,
			MaxRetryDelay: 10 * time.Second,
		},
		Cache: CacheConfig{
			MaxSize: 100,
			TTL:     1 * time.Hour,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}
