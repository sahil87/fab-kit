package parity

import (
	"testing"
)

func TestChangeman(t *testing.T) {
	checkPrereqs(t)

	t.Run("list", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "changeman.sh", "list")
		goRes := runGo(t, tmpGo, "change", "list")

		assertParity(t, "list", bashRes, goRes)
	})

	t.Run("resolve with full name", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "changeman.sh", "resolve", changeName)
		goRes := runGo(t, tmpGo, "change", "resolve", changeName)

		assertParity(t, "resolve full name", bashRes, goRes)
	})

	t.Run("resolve with ID", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "changeman.sh", "resolve", changeID)
		goRes := runGo(t, tmpGo, "change", "resolve", changeID)

		assertParity(t, "resolve ID", bashRes, goRes)
	})

	t.Run("switch", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "changeman.sh", "switch", changeName)
		goRes := runGo(t, tmpGo, "change", "switch", changeName)

		assertParity(t, "switch", bashRes, goRes)
	})

	t.Run("switch blank", func(t *testing.T) {
		tmpBash := setupTempRepo(t)
		tmpGo := setupTempRepo(t)

		bashRes := runBash(t, tmpBash, "changeman.sh", "switch", "--blank")
		goRes := runGo(t, tmpGo, "change", "switch", "--blank")

		assertParity(t, "switch blank", bashRes, goRes)
	})
}
