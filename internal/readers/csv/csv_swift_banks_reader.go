package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	readers "github.com/zdziszkee/swift-codes/internal/readers"
)

type CSVSwiftBanksReader struct {
}

const expectedHeader = "COUNTRY ISO2 CODE,SWIFT CODE,CODE TYPE,NAME,ADDRESS,TOWN NAME,COUNTRY NAME,TIME ZONE"

func (c *CSVSwiftBanksReader) LoadSwiftBanks(reader io.Reader) ([]readers.SwiftBankRecord, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	csvReader.ReuseRecord = true

	header, err := csvReader.Read()
	if err != nil {
		if err == io.EOF {
			return []readers.SwiftBankRecord{}, nil
		}
		return nil, fmt.Errorf("read header: %w", err)
	}

	// Hardcoded header validation
	expectedHeaders := strings.Split(expectedHeader, ",") // Split the string into a slice
	if len(header) != len(expectedHeaders) {
		return nil, fmt.Errorf("invalid header length: expected %d, got %d", len(expectedHeaders), len(header)) // Use expectedHeaders length
	}
	for i, col := range header {
		expectedCol := expectedHeaders[i] // Access element from the slice
		// Case-insensitive and space-trimmed comparison
		if strings.TrimSpace(strings.ToUpper(col)) != strings.TrimSpace(strings.ToUpper(expectedCol)) {
			return nil, fmt.Errorf("invalid header: expected '%s' at index %d, got '%s'", expectedCol, i, col)
		}
	}
	headerMap := map[string]int{}
	for i, col := range header {
		headerMap[strings.ToUpper(col)] = i
	}

	var records []readers.SwiftBankRecord
	rowNum := 1
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		if len(row) != 5 {
			return nil, fmt.Errorf("row %d: invalid length", rowNum)
		}

		getVal := func(field string) string {
			return strings.TrimSpace(row[headerMap[strings.ToUpper(field)]])
		}

		swiftCode := getVal("SWIFT CODE")
		countryISO2 := getVal("COUNTRY ISO2 CODE")
		bankName := getVal("NAME")
		address := getVal("ADDRESS")
		countryName := getVal("COUNTRY NAME")

		records = append(records, readers.SwiftBankRecord{
			Index:          rowNum,
			SwiftCode:      swiftCode,
			BankName:       bankName,
			CountryISOCode: countryISO2,
			Address:        address,
			CountryName:    countryName,
		})
		rowNum++
	}

	return records, nil
}
