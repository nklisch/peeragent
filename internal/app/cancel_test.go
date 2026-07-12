package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nklisch/peeragent/internal/jobs"
	"github.com/nklisch/peeragent/internal/result"
)

func TestCancelJobPersistsCancelledStateAndCleansPIDThroughPort(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 4242); err != nil {
		t.Fatal(err)
	}
	controller := &fakeProcessController{}
	service := NewService(Options{ProcessController: controller})

	got, err := service.CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != result.StatusCancelled || got.Metadata.JobID != job.ID {
		t.Fatalf("result = %#v", got)
	}
	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "cancelled" {
		t.Fatalf("job status = %q, want cancelled", loaded.Status)
	}
	stored, err := os.ReadFile(job.ResultPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) == 0 {
		t.Fatal("cancelled result is empty")
	}
	if _, err := store.ReadPID(job.ID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID after cancel = %v, want missing pid", err)
	}
	if controller.calls() != 1 || controller.pid != 4242 || controller.termGrace != cancelTermGrace || controller.killGrace != cancelKillGrace {
		t.Fatalf("controller call = %#v", controller)
	}
}

func TestCancelJobReportsCorruptResultAsInfrastructureError(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 4242); err != nil {
		t.Fatal(err)
	}
	beforeJob, err := os.ReadFile(filepath.Join(store.Root, job.ID, "job.json"))
	if err != nil {
		t.Fatal(err)
	}
	const corruptResult = "{not valid result json"
	if err := os.WriteFile(job.ResultPath, []byte(corruptResult), 0o644); err != nil {
		t.Fatal(err)
	}
	controller := &fakeProcessController{}
	service := NewService(Options{ProcessController: controller})

	got, err := service.CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err == nil || !strings.Contains(err.Error(), "decode async job result") {
		t.Fatalf("error = %v, want corrupt-result decode error", err)
	}
	if !reflect.DeepEqual(got, result.Result{}) {
		t.Fatalf("result = %#v, want no fabricated cancellation result", got)
	}
	if controller.calls() != 0 {
		t.Fatalf("controller calls = %d, corrupt result must not be signalled", controller.calls())
	}
	afterJob, err := os.ReadFile(filepath.Join(store.Root, job.ID, "job.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterJob, beforeJob) {
		t.Fatalf("job.json changed after corrupt result: before=%q after=%q", beforeJob, afterJob)
	}
	afterResult, err := os.ReadFile(job.ResultPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterResult) != corruptResult {
		t.Fatalf("result.json changed after corrupt result: %q", afterResult)
	}
	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "running" {
		t.Fatalf("job status = %q, want running", loaded.Status)
	}
	pid, err := store.ReadPID(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pid != 4242 {
		t.Fatalf("pid = %d, want unchanged pid 4242", pid)
	}
}

func TestCancelJobCompletionWinsWithoutSignalling(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 4242); err != nil {
		t.Fatal(err)
	}
	winner := result.Result{
		Status:       result.StatusSuccess,
		Summary:      "completion won",
		ChangedFiles: []string{"main.go"},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, JobID: job.ID},
	}
	if err := WriteJobResult(job.ResultPath, winner); err != nil {
		t.Fatal(err)
	}
	job.Status = "complete"
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	controller := &fakeProcessController{}
	service := NewService(Options{ProcessController: controller})

	got, err := service.CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != winner.Status || got.Summary != winner.Summary {
		t.Fatalf("result = %#v, want winner %#v", got, winner)
	}
	if controller.calls() != 0 {
		t.Fatalf("controller calls = %d, complete job must not be signalled", controller.calls())
	}
	if _, err := store.ReadPID(job.ID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID after completion race = %v, want missing pid", err)
	}
}

func TestCancelJobRepairsCompletionResultThatPrecedesJobStatus(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	winner := result.Result{
		Status:       result.StatusFailed,
		Summary:      "target failed",
		ChangedFiles: []string{},
		Verification: []result.Verification{},
		Metadata:     result.Metadata{CWD: cwd, JobID: job.ID, ExitCode: 23},
	}
	if err := WriteJobResult(job.ResultPath, winner); err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 4242); err != nil {
		t.Fatal(err)
	}
	controller := &fakeProcessController{}
	service := NewService(Options{ProcessController: controller})

	got, err := service.CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != winner.Status || got.Summary != winner.Summary || got.Metadata.ExitCode != 23 {
		t.Fatalf("result = %#v, want winner %#v", got, winner)
	}
	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "failed" || controller.calls() != 0 {
		t.Fatalf("job = %#v, controller calls = %d", loaded, controller.calls())
	}
}

func TestCancelJobRepeatedCancellationIsIdempotent(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	controller := &fakeProcessController{}
	service := NewService(Options{ProcessController: controller})

	first, err := service.CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != result.StatusCancelled || second.Status != result.StatusCancelled {
		t.Fatalf("results = %#v, %#v", first, second)
	}
	if controller.calls() != 0 {
		t.Fatalf("controller calls = %d, no pid was present", controller.calls())
	}
}

func TestCancelJobCleanupIgnoresCallerCancellationAfterCommit(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 4242); err != nil {
		t.Fatal(err)
	}
	controller := &fakeProcessController{started: make(chan struct{}), release: make(chan struct{})}
	service := NewService(Options{ProcessController: controller})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	var got result.Result
	var callErr error
	go func() {
		got, callErr = service.CancelJob(ctx, JobRequest{CWD: cwd, JobID: job.ID})
		close(done)
	}()
	select {
	case <-controller.started:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup controller was not called")
	}
	cancel()
	close(controller.release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not finish after cleanup release")
	}
	if callErr != nil || got.Status != result.StatusCancelled {
		t.Fatalf("result = %#v, error = %v", got, callErr)
	}
	if _, err := store.ReadPID(job.ID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID after disconnected cancellation = %v", err)
	}
}

func TestCancelJobReturnsProcessControlErrorAfterRemovingPID(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "do work")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WritePID(job.ID, 4242); err != nil {
		t.Fatal(err)
	}
	expected := errors.New("process unavailable")
	controller := &fakeProcessController{err: expected}
	service := NewService(Options{ProcessController: controller})

	got, err := service.CancelJob(context.Background(), JobRequest{CWD: cwd, JobID: job.ID})
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v, want %v", err, expected)
	}
	if got.Status != result.StatusCancelled {
		t.Fatalf("result = %#v, want persisted cancelled result", got)
	}
	if _, err := store.ReadPID(job.ID); !os.IsNotExist(err) {
		t.Fatalf("ReadPID after process error = %v, want missing pid", err)
	}
}

type fakeProcessController struct {
	mu          sync.Mutex
	pid         int
	termGrace   time.Duration
	killGrace   time.Duration
	count       int
	err         error
	started     chan struct{}
	release     chan struct{}
	startedOnce sync.Once
}

func (f *fakeProcessController) TerminateAndWait(pid int, termGrace, killGrace time.Duration) error {
	f.mu.Lock()
	f.pid = pid
	f.termGrace = termGrace
	f.killGrace = killGrace
	f.count++
	if f.started != nil {
		f.startedOnce.Do(func() { close(f.started) })
	}
	f.mu.Unlock()
	if f.release != nil {
		<-f.release
	}
	return f.err
}

func (f *fakeProcessController) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count
}
