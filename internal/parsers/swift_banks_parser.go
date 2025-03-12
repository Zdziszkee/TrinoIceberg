package parser

import (
	"log"
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
			log.Printf("Validation error at index %d: SwiftCode cannot be empty", record.Index)
			continue
		}
		if !bicRegex.MatchString(record.SwiftCode) {
			log.Printf("Validation error at index %d: SwiftCode '%s' does not match BIC format", record.Index, record.SwiftCode)
			continue
		}
		if len(record.SwiftCode) > 15 { // Example: Max length for SwiftCode
			log.Printf("Validation error at index %d: SwiftCode '%s' exceeds maximum length", record.Index, record.SwiftCode)
			continue
		}

		if record.BankName == "" {
			log.Printf("Validation error for SwiftCode '%s': BankName cannot be empty", record.SwiftCode)
			continue
		}
		if len(record.BankName) > 100 { // Example: Max length for BankName
			log.Printf("Validation error for SwiftCode '%s': BankName '%s' exceeds maximum length", record.SwiftCode, record.BankName)
			continue
		}

		if record.CountryISOCode == "" {
			log.Printf("Validation error for SwiftCode '%s': CountryISOCode cannot be empty", record.SwiftCode)
			continue
		}
		if !countryCodeRegex.MatchString(record.CountryISOCode) {
			log.Printf("Validation error for Bank '%s': CountryISOCode '%s' does not match ISO2 format", record.BankName, record.CountryISOCode)
			continue
		}

		if record.Address == "" {
			log.Printf("Validation error for SwiftCode '%s': Address cannot be empty", record.SwiftCode)
			continue
		}
		if len(record.Address) > 200 { // Example: Max length for Address
			log.Printf("Validation error for SwiftCode '%s': Address exceeds maximum length", record.SwiftCode)
			continue
		}

		if record.CountryName == "" {
			log.Printf("Validation error for SwiftCode '%s': CountryName cannot be empty", record.SwiftCode)
			continue
		}
		if len(record.CountryName) > 100 { // Example: Max length for CountryName
			log.Printf("Validation error for SwiftCode '%s': CountryName '%s' exceeds maximum length", record.SwiftCode, record.BankName)
			continue
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
