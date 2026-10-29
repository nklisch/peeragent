package jobs

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func TestIdentityMatchesLiveWorkerAndRejectsReusedPID(t *testing.T) {
	p := IdentifyProcess(os.Getpid())
	if p.StartTime == "" {
		t.Fatal("supported platform did not record process start time")
	}
	if p.Gone() {
		t.Fatal("live worker reported gone")
	}
	p.StartTime += "-other-process"
	if !p.Gone() {
		t.Fatal("reused PID identity accepted")
	}
}

func TestWritePIDRecordsIdentity(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create(filepath.Dir(filepath.Dir(store.Root)), ExecSpec{}, "task")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(job.ID)
	if err != nil || loaded.Worker == nil || loaded.Worker.PID != os.Getpid() || loaded.Worker.StartTime == "" || loaded.Worker.Gone() {
		t.Fatalf("%#v %v", loaded.Worker, err)
	}
}

func TestStaleLockRecoverySerializesContenders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	p := IdentifyProcess(os.Getpid())
	p.StartTime += "-other-process"
	data, _ := json.Marshal(p)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var count, active atomic.Int32
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := withFileLock(path, func() error {
				if active.Add(1) != 1 {
					t.Error("overlapping lock holders")
				}
				runtime.Gosched()
				count.Add(1)
				active.Add(-1)
				return nil
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if count.Load() != 12 {
		t.Fatalf("lost updates: %d", count.Load())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("owner file left behind: %v", err)
	}
}

func TestDarwinKernelTimevalPreservesSubsecondIdentity(t *testing.T) {
	var data [16]byte
	binary.LittleEndian.PutUint64(data[:8], 1700000000)
	binary.LittleEndian.PutUint32(data[8:12], 123456)
	first := darwinStartTime(data[:], binary.LittleEndian)
	if first != "1700000000:123456" {
		t.Fatalf("decoded %q", first)
	}
	binary.LittleEndian.PutUint32(data[8:12], 123457)
	if second := darwinStartTime(data[:], binary.LittleEndian); second == first {
		t.Fatal("subsecond PID reuse collapsed")
	}
	if got := darwinStartTime(data[:11], binary.LittleEndian); got != "" {
		t.Fatalf("short record %q", got)
	}
}
