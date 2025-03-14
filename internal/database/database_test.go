package database_test

import (
	"database/sql"
	"os"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zdziszkee/swift-codes/internal/database"
)

func TestDatabase(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Database Init Suite")
}

var _ = Describe("Database", func() {
	var (
		mockDB sqlmock.Sqlmock
		db     *sql.DB
		err    error
	)

	BeforeEach(func() {
		db, mockDB, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = db.Close()
	})

	Describe("ExecuteSchema", func() {
		It("should execute all non-empty queries from the schema file", func() {
			// Create a temporary schema file with two SQL queries.
			schemaContent := `
CREATE TABLE IF NOT EXISTS test1 (id INT);
-- a comment line
CREATE TABLE IF NOT EXISTS test2 (name VARCHAR(50));
`
			tmpFile, err := os.CreateTemp("", "schema-*.sql")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpFile.Name())
			_, err = tmpFile.Write([]byte(schemaContent))
			Expect(err).NotTo(HaveOccurred())
			tmpFile.Close()

			// Expect Exec calls for each query.
			mockDB.ExpectExec("CREATE TABLE IF NOT EXISTS test1").WillReturnResult(sqlmock.NewResult(1, 1))
			mockDB.ExpectExec("CREATE TABLE IF NOT EXISTS test2").WillReturnResult(sqlmock.NewResult(1, 1))

			// Create an instance of Database using our mock DB.
			databaseInstance := &database.Database{
				DB:     db,
				Config: database.Config{
					// Config values aren’t used in ExecuteSchema.
				},
			}
			err = databaseInstance.ExecuteSchema(tmpFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(mockDB.ExpectationsWereMet()).NotTo(HaveOccurred())
		})

		It("should return an error if the schema file does not exist", func() {
			databaseInstance := &database.Database{DB: db}
			err := databaseInstance.ExecuteSchema("/nonexistent/path/schema.sql")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read schema file"))
		})
	})
})
