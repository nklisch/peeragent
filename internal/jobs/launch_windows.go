//go:build windows

package jobs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

const (
	createNoWindow         = 0x08000000
	jobBasicAccountingInfo = 1
	jobExtendedLimitInfo   = 9
	jobKillOnClose         = 0x00002000
	jobQuery               = 0x0004
	jobTerminate           = 0x0008
	waitTimeout            = 0x00000102
	errorInvalidParameter  = syscall.Errno(87)
	attachmentWait         = 500 * time.Millisecond
	attachmentPoll         = 20 * time.Millisecond
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	createJobObject           = kernel32.NewProc("CreateJobObjectW")
	openJobObject             = kernel32.NewProc("OpenJobObjectW")
	setInformationJobObject   = kernel32.NewProc("SetInformationJobObject")
	assignProcessToJobObject  = kernel32.NewProc("AssignProcessToJobObject")
	terminateJobObject        = kernel32.NewProc("TerminateJobObject")
	queryInformationJobObject = kernel32.NewProc("QueryInformationJobObject")
)

type basicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type extendedLimitInformation struct {
	BasicLimitInformation basicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

type basicAccountingInformation struct {
	TotalUserTime             int64
	TotalKernelTime           int64
	ThisPeriodTotalUserTime   int64
	ThisPeriodTotalKernelTime int64
	TotalPageFaultCount       uint32
	TotalProcesses            uint32
	ActiveProcesses           uint32
	TotalTerminatedProcesses  uint32
}

func ApplyDetachAttrs(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}

// AttachCurrentProcess confines the worker and all future descendants to a
// named Windows job before the target CLI can start.
func AttachCurrentProcess(jobID string) (func(), error) {
	name, err := jobName(jobID)
	if err != nil {
		return nil, err
	}
	h, _, callErr := createJobObject.Call(0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return nil, fmt.Errorf("create Windows job: %w", callErr)
	}
	closeJob := func() { _ = syscall.CloseHandle(syscall.Handle(h)) }
	info := extendedLimitInformation{}
	info.BasicLimitInformation.LimitFlags = jobKillOnClose
	ok, _, callErr := setInformationJobObject.Call(h, jobExtendedLimitInfo, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
	if ok == 0 {
		closeJob()
		return nil, fmt.Errorf("set Windows job limits: %w", callErr)
	}
	self, err := syscall.GetCurrentProcess()
	if err != nil {
		closeJob()
		return nil, fmt.Errorf("get current process: %w", err)
	}
	ok, _, callErr = assignProcessToJobObject.Call(h, uintptr(self))
	if ok == 0 {
		closeJob()
		return nil, fmt.Errorf("assign async worker to Windows job: %w", callErr)
	}
	return closeJob, nil
}

func SignalProcessTree(jobID string, pid int, _ os.Signal) error {
	if pid <= 1 {
		return fmt.Errorf("refusing to terminate unsafe process %d", pid)
	}
	deadline := time.Now().Add(attachmentWait)
	for {
		h, err := openNamedJob(jobID, jobTerminate|jobQuery)
		if err == nil {
			active, queryErr := activeJobProcesses(h)
			if queryErr != nil {
				_ = syscall.CloseHandle(h)
				return queryErr
			}
			if active != 0 {
				ok, _, callErr := terminateJobObject.Call(uintptr(h), 1)
				_ = syscall.CloseHandle(h)
				if ok == 0 {
					return fmt.Errorf("terminate Windows job %s: %w", jobID, callErr)
				}
				return nil
			}
			_ = syscall.CloseHandle(h)
		} else if !errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
			return err
		}
		// The worker may still be creating and joining its job. A persisted PID
		// is not a safe termination target because Windows can reuse it.
		if !processExists(pid) {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("Windows job %s has no attached worker while process %d is running", jobID, pid)
		}
		time.Sleep(attachmentPoll)
	}
}

func ProcessTreeExists(jobID string, pid int) bool {
	h, err := openNamedJob(jobID, jobQuery)
	if err == nil {
		defer syscall.CloseHandle(h)
		active, queryErr := activeJobProcesses(h)
		return queryErr != nil || active != 0 || processExists(pid)
	}
	if !errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return true
	}
	return processExists(pid)
}

func activeJobProcesses(h syscall.Handle) (uint32, error) {
	info := basicAccountingInformation{}
	ok, _, callErr := queryInformationJobObject.Call(uintptr(h), jobBasicAccountingInfo, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info), 0)
	if ok == 0 {
		return 0, fmt.Errorf("query Windows job: %w", callErr)
	}
	return info.ActiveProcesses, nil
}

func SignalProcessGroup(pid int, _ os.Signal) error { return terminateRoot(pid) }
func ProcessGroupExists(pid int) bool               { return processExists(pid) }
func TerminateSignal() os.Signal                    { return os.Interrupt }
func KillSignal() os.Signal                         { return os.Kill }

func jobName(jobID string) (*uint16, error) {
	if err := ValidateID(jobID); err != nil {
		return nil, err
	}
	return syscall.UTF16PtrFromString("Local\\peeragent-" + jobID)
}

func openNamedJob(jobID string, access uintptr) (syscall.Handle, error) {
	name, err := jobName(jobID)
	if err != nil {
		return 0, err
	}
	h, _, callErr := openJobObject.Call(access, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return 0, callErr
	}
	return syscall.Handle(h), nil
}

func terminateRoot(pid int) error {
	h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid))
	if errors.Is(err, errorInvalidParameter) {
		return nil
	}
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(h)
	return syscall.TerminateProcess(h, 1)
}

func processExists(pid int) bool {
	if pid <= 1 {
		return false
	}
	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if errors.Is(err, errorInvalidParameter) {
		return false
	}
	if err != nil {
		return true
	}
	defer syscall.CloseHandle(h)
	state, err := syscall.WaitForSingleObject(h, 0)
	return err != nil || state == waitTimeout
}
