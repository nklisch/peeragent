package input

import (
	"os"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
)

func TestMain(m *testing.M) {
	os.Exit(testsupport.RunWithPrivateState(m.Run))
}
