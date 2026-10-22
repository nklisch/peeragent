package app

import (
	"os"
	"testing"

	"github.com/nklisch/peeragent/internal/testsupport"
	"github.com/nklisch/peeragent/internal/testsupport/terminalenv"
)

func TestMain(m *testing.M) {
	if terminalenv.WriteTerminalEnvAgentSnapshot() {
		return
	}
	os.Exit(testsupport.RunWithPrivateState(m.Run))
}
