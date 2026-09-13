package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestSuite is the single Ginkgo entry point for this package.
// All Describe/Context/It blocks across the package register into it.
func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Permission Suite")
}
