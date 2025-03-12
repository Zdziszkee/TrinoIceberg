package router

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	handler "github.com/zdziszkee/swift-codes/internal/api/handlers"
)

// SetupRoutes configures all API routes
func SetupRoutes(swiftHandler *handler.SwiftHandler) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			// Default error handler
			code := fiber.StatusInternalServerError

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			return c.Status(code).JSON(fiber.Map{
				"message": "Internal server error",
			})
		},
	})

	// Add global middleware
	app.Use(logger.New())
	app.Use(recover.New())

	// API versioning
	v1 := app.Group("/v1")

	// SWIFT codes endpoints
	v1.Get("/swiftCodes/:swiftCode", swiftHandler.GetByCode)
	v1.Get("/swiftCodes/country/:countryISO2code", swiftHandler.GetByCountry)
	v1.Post("/swiftCodes", swiftHandler.Create)
	v1.Delete("/swiftCodes/:swiftCode", swiftHandler.Delete)
	return app
}
