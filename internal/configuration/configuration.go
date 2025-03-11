package config

import (
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
	// Other configuration sections as needed
	AppName string `koanf:"app_name"`
	Log     struct {
		Level  string `koanf:"level"`
		Format string `koanf:"format"`
	} `koanf:"log"`
	Data struct {
		SwiftCodesFile string `koanf:"swift_codes_file"`
		AutoLoad       bool   `koanf:"auto_load"`
	} `koanf:"data"`
}

// DefaultConfig returns the default configuration for Trino
// config/config.go
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
			Type:            "trino",
			Host:            "trino",
			Port:            8080,
			Catalog:         "swift_catalog",
			Schema:          "default_schema",
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

	// Load default values
	defaultConfig := DefaultConfig()
	if err := k.Load(structs.Provider(defaultConfig, "koanf"), nil); err != nil {
		return nil, fmt.Errorf("error loading default config: %w", err)
	}

	// Load from config file if specified
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
				return nil, fmt.Errorf("error loading TOML config file: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			// Return the error if it's something other than "file not found"
			return nil, fmt.Errorf("error checking config file: %w", err)
		}
	} else {
		// If no specific path provided, try to load from common locations
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
				// Break on first found config file
				break
			}
		}
	}

	// Load environment variables with APP_ prefix
	// e.g., APP_DATABASE_HOST will override database.host
	callback := func(s string) string {
		// Convert APP_DATABASE_HOST to database.host
		s = strings.TrimPrefix(s, "APP_")
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "_", ".")
		return s
	}
	if err := k.Load(env.Provider("APP_", ".", callback), nil); err != nil {
		return nil, fmt.Errorf("error loading environment variables: %w", err)
	}

	// Unmarshal the config into our Config struct
	var config Config
	if err := k.Unmarshal("", &config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// Validate the config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return &config, nil
}

// validateConfig checks that required fields are present and valid
func validateConfig(config *Config) error {
	// Validate database config
	if config.Database.Type != "trino" {
		return fmt.Errorf("database type must be 'trino', got '%s'", config.Database.Type)
	}
	if config.Database.Host == "" {
		return errors.New("database host cannot be empty")
	}
	if config.Database.Port <= 0 {
		return errors.New("database port must be a positive number")
	}
	if config.Database.Catalog == "" {
		return errors.New("database catalog cannot be empty")
	}
	if config.Database.Schema == "" {
		return errors.New("database schema cannot be empty")
	}

	// Validate connection pool settings
	if config.Database.MaxOpenConns < 0 {
		return errors.New("max open connections cannot be negative")
	}
	if config.Database.MaxIdleConns < 0 {
		return errors.New("max idle connections cannot be negative")
	}
	if config.Database.ConnMaxLifetime < 0 {
		return errors.New("connection max lifetime cannot be negative")
	}

	// Validate log config
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

	return nil
}
