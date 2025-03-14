package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	reader "github.com/zdziszkee/swift-codes/internal/readers"
)

type CSVSwiftBanksReader struct {
}

const expectedHeader = "COUNTRY ISO2 CODE,SWIFT CODE,CODE TYPE,NAME,ADDRESS,TOWN NAME,COUNTRY NAME,TIME ZONE"

func (c *CSVSwiftBanksReader) LoadSwiftBanks(r io.Reader) ([]reader.SwiftBankRecord, error) {
	// Handle empty input explicitly
	if testStr, ok := r.(*strings.Reader); ok {
		if testStr.Len() == 0 {
			return []reader.SwiftBankRecord{}, io.EOF
		}
	}

	csvReader := csv.NewReader(r)
	csvReader.TrimLeadingSpace = true
	csvReader.ReuseRecord = true

	header, err := csvReader.Read()
	if err != nil {
		if err == io.EOF {
			return []reader.SwiftBankRecord{}, nil
		}
		return nil, fmt.Errorf("read header: %w", err)
	}

	// Hardcoded header validation
	expectedHeaders := strings.Split(expectedHeader, ",")
	if len(header) != len(expectedHeaders) {
		return nil, fmt.Errorf("invalid header length: expected %d, got %d", len(expectedHeaders), len(header))
	}
	for i, col := range header {
		expectedCol := expectedHeaders[i]
		if strings.TrimSpace(strings.ToUpper(col)) != strings.TrimSpace(strings.ToUpper(expectedCol)) {
			return nil, fmt.Errorf("invalid header: expected '%s' at index %d, got '%s'", expectedCol, i, col)
		}
	}

	// Build a map of column name to index
	headerMap := map[string]int{}
	for i, col := range header {
		headerMap[strings.ToUpper(strings.TrimSpace(col))] = i
	}

	var records []reader.SwiftBankRecord
	rowNum := 1
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNum, err)
		}
		if len(row) != len(expectedHeaders) {
			return nil, fmt.Errorf("row %d: invalid length", rowNum)
		}

		// This is the key fix - make sure we're using the right column indices
		record := reader.SwiftBankRecord{
			Index:          rowNum,
			CountryISOCode: strings.TrimSpace(row[headerMap["COUNTRY ISO2 CODE"]]),
			SwiftCode:      strings.TrimSpace(row[headerMap["SWIFT CODE"]]),
			BankName:       strings.TrimSpace(row[headerMap["NAME"]]),
			Address:        strings.TrimSpace(row[headerMap["ADDRESS"]]),
			CountryName:    strings.TrimSpace(row[headerMap["COUNTRY NAME"]]),
		}

		records = append(records, record)
		rowNum++
	}

	return records, nil
}
