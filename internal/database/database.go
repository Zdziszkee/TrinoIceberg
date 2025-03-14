package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/trinodb/trino-go-client/trino"
	_ "github.com/trinodb/trino-go-client/trino" // Register Trino driver
)

// Config holds configuration for a Trino database connection
type Config struct {
	ServerURI       string        `koanf:"server_uri"`
	Catalog         string        `koanf:"catalog"`
	Schema          string        `koanf:"schema"`
	TableName       string        `koanf:"table_name"`
	MaxOpenConns    int           `koanf:"max_open_conns"`
	MaxIdleConns    int           `koanf:"max_idle_conns"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
}

// Database provides a Trino database connection
type Database struct {
	DB     *sql.DB
	Config Config
}

// New initializes a Trino database connection and executes schema
func New(config Config) (*Database, error) {
	// Build DSN using trino.Config
	trinoConfig := trino.Config{
		ServerURI: config.ServerURI, // e.g., "http://test:password@localhost:8080"
		Catalog:   config.Catalog,
		Schema:    config.Schema,
	}
	dsn, err := trinoConfig.FormatDSN()

	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open Trino connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping Trino: %w", err)
	}

	database := &Database{DB: db, Config: config}

	// Execute schema on startup
	if err := database.ExecuteSchema("schema.sql"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	return database, nil
}

// ExecuteSchema loads and executes the schema.sql file
func (db *Database) ExecuteSchema(filePath string) error {
	fmt.Println("Executing schema from:", filePath)

	schemaSQL, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	queries := strings.Split(string(schemaSQL), ";")
	ctx := context.Background()

	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		fmt.Println("Executing query:", query)
		_, err := db.DB.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	fmt.Println("Schema successfully executed!")
	return nil
}
