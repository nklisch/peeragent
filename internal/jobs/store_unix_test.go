//go:build unix

package jobs

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestApplyDetachAttrsSetsidOnUnix(t *testing.T) {
	cmd := exec.Command("true")
	ApplyDetachAttrs(cmd)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setsid {
		t.Fatalf("Setsid not enabled: %#v", cmd.SysProcAttr)
	}
}

func TestProcessGroupHelpersRejectUnsafePID(t *testing.T) {
	if err := SignalProcessGroup(1, syscall.SIGTERM); err == nil {
		t.Fatal("expected unsafe pid rejection")
	}
	if ProcessGroupExists(1) {
		t.Fatal("unsafe process group should not be reported as existing")
	}
}
