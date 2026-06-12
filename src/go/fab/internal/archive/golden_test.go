package archive

// Full-content golden test for the generated archive index (260612-tb6f,
// F41). The archive index is machine-maintained (updateIndex / backfillIndex /
// removeFromIndex rewrite it atomically); pinning the exact bytes after an
// archive guards the format skills and humans read against accidental churn
// from rendering or serialization changes.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGolden_ArchiveIndexFullContent(t *testing.T) {
	fabRoot := setupArchiveFixture(t)
	folder := "260310-abcd-my-change"

	if _, err := Archive(fabRoot, folder, "My change description"); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	indexFile := filepath.Join(fabRoot, "changes", "archive", "index.md")
	data, err := os.ReadFile(indexFile)
	if err != nil {
		t.Fatalf("read archive index: %v", err)
	}

	want := "# Archive Index\n" +
		"\n" +
		"- **260310-abcd-my-change** — My change description\n"
	if string(data) != want {
		t.Errorf("archive index golden mismatch.\n--- got ---\n%q\n--- want ---\n%q", data, want)
	}
}
