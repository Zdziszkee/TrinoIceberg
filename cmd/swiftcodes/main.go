package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zdziszkee/swift-codes/internal/api/handler"
	"github.com/zdziszkee/swift-codes/internal/api/router"
	config "github.com/zdziszkee/swift-codes/internal/configuration"
	"github.com/zdziszkee/swift-codes/internal/database"
	model "github.com/zdziszkee/swift-codes/internal/model" // Add this import
	"github.com/zdziszkee/swift-codes/internal/parser"
	"github.com/zdziszkee/swift-codes/internal/repository"
	"github.com/zdziszkee/swift-codes/internal/service"
)

// loadSwiftCodesFromFile loads SWIFT codes from a CSV file into the database
func loadSwiftCodesFromFile(ctx context.Context, filePath string, repo repository.SwiftRepository) (int, error) {
	startTime := time.Now()

	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	swiftParser := parser.NewCSVSwiftParser()
	swiftBanks, err := swiftParser.ParseSwiftData(file)
	if err != nil {
		return 0, fmt.Errorf("failed to parse SWIFT data: %w", err)
	}

	const batchSize = 20000
	loadedCount := 0
	batch := make([]*model.SwiftBank, 0, batchSize)

	for i, bank := range swiftBanks {
		localBank := bank
		batch = append(batch, &localBank)

		if len(batch) == batchSize || i == len(swiftBanks)-1 {
			fmt.Printf("Inserting batch of %d rows at %v\n", len(batch), time.Now())
			err := repo.CreateBatch(ctx, batch)
			if err != nil {
				fmt.Printf("Error inserting batch of %d SWIFT codes: %v\n", len(batch), err)
			} else {
				loadedCount += len(batch)
			}
			batch = batch[:0]
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("Loaded %d of %d SWIFT codes in %v\n", loadedCount, len(swiftBanks), duration)
	return loadedCount, nil
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	loadFile := flag.String("load", "", "Path to SWIFT codes CSV file to load")
	flag.Parse()
	time.Sleep(20 * time.Second)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override config with command line flags if provided
	if *loadFile != "" {
		cfg.Data.SwiftCodesFile = *loadFile
		cfg.Data.AutoLoad = true
	}

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repository
	repo := repository.NewSQLSwiftRepository(db)

	// Initialize service
	swiftService := service.NewSwiftService(repo)

	// Auto-load data if configured
	if cfg.Data.AutoLoad && cfg.Data.SwiftCodesFile != "" {
		log.Printf("Loading SWIFT codes from %s", cfg.Data.SwiftCodesFile)

		// Use a timeout context for loading
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		count, err := loadSwiftCodesFromFile(ctx, cfg.Data.SwiftCodesFile, repo)
		if err != nil {
			log.Printf("WARNING: Failed to load SWIFT codes: %v", err)
		} else {
			log.Printf("Successfully loaded %d SWIFT codes", count)
		}
	}

	// Initialize handler
	handler := handler.NewSwiftHandler(swiftService)

	// Setup routes
	app := router.SetupRoutes(handler)

	// Start server in a goroutine so we can handle graceful shutdown
	go func() {
		log.Printf("Starting server on port 8080")
		if err := app.Listen(":8080"); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Provide a timeout context for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
