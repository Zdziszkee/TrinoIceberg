package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	models "github.com/zdziszkee/swift-codes/internal/model"
	"github.com/zdziszkee/swift-codes/internal/repository"
)

var (
	ErrNotFound      = errors.New("swift code not found")
	ErrInvalidInput  = errors.New("invalid input provided")
	ErrAlreadyExists = errors.New("swift code already exists")
)

// SWIFT code validation regex
var swiftCodeRegex = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)
var countryCodeRegex = regexp.MustCompile(`^[A-Z]{2}$`)

// SwiftService handles business logic for SWIFT codes
type SwiftService interface {
	GetSwiftCodeDetails(ctx context.Context, code string) (*repository.SwiftBankDetail, error)
	GetSwiftCodesByCountry(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error)
	CreateSwiftCode(ctx context.Context, bank *models.SwiftBank) error
	DeleteSwiftCode(ctx context.Context, code string) error
}

// swiftService implements SwiftService
type swiftService struct {
	repo repository.SwiftRepository
}

// NewSwiftService creates a new instance of the Swift service
func NewSwiftService(repo repository.SwiftRepository) SwiftService {
	return &swiftService{repo: repo}
}

// GetSwiftCodeDetails retrieves detailed info for a SWIFT code
func (s *swiftService) GetSwiftCodeDetails(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
	if !swiftCodeRegex.MatchString(strings.ToUpper(code)) {
		return nil, ErrInvalidInput
	}

	bank, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return bank, nil
}

// GetSwiftCodesByCountry retrieves all SWIFT codes for a country
func (s *swiftService) GetSwiftCodesByCountry(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
	if !countryCodeRegex.MatchString(strings.ToUpper(countryCode)) {
		return nil, ErrInvalidInput
	}

	codes, err := s.repo.GetByCountry(ctx, countryCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return codes, nil
}

// CreateSwiftCode adds a new SWIFT code to the database
func (s *swiftService) CreateSwiftCode(ctx context.Context, bank *models.SwiftBank) error {
	// Validate SWIFT code
	if !swiftCodeRegex.MatchString(strings.ToUpper(bank.SwiftCode)) {
		return ErrInvalidInput
	}

	// Validate country code
	if !countryCodeRegex.MatchString(strings.ToUpper(bank.CountryISOCode)) {
		return ErrInvalidInput
	}

	// Validate other fields
	if bank.BankName == "" {
		return ErrInvalidInput
	}

	// Ensure SWIFT code is uppercase
	bank.SwiftCode = strings.ToUpper(bank.SwiftCode)
	bank.CountryISOCode = strings.ToUpper(bank.CountryISOCode)

	// Set the entity type if not set
	if bank.EntityType == "" {
		// Default to branch, unless code ends with XXX
		if strings.HasSuffix(bank.SwiftCode, "XXX") {
			bank.EntityType = models.Headquarters
		} else {
			bank.EntityType = models.Branch
		}
	}

	// Set HQ base if not set
	if bank.HQSwiftBase == "" {
		bank.HQSwiftBase = bank.SwiftCode[:8]
	}

	err := s.repo.Create(ctx, bank)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return ErrAlreadyExists
		}
		return err
	}

	return nil
}

// DeleteSwiftCode removes a SWIFT code from the database
func (s *swiftService) DeleteSwiftCode(ctx context.Context, code string) error {
	if !swiftCodeRegex.MatchString(strings.ToUpper(code)) {
		return ErrInvalidInput
	}

	err := s.repo.Delete(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	return nil
}
