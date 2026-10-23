package heavy

import (
	"syscall"
	"unsafe"
)

func processSnapshot() map[int]process {
	out := map[int]process{}
	handle, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return out
	}
	defer syscall.CloseHandle(handle)
	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = syscall.Process32First(handle, &entry); err == nil; err = syscall.Process32Next(handle, &entry) {
		out[int(entry.ProcessID)] = process{parent: int(entry.ParentProcessID)}
	}
	return out
}
