package config

import (
	// (imports remain the same)
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/zdziszkee/swift-codes/internal/database"
)

type Config struct {
	Database database.Config `koanf:"database"`
	AppName  string          `koanf:"app_name"`
	Log      struct {
		Level  string `koanf:"level"`
		Format string `koanf:"format"`
	} `koanf:"log"`
	Data struct {
		SwiftCodesFile string `koanf:"swift_codes_file"`
		AutoLoad       bool   `koanf:"auto_load"`
	} `koanf:"data"`
}

// DefaultConfig returns the default configuration for swift-codes
func DefaultConfig() *Config {
	cfg := &Config{
		AppName: "swift-codes",
		Log: struct {
			Level  string `koanf:"level"`
			Format string `koanf:"format"`
		}{
			Level:  "info",
			Format: "text",
		},
		Database: database.Config{
			ServerURI:       "http://test:password@trino:8080",
			Catalog:         "swift_catalog",
			Schema:          "default_schema",
			TableName:       "swift_banks",
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 1 * time.Hour,
		},
		Data: struct {
			SwiftCodesFile string `koanf:"swift_codes_file"`
			AutoLoad       bool   `koanf:"auto_load"`
		}{
			SwiftCodesFile: "/app/swift_codes.csv",
			AutoLoad:       true,
		},
	}
	return cfg
}

// Load loads the configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	var k = koanf.New(".")

	// Load default values.
	defaultConfig := DefaultConfig()
	if err := k.Load(structs.Provider(defaultConfig, "koanf"), nil); err != nil {
		return nil, fmt.Errorf("error loading default config: %w", err)
	}

	// Load from config file if specified.
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
				return nil, fmt.Errorf("error loading TOML config file: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("error checking config file: %w", err)
		}
	} else {
		commonPaths := []string{
			"./config.toml",
			"./config/config.toml",
			"/etc/swift-codes/config.toml",
		}
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				if err := k.Load(file.Provider(path), toml.Parser()); err != nil {
					return nil, fmt.Errorf("error loading TOML config file from %s: %w", path, err)
				}
				break
			}
		}
	}

	// New callback: split the env variable by double underscores, lowercase each part, and join with "."
	callback := func(s string) string {
		// Remove the prefix (e.g. "APP_")
		s = strings.TrimPrefix(s, "APP_")
		// Split on double underscore to separate nested keys.
		parts := strings.Split(s, "__")
		for i, part := range parts {
			parts[i] = strings.ToLower(part)
		}
		return strings.Join(parts, ".")
	}
	if err := k.Load(env.Provider("APP_", ".", callback), nil); err != nil {
		return nil, fmt.Errorf("error loading environment variables: %w", err)
	}

	// Unmarshal the config into our Config struct.
	var config Config
	if err := k.Unmarshal("", &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// Validate the config.
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return &config, nil
}

// validateConfig checks required fields.
func validateConfig(config *Config) error {
	// Database config validations.
	if config.Database.ServerURI == "" {
		return errors.New("database server_uri cannot be empty")
	}
	if !strings.HasPrefix(config.Database.ServerURI, "http://") && !strings.HasPrefix(config.Database.ServerURI, "https://") {
		return fmt.Errorf("database server_uri must start with 'http://' or 'https://', got '%s'", config.Database.ServerURI)
	}
	if config.Database.Catalog == "" {
		return errors.New("database catalog cannot be empty")
	}
	if config.Database.Schema == "" {
		return errors.New("database schema cannot be empty")
	}
	// Connection pool validations.
	if config.Database.MaxOpenConns < 0 {
		return errors.New("max open connections cannot be negative")
	}
	if config.Database.MaxIdleConns < 0 {
		return errors.New("max idle connections cannot be negative")
	}
	if config.Database.ConnMaxLifetime < 0 {
		return errors.New("connection max lifetime cannot be negative")
	}

	// Log config validations.
	if config.Log.Level == "" {
		return errors.New("log level cannot be empty")
	}
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}
	if !validLogLevels[strings.ToLower(config.Log.Level)] {
		return errors.New("invalid log level: must be one of debug, info, warn, error, fatal")
	}
	validLogFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validLogFormats[strings.ToLower(config.Log.Format)] {
		return errors.New("invalid log format: must be text or json")
	}

	// Data config validations.
	if config.Data.SwiftCodesFile == "" {
		return errors.New("data.swift_codes_file cannot be empty")
	}

	return nil
}
