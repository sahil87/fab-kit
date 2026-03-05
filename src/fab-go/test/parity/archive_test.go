package parity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchive(t *testing.T) {
	checkPrereqs(t)

	t.Run("list empty", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		if err := os.MkdirAll(filepath.Join(tmpBash, "fab", "changes", "archive"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(tmpGo, "fab", "changes", "archive"), 0o755); err != nil {
			t.Fatal(err)
		}

		bashRes := runBash(t, tmpBash, "archiveman.sh", "list")
		goRes := runGo(t, tmpGo, "archive", "list")

		assertParity(t, "list empty", bashRes, goRes)
	})

	t.Run("archive change", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		if err := os.MkdirAll(filepath.Join(tmpBash, "fab", "changes", "archive"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(tmpGo, "fab", "changes", "archive"), 0o755); err != nil {
			t.Fatal(err)
		}

		bashRes := runBash(t, tmpBash, "archiveman.sh", "archive", changeID, "--description", "test archive")
		goRes := runGo(t, tmpGo, "archive", changeID, "--description", "test archive")

		assertParity(t, "archive", bashRes, goRes)
	})
}
