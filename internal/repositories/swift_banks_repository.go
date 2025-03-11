package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zdziszkee/swift-codes/internal/database"
	model "github.com/zdziszkee/swift-codes/internal/models"
)

var (
	ErrNotFound    = errors.New("swift code not found")
	ErrDuplicate   = errors.New("swift code already exists")
	ErrInvalidData = errors.New("invalid data provided")
)

// SwiftBankDetail represents detailed bank information including branches
type SwiftBankDetail struct {
	Bank     model.SwiftBank   `json:"bank"`
	Branches []model.SwiftBank `json:"branches,omitempty"`
}

// CountrySwiftCodes holds all SWIFT codes for a specific country
type CountrySwiftCodes struct {
	CountryISO2 string            `json:"country_iso2"`
	CountryName string            `json:"country_name"`
	SwiftCodes  []model.SwiftBank `json:"swift_codes"`
}

// SwiftRepository defines the interface for SWIFT code data operations
type SwiftRepository interface {
	GetByCode(ctx context.Context, code string) (*SwiftBankDetail, error)
	GetByCountry(ctx context.Context, countryCode string) (*CountrySwiftCodes, error)
	Create(ctx context.Context, bank *model.SwiftBank) error
	CreateBatch(ctx context.Context, banks []*model.SwiftBank) error
	Delete(ctx context.Context, code string) error
	GetBranchesByHQBase(ctx context.Context, hqBase string) ([]model.SwiftBank, error)
	LoadCSV(ctx context.Context, csvPath string) error
}

// SQLSwiftRepository implements SwiftRepository using Trino via database/sql
type SQLSwiftRepository struct {
	db *sql.DB
}

// NewSQLSwiftRepository creates a new repository instance with Trino
func NewSQLSwiftRepository(db *database.Database) SwiftRepository {
	return &SQLSwiftRepository{db: db.DB}
}

const batchSize = 100

// CreateBatch inserts multiple SWIFT banks in batches using parameterized queries
func (r *SQLSwiftRepository) CreateBatch(ctx context.Context, banks []*model.SwiftBank) error {
	if len(banks) == 0 {
		return nil
	}

	totalRows := len(banks)
	insertedRows := 0

	for i := 0; i < totalRows; i += batchSize {
		endIdx := i + batchSize
		if endIdx > totalRows {
			endIdx = totalRows
		}
		batch := banks[i:endIdx]

		// Build parameterized INSERT query
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("INSERT INTO %s (swift_code, swift_code_base, country_iso_code, bank_name, is_headquarter, address, country_name) VALUES ", r.tableName()))
		placeholders := make([]string, 0, len(batch))
		args := make([]interface{}, 0, len(batch)*7)

		for _, bank := range batch {
			bank.SwiftCode = strings.ToUpper(bank.SwiftCode)
			bank.CountryISOCode = strings.ToUpper(bank.CountryISOCode)
			if bank.SwiftCodeBase == "" {
				bank.SwiftCodeBase = bank.SwiftCode[:8]
			}

			placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?)")
			args = append(args,
				bank.SwiftCode,
				bank.SwiftCodeBase,
				bank.CountryISOCode,
				bank.BankName,
				bank.IsHeadquarter,
				bank.Address,
				bank.CountryName,
			)
		}

		sb.WriteString(strings.Join(placeholders, ","))
		query := sb.String()

		fmt.Printf("Executing Trino batch INSERT with %d rows: %s\n", len(batch), query[:min(200, len(query))])
		start := time.Now()
		result, err := r.db.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("trino batch insert failed for batch %d-%d: %v (query: %s)", i+1, endIdx, err, query[:min(500, len(query))])
		}
		rowsAffected, _ := result.RowsAffected()
		insertedRows += int(rowsAffected)
		fmt.Printf("Completed Trino batch INSERT of %d rows in %v\n", len(batch), time.Since(start))
	}

	fmt.Printf("Successfully loaded %d SWIFT codes\n", insertedRows)
	return nil
}

// Create adds a single SWIFT bank to the database
func (r *SQLSwiftRepository) Create(ctx context.Context, bank *model.SwiftBank) error {
	if err := r.checkDuplicate(ctx, bank.SwiftCode); err != nil {
		return err
	}

	bank.SwiftCode = strings.ToUpper(bank.SwiftCode)
	bank.CountryISOCode = strings.ToUpper(bank.CountryISOCode)
	if bank.SwiftCodeBase == "" {
		bank.SwiftCodeBase = bank.SwiftCode[:8]
	}

	query := fmt.Sprintf("INSERT INTO %s (swift_code, swift_code_base, country_iso_code, bank_name, is_headquarter, address, country_name) VALUES (?, ?, ?, ?, ?, ?, ?)", r.tableName())
	_, err := r.db.ExecContext(ctx, query,
		bank.SwiftCode,
		bank.SwiftCodeBase,
		bank.CountryISOCode,
		bank.BankName,
		bank.IsHeadquarter,
		bank.Address,
		bank.CountryName,
	)
	if err != nil {
		return fmt.Errorf("trino insert failed: %w", err)
	}
	return nil
}

// LoadCSV is a placeholder
func (r *SQLSwiftRepository) LoadCSV(ctx context.Context, csvPath string) error {
	return fmt.Errorf("LoadCSV not implemented for Trino; use CreateBatch instead")
}

