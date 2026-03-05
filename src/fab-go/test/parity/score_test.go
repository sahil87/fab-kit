package parity

import (
	"testing"
)

func TestScore(t *testing.T) {
	checkPrereqs(t)

	t.Run("normal scoring from spec", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "calc-score.sh", changeID)
		goRes := runGo(t, tmpGo, "score", changeID)

		assertParity(t, "normal scoring", bashRes, goRes)
		assertFileParity(t, "normal scoring", tmpBash, tmpGo, statusPath)
	})

	t.Run("check-gate pass", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "calc-score.sh", "--check-gate", changeID)
		goRes := runGo(t, tmpGo, "score", "--check-gate", changeID)

		assertParity(t, "check-gate pass", bashRes, goRes)
	})

	t.Run("intake stage scoring", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "calc-score.sh", "--stage", "intake", changeID)
		goRes := runGo(t, tmpGo, "score", "--stage", "intake", changeID)

		assertParity(t, "intake scoring", bashRes, goRes)
		assertFileParity(t, "intake scoring", tmpBash, tmpGo, statusPath)
	})

	t.Run("intake gate check", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "calc-score.sh", "--check-gate", "--stage", "intake", changeID)
		goRes := runGo(t, tmpGo, "score", "--check-gate", "--stage", "intake", changeID)

		assertParity(t, "intake gate", bashRes, goRes)
	})
}
