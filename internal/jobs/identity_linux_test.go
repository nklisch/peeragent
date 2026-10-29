package jobs

import (
	"os"
	"strings"
	"testing"
)

func TestIdentityDetectsReboot(t *testing.T) {
	p := IdentifyProcess(os.Getpid())
	if p.BootID == "" {
		t.Fatal("missing boot id")
	}
	p.BootID = "previous-boot"
	if !p.Gone() {
		t.Fatal("earlier boot still live")
	}
	p = IdentifyProcess(os.Getpid())
	p.BootID = "" // a platform without a boot id still verifies start time
	if p.Gone() {
		t.Fatal("pid fallback rejected live process")
	}
	p.StartTime += "-reused"
	if !p.Gone() {
		t.Fatal("pid fallback accepted reused pid")
	}
}
func TestLinuxStartTimeHandlesParenthesizedNames(t *testing.T) {
	stat := "123 (worker (with spaces)) S " + strings.Repeat("0 ", 18) + "987654 0"
	if got := linuxStartTime(stat); got != "987654" {
		t.Fatalf("start time %q", got)
	}
}
