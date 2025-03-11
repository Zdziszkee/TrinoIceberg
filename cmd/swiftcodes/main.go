package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	handler "github.com/zdziszkee/swift-codes/internal/api/handlers"
	"github.com/zdziszkee/swift-codes/internal/api/router"
	config "github.com/zdziszkee/swift-codes/internal/configurations"
	"github.com/zdziszkee/swift-codes/internal/database"
	"github.com/zdziszkee/swift-codes/internal/models"
	parser "github.com/zdziszkee/swift-codes/internal/parsers"
	csvreader "github.com/zdziszkee/swift-codes/internal/readers/csv"
	repository "github.com/zdziszkee/swift-codes/internal/repositories"
	service "github.com/zdziszkee/swift-codes/internal/services"
)

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
	defer db.DB.Close()

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

		// Open the CSV file
		file, err := os.Open(cfg.Data.SwiftCodesFile)
		if err != nil {
			log.Printf("WARNING: Failed to open SWIFT codes file: %v", err)
		} else {
			defer file.Close()

			// Load SWIFT bank records from CSV
			reader := csvreader.CSVSwiftBanksReader{}
			records, err := reader.LoadSwiftBanks(file)
			if err != nil {
				log.Printf("WARNING: Failed to read CSV file: %v", err)
			} else {
				// Parse the records into SwiftBank models
				defaultParser := parser.DefaultSwiftBanksParser{}
				banks, err := defaultParser.ParseSwiftBanks(records)
				if err != nil {
					log.Printf("WARNING: Failed to parse SWIFT bank records: %v", err)
				} else {
					// Convert banks slice to a slice of pointers to models.SwiftBank
					var bankPtrs []*models.SwiftBank
					for i := range banks {
						bankPtrs = append(bankPtrs, &banks[i])
					}
					// Create banks in batches
					err = repo.CreateBatch(ctx, bankPtrs)
					if err != nil {
						log.Printf("WARNING: Failed to load SWIFT codes into database: %v", err)
					} else {
						log.Printf("Successfully loaded %d SWIFT codes", len(bankPtrs))
					}
				}
			}
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
