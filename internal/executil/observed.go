package executil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

const completionGrace = 5 * time.Second

// CompletionObservation describes a change, rather than repeating a sticky
// terminal flag on every stdout line. Continued activity can revoke a candidate.
type CompletionObservation uint8

const (
	CompletionUnchanged CompletionObservation = iota
	CompletionPending
	CompletionActive
)

type completionTracker struct {
	mu         sync.Mutex
	pending    bool
	generation uint64
	changed    chan struct{}
}

func (s *completionTracker) update(observation CompletionObservation) {
	if observation == CompletionUnchanged {
		return
	}
	s.mu.Lock()
	s.pending = observation == CompletionPending
	s.generation++
	s.mu.Unlock()
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

func (s *completionTracker) snapshot() (bool, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pending, s.generation
}

// lineObserver keeps all raw bytes and observes each line once, including an
// unterminated last line. Only the stdout-copy goroutine writes it.
type lineObserver struct {
	raw        *bytes.Buffer
	pending    []byte
	observe    func(string) (CompletionObservation, error)
	completion *completionTracker
	failure    error
}

func (w *lineObserver) emit(line []byte) error {
	observation, err := w.observe(string(line))
	if err == nil {
		w.completion.update(observation)
	} else {
		w.failure = err
	}
	return err
}

func (w *lineObserver) Write(p []byte) (int, error) {
	w.raw.Write(p)
	w.pending = append(w.pending, p...)
	for {
		i := bytes.IndexByte(w.pending, '\n')
		if i < 0 {
			break
		}
		if err := w.emit(w.pending[:i]); err != nil {
			return len(p), err
		}
		w.pending = w.pending[i+1:]
	}
	return len(p), nil
}

func (w *lineObserver) finish() error {
	if len(w.pending) == 0 {
		return nil
	}
	return w.emit(w.pending)
}

// waitExit observes exit without releasing Unix group ownership; reap is the
// only operation allowed to release that reservation. Windows retains a Job
// Object; unsupported platforms use only the safe direct-process handle.
type ownedCommand struct {
	stop     func() error
	waitExit func() error
	reap     func() error
	release  func()
}

func (r OSRunner) runObserved(ctx context.Context, cmd *exec.Cmd, stdout *bytes.Buffer) (bool, error) {
	prepareOwnedCommand(cmd)
	// This loop owns cancellation and draining. In particular, launcher exit alone
	// must not start an os/exec pipe deadline or reap the Unix group leader.
	cmd.Cancel = nil
	cmd.WaitDelay = 0
	stderr := cmd.Stderr
	cmd.Stdout, cmd.Stderr = nil, nil
	out, err := cmd.StdoutPipe()
	if err != nil {
		return false, err
	}
	defer out.Close()
	errout, err := cmd.StderrPipe()
	if err != nil {
		return false, err
	}
	defer errout.Close()
	if err := cmd.Start(); err != nil {
		return false, err
	}
	owned, err := ownProcess(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = out.Close()
		_ = errout.Close()
		_ = cmd.Wait()
		return false, fmt.Errorf("own target process: %w", err)
	}
	defer owned.release()
	completion := &completionTracker{changed: make(chan struct{}, 1)}
	writer := &lineObserver{raw: stdout, observe: r.ObserveStdout, completion: completion}
	drained := make(chan error, 2)
	go func() {
		_, err := io.Copy(writer, out)
		if err == nil {
			err = writer.finish()
		}
		drained <- err
	}()
	go func() { _, err := io.Copy(stderr, errout); drained <- err }()
	exited := make(chan error, 1)
	go func() { exited <- owned.waitExit() }()
	remaining := 2
	exitObserved := false
	var timer *time.Timer
	var deadline <-chan time.Time
	var generation uint64
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	// A forced stop closes only our pipe ends, so an escaped descriptor holder
	// cannot keep cleanup blocked. All copy goroutines finish before buffers read.
	force := func() (error, error) {
		stopErr := owned.stop()
		if stopErr != nil {
			_ = cmd.Process.Kill()
		}
		_ = out.Close()
		_ = errout.Close()
		for remaining > 0 {
			<-drained
			remaining--
		}
		if !exitObserved {
			<-exited
			exitObserved = true
		}
		return stopErr, owned.reap()
	}
	for {
		if exitObserved && remaining == 0 {
			// No more observations are possible. Revoke group signals before reaping;
			// a natural nonzero exit never causes a late numeric group signal.
			return false, owned.reap()
		}
		select {
		case err := <-exited:
			exitObserved = true
			if err != nil {
				_, _ = force()
				return false, fmt.Errorf("observe target exit: %w", err)
			}
		case err := <-drained:
			remaining--
			if err != nil {
				_, _ = force()
				return false, err
			}
		case <-completion.changed:
			pending, current := completion.snapshot()
			if timer != nil {
				timer.Stop()
			}
			deadline = nil
			if pending {
				grace := r.CompletionGrace
				if grace <= 0 {
					grace = completionGrace
				}
				timer = time.NewTimer(grace)
				deadline, generation = timer.C, current
			}
		case <-ctx.Done():
			_, _ = force()
			return false, ctx.Err()
		case <-deadline:
			// An activity observation can race a ready timer. Check its generation
			// before signalling, rather than letting select choose a stale deadline.
			pending, current := completion.snapshot()
			deadline = nil
			if !pending || current != generation {
				continue
			}
			stopErr, waitErr := force()
			if writer.failure != nil {
				return false, writer.failure
			}
			if stopErr != nil {
				return false, fmt.Errorf("stop completed target: %w", stopErr)
			}
			var exitErr *exec.ExitError
			if waitErr != nil && !errors.As(waitErr, &exitErr) {
				return false, waitErr
			}
			return true, nil
		}
	}
}
