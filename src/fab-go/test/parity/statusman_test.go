package parity

import (
	"testing"
)

func TestStatusman(t *testing.T) {
	checkPrereqs(t)

	t.Run("progress-map", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "statusman.sh", "progress-map", changeID)
		goRes := runGo(t, tmpGo, "status", "progress-map", changeID)

		assertParity(t, "progress-map", bashRes, goRes)
	})

	t.Run("progress-line", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "statusman.sh", "progress-line", changeID)
		goRes := runGo(t, tmpGo, "status", "progress-line", changeID)

		assertParity(t, "progress-line", bashRes, goRes)
	})

	t.Run("current-stage", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "statusman.sh", "current-stage", changeID)
		goRes := runGo(t, tmpGo, "status", "current-stage", changeID)

		assertParity(t, "current-stage", bashRes, goRes)
	})

	t.Run("finish tasks", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "statusman.sh", "finish", changeID, "tasks", "parity-test")
		goRes := runGo(t, tmpGo, "status", "finish", changeID, "tasks", "parity-test")

		assertParity(t, "finish tasks", bashRes, goRes)
		assertFileParity(t, "finish tasks", tmpBash, tmpGo, statusPath)
	})

	t.Run("start apply", func(t *testing.T) {
		// First finish tasks to get apply into pending→active transition
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		// finish tasks (auto-activates apply)
		runBash(t, tmpBash, "statusman.sh", "finish", changeID, "tasks", "parity-test")
		runGo(t, tmpGo, "status", "finish", changeID, "tasks", "parity-test")

		// Now both should have apply=active — verify state
		bashRes := runBash(t, tmpBash, "statusman.sh", "progress-map", changeID)
		goRes := runGo(t, tmpGo, "status", "progress-map", changeID)

		assertParity(t, "start apply (after finish tasks)", bashRes, goRes)
		assertFileParity(t, "start apply", tmpBash, tmpGo, statusPath)
	})

	t.Run("set-change-type", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "statusman.sh", "set-change-type", changeID, "fix")
		goRes := runGo(t, tmpGo, "status", "set-change-type", changeID, "fix")

		assertParity(t, "set-change-type", bashRes, goRes)
		assertFileParity(t, "set-change-type", tmpBash, tmpGo, statusPath)
	})

	t.Run("set-checklist", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "statusman.sh", "set-checklist", changeID, "generated", "true")
		goRes := runGo(t, tmpGo, "status", "set-checklist", changeID, "generated", "true")

		assertParity(t, "set-checklist", bashRes, goRes)
		assertFileParity(t, "set-checklist", tmpBash, tmpGo, statusPath)
	})

	t.Run("set-confidence", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "statusman.sh", "set-confidence", changeID, "5", "1", "0", "0", "4.7", "--indicative")
		goRes := runGo(t, tmpGo, "status", "set-confidence", changeID, "5", "1", "0", "0", "4.7", "--indicative")

		assertParity(t, "set-confidence", bashRes, goRes)
		assertFileParity(t, "set-confidence", tmpBash, tmpGo, statusPath)
	})

	t.Run("advance", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "statusman.sh", "advance", changeID, "tasks", "parity-test")
		goRes := runGo(t, tmpGo, "status", "advance", changeID, "tasks", "parity-test")

		assertParity(t, "advance", bashRes, goRes)
		assertFileParity(t, "advance", tmpBash, tmpGo, statusPath)
	})

	t.Run("reset", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		// First finish intake→done to allow reset
		// Intake is already done in fixture, so reset spec (which is done)
		bashRes := runBash(t, tmpBash, "statusman.sh", "reset", changeID, "spec", "parity-test")
		goRes := runGo(t, tmpGo, "status", "reset", changeID, "spec", "parity-test")

		assertParity(t, "reset", bashRes, goRes)
		assertFileParity(t, "reset", tmpBash, tmpGo, statusPath)
	})

	t.Run("skip", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		// Skip apply (which is pending)
		bashRes := runBash(t, tmpBash, "statusman.sh", "skip", changeID, "apply", "parity-test")
		goRes := runGo(t, tmpGo, "status", "skip", changeID, "apply", "parity-test")

		assertParity(t, "skip", bashRes, goRes)
		assertFileParity(t, "skip", tmpBash, tmpGo, statusPath)
	})
}
