package models

import (
	"time"
)

// SwiftCodeEntity represents the type of SWIFT code entity
type SwiftCodeEntity string

const (
	Headquarters SwiftCodeEntity = "HEADQUARTERS"
	Branch       SwiftCodeEntity = "BRANCH"
)

// SwiftBank represents a bank entity in the swift_banks table
type SwiftBank struct {
	SwiftCode      string          `db:"swift_code"`
	HQSwiftBase    string          `db:"hq_swift_base"`
	CountryISOCode string          `db:"country_iso_code"`
	BankName       string          `db:"bank_name"`
	EntityType     SwiftCodeEntity `db:"entity_type"`
	CreatedAt      time.Time       `db:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at"`
}
