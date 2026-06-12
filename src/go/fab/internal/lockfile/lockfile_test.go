package lockfile

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLock_AcquireRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.yaml")

	unlock, err := Lock(path)
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	unlock()

	// Re-acquire after release must succeed immediately.
	unlock2, err := Lock(path)
	if err != nil {
		t.Fatalf("re-Lock after unlock failed: %v", err)
	}
	unlock2()
}

func TestLock_CreatesSiblingLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.yaml")

	unlock, err := Lock(path)
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}
	defer unlock()

	if _, err := os.Stat(path + ".lock"); err != nil {
		t.Errorf("expected sibling lock file %s.lock to exist: %v", path, err)
	}
	// The guarded file itself is never created by the lock.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Lock must not create the guarded file itself")
	}
}

func TestLock_Timeout(t *testing.T) {
	origTimeout, origRetry := acquireTimeout, retryInterval
	acquireTimeout, retryInterval = 100*time.Millisecond, 10*time.Millisecond
	defer func() { acquireTimeout, retryInterval = origTimeout, origRetry }()

	path := filepath.Join(t.TempDir(), "state.yaml")

	unlock, err := Lock(path)
	if err != nil {
		t.Fatalf("first Lock failed: %v", err)
	}
	defer unlock()

	// flock is per open-file-description, so a second Lock in the same
	// process contends with the first.
	_, err = Lock(path)
	if err == nil {
		t.Fatal("expected second Lock to time out while first is held")
	}
	if !strings.Contains(err.Error(), "timed out waiting for lock") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestWithLock_MutualExclusion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.yaml")

	const workers = 8
	inCritical := 0
	maxInCritical := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := WithLock(path, func() error {
				mu.Lock()
				inCritical++
				if inCritical > maxInCritical {
					maxInCritical = inCritical
				}
				mu.Unlock()

				time.Sleep(5 * time.Millisecond)

				mu.Lock()
				inCritical--
				mu.Unlock()
				return nil
			})
			if err != nil {
				t.Errorf("WithLock failed: %v", err)
			}
		}()
	}
	wg.Wait()

	if maxInCritical != 1 {
		t.Errorf("expected at most 1 worker in critical section, observed %d", maxInCritical)
	}
}

func TestWithLock_PropagatesError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.yaml")

	want := os.ErrPermission
	err := WithLock(path, func() error { return want })
	if err != want {
		t.Errorf("expected fn error to propagate, got: %v", err)
	}
}
