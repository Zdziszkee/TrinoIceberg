package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	handlers "github.com/zdziszkee/swift-codes/internal/api/handlers"
	models "github.com/zdziszkee/swift-codes/internal/models"
	repository "github.com/zdziszkee/swift-codes/internal/repositories"
	service "github.com/zdziszkee/swift-codes/internal/services"
	mocks "github.com/zdziszkee/swift-codes/tests/mocks"
)

func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Swift Handler Suite")
}

// A helper to create a Fiber app with our handler mounted on a route.
func setupApp(svc service.SwiftService) *fiber.App {
	app := fiber.New()
	// Create a new handler that uses the provided service.
	h := handlers.NewSwiftHandler(svc)

	// Mount routes for testing.
	app.Get("/swift/:swiftCode", h.GetByCode)
	app.Get("/country/:countryISO2code", h.GetByCountry)
	app.Post("/swift", h.Create)
	app.Delete("/swift/:swiftCode", h.Delete)

	return app
}

func TestGetByCode_Success(t *testing.T) {
	// Arrange: create a mock service that returns a valid SwiftBankDetail.
	mockSvc := &mocks.MockSwiftService{
		GetSwiftCodeDetailsFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
			// assume repository.SwiftBankDetail has a Bank field containing SwiftCode and BankName.
			return &repository.SwiftBankDetail{
				Bank: models.SwiftBank{
					SwiftCode: strings.ToUpper(code),
					BankName:  "Test Bank",
				},
			}, nil
		},
	}
	app := setupApp(mockSvc)

	// Act: perform a GET request for a known code.
	req := httptest.NewRequest(http.MethodGet, "/swift/abc", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	// Assert: decode the returned JSON.
	var bank repository.SwiftBankDetail
	err = json.NewDecoder(resp.Body).Decode(&bank)
	Expect(err).NotTo(HaveOccurred())
	Expect(bank.Bank.SwiftCode).To(Equal("ABC"))
	Expect(bank.Bank.BankName).To(Equal("Test Bank"))
}

func TestGetByCode_NotFound(t *testing.T) {
	// Arrange: use a mock service that returns the predeclared error.
	mockSvc := &mocks.MockSwiftService{
		GetSwiftCodeDetailsFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
			return nil, service.ErrNotFound
		},
	}
	app := setupApp(mockSvc)

	// Act.
	req := httptest.NewRequest(http.MethodGet, "/swift/xyz", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// Assert: decode error message.
	var body map[string]string
	err = json.NewDecoder(resp.Body).Decode(&body)
	Expect(err).NotTo(HaveOccurred())
	Expect(body["message"]).To(Equal("SWIFT code not found"))
}

func TestGetByCountry_Success(t *testing.T) {
	// Arrange: mock service returns a CountrySwiftCodes object.
	mockSvc := &mocks.MockSwiftService{
		GetSwiftCodesByCountryFunc: func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
			return &repository.CountrySwiftCodes{
				CountryISO2: strings.ToUpper(countryCode),
				CountryName: "Test Country",
				SwiftCodes: []models.SwiftBank{
					{SwiftCode: "ABC", BankName: "Bank A"},
					{SwiftCode: "DEF", BankName: "Bank B"},
				},
			}, nil
		},
	}
	app := setupApp(mockSvc)

	// Act.
	req := httptest.NewRequest(http.MethodGet, "/country/us", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	// Assert.
	var countryCodes repository.CountrySwiftCodes
	err = json.NewDecoder(resp.Body).Decode(&countryCodes)
	Expect(err).NotTo(HaveOccurred())
	Expect(countryCodes.SwiftCodes).To(HaveLen(2))
	Expect(countryCodes.SwiftCodes[0].SwiftCode).To(Equal("ABC"))
}

func TestCreate_Success(t *testing.T) {
	// Arrange: for creation provide a valid JSON payload.
	mockSvc := &mocks.MockSwiftService{
		CreateSwiftCodeFunc: func(ctx context.Context, bank *models.SwiftBank) error {
			return nil
		},
	}
	app := setupApp(mockSvc)

	// Create a sample bank. Adjust the JSON structure according to your models.
	bankData := models.SwiftBank{
		SwiftCode: "LMN",
		BankName:  "New Bank",
	}
	bodyBytes, err := json.Marshal(bankData)
	Expect(err).NotTo(HaveOccurred())

	req := httptest.NewRequest(http.MethodPost, "/swift", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, fiber.TestConfig{})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	var respBody map[string]string
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	Expect(err).NotTo(HaveOccurred())
	Expect(respBody["message"]).To(Equal("SWIFT code created successfully"))
}

func TestCreate_InvalidBody(t *testing.T) {
	// In case of a bad body the bind should fail; here we simulate that by sending invalid JSON.
	// Note: Depending on your actual Bind() implementation this test may need adjusting.
	mockSvc := &mocks.MockSwiftService{
		CreateSwiftCodeFunc: func(ctx context.Context, bank *models.SwiftBank) error {
			return nil // This should not be reached if binding fails.
		},
	}
	app := setupApp(mockSvc)

	invalidJSON := `{"swiftCode": "LMN",` // truncated JSON
	req := httptest.NewRequest(http.MethodPost, "/swift", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, fiber.TestConfig{})
	Expect(err).NotTo(HaveOccurred())
	// The handler should return a Bad Request.
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
}

func TestDelete_Success(t *testing.T) {
	// Arrange: mock deletion succeeds.
	mockSvc := &mocks.MockSwiftService{
		DeleteSwiftCodeFunc: func(ctx context.Context, code string) error {
			return nil
		},
	}
	app := setupApp(mockSvc)

	// Act.
	req := httptest.NewRequest(http.MethodDelete, "/swift/def", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var body map[string]string
	err = json.NewDecoder(resp.Body).Decode(&body)
	Expect(err).NotTo(HaveOccurred())
	Expect(body["message"]).To(Equal("SWIFT code deleted successfully"))
}

func TestDelete_NotFound(t *testing.T) {
	// Arrange: deletion returns error when code is not found.
	mockSvc := &mocks.MockSwiftService{
		DeleteSwiftCodeFunc: func(ctx context.Context, code string) error {
			return service.ErrNotFound
		},
	}
	app := setupApp(mockSvc)

	// Act.
	req := httptest.NewRequest(http.MethodDelete, "/swift/ghi", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	var body map[string]string
	err = json.NewDecoder(resp.Body).Decode(&body)
	Expect(err).NotTo(HaveOccurred())
	Expect(body["message"]).To(Equal("SWIFT code not found"))
}

func TestDelete_InvalidInput(t *testing.T) {
	// Arrange: deletion returns an invalid input error.
	mockSvc := &mocks.MockSwiftService{
		DeleteSwiftCodeFunc: func(ctx context.Context, code string) error {
			return service.ErrInvalidInput
		},
	}
	app := setupApp(mockSvc)

	// Act.
	req := httptest.NewRequest(http.MethodDelete, "/swift/JKL", nil)
	resp, err := app.Test(req, fiber.TestConfig{})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	var body map[string]string
	err = json.NewDecoder(resp.Body).Decode(&body)
	Expect(err).NotTo(HaveOccurred())
	Expect(body["message"]).To(Equal("Invalid input provided"))
}
