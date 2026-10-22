package executil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const completionGrace = 5 * time.Second

// lineObserver keeps all raw bytes and observes each line once, including an
// unterminated last line. Only the os/exec stdout-copy goroutine writes it.
type lineObserver struct {
	raw       *bytes.Buffer
	pending   []byte
	observe   func(string) (bool, error)
	terminal  chan struct{}
	signalled bool
	failure   chan error
}

func (w *lineObserver) emit(line []byte) error {
	terminal, err := w.observe(string(line))
	if err != nil {
		select {
		case w.failure <- err:
		default:
		}
	}
	if terminal && !w.signalled {
		w.signalled = true
		close(w.terminal)
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

func (r OSRunner) runObserved(ctx context.Context, cmd *exec.Cmd, stdout *bytes.Buffer) (bool, error) {
	prepareOwnedCommand(cmd)
	terminal := make(chan struct{})
	failure := make(chan error, 1)
	writer := &lineObserver{raw: stdout, observe: r.ObserveStdout, terminal: terminal, failure: failure}
	cmd.Stdout = writer
	// Descendants must not hold stdout/stderr open forever after the child exits.
	cmd.WaitDelay = completionGrace
	if err := cmd.Start(); err != nil {
		return false, err
	}
	stop, release, err := ownProcess(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return false, fmt.Errorf("own target process: %w", err)
	}
	defer release()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var timer *time.Timer
	var deadline <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case err := <-done:
			select {
			case observeErr := <-failure:
				return false, observeErr
			default:
			}
			if len(writer.pending) > 0 {
				if observeErr := writer.emit(writer.pending); err == nil {
					err = observeErr
				}
			}
			if err != nil && (writer.signalled || errors.Is(err, exec.ErrWaitDelay)) {
				// The direct child exited, but a descendant retained its pipes.
				// WaitDelay and the completion timer can expire together.
				if stopErr := stop(); stopErr != nil {
					return false, fmt.Errorf("stop target retaining output: %w", stopErr)
				}
				if writer.signalled && errors.Is(err, exec.ErrWaitDelay) {
					return true, nil
				}
			}
			return false, err
		case <-terminal:
			terminal = nil
			grace := r.CompletionGrace
			if grace <= 0 {
				grace = completionGrace
			}
			timer = time.NewTimer(grace)
			deadline = timer.C
		case <-ctx.Done():
			if err := stop(); err != nil {
				_ = cmd.Process.Kill()
			}
			_ = <-done
			return false, ctx.Err()
		case failureErr := <-failure:
			if err := stop(); err != nil {
				_ = cmd.Process.Kill()
			}
			_ = <-done
			return false, failureErr
		case <-deadline:
			stopErr := stop()
			if stopErr != nil {
				_ = cmd.Process.Kill()
			}
			waitErr := <-done
			select {
			case observeErr := <-failure:
				return false, observeErr
			default:
			}
			if stopErr != nil {
				return false, fmt.Errorf("stop completed target: %w", stopErr)
			}
			var exitErr *exec.ExitError
			if waitErr != nil && !errors.As(waitErr, &exitErr) && !errors.Is(waitErr, exec.ErrWaitDelay) {
				return false, waitErr
			}
			return true, nil
		}
	}
}
