package repository_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/zdziszkee/swift-codes/internal/database"
	"github.com/zdziszkee/swift-codes/internal/models"
	repo "github.com/zdziszkee/swift-codes/internal/repositories"
)

func TestServices(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Repositories Suite")
}

var _ = Describe("SQLSwiftRepository", func() {
	var (
		mockDB      *sql.DB
		mock        sqlmock.Sqlmock
		repository  repo.SwiftRepository
		ctx         context.Context
		tableName   = "swift_catalog.default_schema.swift_banks"
		sampleBank  *models.SwiftBank
		sampleBanks []*models.SwiftBank
	)

	BeforeEach(func() {
		var err error
		mockDB, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		db := &database.Database{DB: mockDB}
		repository = repo.NewSQLSwiftRepository(db, database.Config{
			Catalog:   "swift_catalog",
			Schema:    "default_schema",
			TableName: "swift_banks",
		})
		ctx = context.Background()

		sampleBank = &models.SwiftBank{
			SwiftCode:      "TESTCODE123",
			SwiftCodeBase:  "TESTCODE",
			CountryISOCode: "US",
			BankName:       "Test Bank",
			IsHeadquarter:  true,
			Address:        "123 Test St",
			CountryName:    "United States",
		}

		sampleBanks = []*models.SwiftBank{
			sampleBank,
			{
				SwiftCode:      "TESTCODE456",
				SwiftCodeBase:  "TESTCODE",
				CountryISOCode: "US",
				BankName:       "Test Bank Branch",
				IsHeadquarter:  false,
				Address:        "456 Branch St",
				CountryName:    "United States",
			},
		}
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
		mockDB.Close()
	})

	Describe("Create", func() {
		Context("when creating a new bank", func() {
			It("should succeed for valid data", func() {
				// Check if code exists
				mock.ExpectQuery(`SELECT 1 FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnError(sql.ErrNoRows)

				// Insert new record
				mock.ExpectExec(`INSERT INTO `+tableName+` \(swift_code, swift_code_base, country_iso_code, bank_name, is_headquarter, address, country_name\) VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
					WithArgs("TESTCODE123", "TESTCODE", "US", "Test Bank", true, "123 Test St", "United States").
					WillReturnResult(sqlmock.NewResult(1, 1))

				err := repository.Create(ctx, sampleBank)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle duplicate entries", func() {
				mock.ExpectQuery(`SELECT 1 FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

				err := repository.Create(ctx, sampleBank)
				Expect(err).To(Equal(repo.ErrDuplicate))
			})

			It("should handle database errors during existence check", func() {
				mock.ExpectQuery(`SELECT 1 FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnError(errors.New("database connection error"))

				err := repository.Create(ctx, sampleBank)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino check duplicate failed"))
			})

			It("should handle database errors during insertion", func() {
				mock.ExpectQuery(`SELECT 1 FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnError(sql.ErrNoRows)

				mock.ExpectExec(`INSERT INTO `+tableName+` \(swift_code, swift_code_base, country_iso_code, bank_name, is_headquarter, address, country_name\) VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
					WithArgs("TESTCODE123", "TESTCODE", "US", "Test Bank", true, "123 Test St", "United States").
					WillReturnError(errors.New("insert error"))

				err := repository.Create(ctx, sampleBank)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino insert failed"))
			})

			It("should auto-generate swift code base if not provided", func() {
				bankWithoutBase := &models.SwiftBank{
					SwiftCode:      "TESTCODE123",
					CountryISOCode: "US",
					BankName:       "Test Bank",
					IsHeadquarter:  true,
					Address:        "123 Test St",
					CountryName:    "United States",
				}

				mock.ExpectQuery(`SELECT 1 FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnError(sql.ErrNoRows)

				mock.ExpectExec(`INSERT INTO `+tableName+` \(swift_code, swift_code_base, country_iso_code, bank_name, is_headquarter, address, country_name\) VALUES \(\?, \?, \?, \?, \?, \?, \?\)`).
					WithArgs("TESTCODE123", "TESTCODE", "US", "Test Bank", true, "123 Test St", "United States").
					WillReturnResult(sqlmock.NewResult(1, 1))

				err := repository.Create(ctx, bankWithoutBase)
				Expect(err).NotTo(HaveOccurred())
				Expect(bankWithoutBase.SwiftCodeBase).To(Equal("TESTCODE"))
			})
		})
	})
	Describe("CreateBatch", func() {
		Context("when creating multiple banks in batch", func() {
			It("should succeed with valid data", func() {
				mock.ExpectExec(`INSERT INTO `+tableName+` \(swift_code, swift_code_base, country_iso_code, bank_name, is_headquarter, address, country_name\) VALUES \(\?, \?, \?, \?, \?, \?, \?\),\(\?, \?, \?, \?, \?, \?, \?\)`).
					WithArgs(
						"TESTCODE123", "TESTCODE", "US", "Test Bank", true, "123 Test St", "United States",
						"TESTCODE456", "TESTCODE", "US", "Test Bank Branch", false, "456 Branch St", "United States",
					).
					WillReturnResult(sqlmock.NewResult(2, 2))

				err := repository.CreateBatch(ctx, sampleBanks)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle empty batch", func() {
				err := repository.CreateBatch(ctx, []*models.SwiftBank{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle database errors during batch insert", func() {
				mock.ExpectExec(`INSERT INTO .*`).
					WithArgs(
						"TESTCODE123", "TESTCODE", "US", "Test Bank", true, "123 Test St", "United States",
						"TESTCODE456", "TESTCODE", "US", "Test Bank Branch", false, "456 Branch St", "United States",
					).
					WillReturnError(errors.New("batch insert error"))

				err := repository.CreateBatch(ctx, sampleBanks)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino batch insert failed"))
			})

			It("should handle large batches by splitting them", func() {
				// Create 150 banks (more than batchSize)
				largeBatch := make([]*models.SwiftBank, 150)
				for i := range largeBatch {
					largeBatch[i] = &models.SwiftBank{
						SwiftCode:      fmt.Sprintf("BANK%c%c", rune('A'+i%26), rune('0'+i%10)),
						SwiftCodeBase:  fmt.Sprintf("BANK%c", rune('A'+i%26)),
						CountryISOCode: "US",
						BankName:       fmt.Sprintf("Bank %c", rune('A'+i%26)),
						IsHeadquarter:  i%5 == 0,
						Address:        fmt.Sprintf("Address %c", rune('A'+i%26)),
						CountryName:    "United States",
					}
				}

				// For the first batch of 100, match exact arguments count (7 fields * 100 items)
				firstBatchArgs := make([]driver.Value, 7*100)
				for i := 0; i < len(firstBatchArgs); i++ {
					firstBatchArgs[i] = sqlmock.AnyArg()
				}
				mock.ExpectExec(`INSERT INTO .*`).
					WithArgs(firstBatchArgs...).
					WillReturnResult(sqlmock.NewResult(100, 100))

				// For the second batch of 50, match exact arguments count (7 fields * 50 items)
				secondBatchArgs := make([]driver.Value, 7*50)
				for i := 0; i < len(secondBatchArgs); i++ {
					secondBatchArgs[i] = sqlmock.AnyArg()
				}
				mock.ExpectExec(`INSERT INTO .*`).
					WithArgs(secondBatchArgs...).
					WillReturnResult(sqlmock.NewResult(50, 50))

				err := repository.CreateBatch(ctx, largeBatch)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("GetByCode", func() {
		Context("when retrieving a bank by code", func() {
			It("should return the correct bank", func() {
				rows := sqlmock.NewRows([]string{"swift_code", "swift_code_base", "country_iso_code", "bank_name", "is_headquarter", "address", "country_name"}).
					AddRow("TESTCODE123", "TESTCODE", "US", "Test Bank", true, "123 Test St", "United States")

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnRows(rows)

				// For the branches query as it's a headquarters
				branchRows := sqlmock.NewRows([]string{"swift_code", "swift_code_base", "country_iso_code", "bank_name", "is_headquarter", "address", "country_name"}).
					AddRow("TESTCODE456", "TESTCODE", "US", "Test Branch", false, "456 Branch St", "United States")

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code_base = \? AND is_headquarter = false`).
					WithArgs("TESTCODE").
					WillReturnRows(branchRows)

				result, err := repository.GetByCode(ctx, "TESTCODE123")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Bank.SwiftCode).To(Equal("TESTCODE123"))
				Expect(result.Bank.BankName).To(Equal("Test Bank"))
				Expect(result.Branches).To(HaveLen(1))
				Expect(result.Branches[0].SwiftCode).To(Equal("TESTCODE456"))
			})

			It("should handle non-headquarters banks", func() {
				nonHQBank := &models.SwiftBank{
					SwiftCode:      "BRANCH456",
					SwiftCodeBase:  "TESTCODE",
					CountryISOCode: "US",
					BankName:       "Branch Bank",
					IsHeadquarter:  false,
					Address:        "456 Branch St",
					CountryName:    "United States",
				}

				rows := sqlmock.NewRows([]string{"swift_code", "swift_code_base", "country_iso_code", "bank_name", "is_headquarter", "address", "country_name"}).
					AddRow(nonHQBank.SwiftCode, nonHQBank.SwiftCodeBase, nonHQBank.CountryISOCode, nonHQBank.BankName, nonHQBank.IsHeadquarter, nonHQBank.Address, nonHQBank.CountryName)

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("BRANCH456").
					WillReturnRows(rows)

				// Should not query for branches since it's not a headquarters
				result, err := repository.GetByCode(ctx, "BRANCH456")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Bank.SwiftCode).To(Equal("BRANCH456"))
				Expect(result.Bank.IsHeadquarter).To(BeFalse())
				Expect(result.Branches).To(BeEmpty())
			})

			It("should handle not found error", func() {
				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("NOTFOUND").
					WillReturnError(sql.ErrNoRows)

				result, err := repository.GetByCode(ctx, "NOTFOUND")
				Expect(err).To(Equal(repo.ErrNotFound))
				Expect(result).To(BeNil())
			})

			It("should handle database errors", func() {
				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnError(errors.New("database error"))

				result, err := repository.GetByCode(ctx, "TESTCODE123")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino query failed"))
				Expect(result).To(BeNil())
			})

			It("should handle errors when fetching branches", func() {
				rows := sqlmock.NewRows([]string{"swift_code", "swift_code_base", "country_iso_code", "bank_name", "is_headquarter", "address", "country_name"}).
					AddRow("TESTCODE123", "TESTCODE", "US", "Test Bank", true, "123 Test St", "United States")

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnRows(rows)

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code_base = \? AND is_headquarter = false`).
					WithArgs("TESTCODE").
					WillReturnError(errors.New("branch query error"))

				result, err := repository.GetByCode(ctx, "TESTCODE123")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino fetch branches failed"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("GetBranchesByHQBase", func() {
		Context("when fetching branches for a headquarters", func() {
			It("should return all branches", func() {
				branchRows := sqlmock.NewRows([]string{"swift_code", "swift_code_base", "country_iso_code", "bank_name", "is_headquarter", "address", "country_name"}).
					AddRow("BRANCH123", "TESTCODE", "US", "Branch 1", false, "123 Branch St", "United States").
					AddRow("BRANCH456", "TESTCODE", "US", "Branch 2", false, "456 Branch St", "United States")

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code_base = \? AND is_headquarter = false`).
					WithArgs("TESTCODE").
					WillReturnRows(branchRows)

				branches, err := repository.GetBranchesByHQBase(ctx, "TESTCODE")
				Expect(err).NotTo(HaveOccurred())
				Expect(branches).To(HaveLen(2))
				Expect(branches[0].SwiftCode).To(Equal("BRANCH123"))
				Expect(branches[1].SwiftCode).To(Equal("BRANCH456"))
			})

			It("should return empty slice when no branches found", func() {
				emptyRows := sqlmock.NewRows([]string{"swift_code", "swift_code_base", "country_iso_code", "bank_name", "is_headquarter", "address", "country_name"})

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code_base = \? AND is_headquarter = false`).
					WithArgs("TESTCODE").
					WillReturnRows(emptyRows)

				branches, err := repository.GetBranchesByHQBase(ctx, "TESTCODE")
				Expect(err).NotTo(HaveOccurred())
				Expect(branches).To(BeEmpty())
			})

			It("should handle database errors", func() {
				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code_base = \? AND is_headquarter = false`).
					WithArgs("TESTCODE").
					WillReturnError(errors.New("database error"))

				branches, err := repository.GetBranchesByHQBase(ctx, "TESTCODE")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino query failed"))
				Expect(branches).To(BeNil())
			})

			It("should handle row scan errors", func() {
				// Return rows with incorrect number of columns to cause a scan error
				incorrectRows := sqlmock.NewRows([]string{"swift_code", "swift_code_base"}).
					AddRow("BRANCH123", "TESTCODE")

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE swift_code_base = \? AND is_headquarter = false`).
					WithArgs("TESTCODE").
					WillReturnRows(incorrectRows)

				branches, err := repository.GetBranchesByHQBase(ctx, "TESTCODE")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino scan failed"))
				Expect(branches).To(BeNil())
			})
		})
	})

	Describe("GetByCountry", func() {
		Context("when retrieving banks by country", func() {
			It("should return all banks for a country", func() {
				// First mock the country name query
				countryNameRow := sqlmock.NewRows([]string{"country_name"}).
					AddRow("United States")

				mock.ExpectQuery(`SELECT country_name FROM ` + tableName + ` WHERE country_iso_code = \? LIMIT 1`).
					WithArgs("US").
					WillReturnRows(countryNameRow)

				// Then mock the banks query
				bankRows := sqlmock.NewRows([]string{"swift_code", "swift_code_base", "country_iso_code", "bank_name", "is_headquarter", "address", "country_name"}).
					AddRow("TESTCODE123", "TESTCODE", "US", "Test Bank", true, "123 Test St", "United States").
					AddRow("BRANCH456", "TESTCODE", "US", "Branch Bank", false, "456 Branch St", "United States")

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE country_iso_code = \?`).
					WithArgs("US").
					WillReturnRows(bankRows)

				result, err := repository.GetByCountry(ctx, "US")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.CountryISO2).To(Equal("US"))
				Expect(result.CountryName).To(Equal("United States"))
				Expect(result.SwiftCodes).To(HaveLen(2))
				Expect(result.SwiftCodes[0].SwiftCode).To(Equal("TESTCODE123"))
				Expect(result.SwiftCodes[1].SwiftCode).To(Equal("BRANCH456"))
			})

			It("should handle country not found", func() {
				mock.ExpectQuery(`SELECT country_name FROM ` + tableName + ` WHERE country_iso_code = \? LIMIT 1`).
					WithArgs("XX").
					WillReturnError(sql.ErrNoRows)

				result, err := repository.GetByCountry(ctx, "XX")
				Expect(err).To(Equal(repo.ErrNotFound))
				Expect(result).To(BeNil())
			})

			It("should handle database errors during country fetch", func() {
				mock.ExpectQuery(`SELECT country_name FROM ` + tableName + ` WHERE country_iso_code = \? LIMIT 1`).
					WithArgs("US").
					WillReturnError(errors.New("database error"))

				result, err := repository.GetByCountry(ctx, "US")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino query failed"))
				Expect(result).To(BeNil())
			})

			It("should handle database errors during banks fetch", func() {
				// First mock the country name query
				countryNameRow := sqlmock.NewRows([]string{"country_name"}).
					AddRow("United States")

				mock.ExpectQuery(`SELECT country_name FROM ` + tableName + ` WHERE country_iso_code = \? LIMIT 1`).
					WithArgs("US").
					WillReturnRows(countryNameRow)

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE country_iso_code = \?`).
					WithArgs("US").
					WillReturnError(errors.New("database error"))

				result, err := repository.GetByCountry(ctx, "US")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino query failed"))
				Expect(result).To(BeNil())
			})

			It("should handle empty results", func() {
				// First mock the country name query
				countryNameRow := sqlmock.NewRows([]string{"country_name"}).
					AddRow("United States")

				mock.ExpectQuery(`SELECT country_name FROM ` + tableName + ` WHERE country_iso_code = \? LIMIT 1`).
					WithArgs("US").
					WillReturnRows(countryNameRow)

				// Then mock empty banks results
				emptyRows := sqlmock.NewRows([]string{"swift_code", "swift_code_base", "country_iso_code", "bank_name", "is_headquarter", "address", "country_name"})

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE country_iso_code = \?`).
					WithArgs("US").
					WillReturnRows(emptyRows)

				result, err := repository.GetByCountry(ctx, "US")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.CountryISO2).To(Equal("US"))
				Expect(result.CountryName).To(Equal("United States"))
				Expect(result.SwiftCodes).To(BeEmpty())
			})

			It("should handle row scan errors", func() {
				// First mock the country name query
				countryNameRow := sqlmock.NewRows([]string{"country_name"}).
					AddRow("United States")

				mock.ExpectQuery(`SELECT country_name FROM ` + tableName + ` WHERE country_iso_code = \? LIMIT 1`).
					WithArgs("US").
					WillReturnRows(countryNameRow)

				// Return rows with incorrect number of columns to cause a scan error
				incorrectRows := sqlmock.NewRows([]string{"swift_code", "swift_code_base"}).
					AddRow("TESTCODE123", "TESTCODE")

				mock.ExpectQuery(`SELECT .* FROM ` + tableName + ` WHERE country_iso_code = \?`).
					WithArgs("US").
					WillReturnRows(incorrectRows)

				result, err := repository.GetByCountry(ctx, "US")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino scan failed"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("Delete", func() {
		Context("when deleting a bank", func() {
			It("should delete an existing bank", func() {
				// Check if exists first
				mock.ExpectQuery(`SELECT 1 FROM ` + tableName + ` WHERE swift_code = \? LIMIT 1`).
					WithArgs("TESTCODE123").
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

				// Then delete
				mock.ExpectExec(`DELETE FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnResult(sqlmock.NewResult(0, 1))

				err := repository.Delete(ctx, "TESTCODE123")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle not found error", func() {
				mock.ExpectQuery(`SELECT 1 FROM ` + tableName + ` WHERE swift_code = \? LIMIT 1`).
					WithArgs("NOTFOUND").
					WillReturnError(sql.ErrNoRows)

				err := repository.Delete(ctx, "NOTFOUND")
				Expect(err).To(Equal(repo.ErrNotFound))
			})

			It("should handle database errors during existence check", func() {
				mock.ExpectQuery(`SELECT 1 FROM ` + tableName + ` WHERE swift_code = \? LIMIT 1`).
					WithArgs("TESTCODE123").
					WillReturnError(errors.New("database error"))

				err := repository.Delete(ctx, "TESTCODE123")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino check exists failed"))
			})

			It("should handle database errors during delete", func() {
				// Check if exists first
				mock.ExpectQuery(`SELECT 1 FROM ` + tableName + ` WHERE swift_code = \? LIMIT 1`).
					WithArgs("TESTCODE123").
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

				mock.ExpectExec(`DELETE FROM ` + tableName + ` WHERE swift_code = \?`).
					WithArgs("TESTCODE123").
					WillReturnError(errors.New("delete error"))

				err := repository.Delete(ctx, "TESTCODE123")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("trino delete failed"))
			})
		})
	})

	Describe("LoadCSV", func() {
		Context("when trying to load CSV", func() {
			It("should return not implemented error", func() {
				err := repository.LoadCSV(ctx, "path/to/file.csv")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not implemented for Trino"))
			})
		})
	})
})
