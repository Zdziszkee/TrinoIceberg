package mocks

import (
	"context"

	models "github.com/zdziszkee/swift-codes/internal/models"
	repository "github.com/zdziszkee/swift-codes/internal/repositories"
)

// MockSwiftService implements service.SwiftService.
type MockSwiftService struct {
	GetSwiftCodeDetailsFunc    func(ctx context.Context, code string) (*repository.SwiftBankDetail, error)
	GetSwiftCodesByCountryFunc func(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error)
	CreateSwiftCodeFunc        func(ctx context.Context, bank *models.SwiftBank) error
	DeleteSwiftCodeFunc        func(ctx context.Context, code string) error
}

func (m *MockSwiftService) GetSwiftCodeDetails(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
	return m.GetSwiftCodeDetailsFunc(ctx, code)
}

func (m *MockSwiftService) GetSwiftCodesByCountry(ctx context.Context, countryCode string) (*repository.CountrySwiftCodes, error) {
	return m.GetSwiftCodesByCountryFunc(ctx, countryCode)
}

func (m *MockSwiftService) CreateSwiftCode(ctx context.Context, bank *models.SwiftBank) error {
	return m.CreateSwiftCodeFunc(ctx, bank)
}

func (m *MockSwiftService) DeleteSwiftCode(ctx context.Context, code string) error {
	return m.DeleteSwiftCodeFunc(ctx, code)
}
