package internal

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireLock_AcquireAndRelease(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "0.1.0.lock")

	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("acquireLock failed: %v", err)
	}
	release()

	// Lock file is left in place after release (by design).
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("expected lock file to remain after release: %v", err)
	}

	// Re-acquiring after release must succeed.
	release2, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("re-acquire after release failed: %v", err)
	}
	release2()
}

func TestAcquireLock_MutualExclusion(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "0.1.0.lock")

	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("acquireLock failed: %v", err)
	}

	var inCritical atomic.Bool
	inCritical.Store(true)

	var wg sync.WaitGroup
	wg.Add(1)
	violation := make(chan struct{}, 1)
	go func() {
		defer wg.Done()
		// flock is per-open-file-description, so a second open in the same
		// process still contends with the first.
		release2, err := acquireLock(lockPath)
		if err != nil {
			t.Errorf("second acquireLock failed: %v", err)
			return
		}
		defer release2()
		if inCritical.Load() {
			violation <- struct{}{}
		}
	}()

	// Hold the lock briefly, then leave the critical section and release.
	time.Sleep(50 * time.Millisecond)
	inCritical.Store(false)
	release()
	wg.Wait()

	select {
	case <-violation:
		t.Error("second holder entered critical section while first held the lock")
	default:
	}
}
