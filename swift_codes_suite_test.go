package swift_codes_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSwiftCodes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SwiftCodes Suite")
}
