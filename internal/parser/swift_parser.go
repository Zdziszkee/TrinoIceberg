package parser

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode"

	models "github.com/zdziszkee/swift-codes/internal/model"
)

// Error definitions for better error handling
var (
	ErrHeaderInsufficient   = errors.New("header has insufficient columns")
	ErrRecordInsufficient   = errors.New("record has insufficient columns")
	ErrMissingRequiredField = errors.New("missing required fields: country code, swift code, or bank name")
	ErrInvalidSwiftCode     = errors.New("invalid SWIFT/BIC code format")
	ErrInvalidCountryCode   = errors.New("invalid country ISO code")
)

// Regular expressions for validation
var (
	// SWIFT/BIC code format: 4 letters (bank code) + 2 letters (country code) + 2 alphanumeric (location code) + optional 3 alphanumeric (branch code)
	swiftCodeRegex = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)

	// ISO 3166-1 alpha-2 country code format: 2 uppercase letters
	countryCodeRegex = regexp.MustCompile(`^[A-Z]{2}$`)
)

// SwiftParser is an interface for parsing SWIFT bank code data
type SwiftParser interface {
	ParseSwiftData(input io.Reader) ([]models.SwiftBank, error)
}

// CSVSwiftParser implements the SwiftParser interface for CSV format
type CSVSwiftParser struct {
	// Configuration options could be added here
	MaxRecordSize int // Maximum allowed size for a CSV record
}

// NewCSVSwiftParser creates a new SWIFT parser for CSV format
func NewCSVSwiftParser() SwiftParser {
	return &CSVSwiftParser{
		MaxRecordSize: 1024, // Set a reasonable limit
	}
}

// validateSwiftCode checks if the SWIFT code adheres to the standard format
func validateSwiftCode(code string) error {
	if !swiftCodeRegex.MatchString(code) {
		return fmt.Errorf("%w: %s", ErrInvalidSwiftCode, code)
	}
	return nil
}

// validateCountryCode checks if the country code is a valid ISO 3166-1 alpha-2 code
func validateCountryCode(code string) error {
	if !countryCodeRegex.MatchString(code) {
		return fmt.Errorf("%w: %s", ErrInvalidCountryCode, code)
	}
	return nil
}

// sanitizeBankName ensures the bank name is properly formatted and free of problematic characters
func sanitizeBankName(name string) string {
	// Trim whitespace and normalize spaces
	name = strings.Join(strings.Fields(name), " ")

	// Remove control characters
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
}

// validateSwiftBankEntry performs comprehensive validation on a SwiftBank entry
func validateSwiftBankEntry(bank *models.SwiftBank) error {
	// Validate SWIFT code
	if err := validateSwiftCode(bank.SwiftCode); err != nil {
		return err
	}

	// Validate country code
	if err := validateCountryCode(bank.CountryISOCode); err != nil {
		return err
	}

	// Verify entity type is valid
	if bank.EntityType != models.Headquarters && bank.EntityType != models.Branch {
		return fmt.Errorf("invalid entity type: %s", bank.EntityType)
	}

	// Ensure bank name isn't too short or too long
	if len(bank.BankName) < 2 {
		return errors.New("bank name too short")
	}
	if len(bank.BankName) > 255 {
		return errors.New("bank name too long")
	}

	// Verify consistency between SWIFT code and entity type
	if bank.EntityType == models.Headquarters && !strings.HasSuffix(bank.SwiftCode, "XXX") {
		return errors.New("headquarters must have 'XXX' suffix in SWIFT code")
	}
	if bank.EntityType == models.Branch && strings.HasSuffix(bank.SwiftCode, "XXX") {
		return errors.New("branch cannot have 'XXX' suffix in SWIFT code")
	}

	// Verify HQ base is correctly derived from SWIFT code
	if bank.HQSwiftBase != bank.SwiftCode[:8] {
		return errors.New("HQ swift base must be first 8 characters of SWIFT code")
	}

	return nil
}

// ParseSwiftData parses the data into SwiftBank entities
func (p *CSVSwiftParser) ParseSwiftData(input io.Reader) ([]models.SwiftBank, error) {
	reader := csv.NewReader(input)

	// Set limits to prevent potential DoS
	reader.FieldsPerRecord = -1 // Allow variable number of fields
	reader.LazyQuotes = true    // Handle quotes more flexibly

	// Read header line
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Verify header matches expected format
	expectedHeader := []string{"COUNTRY ISO2 CODE", "SWIFT CODE", "CODE TYPE", "NAME"}
	if len(header) < len(expectedHeader) {
		return nil, ErrHeaderInsufficient
	}

	var swiftBanks []models.SwiftBank
	now := time.Now().UTC() // Use UTC for consistency
	var lineNumber int = 1  // For error reporting (header is line 1)

	// Track unique SWIFT codes to prevent duplicates
	uniqueCodes := make(map[string]bool)

	// Parse all records
	for {
		lineNumber++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading line %d: %w", lineNumber, err)
		}

		if len(record) < 4 { // We need at least 4 essential columns
			return nil, fmt.Errorf("%w at line %d", ErrRecordInsufficient, lineNumber)
		}

		// Extract essential data from record
		countryISOCode := strings.ToUpper(strings.TrimSpace(record[0]))
		swiftCode := strings.ToUpper(strings.TrimSpace(record[1]))
		bankName := sanitizeBankName(record[3])

		// Validate essential fields
		if countryISOCode == "" || swiftCode == "" || bankName == "" {
			return nil, fmt.Errorf("%w at line %d", ErrMissingRequiredField, lineNumber)
		}

		// Skip already processed SWIFT codes (prevent duplicates)
		if uniqueCodes[swiftCode] {
			continue
		}

		// Determine entity type and HQ base
		var entityType models.SwiftCodeEntity

		// Validate SWIFT code format
		if err := validateSwiftCode(swiftCode); err != nil {
			return nil, fmt.Errorf("at line %d: %w", lineNumber, err)
		}

		// Validate country code
		if err := validateCountryCode(countryISOCode); err != nil {
			return nil, fmt.Errorf("at line %d: %w", lineNumber, err)
		}

		// Extract the first 8 chars as the HQ base
		hqSwiftBase := swiftCode[:8]

		// Determine if this is a headquarters (ending with XXX) or branch
		if strings.HasSuffix(swiftCode, "XXX") {
			entityType = models.Headquarters
		} else {
			entityType = models.Branch
		}

		swiftBank := models.SwiftBank{
			SwiftCode:      swiftCode,
			HQSwiftBase:    hqSwiftBase,
			CountryISOCode: countryISOCode,
			BankName:       bankName,
			EntityType:     entityType,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		// Comprehensive validation
		if err := validateSwiftBankEntry(&swiftBank); err != nil {
			return nil, fmt.Errorf("validation failed at line %d: %w", lineNumber, err)
		}

		// Mark this SWIFT code as processed
		uniqueCodes[swiftCode] = true

		swiftBanks = append(swiftBanks, swiftBank)
	}

	// Final verification: ensure we have at least some valid data
	if len(swiftBanks) == 0 {
		return nil, errors.New("no valid SWIFT bank entries found in input")
	}

	return swiftBanks, nil
}
