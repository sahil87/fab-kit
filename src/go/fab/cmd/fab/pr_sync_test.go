package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/prmeta"
)

func TestPrSyncCmd_RegisteredWithExpectedUse(t *testing.T) {
	cmd := prSyncCmd()
	if !strings.HasPrefix(cmd.Use, "pr-sync ") {
		t.Errorf("prSyncCmd Use = %q, want prefix \"pr-sync \"", cmd.Use)
	}
	if cmd.Flags().Lookup("type") == nil {
		t.Error("prSyncCmd missing --type flag")
	}
	if cmd.Flags().Lookup("issues") == nil {
		t.Error("prSyncCmd missing --issues flag")
	}
}

func TestPrSyncCmd_TypeRequired(t *testing.T) {
	cmd := prSyncCmd()
	cmd.SetArgs([]string{"some-change"}) // no --type
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --type is omitted")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Errorf("error should mention the required type flag, got: %v", err)
	}
}

func TestPrSyncCmd_NoFabContextExitsNonZero(t *testing.T) {
	// Mirror TestPrMetaCmd_NoFabContextExitsNonZero: run from a temp dir with no
	// fab/ ancestor; the command must error and print nothing to stdout, with
	// no gh call ever reached.
	tmp := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	// Stub the seams to prove they are never touched on the no-context path.
	origReader, origWriter := prBodyReader, prBodyWriter
	defer func() { prBodyReader, prBodyWriter = origReader, origWriter }()
	prBodyReader = func(string) (string, error) {
		t.Fatal("prBodyReader must not run without fab context")
		return "", nil
	}
	prBodyWriter = func(string, string) error {
		t.Fatal("prBodyWriter must not run without fab context")
		return nil
	}

	cmd := prSyncCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"nonexistent", "--type", "feat"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected non-zero exit when there is no fab context")
	}
	if out.Len() != 0 {
		t.Errorf("expected empty stdout on no-fab-context error, got %q", out.String())
	}
}

// TestBranchMatchesChange covers the pr-sync branch guard (the analog of
// git-pr.md Step 0 item 4): exact match, substring match, detached HEAD, and
// mismatch.
func TestBranchMatchesChange(t *testing.T) {
	const folder = "261007-gyp9-early-draft-pr-live-meta"

	t.Run("exact match", func(t *testing.T) {
		if err := branchMatchesChange(folder, folder); err != nil {
			t.Errorf("exact match must pass, got: %v", err)
		}
	})

	t.Run("substring match", func(t *testing.T) {
		if err := branchMatchesChange("feat/"+folder, folder); err != nil {
			t.Errorf("substring match must pass, got: %v", err)
		}
	})

	t.Run("detached HEAD", func(t *testing.T) {
		err := branchMatchesChange("", folder)
		if err == nil || !strings.Contains(err.Error(), "detached HEAD") {
			t.Errorf("empty branch must error naming detached HEAD, got: %v", err)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		err := branchMatchesChange("261001-abcd-some-other-change", folder)
		if err == nil || !strings.Contains(err.Error(), "does not match") {
			t.Errorf("mismatched branch must error, got: %v", err)
		}
	})
}

// TestSyncPRBody covers the network-free core: the no-op-on-equal-body path
// (no write issued), the apply-on-change path, and the read-failure path
// (no open PR → error, no write).
func TestSyncPRBody(t *testing.T) {
	rendered := prmeta.Render(prmeta.Data{ID: "rj31", Type: "feat"})

	t.Run("equal body: no-op, no write", func(t *testing.T) {
		body := rendered + "\n\n## Summary\n\nhand-edited\n"
		wrote := false
		applied, err := syncPRBody(rendered,
			func() (string, error) { return body, nil },
			func(string) error { wrote = true; return nil })
		if err != nil {
			t.Fatalf("syncPRBody: %v", err)
		}
		if applied {
			t.Error("expected no-op on equal body, got applied=true")
		}
		if wrote {
			t.Error("write must not be called when the spliced body equals the current body")
		}
	})

	t.Run("stale body: applies the spliced body exactly once", func(t *testing.T) {
		stale := "## Meta\n\nold unmarked block\n\n## Summary\n\nhand-edited\n"
		var written string
		applied, err := syncPRBody(rendered,
			func() (string, error) { return stale, nil },
			func(b string) error { written = b; return nil })
		if err != nil {
			t.Fatalf("syncPRBody: %v", err)
		}
		if !applied {
			t.Error("expected applied=true on a stale body")
		}
		want := rendered + "\n\n## Summary\n\nhand-edited\n"
		if written != want {
			t.Errorf("written body =\n%q\nwant\n%q", written, want)
		}
	})

	t.Run("read failure (no open PR): error, no write", func(t *testing.T) {
		wrote := false
		_, err := syncPRBody(rendered,
			func() (string, error) { return "", errors.New("no pull requests found") },
			func(string) error { wrote = true; return nil })
		if err == nil {
			t.Fatal("expected error when the branch has no open PR")
		}
		if !strings.Contains(err.Error(), "no open PR") {
			t.Errorf("error should name the missing open PR, got: %v", err)
		}
		if wrote {
			t.Error("write must not be called when the read fails")
		}
	})
}
