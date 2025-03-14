package mocks

import (
	"context"
	"errors"

	models "github.com/zdziszkee/swift-codes/internal/models"
	repository "github.com/zdziszkee/swift-codes/internal/repositories"
)

// MockSwiftRepository implements the SwiftRepository interface for testing
type MockSwiftRepository struct {
	GetByCodeFunc           func(ctx context.Context, code string) (*repository.SwiftBankDetail, error)
	GetByCountryFunc        func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error)
	CreateFunc              func(ctx context.Context, bank *models.SwiftBank) error
	CreateBatchFunc         func(ctx context.Context, banks []*models.SwiftBank) error
	DeleteFunc              func(ctx context.Context, code string) error
	GetBranchesByHQBaseFunc func(ctx context.Context, hqBase string) ([]models.SwiftBank, error)
	LoadCSVFunc             func(ctx context.Context, file string) error
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

func (m *MockSwiftRepository) LoadCSV(ctx context.Context, file string) error {
	if m.LoadCSVFunc != nil {
		return m.LoadCSVFunc(ctx, file)
	}
	return errors.New("LoadCSV not implemented")
}
