package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zdziszkee/swift-codes/internal/models"
	repository "github.com/zdziszkee/swift-codes/internal/repositories"
)

// MockSwiftRepository updated to include CreateBatch and GetBranchesByHQBase
type MockSwiftRepository struct {
	GetByCodeFunc           func(ctx context.Context, code string) (*repository.SwiftBankDetail, error)
	GetByCountryFunc        func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error)
	CreateFunc              func(ctx context.Context, bank *models.SwiftBank) error
	CreateBatchFunc         func(ctx context.Context, banks []*models.SwiftBank) error // Added missing method
	DeleteFunc              func(ctx context.Context, code string) error
	GetBranchesByHQBaseFunc func(ctx context.Context, hqBase string) ([]models.SwiftBank, error) // Updated function pointer signature to match interface
}

func (m *MockSwiftRepository) GetByCode(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
	return m.GetByCodeFunc(ctx, code)
}

func (m *MockSwiftRepository) GetByCountry(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
	return m.GetByCountryFunc(ctx, countryCode)
}

func (m *MockSwiftRepository) Create(ctx context.Context, bank *models.SwiftBank) error {
	return m.CreateFunc(ctx, bank)
}

func (m *MockSwiftRepository) CreateBatch(ctx context.Context, banks []*models.SwiftBank) error {
	return m.CreateBatchFunc(ctx, banks)
}

func (m *MockSwiftRepository) Delete(ctx context.Context, code string) error {
	return m.DeleteFunc(ctx, code)
}

func (m *MockSwiftRepository) GetBranchesByHQBase(ctx context.Context, hqBase string) ([]models.SwiftBank, error) {
	if m.GetBranchesByHQBaseFunc != nil {
		return m.GetBranchesByHQBaseFunc(ctx, hqBase)
	}
	return nil, errors.New("GetBranchesByHQBase not implemented")
}

func TestGetSwiftCodeDetails(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		code    string
		repo    *MockSwiftRepository
		want    *repository.SwiftBankDetail
		wantErr error
	}{
		{
			name: "Valid SWIFT code",
			code: "ABCDUS33XXX",
			repo: &MockSwiftRepository{
				GetByCodeFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
					return &repository.SwiftBankDetail{}, nil // Adjust fields as per your struct
				},
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			want:    &repository.SwiftBankDetail{}, // Adjust fields as per your struct
			wantErr: nil,
		},
		{
			name: "Invalid SWIFT code",
			code: "ABC123",
			repo: &MockSwiftRepository{
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name: "Not found",
			code: "ABCDUS33XXX",
			repo: &MockSwiftRepository{
				GetByCodeFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
					return nil, repository.ErrNotFound
				},
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			want:    nil,
			wantErr: ErrNotFound,
		},
		{
			name: "Repository error",
			code: "ABCDUS33XXX",
			repo: &MockSwiftRepository{
				GetByCodeFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
					return nil, errors.New("db error")
				},
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			want:    nil,
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSwiftService(tt.repo)
			got, err := s.GetSwiftCodeDetails(ctx, tt.code)
			if (err != nil) != (tt.wantErr != nil) || (err != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("GetSwiftCodeDetails() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want { // Note: For real comparison, use reflect.DeepEqual or specific fields
				t.Errorf("GetSwiftCodeDetails() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSwiftCodesByCountry(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		countryCode string
		repo        *MockSwiftRepository
		want        *repository.CountrySwiftCodes
		wantErr     error
	}{
		{
			name:        "Valid country code",
			countryCode: "US",
			repo: &MockSwiftRepository{
				GetByCountryFunc: func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
					return &repository.CountrySwiftCodes{}, nil // Adjust fields
				},
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			want:    &repository.CountrySwiftCodes{}, // Adjust fields
			wantErr: nil,
		},
		{
			name:        "Invalid country code",
			countryCode: "USA",
			repo: &MockSwiftRepository{
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:        "Not found",
			countryCode: "US",
			repo: &MockSwiftRepository{
				GetByCountryFunc: func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
					return nil, repository.ErrNotFound
				},
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			want:    nil,
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSwiftService(tt.repo)
			got, err := s.GetSwiftCodesByCountry(ctx, tt.countryCode)
			if (err != nil) != (tt.wantErr != nil) || (err != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("GetSwiftCodesByCountry() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("GetSwiftCodesByCountry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateSwiftCode(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		bank    *models.SwiftBank
		repo    *MockSwiftRepository
		wantErr error
	}{
		{
			name: "Valid bank",
			bank: &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "US", BankName: "Test Bank"},
			repo: &MockSwiftRepository{
				CreateFunc:      func(ctx context.Context, bank *models.SwiftBank) error { return nil },
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			wantErr: nil,
		},
		{
			name: "Invalid SWIFT code",
			bank: &models.SwiftBank{SwiftCode: "ABC123", CountryISOCode: "US", BankName: "Test Bank"},
			repo: &MockSwiftRepository{
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "Invalid country code",
			bank: &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "USA", BankName: "Test Bank"},
			repo: &MockSwiftRepository{
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "Empty bank name",
			bank: &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "US", BankName: ""},
			repo: &MockSwiftRepository{
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "Duplicate SWIFT code",
			bank: &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "US", BankName: "Test Bank"},
			repo: &MockSwiftRepository{
				CreateFunc:      func(ctx context.Context, bank *models.SwiftBank) error { return repository.ErrDuplicate },
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			wantErr: ErrAlreadyExists,
		},
		{
			name: "Nil bank",
			bank: nil,
			repo: &MockSwiftRepository{
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			wantErr: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSwiftService(tt.repo)
			err := s.CreateSwiftCode(ctx, tt.bank)
			if (err != nil) != (tt.wantErr != nil) || (err != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("CreateSwiftCode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.bank != nil {
				if tt.bank.SwiftCode != strings.ToUpper(tt.bank.SwiftCode) ||
					tt.bank.CountryISOCode != strings.ToUpper(tt.bank.CountryISOCode) ||
					tt.bank.IsHeadquarter != strings.HasSuffix(tt.bank.SwiftCode, "XXX") ||
					tt.bank.SwiftCodeBase != tt.bank.SwiftCode[:8] {
					t.Errorf("CreateSwiftCode() did not transform bank correctly: %v", tt.bank)
				}
			}
		})
	}
}

func TestDeleteSwiftCode(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		code    string
		repo    *MockSwiftRepository
		wantErr error
	}{
		{
			name: "Valid deletion",
			code: "ABCDUS33XXX",
			repo: &MockSwiftRepository{
				DeleteFunc:      func(ctx context.Context, code string) error { return nil },
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			wantErr: nil,
		},
		{
			name: "Invalid SWIFT code",
			code: "ABC123",
			repo: &MockSwiftRepository{
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "Not found",
			code: "ABCDUS33XXX",
			repo: &MockSwiftRepository{
				DeleteFunc:      func(ctx context.Context, code string) error { return repository.ErrNotFound },
				CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
			},
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSwiftService(tt.repo)
			err := s.DeleteSwiftCode(ctx, tt.code)
			if (err != nil) != (tt.wantErr != nil) || (err != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("DeleteSwiftCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
