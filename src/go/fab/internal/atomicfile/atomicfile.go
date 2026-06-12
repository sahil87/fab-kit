// Package atomicfile provides crash-safe file replacement via the
// temp-file + rename pattern: content lands in a temp file in the
// destination directory (same filesystem, so the final rename is atomic),
// then replaces the target in one step. It mirrors the statusfile.Save
// pattern; statusfile.Save and runtime.SaveFile keep their own inline
// variants because they carry deliberately different fsync postures
// (mz4q F03/F04) — this always-fsync helper serves writers like the
// archive index where durability is wanted and the path is cold.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes data to path atomically. The temp file is synced and
// chmodded to perm before the rename, and removed on any failure — a crash
// or error mid-write leaves the original file untouched.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Ensure the temp file is cleaned up on any error path.
	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	success = true
	return nil
}
