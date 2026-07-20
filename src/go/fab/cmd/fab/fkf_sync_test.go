package main

import (
	"bytes"
	"os"
	"testing"
)

// canonicalFKFRelPath is the repo-relative path of the authoritative FKF
// standard (published at https://shll.ai/fab-kit/fkf); shippedFKFRelPath is the
// kit copy that ships to the cache as $(fab kit-path)/reference/fkf.md and MUST
// track it byte-for-byte.
const (
	canonicalFKFRelPath = "docs/site/fkf.md"
	shippedFKFRelPath   = "src/kit/reference/fkf.md"
)

// TestFKFShippedCopyMatchesCanonical is the drift guard: the shipped kit copy
// src/kit/reference/fkf.md MUST equal the canonical docs/site/fkf.md. Unlike
// skill.md (go:embed-ed and guarded in skill_test.go), the FKF standard is not
// embedded — both files live outside the module root (src/go/fab/) — so the
// test reads both via repo-relative paths resolved with findRepoFile (shared
// with skill_test.go / lifecycle_collision_test.go). When someone edits
// docs/site/fkf.md without re-running scripts/sync-fkf.sh, this fails, naming
// the drifted file. Runs on every `go test ./...` and in CI.
func TestFKFShippedCopyMatchesCanonical(t *testing.T) {
	canonical, err := os.ReadFile(findRepoFile(t, canonicalFKFRelPath))
	if err != nil {
		t.Fatalf("read canonical %s: %v", canonicalFKFRelPath, err)
	}
	shipped, err := os.ReadFile(findRepoFile(t, shippedFKFRelPath))
	if err != nil {
		t.Fatalf("read shipped copy %s: %v", shippedFKFRelPath, err)
	}
	if !bytes.Equal(shipped, canonical) {
		t.Errorf("%s has drifted from canonical %s — run scripts/sync-fkf.sh and commit the refreshed copy",
			shippedFKFRelPath, canonicalFKFRelPath)
	}
}
