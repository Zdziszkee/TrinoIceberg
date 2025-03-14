package service_test

import (
	"context"
	"errors"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/zdziszkee/swift-codes/internal/models"
	repository "github.com/zdziszkee/swift-codes/internal/repositories"
	service "github.com/zdziszkee/swift-codes/internal/services"
)

func TestServices(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Services Suite")
}

// compareErrors compares two errors by their string representation
func compareErrors(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}
	if err1 == nil || err2 == nil {
		return false
	}
	return err1.Error() == err2.Error()
}

var _ = Describe("SwiftService", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("GetSwiftCodeDetails", func() {
		Context("when called with a valid SWIFT code", func() {
			It("should return the bank details", func() {
				repo := &mocks.MockSwiftRepository{
					GetByCodeFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
						return &repository.SwiftBankDetail{
							Bank:     models.SwiftBank{SwiftCode: "ABCDUS33XXX"},
							Branches: []models.SwiftBank{},
						}, nil
					},
				}

				s := service.NewSwiftService(repo)
				got, err := s.GetSwiftCodeDetails(ctx, "ABCDUS33XXX")

				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(&repository.SwiftBankDetail{
					Bank:     models.SwiftBank{SwiftCode: "ABCDUS33XXX"},
					Branches: []models.SwiftBank{},
				}))
			})
		})

		Context("when called with an invalid SWIFT code", func() {
			It("should return an invalid input error", func() {
				repo := &mocks.MockSwiftRepository{}
				s := service.NewSwiftService(repo)

				_, err := s.GetSwiftCodeDetails(ctx, "ABC123")

				Expect(err).To(MatchError(service.ErrInvalidInput))
			})
		})

		Context("when the code is not found", func() {
			It("should return not found error", func() {
				repo := &mocks.MockSwiftRepository{
					GetByCodeFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
						return nil, repository.ErrNotFound
					},
				}

				s := service.NewSwiftService(repo)
				_, err := s.GetSwiftCodeDetails(ctx, "ABCDUS33XXX")

				Expect(err).To(MatchError(service.ErrNotFound))
			})
		})

		Context("when repository returns an error", func() {
			It("should return the error", func() {
				expectedError := errors.New("db error")
				repo := &mocks.MockSwiftRepository{
					GetByCodeFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
						return nil, expectedError
					},
				}

				s := service.NewSwiftService(repo)
				_, err := s.GetSwiftCodeDetails(ctx, "ABCDUS33XXX")

				Expect(err.Error()).To(Equal(expectedError.Error()))
			})
		})

		Context("when called with a valid 8-character SWIFT code", func() {
			It("should return the bank details", func() {
				repo := &mocks.MockSwiftRepository{
					GetByCodeFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
						return &repository.SwiftBankDetail{
							Bank:     models.SwiftBank{SwiftCode: "ABCDUS33"},
							Branches: []models.SwiftBank{},
						}, nil
					},
				}

				s := service.NewSwiftService(repo)
				got, err := s.GetSwiftCodeDetails(ctx, "ABCDUS33")

				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(&repository.SwiftBankDetail{
					Bank:     models.SwiftBank{SwiftCode: "ABCDUS33"},
					Branches: []models.SwiftBank{},
				}))
			})
		})
	})

	Describe("GetSwiftCodesByCountry", func() {
		Context("when called with a valid country code", func() {
			It("should return the country codes", func() {
				repo := &mocks.MockSwiftRepository{
					GetByCountryFunc: func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
						return &repository.CountrySwiftCodes{
							SwiftCodes: []models.SwiftBank{},
						}, nil
					},
				}

				s := service.NewSwiftService(repo)
				got, err := s.GetSwiftCodesByCountry(ctx, "US")

				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(&repository.CountrySwiftCodes{
					SwiftCodes: []models.SwiftBank{},
				}))
			})
		})

		Context("when called with an invalid country code", func() {
			It("should return an invalid input error", func() {
				repo := &mocks.MockSwiftRepository{}
				s := service.NewSwiftService(repo)

				_, err := s.GetSwiftCodesByCountry(ctx, "USA")

				Expect(err).To(MatchError(service.ErrInvalidInput))
			})
		})

		Context("when called with an empty country code", func() {
			It("should return an invalid input error", func() {
				repo := &mocks.MockSwiftRepository{}
				s := service.NewSwiftService(repo)

				_, err := s.GetSwiftCodesByCountry(ctx, "")

				Expect(err).To(MatchError(service.ErrInvalidInput))
			})
		})

		Context("when the country code is not found", func() {
			It("should return not found error", func() {
				repo := &mocks.MockSwiftRepository{
					GetByCountryFunc: func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
						return nil, repository.ErrNotFound
					},
				}

				s := service.NewSwiftService(repo)
				_, err := s.GetSwiftCodesByCountry(ctx, "US")

				Expect(err).To(MatchError(service.ErrNotFound))
			})
		})

		Context("when repository returns an error", func() {
			It("should return the error", func() {
				expectedError := errors.New("db error")
				repo := &mocks.MockSwiftRepository{
					GetByCountryFunc: func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
						return nil, expectedError
					},
				}

				s := service.NewSwiftService(repo)
				_, err := s.GetSwiftCodesByCountry(ctx, "US")

				Expect(err.Error()).To(Equal(expectedError.Error()))
			})
		})

		Context("when called with a lowercase country code", func() {
			It("should convert and return the codes", func() {
				repo := &mocks.MockSwiftRepository{
					GetByCountryFunc: func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
						countryCode = strings.ToUpper(countryCode)
						if countryCode == "US" {
							return &repository.CountrySwiftCodes{
								SwiftCodes: []models.SwiftBank{},
							}, nil
						}
						return nil, repository.ErrNotFound
					},
				}

				s := service.NewSwiftService(repo)
				got, err := s.GetSwiftCodesByCountry(ctx, "us")

				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(&repository.CountrySwiftCodes{
					SwiftCodes: []models.SwiftBank{},
				}))
			})
		})
	})

	Describe("CreateSwiftCode", func() {
		Context("when called with a valid bank", func() {
			It("should create the bank", func() {
				repo := &mocks.MockSwiftRepository{
					CreateFunc: func(ctx context.Context, bank *models.SwiftBank) error { return nil },
				}

				s := service.NewSwiftService(repo)
				bank := &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "US", BankName: "Test Bank"}
				err := s.CreateSwiftCode(ctx, bank)

				Expect(err).ToNot(HaveOccurred())
				Expect(bank.SwiftCode).To(Equal("ABCDUS33XXX"))
				Expect(bank.CountryISOCode).To(Equal("US"))
				Expect(bank.IsHeadquarter).To(BeTrue())
				Expect(bank.SwiftCodeBase).To(Equal("ABCDUS33"))
			})
		})

		Context("when called with an invalid SWIFT code", func() {
			It("should return an invalid input error", func() {
				repo := &mocks.MockSwiftRepository{}
				s := service.NewSwiftService(repo)

				bank := &models.SwiftBank{SwiftCode: "ABC123", CountryISOCode: "US", BankName: "Test Bank"}
				err := s.CreateSwiftCode(ctx, bank)

				Expect(err).To(MatchError(service.ErrInvalidInput))
			})
		})

		Context("when called with an invalid country code", func() {
			It("should return an invalid input error", func() {
				repo := &mocks.MockSwiftRepository{}
				s := service.NewSwiftService(repo)

				bank := &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "USA", BankName: "Test Bank"}
				err := s.CreateSwiftCode(ctx, bank)

				Expect(err).To(MatchError(service.ErrInvalidInput))
			})
		})

		Context("when called with an empty bank name", func() {
			It("should return an invalid input error", func() {
				repo := &mocks.MockSwiftRepository{}
				s := service.NewSwiftService(repo)

				bank := &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "US", BankName: ""}
				err := s.CreateSwiftCode(ctx, bank)

				Expect(err).To(MatchError(service.ErrInvalidInput))
			})
		})

		Context("when the SWIFT code already exists", func() {
			It("should return an already exists error", func() {
				repo := &mocks.MockSwiftRepository{
					CreateFunc: func(ctx context.Context, bank *models.SwiftBank) error {
						return repository.ErrDuplicate
					},
				}

				s := service.NewSwiftService(repo)
				bank := &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "US", BankName: "Test Bank"}
				err := s.CreateSwiftCode(ctx, bank)

				Expect(err).To(MatchError(service.ErrAlreadyExists))
			})
		})

		Context("when bank is nil", func() {
			It("should return an invalid input error", func() {
				repo := &mocks.MockSwiftRepository{}
				s := service.NewSwiftService(repo)

				err := s.CreateSwiftCode(ctx, nil)

				Expect(err).To(MatchError(service.ErrInvalidInput))
			})
		})

		Context("when repository returns an error", func() {
			It("should return the error", func() {
				expectedError := errors.New("db error")
				repo := &mocks.MockSwiftRepository{
					CreateFunc: func(ctx context.Context, bank *models.SwiftBank) error {
						return expectedError
					},
				}

				s := service.NewSwiftService(repo)
				bank := &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "US", BankName: "Test Bank"}
				err := s.CreateSwiftCode(ctx, bank)

				Expect(err.Error()).To(Equal(expectedError.Error()))
			})
		})

		Context("when called with lowercase codes", func() {
			It("should convert them to uppercase", func() {
				repo := &mocks.MockSwiftRepository{
					CreateFunc: func(ctx context.Context, bank *models.SwiftBank) error {
						if bank.SwiftCode != "ABCDUS33XXX" || bank.CountryISOCode != "US" {
							return errors.New("codes not properly uppercased")
						}
						return nil
					},
				}

				s := service.NewSwiftService(repo)
				bank := &models.SwiftBank{SwiftCode: "abcdus33xxx", CountryISOCode: "us", BankName: "Test Bank"}
				err := s.CreateSwiftCode(ctx, bank)

				Expect(err).ToNot(HaveOccurred())
				Expect(bank.SwiftCode).To(Equal("ABCDUS33XXX"))
				Expect(bank.CountryISOCode).To(Equal("US"))
			})
		})
	})

	Describe("DeleteSwiftCode", func() {
		Context("when called with a valid SWIFT code", func() {
			It("should delete the bank", func() {
				repo := &mocks.MockSwiftRepository{
					DeleteFunc: func(ctx context.Context, code string) error { return nil },
				}

				s := service.NewSwiftService(repo)
				err := s.DeleteSwiftCode(ctx, "ABCDUS33XXX")

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when called with an invalid SWIFT code", func() {
			It("should return an invalid input error", func() {
				repo := &mocks.MockSwiftRepository{}
				s := service.NewSwiftService(repo)

				err := s.DeleteSwiftCode(ctx, "ABC123")

				Expect(err).To(MatchError(service.ErrInvalidInput))
			})
		})

		Context("when the code is not found", func() {
			It("should return not found error", func() {
				repo := &mocks.MockSwiftRepository{
					DeleteFunc: func(ctx context.Context, code string) error {
						return repository.ErrNotFound
					},
				}

				s := service.NewSwiftService(repo)
				err := s.DeleteSwiftCode(ctx, "ABCDUS33XXX")

				Expect(err).To(MatchError(service.ErrNotFound))
			})
		})

		Context("when repository returns an error", func() {
			It("should return the error", func() {
				expectedError := errors.New("db error")
				repo := &mocks.MockSwiftRepository{
					DeleteFunc: func(ctx context.Context, code string) error {
						return expectedError
					},
				}

				s := service.NewSwiftService(repo)
				err := s.DeleteSwiftCode(ctx, "ABCDUS33XXX")

				Expect(err.Error()).To(Equal(expectedError.Error()))
			})
		})

		Context("when called with a lowercase SWIFT code", func() {
			It("should convert it to uppercase", func() {
				repo := &mocks.MockSwiftRepository{
					DeleteFunc: func(ctx context.Context, code string) error {
						if code != "ABCDUS33XXX" {
							return errors.New("code not properly uppercased")
						}
						return nil
					},
				}

				s := service.NewSwiftService(repo)
				err := s.DeleteSwiftCode(ctx, "abcdus33xxx")

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
