//go:build integration

package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zdziszkee/swift-codes/internal/models"
	repository "github.com/zdziszkee/swift-codes/internal/repositories"
)

func TestSwiftServiceIntegration(t *testing.T) {
	ctx := context.Background()
	repo := &MockSwiftRepository{
		GetByCodeFunc: func(ctx context.Context, code string) (*repository.SwiftBankDetail, error) {
			if code == "ABCDUS33XXX" {
				return &repository.SwiftBankDetail{}, nil // Adjust fields
			}
			return nil, repository.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, bank *models.SwiftBank) error {
			if bank.SwiftCode == "ABCDUS33XXX" {
				return repository.ErrDuplicate
			}
			return nil
		},
		CreateBatchFunc: func(ctx context.Context, banks []*models.SwiftBank) error { return nil },
		DeleteFunc: func(ctx context.Context, code string) error {
			if code == "ABCDUS33XXX" {
				return nil
			}
			return repository.ErrNotFound
		},
	}
	s := NewSwiftService(repo)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		details, err := s.GetSwiftCodeDetails(ctx, code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Write([]byte("OK")) // Simplified response; adjust based on your struct
	})

	t.Run("Get valid SWIFT code", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/swift?code=ABCDUS33XXX", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Expected %d, got %d", http.StatusOK, rr.Code)
		}
		if rr.Body.String() != "OK" {
			t.Errorf("Expected 'OK', got %s", rr.Body.String())
		}
	})

	t.Run("Get invalid SWIFT code", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/swift?code=ABC123", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("Create duplicate SWIFT code", func(t *testing.T) {
		err := s.CreateSwiftCode(ctx, &models.SwiftBank{SwiftCode: "ABCDUS33XXX", CountryISOCode: "US", BankName: "Test Bank"})
		if !errors.Is(err, ErrAlreadyExists) {
			t.Errorf("Expected ErrAlreadyExists, got %v", err)
		}
	})

	t.Run("Delete non-existent SWIFT code", func(t *testing.T) {
		err := s.DeleteSwiftCode(ctx, "XYZ12345XXX")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}
