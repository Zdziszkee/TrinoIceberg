package service

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"

	models "github.com/zdziszkee/swift-codes/internal/models"
	repository "github.com/zdziszkee/swift-codes/internal/repositories"
)

var (
	ErrNotFound      = errors.New("swift code not found")
	ErrInvalidInput  = errors.New("invalid input provided")
	ErrAlreadyExists = errors.New("swift code already exists")
)

// SWIFT code validation regex - Updated to be more accurate
// Format: 4 letters (bank code) + 2 letters (country code) + 2 alphanumeric (location) + optional 3 alphanumeric (branch)
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
	log.Printf("GetSwiftCodeDetails called with code: %s", code)

	// Convert to uppercase before validation
	code = strings.ToUpper(code)

	if !swiftCodeRegex.MatchString(code) {
		log.Printf("Invalid swift code format: %s", code)
		return nil, ErrInvalidInput
	}

	bank, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			log.Printf("Swift code not found: %s", code)
			return nil, ErrNotFound
		}
		log.Printf("Error retrieving swift code details for %s: %v", code, err)
		return nil, err
	}

	log.Printf("Successfully retrieved swift code details for %s", code)
	return bank, nil
}

// GetSwiftCodesByCountry retrieves all SWIFT codes for a country
func (s *swiftService) GetSwiftCodesByCountry(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
	// Convert to uppercase before validation
	countryCode = strings.ToUpper(countryCode)

	if !countryCodeRegex.MatchString(countryCode) {
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
	// Check for nil bank to prevent panic
	if bank == nil {
		return ErrInvalidInput
	}

	// Convert to uppercase before validation
	bank.SwiftCode = strings.ToUpper(bank.SwiftCode)
	bank.CountryISOCode = strings.ToUpper(bank.CountryISOCode)

	// Validate SWIFT code
	if !swiftCodeRegex.MatchString(bank.SwiftCode) {
		return ErrInvalidInput
	}

	// Validate country code
	if !countryCodeRegex.MatchString(bank.CountryISOCode) {
		return ErrInvalidInput
	}

	// Validate other fields
	if bank.BankName == "" {
		return ErrInvalidInput
	}

	// Set headquarter flag based on SWIFT code suffix
	bank.IsHeadquarter = strings.HasSuffix(bank.SwiftCode, "XXX")

	// Set SwiftCodeBase if not set
	if bank.SwiftCodeBase == "" {
		bank.SwiftCodeBase = bank.SwiftCode[:8]
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
	// Convert to uppercase before validation
	code = strings.ToUpper(code)

	if !swiftCodeRegex.MatchString(code) {
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
