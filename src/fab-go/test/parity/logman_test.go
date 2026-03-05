package parity

import (
	"testing"
)

func TestLogman(t *testing.T) {
	checkPrereqs(t)

	tests := []struct {
		name     string
		bashArgs []string
		goArgs   []string
		checkFile string // relative path to check for file mutations
	}{
		{
			name:      "command log",
			bashArgs:  []string{"command", "fab-test", changeID},
			goArgs:    []string{"log", "command", "fab-test", changeID},
			checkFile: historyPath,
		},
		{
			name:      "confidence log",
			bashArgs:  []string{"confidence", changeID, "4.5", "+0.3", "spec"},
			goArgs:    []string{"log", "confidence", changeID, "4.5", "+0.3", "spec"},
			checkFile: historyPath,
		},
		{
			name:      "review passed log",
			bashArgs:  []string{"review", changeID, "passed"},
			goArgs:    []string{"log", "review", changeID, "passed"},
			checkFile: historyPath,
		},
		{
			name:      "review failed log",
			bashArgs:  []string{"review", changeID, "failed", "fix-code"},
			goArgs:    []string{"log", "review", changeID, "failed", "fix-code"},
			checkFile: historyPath,
		},
		{
			name:      "transition log",
			bashArgs:  []string{"transition", changeID, "tasks", "finish", "", "", "fab-ff"},
			goArgs:    []string{"log", "transition", changeID, "tasks", "finish", "", "", "fab-ff"},
			checkFile: historyPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpBash := setupTempRepo(t)
			tmpGo := setupTempRepo(t)

			bashRes := runBash(t, tmpBash, "logman.sh", tt.bashArgs...)
			goRes := runGo(t, tmpGo, tt.goArgs...)

			assertParity(t, tt.name, bashRes, goRes)

			if tt.checkFile != "" {
				assertFileParity(t, tt.name, tmpBash, tmpGo, tt.checkFile)
			}
		})
	}
}
