package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
)

// CustomLogger creates a custom logging middleware
func CustomLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		// Call the next handler
		err := c.Next()

		// Log after request is processed
		log.Printf(
			"%s %s %s %s",
			c.Method(),
			c.Path(),
			c.IP(),
			time.Since(start),
		)

		return err
	}
}
