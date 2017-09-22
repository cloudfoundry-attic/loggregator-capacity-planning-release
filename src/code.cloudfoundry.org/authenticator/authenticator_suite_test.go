package authenticator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAuthenticator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authenticator Suite")
}
