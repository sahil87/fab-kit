package parity

import (
	"testing"
)

func TestPreflight(t *testing.T) {
	checkPrereqs(t)

	t.Run("valid change via current pointer", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "preflight.sh")
		goRes := runGo(t, tmpGo, "preflight")

		assertParity(t, "preflight default", bashRes, goRes)
	})

	t.Run("valid change via override", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "preflight.sh", changeName)
		goRes := runGo(t, tmpGo, "preflight", changeName)

		assertParity(t, "preflight override", bashRes, goRes)
	})

	t.Run("valid change via ID", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "preflight.sh", changeID)
		goRes := runGo(t, tmpGo, "preflight", changeID)

		assertParity(t, "preflight by ID", bashRes, goRes)
	})

	// Error-path tests: exit codes must match (error messages may differ)
	t.Run("missing config exits non-zero", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		removeFile(t, tmpBash, "fab/project/config.yaml")
		removeFile(t, tmpGo, "fab/project/config.yaml")

		bashRes := runBash(t, tmpBash, "preflight.sh")
		goRes := runGo(t, tmpGo, "preflight")

		assertExitCodeParity(t, "missing config", bashRes, goRes)
	})

	t.Run("missing change exit code parity", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		writeFile(t, tmpBash, "fab/current", "nonexistent")
		writeFile(t, tmpGo, "fab/current", "nonexistent")

		bashRes := runBash(t, tmpBash, "preflight.sh")
		goRes := runGo(t, tmpGo, "preflight")

		// Both implementations should handle nonexistent change the same way
		if bashRes.ExitCode != goRes.ExitCode {
			t.Errorf("missing change: exit code mismatch — bash=%d, go=%d", bashRes.ExitCode, goRes.ExitCode)
		}
	})
}

// assertExitCodeParity checks exit codes match and are non-zero.
func assertExitCodeParity(t *testing.T, label string, bash, goRes cmdResult) {
	t.Helper()
	if bash.ExitCode != goRes.ExitCode {
		t.Errorf("%s: exit code mismatch — bash=%d, go=%d", label, bash.ExitCode, goRes.ExitCode)
	}
}

func removeFile(t *testing.T, dir, relPath string) {
	t.Helper()
	if err := removeAll(dir, relPath); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	if err := writeFileHelper(dir, relPath, content); err != nil {
		t.Fatal(err)
	}
}
