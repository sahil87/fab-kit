package parity

import (
	"testing"
)

func TestResolve(t *testing.T) {
	checkPrereqs(t)

	tests := []struct {
		name     string
		bashArgs []string
		goArgs   []string
	}{
		{
			name:     "default (--id) with full name",
			bashArgs: []string{"--id", changeName},
			goArgs:   []string{"resolve", "--id", changeName},
		},
		{
			name:     "default (--id) with change ID",
			bashArgs: []string{"--id", changeID},
			goArgs:   []string{"resolve", "--id", changeID},
		},
		{
			name:     "--folder with full name",
			bashArgs: []string{"--folder", changeName},
			goArgs:   []string{"resolve", "--folder", changeName},
		},
		{
			name:     "--folder with substring",
			bashArgs: []string{"--folder", "parity-test"},
			goArgs:   []string{"resolve", "--folder", "parity-test"},
		},
		{
			name:     "--dir with change ID",
			bashArgs: []string{"--dir", changeID},
			goArgs:   []string{"resolve", "--dir", changeID},
		},
		{
			name:     "--status with change ID",
			bashArgs: []string{"--status", changeID},
			goArgs:   []string{"resolve", "--status", changeID},
		},
		{
			name:     "no flag defaults to --id",
			bashArgs: []string{changeName},
			goArgs:   []string{"resolve", changeName},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpBash := setupTempRepo(t)
			tmpGo := setupTempRepo(t)

			bashRes := runBash(t, tmpBash, "resolve.sh", tt.bashArgs...)
			goRes := runGo(t, tmpGo, tt.goArgs...)

			assertParity(t, tt.name, bashRes, goRes)
		})
	}
}