// GetByCode retrieves a SWIFT bank and its branches if it's a headquarters
func (r *SQLSwiftRepository) GetByCode(ctx context.Context, code string) (*SwiftBankDetail, error) {
	bank, err := r.getBankByCode(ctx, strings.ToUpper(code))
	if err != nil {
		return nil, err
	}

	result := &SwiftBankDetail{Bank: *bank}

	if bank.IsHeadquarter {
		branches, err := r.GetBranchesByHQBase(ctx, bank.SwiftCodeBase)
		if err != nil {
			return nil, fmt.Errorf("trino fetch branches failed: %w", err)
		}
		result.Branches = branches
	}

	return result, nil
}

// GetBranchesByHQBase retrieves all branches for a headquarters
func (r *SQLSwiftRepository) GetBranchesByHQBase(ctx context.Context, hqBase string) ([]model.SwiftBank, error) {
	query := fmt.Sprintf("SELECT swift_code, swift_code_base, country_iso_code, bank_name, is_headquarter, address, country_name FROM %s WHERE swift_code_base = ? AND is_headquarter = false", r.tableName())
	rows, err := r.db.QueryContext(ctx, query, hqBase)
	if err != nil {
		return nil, fmt.Errorf("trino query failed: %w", err)
	}
	defer rows.Close()

	var branches []model.SwiftBank
	for rows.Next() {
		branch, err := scanBank(rows)
		if err != nil {
			return nil, fmt.Errorf("trino scan failed: %w", err)
		}
		branches = append(branches, *branch)
	}

	return branches, rows.Err()
}

// GetByCountry retrieves all SWIFT banks for a country
func (r *SQLSwiftRepository) GetByCountry(ctx context.Context, countryCode string) (*CountrySwiftCodes, error) {
	countryCode = strings.ToUpper(countryCode)
	countryName, err := r.getCountryName(ctx, countryCode)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT swift_code, swift_code_base, country_iso_code, bank_name, is_headquarter, address, country_name FROM %s WHERE country_iso_code = ?", r.tableName())
	rows, err := r.db.QueryContext(ctx, query, countryCode)
	if err != nil {
		return nil, fmt.Errorf("trino query failed: %w", err)
	}
	defer rows.Close()

	result := &CountrySwiftCodes{
		CountryISO2: countryCode,
		CountryName: countryName,
	}

	for rows.Next() {
		bank, err := scanBank(rows)
		if err != nil {
			return nil, fmt.Errorf("trino scan failed: %w", err)
		}
		result.SwiftCodes = append(result.SwiftCodes, *bank)
	}

	return result, rows.Err()
}

// Delete removes a SWIFT bank from the database
func (r *SQLSwiftRepository) Delete(ctx context.Context, code string) error {
	code = strings.ToUpper(code)
	if err := r.checkExists(ctx, code); err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE swift_code = ?", r.tableName())
	_, err := r.db.ExecContext(ctx, query, code)
	if err != nil {
		return fmt.Errorf("trino delete failed: %w", err)
	}

	return nil
}

// Helper methods

func (r *SQLSwiftRepository) tableName() string {
	return "swift_catalog.default_schema.swift_banks"
}

func (r *SQLSwiftRepository) getBankByCode(ctx context.Context, code string) (*model.SwiftBank, error) {
	query := fmt.Sprintf("SELECT swift_code, swift_code_base, country_iso_code, bank_name, is_headquarter, address, country_name FROM %s WHERE swift_code = ?", r.tableName())
	row := r.db.QueryRowContext(ctx, query, code)
	bank, err := scanBank(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("trino query failed: %w", err)
	}
	return bank, nil
}

func (r *SQLSwiftRepository) getCountryName(ctx context.Context, countryCode string) (string, error) {
	query := fmt.Sprintf("SELECT country_name FROM %s WHERE country_iso_code = ? LIMIT 1", r.tableName())
	var countryName string
	err := r.db.QueryRowContext(ctx, query, countryCode).Scan(&countryName)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("trino query failed: %w", err)
	}
	return countryName, nil
}

func (r *SQLSwiftRepository) checkDuplicate(ctx context.Context, code string) error {
	query := fmt.Sprintf("SELECT 1 FROM %s WHERE swift_code = ? LIMIT 1", r.tableName())
	var exists int
	err := r.db.QueryRowContext(ctx, query, strings.ToUpper(code)).Scan(&exists)
	if err == nil {
		return ErrDuplicate
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("trino check duplicate failed: %w", err)
	}
	return nil
}

func (r *SQLSwiftRepository) checkExists(ctx context.Context, code string) error {
	query := fmt.Sprintf("SELECT 1 FROM %s WHERE swift_code = ? LIMIT 1", r.tableName())
	var exists int
	err := r.db.QueryRowContext(ctx, query, code).Scan(&exists)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("trino check exists failed: %w", err)
	}
	return nil
}

func scanBank(scanner interface {
	Scan(dest ...any) error
}) (*model.SwiftBank, error) {
	var bank model.SwiftBank

	err := scanner.Scan(
		&bank.SwiftCode,
		&bank.SwiftCodeBase,
		&bank.CountryISOCode,
		&bank.BankName,
		&bank.IsHeadquarter,
		&bank.Address,
		&bank.CountryName,
	)
	if err != nil {
		return nil, err
	}

	return &bank, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
