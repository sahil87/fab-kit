// Package lockfile provides bounded exclusive advisory locking for fab's
// shared state files — the current consumer is .status.yaml (via
// cmd/fab/status.go, preflight.go, and internal/score). (It formerly also
// guarded .fab-runtime.yaml; that file was removed when agent-state
// production was divested — see cmd/fab/hook.go.)
//
// Concurrency posture: the existing temp+rename writes prevent torn files but
// not lost updates — an unlocked load-mutate-save cycle from two processes
// (e.g. fab status commands in different panes over one change) is
// last-writer-wins over the whole document. Every load-mutate-save cycle on a
// shared state file must therefore run while holding this lock.
//
// The lock is a sibling file (<guarded-path>.lock) held via flock(2) —
// advisory, cross-process, and released automatically when the holding
// process exits. Lock files are never deleted: flock state, not file
// existence, carries the lock. Both supported GOOS (linux, darwin) implement
// flock; no build-tag split is needed.
package lockfile

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// acquireTimeout bounds lock acquisition so a pathological holder (e.g. a
// stage hook re-invoking `fab status` on the same change while the parent
// command holds the lock) yields a clear error instead of an indefinite
// deadlock. Variables (not constants) so tests can shorten them.
var (
	acquireTimeout = 10 * time.Second

	// retryInterval is the poll cadence between non-blocking flock attempts.
	// The uncontended path acquires on the first attempt with no sleep.
	retryInterval = 25 * time.Millisecond
)

// Lock acquires an exclusive advisory lock guarding path, creating the
// sibling lock file (<path>.lock) when absent. It returns an unlock function
// that releases the lock and closes the descriptor. Acquisition is bounded:
// after acquireTimeout of contention the call fails with an error naming the
// lock file.
func Lock(path string) (func(), error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file %s: %w", lockPath, err)
	}

	deadline := time.Now().Add(acquireTimeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN && err != syscall.EINTR {
			_ = f.Close()
			return nil, fmt.Errorf("lock %s: %w", lockPath, err)
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("timed out waiting for lock %s (held by another fab process)", lockPath)
		}
		time.Sleep(retryInterval)
	}

	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

// WithLock runs fn while holding the exclusive lock guarding path.
func WithLock(path string, fn func() error) error {
	unlock, err := Lock(path)
	if err != nil {
		return err
	}
	defer unlock()
	return fn()
}
