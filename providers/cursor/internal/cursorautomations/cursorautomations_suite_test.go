package cursorautomations

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCursorAutomations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CursorAutomations Suite")
}
