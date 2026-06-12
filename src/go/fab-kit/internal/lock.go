package internal

import (
	"fmt"
	"os"
	"syscall"
)

// acquireLock opens (creating if needed) the lock file at path and takes a
// blocking exclusive advisory flock on it. The returned release function
// unlocks and closes the file.
//
// The lock file is deliberately left in place after release — unlinking it
// would race with other processes already blocked on the same file (they
// would hold a lock on an orphaned inode while a newcomer locks a fresh one).
//
// This helper is intentionally local to the fab-kit module: the two Go
// modules (src/go/fab and src/go/fab-kit) deliberately share no code — small
// utilities are replicated across the separate go.mod boundaries rather than
// imported (the documented two-module replication pattern, cf. the hooklib
// replication in 260402-ktbg).
func acquireLock(path string) (release func(), err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot create lock file %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("cannot acquire lock on %s: %w", path, err)
	}
	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}
