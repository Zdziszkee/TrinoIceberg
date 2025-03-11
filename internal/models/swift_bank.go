package models

type SwiftBank struct {
	SwiftCode      string `db:"swift_code"`
	SwiftCodeBase  string `db:"swift_code_base"`
	CountryISOCode string `db:"country_iso_code"`
	BankName       string `db:"bank_name"`
	IsHeadquarter  bool   `db:"is_headquarter"`
	Address        string `db:"address"`
	CountryName    string `db:"country_name"`
}
