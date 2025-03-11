package handler

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	models "github.com/zdziszkee/swift-codes/internal/model"
	"github.com/zdziszkee/swift-codes/internal/service"
)

// SwiftHandler handles API requests for SWIFT codes
type SwiftHandler struct {
	service service.SwiftService
}

// NewSwiftHandler creates a new handler instance
func NewSwiftHandler(service service.SwiftService) *SwiftHandler {
	return &SwiftHandler{service: service}
}

// GetByCode handles requests for a specific SWIFT code
func (h *SwiftHandler) GetByCode(c fiber.Ctx) error {
	code := strings.ToUpper(c.Params("swift-code"))

	bank, err := h.service.GetSwiftCodeDetails(c.Context(), code)
	if err != nil {
		return handleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(bank)
}

// GetByCountry handles requests for all SWIFT codes by country
func (h *SwiftHandler) GetByCountry(c fiber.Ctx) error {
	countryCode := strings.ToUpper(c.Params("countryISO2code"))

	codes, err := h.service.GetSwiftCodesByCountry(c.Context(), countryCode)
	if err != nil {
		return handleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(codes)
}

// Create handles creation of a new SWIFT code
func (h *SwiftHandler) Create(c fiber.Ctx) error {
	var bank models.SwiftBank

	if err := c.Bind().Body(&bank); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	err := h.service.CreateSwiftCode(c.Context(), &bank)
	if err != nil {
		return handleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "SWIFT code created successfully",
	})
}

// Delete handles deletion of a SWIFT code
func (h *SwiftHandler) Delete(c fiber.Ctx) error {
	code := strings.ToUpper(c.Params("swift-code"))

	err := h.service.DeleteSwiftCode(c.Context(), code)
	if err != nil {
		return handleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "SWIFT code deleted successfully",
	})
}

// Helper function for error handling
func handleError(c fiber.Ctx, err error) error {
	switch {
	case err == service.ErrNotFound:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "SWIFT code not found",
		})
	case err == service.ErrInvalidInput:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid input provided",
		})
	case err == service.ErrAlreadyExists:
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "SWIFT code already exists",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
		})
	}
}
