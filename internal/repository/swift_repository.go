package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zdziszkee/swift-codes/internal/database"
	model "github.com/zdziszkee/swift-codes/internal/model"
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
	return &SQLSwiftRepository{db: db.DB} // Ensures Trino driver from database package
}

// CreateBatch inserts multiple SWIFT banks in a single Trino query
func (r *SQLSwiftRepository) CreateBatch(ctx context.Context, banks []*model.SwiftBank) error {
	if len(banks) == 0 {
		return nil
	}

	// Single INSERT into main table (no staging for small datasets)
	query := fmt.Sprintf("INSERT INTO %s (swift_code, hq_swift_base, country_iso_code, bank_name, entity_type, created_at, updated_at) VALUES ", r.tableName())
	values := make([]interface{}, 0, len(banks)*7)
	placeholders := make([]string, 0, len(banks))
	now := time.Now()

	for i, bank := range banks {
		bank.SwiftCode = strings.ToUpper(bank.SwiftCode)
		bank.CountryISOCode = strings.ToUpper(bank.CountryISOCode)
		if bank.HQSwiftBase == "" {
			bank.HQSwiftBase = bank.SwiftCode[:8]
		}
		bank.CreatedAt = now
		bank.UpdatedAt = now
		startIdx := i * 7
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", startIdx+1, startIdx+2, startIdx+3, startIdx+4, startIdx+5, startIdx+6, startIdx+7))
		values = append(values, bank.SwiftCode, bank.HQSwiftBase, bank.CountryISOCode, bank.BankName, bank.EntityType, bank.CreatedAt, bank.UpdatedAt)
	}

	query += strings.Join(placeholders, ",")
	fmt.Printf("Executing Trino batch INSERT with %d rows: %s\n", len(banks), query[:200]) // Log for verification
	start := time.Now()
	_, err := r.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("trino batch insert failed: %v", err)
	}
	fmt.Printf("Completed Trino batch INSERT of %d rows in %v\n", len(banks), time.Since(start))

	// No staging table move needed for 1,000 rows
	return nil
}

// Create adds a single SWIFT bank to the database
func (r *SQLSwiftRepository) Create(ctx context.Context, bank *model.SwiftBank) error {
	if err := r.checkDuplicate(ctx, bank.SwiftCode); err != nil {
		return err
	}

	bank.SwiftCode = strings.ToUpper(bank.SwiftCode)
	bank.CountryISOCode = strings.ToUpper(bank.CountryISOCode)
	if bank.HQSwiftBase == "" {
		bank.HQSwiftBase = bank.SwiftCode[:8]
	}

	now := time.Now()
	bank.CreatedAt = now
	bank.UpdatedAt = now

	query := fmt.Sprintf("INSERT INTO %s (swift_code, hq_swift_base, country_iso_code, bank_name, entity_type, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)", r.tableName())
	_, err := r.db.ExecContext(ctx, query,
		bank.SwiftCode,
		bank.HQSwiftBase,
		bank.CountryISOCode,
		bank.BankName,
		bank.EntityType,
		bank.CreatedAt,
		bank.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("trino insert failed: %w", err)
	}
	return nil
}

// LoadCSV loads data from a CSV file using Trino's COPY (adjusted for Trino compatibility)
func (r *SQLSwiftRepository) LoadCSV(ctx context.Context, csvPath string) error {
	// Trino doesn't natively support COPY; use INSERT FROM EXTERNAL instead if available
	// For now, assume CSV is loaded via app logic or staging table
	return fmt.Errorf("LoadCSV not implemented for Trino; use CreateBatch instead")
}

// GetByCode retrieves a SWIFT bank and its branches if it's a headquarters
func (r *SQLSwiftRepository) GetByCode(ctx context.Context, code string) (*SwiftBankDetail, error) {
	bank, err := r.getBankByCode(ctx, strings.ToUpper(code))
	if err != nil {
		return nil, err
	}

	result := &SwiftBankDetail{Bank: *bank}

	if bank.EntityType == model.Headquarters {
		branches, err := r.GetBranchesByHQBase(ctx, bank.HQSwiftBase)
		if err != nil {
			return nil, fmt.Errorf("trino fetch branches failed: %w", err)
		}
		result.Branches = branches
	}

	return result, nil
}

// GetBranchesByHQBase retrieves all branches for a headquarters
func (r *SQLSwiftRepository) GetBranchesByHQBase(ctx context.Context, hqBase string) ([]model.SwiftBank, error) {
	query := fmt.Sprintf("SELECT swift_code, bank_name, country_iso_code, entity_type, hq_swift_base, created_at, updated_at FROM %s WHERE hq_swift_base = $1 AND entity_type = $2", r.tableName())
	rows, err := r.db.QueryContext(ctx, query, hqBase, model.Branch)
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
	bankName, err := r.getSampleBankName(ctx, countryCode)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT swift_code, bank_name, country_iso_code, entity_type, hq_swift_base, created_at, updated_at FROM %s WHERE country_iso_code = $1", r.tableName())
	rows, err := r.db.QueryContext(ctx, query, countryCode)
	if err != nil {
		return nil, fmt.Errorf("trino query failed: %w", err)
	}
	defer rows.Close()

	result := &CountrySwiftCodes{
		CountryISO2: countryCode,
		CountryName: bankName, // Consider a proper country name mapping
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

	query := fmt.Sprintf("DELETE FROM %s WHERE swift_code = $1", r.tableName())
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
	query := fmt.Sprintf("SELECT swift_code, bank_name, country_iso_code, entity_type, hq_swift_base, created_at, updated_at FROM %s WHERE swift_code = $1", r.tableName())
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

func (r *SQLSwiftRepository) getSampleBankName(ctx context.Context, countryCode string) (string, error) {
	query := fmt.Sprintf("SELECT bank_name FROM %s WHERE country_iso_code = $1 LIMIT 1", r.tableName())
	var bankName string
	err := r.db.QueryRowContext(ctx, query, countryCode).Scan(&bankName)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("trino query failed: %w", err)
	}
	return bankName, nil
}

func (r *SQLSwiftRepository) checkDuplicate(ctx context.Context, code string) error {
	query := fmt.Sprintf("SELECT 1 FROM %s WHERE swift_code = $1 LIMIT 1", r.tableName())
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
	query := fmt.Sprintf("SELECT 1 FROM %s WHERE swift_code = $1 LIMIT 1", r.tableName())
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
	var createdAt, updatedAt string

	err := scanner.Scan(
		&bank.SwiftCode,
		&bank.BankName,
		&bank.CountryISOCode,
		&bank.EntityType,
		&bank.HQSwiftBase,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse timestamps (Trino returns RFC3339 strings)
	bank.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	bank.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &bank, nil
}
