package parser_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	parser "github.com/zdziszkee/swift-codes/internal/parsers"
	readers "github.com/zdziszkee/swift-codes/internal/readers"
)

func TestSwiftBanksParser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SwiftBanksParser Suite")
}

var _ = Describe("DefaultSwiftBanksParser", func() {
	var (
		p       parser.SwiftBanksParser
		records []readers.SwiftBankRecord
	)

	BeforeEach(func() {
		// Use the default parser implementation
		p = parser.DefaultSwiftBanksParser{}
		records = []readers.SwiftBankRecord{}
	})

	Describe("ParseSwiftBanks", func() {
		Context("with valid record", func() {
			BeforeEach(func() {
				records = []readers.SwiftBankRecord{
					{
						Index:          1,
						SwiftCode:      "ABCDEF12XXX", // valid BIC matching regex and ends with "XXX"
						BankName:       "Bank of America",
						CountryISOCode: "US",
						Address:        "123 Main St",
						CountryName:    "United States",
					},
				}
			})

			It("should parse the record correctly", func() {
				banks, err := p.ParseSwiftBanks(records)
				Expect(err).NotTo(HaveOccurred())
				// Only one valid bank record should be returned.
				Expect(banks).To(HaveLen(1))

				parsed := banks[0]
				Expect(parsed.SwiftCode).To(Equal("ABCDEF12XXX"))
				// SwiftCodeBase is the first 8 characters.
				Expect(parsed.SwiftCodeBase).To(Equal("ABCDEF12"))
				Expect(parsed.CountryISOCode).To(Equal("US"))
				Expect(parsed.BankName).To(Equal("Bank of America"))
				// Since the SwiftCode ends with "XXX", then IsHeadquarter should be true.
				Expect(parsed.IsHeadquarter).To(BeTrue())
				Expect(parsed.Address).To(Equal("123 Main St"))
				Expect(parsed.CountryName).To(Equal("United States"))
			})
		})

		Context("with a record that is not a headquarter", func() {
			BeforeEach(func() {
				records = []readers.SwiftBankRecord{
					{
						Index:          2,
						SwiftCode:      "GHIJKL34ABC", // does not end with "XXX"
						BankName:       "Citibank",
						CountryISOCode: "US",
						Address:        "456 Elm St",
						CountryName:    "United States",
					},
				}
			})

			It("should mark IsHeadquarter as false", func() {
				banks, err := p.ParseSwiftBanks(records)
				Expect(err).NotTo(HaveOccurred())
				Expect(banks).To(HaveLen(1))

				parsed := banks[0]
				Expect(parsed.SwiftCode).To(Equal("GHIJKL34ABC"))
				// SwiftCodeBase is the first 8 characters.
				Expect(parsed.SwiftCodeBase).To(Equal("GHIJKL34"))
				Expect(parsed.IsHeadquarter).To(BeFalse())
			})
		})

		Context("with record having SwiftCode too long", func() {
			BeforeEach(func() {
				// SwiftCode length > 15 should be skipped.
				records = []readers.SwiftBankRecord{
					{
						Index:          3,
						SwiftCode:      "LONGSWIFTCODE1234",
						BankName:       "Dummy Bank",
						CountryISOCode: "GB",
						Address:        "789 Oak St",
						CountryName:    "United Kingdom",
					},
				}
			})

			It("should skip the invalid record", func() {
				banks, err := p.ParseSwiftBanks(records)
				Expect(err).NotTo(HaveOccurred())
				// Since the record is invalid, banks should be empty.
				Expect(banks).To(HaveLen(0))
			})
		})

		Context("with records that fail validation", func() {
			BeforeEach(func() {
				records = []readers.SwiftBankRecord{
					{
						Index:          1,
						SwiftCode:      "", // empty swift code, invalid record
						BankName:       "Invalid Bank",
						CountryISOCode: "US",
						Address:        "Address 1",
						CountryName:    "United States",
					},
					{
						Index:          2,
						SwiftCode:      "BADFORMAT", // does not match BIC regex
						BankName:       "Another Bank",
						CountryISOCode: "US",
						Address:        "Address 2",
						CountryName:    "United States",
					},
					{
						Index:          3,
						SwiftCode:      "VALID12XXX", // valid format provided below
						BankName:       "", // missing bank name
						CountryISOCode: "US",
						Address:        "Address 3",
						CountryName:    "United States",
					},
					{
						Index:          4,
						SwiftCode:      "VALID34XXX", // valid
						BankName:       "Valid Bank",
						CountryISOCode: "USA", // invalid country code (should be 2 letters)
						Address:        "Address 4",
						CountryName:    "United States",
					},
				}
			})

			It("should skip all invalid records and return no banks", func() {
				banks, err := p.ParseSwiftBanks(records)
				Expect(err).NotTo(HaveOccurred())
				// None of these records pass all validations.
				Expect(banks).To(HaveLen(0))
			})
		})

		Context("with a mix of valid and invalid records", func() {
			BeforeEach(func() {
				records = []readers.SwiftBankRecord{
					{
						Index:          1,
						SwiftCode:      "ABCDEF12XXX", // valid
						BankName:       "Bank One",
						CountryISOCode: "US",
						Address:        "Address 1",
						CountryName:    "United States",
					},
					{
						Index:          2,
						SwiftCode:      "BADFORMAT", // invalid SwiftCode (no match)
						BankName:       "Bank Two",
						CountryISOCode: "US",
						Address:        "Address 2",
						CountryName:    "United States",
					},
					{
						Index:          3,
						SwiftCode:      "GHIJKL34ABC", // valid
						BankName:       "Bank Three",
						CountryISOCode: "GB",
						Address:        "Address 3",
						CountryName:    "United Kingdom",
					},
				}
			})

			It("should only return valid banks", func() {
				banks, err := p.ParseSwiftBanks(records)
				Expect(err).NotTo(HaveOccurred())
				// Only records 1 and 3 are valid.
				Expect(banks).To(HaveLen(2))

				// Validate the first valid record.
				Expect(banks[0].SwiftCode).To(Equal("ABCDEF12XXX"))
				Expect(banks[0].SwiftCodeBase).To(Equal("ABCDEF12"))
				Expect(banks[0].IsHeadquarter).To(BeTrue())

				// Validate the second valid record.
				Expect(banks[1].SwiftCode).To(Equal("GHIJKL34ABC"))
				Expect(banks[1].SwiftCodeBase).To(Equal("GHIJKL34"))
				Expect(banks[1].IsHeadquarter).To(BeFalse())
			})
		})
	})
})
