package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionGuard_DevBypass(t *testing.T) {
	if err := versionGuard("0.99.0", "dev"); err != nil {
		t.Errorf("expected dev build to bypass guard, got: %v", err)
	}
}

func TestVersionGuard_SufficientVersion(t *testing.T) {
	if err := versionGuard("0.44.10", "0.44.10"); err != nil {
		t.Errorf("expected equal versions to pass, got: %v", err)
	}
	if err := versionGuard("0.44.9", "0.45.0"); err != nil {
		t.Errorf("expected older fab_version to pass, got: %v", err)
	}
}

// overrideGuardSeams overrides the two versionGuard seams and restores them on cleanup.
func overrideGuardSeams(t *testing.T, brewInstalled bool, installedVersion string, installedErr error) {
	t.Helper()
	origBrew := isBrewInstalled
	origInstalled := installedBinaryVersion
	isBrewInstalled = func() bool { return brewInstalled }
	installedBinaryVersion = func() (string, error) { return installedVersion, installedErr }
	t.Cleanup(func() {
		isBrewInstalled = origBrew
		installedBinaryVersion = origInstalled
	})
}

func TestVersionGuard_NotBrewInstalledFails(t *testing.T) {
	// Non-brew install: Update returns ErrNotBrewInstalled, installed binary
	// stays old — the guard must fail with actionable instructions (it used
	// to silently pass on Update's old nil return).
	overrideGuardSeams(t, false, "0.9.0", nil)

	err := versionGuard("1.0.0", "0.9.0")
	if err == nil {
		t.Fatal("expected guard to fail for non-brew install with too-old binary")
	}
	if !strings.Contains(err.Error(), "manually") {
		t.Errorf("expected actionable manual-update instructions, got: %v", err)
	}
}

func TestVersionGuard_PostStateUpdatedFailsCurrentSync(t *testing.T) {
	// The installed binary on PATH is new enough after the update attempt —
	// post-state decides (even though Update itself errored). The guard must
	// still fail the CURRENT sync so the next run uses the new binary.
	overrideGuardSeams(t, false, "1.0.0", nil)

	err := versionGuard("1.0.0", "0.9.0")
	if err == nil {
		t.Fatal("expected guard to fail the current sync after a successful update")
	}
	if !strings.Contains(err.Error(), "re-run 'fab sync'") {
		t.Errorf("expected re-run guidance, got: %v", err)
	}
}

func TestVersionGuard_BrewReleaseLagFails(t *testing.T) {
	// Brew-installed, but the tap's latest equals the current version
	// (release lag): Update returns nil having upgraded nothing. The
	// post-state check must catch the still-old binary.
	tmpDir := t.TempDir()
	brewScript := "#!/bin/sh\nif [ \"$1\" = \"info\" ]; then printf '%s' '{\"formulae\":[{\"versions\":{\"stable\":\"0.9.0\"}}]}'; fi\nexit 0\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "brew"), []byte(brewScript), 0755); err != nil {
		t.Fatalf("write fake brew: %v", err)
	}
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	overrideGuardSeams(t, true, "0.9.0", nil)

	err := versionGuard("1.0.0", "0.9.0")
	if err == nil {
		t.Fatal("expected guard to fail on brew release lag (Update no-op, binary still old)")
	}
	if !strings.Contains(err.Error(), "still older") {
		t.Errorf("expected release-lag error, got: %v", err)
	}
}

func TestVersionGuard_UnverifiablePostStateFails(t *testing.T) {
	// If the installed version cannot be verified after the update attempt,
	// the guard must fail rather than trust the nil return.
	overrideGuardSeams(t, false, "", fmt.Errorf("fab-kit not found on PATH"))

	err := versionGuard("1.0.0", "0.9.0")
	if err == nil {
		t.Fatal("expected guard to fail when post-state cannot be verified")
	}
}
