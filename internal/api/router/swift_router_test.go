package router_test

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

	// Import the handlers package for creating a new handler.
	handlers "github.com/zdziszkee/swift-codes/internal/api/handlers"
	models "github.com/zdziszkee/swift-codes/internal/models"
	repository "github.com/zdziszkee/swift-codes/internal/repositories"
	service "github.com/zdziszkee/swift-codes/internal/services"
	mocks "github.com/zdziszkee/swift-codes/tests/mocks"
)

func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Swift Router Suite")
}

// setupRouter initializes a new Fiber app and registers the Swift routes using your handler.
func setupRouter(svc service.SwiftService) *fiber.App {
	app := fiber.New()

	// Instead of using router.SetupSwiftRoutes,
	// create a new handler and register the routes.
	h := handlers.NewSwiftHandler(svc)
	app.Get("/swift/:swiftCode", h.GetByCode)
	app.Get("/country/:countryISO2code", h.GetByCountry)
	app.Post("/swift", h.Create)
	app.Delete("/swift/:swiftCode", h.Delete)

	return app
}

var _ = Describe("Swift Router", func() {
	var (
		app     *fiber.App
		mockSvc *mocks.MockSwiftService
	)

	BeforeEach(func() {
		mockSvc = &mocks.MockSwiftService{}
		app = setupRouter(mockSvc)
	})

	Describe("GET /swift/:swiftCode", func() {
		Context("when the swift code exists", func() {
			It("should return status 200 and swift bank details", func() {
				mockSvc.GetSwiftCodeDetailsFunc = func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
					return &repository.SwiftBankDetail{
						Bank: models.SwiftBank{
							SwiftCode: strings.ToUpper(code),
							BankName:  "Test Bank via Router",
						},
					}, nil
				}

				req := httptest.NewRequest(http.MethodGet, "/swift/abc", nil)
				resp, err := app.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var bank repository.SwiftBankDetail
				err = json.NewDecoder(resp.Body).Decode(&bank)
				Expect(err).NotTo(HaveOccurred())
				Expect(bank.Bank.SwiftCode).To(Equal("ABC"))
				Expect(bank.Bank.BankName).To(Equal("Test Bank via Router"))
			})
		})

		Context("when the swift code does not exist", func() {
			It("should return status 404", func() {
				mockSvc.GetSwiftCodeDetailsFunc = func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
					return nil, service.ErrNotFound
				}

				req := httptest.NewRequest(http.MethodGet, "/swift/unknown", nil)
				resp, err := app.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

				var body map[string]string
				err = json.NewDecoder(resp.Body).Decode(&body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body["message"]).To(Equal("SWIFT code not found"))
			})
		})
	})

	Describe("GET /country/:countryISO2code", func() {
		Context("when the country has swift codes", func() {
			It("should return status 200 and the swift codes list", func() {
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

				req := httptest.NewRequest(http.MethodGet, "/country/us", nil)
				resp, err := app.Test(req)
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

	Describe("POST /swift", func() {
		Context("when provided with valid swift code data", func() {
			It("should create a new swift code and return status 201", func() {
				mockSvc.CreateSwiftCodeFunc = func(ctx context.Context, bank *models.SwiftBank) error {
					return nil
				}

				bankData := models.SwiftBank{
					SwiftCode: "LMN",
					BankName:  "New Bank via Router",
				}
				bodyBytes, err := json.Marshal(bankData)
				Expect(err).NotTo(HaveOccurred())

				req := httptest.NewRequest(http.MethodPost, "/swift", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))

				var respBody map[string]string
				err = json.NewDecoder(resp.Body).Decode(&respBody)
				Expect(err).NotTo(HaveOccurred())
				Expect(respBody["message"]).To(Equal("SWIFT code created successfully"))
			})
		})

		Context("when provided with an invalid request body", func() {
			It("should return status 400", func() {
				invalidJSON := `{"swiftCode": "LMN",`
				req := httptest.NewRequest(http.MethodPost, "/swift", strings.NewReader(invalidJSON))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("DELETE /swift/:swiftCode", func() {
		Context("when deletion is successful", func() {
			It("should return status 200", func() {
				mockSvc.DeleteSwiftCodeFunc = func(ctx context.Context, code string) error {
					return nil
				}

				req := httptest.NewRequest(http.MethodDelete, "/swift/def", nil)
				resp, err := app.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var body map[string]string
				err = json.NewDecoder(resp.Body).Decode(&body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body["message"]).To(Equal("SWIFT code deleted successfully"))
			})
		})

		Context("when the swift code is not found", func() {
			It("should return status 404", func() {
				mockSvc.DeleteSwiftCodeFunc = func(ctx context.Context, code string) error {
					return service.ErrNotFound
				}

				req := httptest.NewRequest(http.MethodDelete, "/swift/ghi", nil)
				resp, err := app.Test(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

				var body map[string]string
				err = json.NewDecoder(resp.Body).Decode(&body)
				Expect(err).NotTo(HaveOccurred())
				Expect(body["message"]).To(Equal("SWIFT code not found"))
			})
		})

		Context("when invalid input is provided", func() {
			It("should return status 400", func() {
				mockSvc.DeleteSwiftCodeFunc = func(ctx context.Context, code string) error {
					return service.ErrInvalidInput
				}

				req := httptest.NewRequest(http.MethodDelete, "/swift/JKL", nil)
				resp, err := app.Test(req)
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
