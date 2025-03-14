{{REWRITTEN_CODE}}
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

var _ = Describe("Swift Handler", func() {
	var (
		app     *fiber.App
		mockSvc *mocks.MockSwiftService
		ctx     = context.Background()
	)

	BeforeEach(func() {
		mockSvc = &mocks.MockSwiftService{}
	})

	Describe("GetByCode", func() {
		Context("when called with a valid SWIFT code", func() {
			It("should return the swift bank details", func() {
				mockSvc.GetSwiftCodeDetailsFunc = func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
					return &repository.SwiftBankDetail{
						Bank: models.SwiftBank{
							SwiftCode: strings.ToUpper(code),
							BankName:  "Test Bank",
						},
					}, nil
				}
				app = setupApp(mockSvc)
				req := httptest.NewRequest(http.MethodGet, "/swift/abc", nil)
				resp, err := app.Test(req, fiber.TestConfig{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var bank repository.SwiftBankDetail
				err = json.NewDecoder(resp.Body).Decode(&bank)
				Expect(err).NotTo(HaveOccurred())
				Expect(bank.Bank.SwiftCode).To(Equal("ABC"))
				Expect(bank.Bank.BankName).To(Equal("Test Bank"))
			})
		})

		Context("when called with a SWIFT code that is not found", func() {
			It("should return a not found error", func() {
				mockSvc.GetSwiftCodeDetailsFunc = func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
					return nil, service.ErrNotFound
				}
				app = setupApp(mockSvc)
				req := httptest.NewRequest(http.MethodGet, "/swift/xyz", nil)
				resp, err := app.Test(req, fiber.TestConfig{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

				var body map[string]string
				err = json.NewDecoder(resp.Body).Decode(&body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body["message"]).To(Equal("SWIFT code not found"))
			})
		})

		Context("when called with an invalid SWIFT code", func() {
			It("should return an invalid input error", func() {
				mockSvc.GetSwiftCodeDetailsFunc = func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
					return nil, service.ErrInvalidInput
				}
				app = setupApp(mockSvc)
				req := httptest.NewRequest(http.MethodGet, "/swift/ABC123", nil)
				resp, err := app.Test(req, fiber.TestConfig{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

				var body map[string]string
				err = json.NewDecoder(resp.Body).Decode(&body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body["message"]).To(Equal("Invalid input provided"))
			})
		})
	})

	Describe("GetByCountry", func() {
		Context("when called with a country that has swift codes", func() {
			It("should return a list of swift codes", func() {
				mockSvc.GetSwiftCodesByCountryFunc = func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
					return &repository.CountrySwiftCodes{
						CountryISO2: strings.ToUpper(countryCode),
						CountryName: "Test Country",
						SwiftCodes: []models.SwiftBank{
							{SwiftCode: "ABC", BankName: "Bank A"},
							{SwiftCode: "DEF", BankName: "Bank B"},
						},
					}, nil
				}
				app = setupApp(mockSvc)
				req := httptest.NewRequest(http.MethodGet, "/country/us", nil)
				resp, err := app.Test(req, fiber.TestConfig{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var countryCodes repository.CountrySwiftCodes
				err = json.NewDecoder(resp.Body).Decode(&countryCodes)
				Expect(err).NotTo(HaveOccurred())
				Expect(countryCodes.SwiftCodes).To(HaveLen(2))
				Expect(countryCodes.SwiftCodes[0].SwiftCode).To(Equal("ABC"))
			})
		})
	})

	Describe("Create", func() {
		Context("when provided with valid swift code data", func() {
			It("should create a new swift code", func() {
				mockSvc.CreateSwiftCodeFunc = func(ctx context.Context, bank *models.SwiftBank) error {
					return nil
				}
				app = setupApp(mockSvc)
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
			})
		})

		Context("when provided with an invalid request body", func() {
			It("should return a bad request error", func() {
				mockSvc.CreateSwiftCodeFunc = func(ctx context.Context, bank *models.SwiftBank) error {
					return nil
				}
				app = setupApp(mockSvc)
				invalidJSON := `{"swiftCode": "LMN",`
				req := httptest.NewRequest(http.MethodPost, "/swift", strings.NewReader(invalidJSON))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req, fiber.TestConfig{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("Delete", func() {
		Context("when deletion is successful", func() {
			It("should delete the swift code successfully", func() {
				mockSvc.DeleteSwiftCodeFunc = func(ctx context.Context, code string) error {
					return nil
				}
				app = setupApp(mockSvc)
				req := httptest.NewRequest(http.MethodDelete, "/swift/def", nil)
				resp, err := app.Test(req, fiber.TestConfig{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var body map[string]string
				err = json.NewDecoder(resp.Body).Decode(&body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body["message"]).To(Equal("SWIFT code deleted successfully"))
			})
		})

		Context("when deletion fails because the swift code is not found", func() {
			It("should return a not found error", func() {
				mockSvc.DeleteSwiftCodeFunc = func(ctx context.Context, code string) error {
					return service.ErrNotFound
				}
				app = setupApp(mockSvc)
				req := httptest.NewRequest(http.MethodDelete, "/swift/ghi", nil)
				resp, err := app.Test(req, fiber.TestConfig{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

				var body map[string]string
				err = json.NewDecoder(resp.Body).Decode(&body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body["message"]).To(Equal("SWIFT code not found"))
			})
		})

		Context("when deletion fails due to invalid input", func() {
			It("should return an invalid input error", func() {
				mockSvc.DeleteSwiftCodeFunc = func(ctx context.Context, code string) error {
					return service.ErrInvalidInput
				}
				app = setupApp(mockSvc)
				req := httptest.NewRequest(http.MethodDelete, "/swift/JKL", nil)
				resp, err := app.Test(req, fiber.TestConfig{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

				var body map[string]string
				err = json.NewDecoder(resp.Body).Decode(&body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body["message"]).To(Equal("Invalid input provided"))
			})
		})
	})
})
