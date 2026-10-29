package jobs

import (
	"encoding/binary"
	"syscall"
	"unsafe"
)

func bootID() string { return "" }
func processStartTime(pid int) string {
	// CTL_KERN / KERN_PROC / KERN_PROC_PID returns kinfo_proc, whose
	// first member is extern_proc's start timeval (64-bit seconds, 32-bit usec).
	// ABI: https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/proc.h
	// Use kernel time rather than ps's locale-dependent, whole-second display.
	mib := [4]int32{1, 14, 1, int32(pid)}
	var info [4096]byte
	size := uintptr(len(info))
	_, _, err := syscall.Syscall6(syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])), uintptr(len(mib)),
		uintptr(unsafe.Pointer(&info[0])), uintptr(unsafe.Pointer(&size)), 0, 0)
	if err != 0 || size > uintptr(len(info)) {
		return ""
	}
	return darwinStartTime(info[:size], binary.NativeEndian)
}
