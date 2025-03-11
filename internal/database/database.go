package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/trinodb/trino-go-client/trino" // Trino driver
)

// Config holds configuration for a Trino database connection
type Config struct {
	Type            string        `koanf:"type"`
	Host            string        `koanf:"host"`
	Port            int           `koanf:"port"`
	Catalog         string        `koanf:"catalog"`
	Schema          string        `koanf:"schema"`
	MaxOpenConns    int           `koanf:"max_open_conns"`
	MaxIdleConns    int           `koanf:"max_idle_conns"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`
}

// Database provides a Trino database connection
type Database struct {
	*sql.DB
	Config Config
}

// New initializes a Trino database connection and executes schema
func New(config Config) (*Database, error) {
	if config.Type != "trino" {
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	username := "test"
	password := "password"

	dsn := fmt.Sprintf("http://%s:%s@%s:%d?catalog=%s&schema=%s",
		username,
		password,
		config.Host,
		config.Port,
		config.Catalog,
		config.Schema,
	)

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

	// Read schema.sql file
	schemaSQL, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Split SQL statements (Trino does not support multi-statement execution)
	queries := strings.Split(string(schemaSQL), ";")

	// Execute each query separately
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue // Skip empty statements
		}

		fmt.Println("Executing query:", query)
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to execute query: %s, error: %w", query, err)
		}
	}

	fmt.Println("Schema successfully executed!")
	return nil
}
