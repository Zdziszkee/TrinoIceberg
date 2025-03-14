// File: internal/loader/csv_loader.go
package reader

import (
	"io"
)

type SwiftBankRecord struct {
	Index          int
	CountryISOCode string // COUNTRY ISO2 CODE
	SwiftCode      string // SWIFT CODE
	BankName       string // NAME
	Address        string // ADDRESS
	CountryName    string // COUNTRY NAME
}

// SwiftBanksLoader defines the interface for loading bank data
type SwiftBanksReader interface {
	LoadSwiftBanks(reader io.Reader) ([]SwiftBankRecord, error) // Changed to accept io.Reader and return []models.SwiftBank
}
