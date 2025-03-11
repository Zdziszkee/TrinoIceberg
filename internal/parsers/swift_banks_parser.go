package parser

import (
	"fmt"
	"regexp"
	"strings"

	models "github.com/zdziszkee/swift-codes/internal/models"
	readers "github.com/zdziszkee/swift-codes/internal/readers"
)

type SwiftBanksParser interface {
	ParseSwiftBanks(swiftBankRecords []readers.SwiftBankRecord) ([]models.SwiftBank, error)
}

type DefaultSwiftBanksParser struct{}

func (p DefaultSwiftBanksParser) ParseSwiftBanks(swiftBankRecords []readers.SwiftBankRecord) ([]models.SwiftBank, error) {
	var banks []models.SwiftBank
	bicRegex := regexp.MustCompile(`^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$`) // BIC format regex
	countryCodeRegex := regexp.MustCompile(`^[A-Z]{2}$`)                  // ISO2 country code regex

	for _, record := range swiftBankRecords {
		// --- Enhanced Content Validations ---
		if record.SwiftCode == "" {
			return nil, fmt.Errorf("validation error: SwiftCode cannot be empty for record with Index %d", record.Index)
		}
		if !bicRegex.MatchString(record.SwiftCode) {
			return nil, fmt.Errorf("validation error: SwiftCode '%s' at Index %d does not match BIC format", record.SwiftCode, record.Index)
		}
		if len(record.SwiftCode) > 15 { // Example: Max length for SwiftCode
			return nil, fmt.Errorf("validation error: SwiftCode '%s' at Index %d exceeds maximum length", record.SwiftCode, record.Index)
		}

		if record.BankName == "" {
			return nil, fmt.Errorf("validation error: BankName cannot be empty for SwiftCode '%s'", record.SwiftCode)
		}
		if len(record.BankName) > 100 { // Example: Max length for BankName
			return nil, fmt.Errorf("validation error: BankName '%s' for SwiftCode '%s' exceeds maximum length", record.BankName, record.SwiftCode)
		}

		if record.CountryISOCode == "" {
			return nil, fmt.Errorf("validation error: CountryISOCode cannot be empty for SwiftCode '%s'", record.SwiftCode)
		}
		if !countryCodeRegex.MatchString(record.CountryISOCode) {
			return nil, fmt.Errorf("validation error: CountryISOCode '%s' for Bank '%s' does not match ISO2 format", record.CountryISOCode, record.BankName)
		}

		if record.Address == "" {
			return nil, fmt.Errorf("validation error: Address cannot be empty for SwiftCode '%s'", record.SwiftCode)
		}
		if len(record.Address) > 200 { // Example: Max length for Address
			return nil, fmt.Errorf("validation error: Address for SwiftCode '%s' exceeds maximum length", record.SwiftCode)
		}

		if record.CountryName == "" {
			return nil, fmt.Errorf("validation error: CountryName cannot be empty for SwiftCode '%s'", record.SwiftCode)
		}
		if len(record.CountryName) > 100 { // Example: Max length for CountryName
			return nil, fmt.Errorf("validation error: CountryName '%s' for Bank '%s' exceeds maximum length", record.CountryName, record.BankName)
		}

		// --- Determine IsHeadquarter in Parser ---
		isHeadquarter := strings.HasSuffix(record.SwiftCode, "XXX") // Check for "XXX" suffix

		// --- Conversion to models.SwiftBank ---
		swiftCodeBase := "" // Calculate SwiftCodeBase
		if len(record.SwiftCode) >= 8 {
			swiftCodeBase = record.SwiftCode[:8]
		} else {
			swiftCodeBase = record.SwiftCode
		}

		bank := models.SwiftBank{
			SwiftCode:      record.SwiftCode,
			SwiftCodeBase:  swiftCodeBase,
			CountryISOCode: record.CountryISOCode,
			BankName:       record.BankName,
			IsHeadquarter:  isHeadquarter,
			Address:        record.Address,
			CountryName:    record.CountryName,
		}
		banks = append(banks, bank)
	}

	return banks, nil
}
