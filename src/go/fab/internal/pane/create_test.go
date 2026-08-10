package pane

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestSplitArgs pins the sized-split argv composition: `-l <n>%` appears only when
// the placement carries a size (i.e. only for a column-carving split), always as a
// PERCENTAGE so the column scales with the window, and the pane-id format request
// is present in both shapes so no follow-up lookup can race a fast-exiting worker.
func TestSplitArgs(t *testing.T) {
	carve := splitArgs(SplitPlacement{Target: "%1", Direction: SplitRight, SizePercent: 35},
		"/repo", "claude 'go'")
	wantCarve := []string{"split-window", "-h", "-t", "%1", "-l", "35%",
		"-P", "-F", "#{pane_id}", "-c", "/repo", "claude 'go'"}
	if !reflect.DeepEqual(carve, wantCarve) {
		t.Errorf("carving argv = %q, want %q", carve, wantCarve)
	}

	stack := splitArgs(SplitPlacement{Target: "%2", Direction: SplitBelow},
		"/repo", "claude 'go'")
	wantStack := []string{"split-window", "-v", "-t", "%2",
		"-P", "-F", "#{pane_id}", "-c", "/repo", "claude 'go'"}
	if !reflect.DeepEqual(stack, wantStack) {
		t.Errorf("stacking argv = %q, want %q", stack, wantStack)
	}

	// A zero/absent size is the UNSIZED signal in either direction — tmux's own even
	// split — never a literal `-l 0%`, which tmux would reject.
	unsizedCarve := splitArgs(SplitPlacement{Target: "%1", Direction: SplitRight}, "/repo", "cmd")
	for _, arg := range unsizedCarve {
		if arg == SizeFlag {
			t.Errorf("a zero-size placement must emit no %s argument, got %q", SizeFlag, unsizedCarve)
		}
	}
}

// TestSplitPlacementDescribe pins the placement's own vocabulary: Describe is the
// only cross-package reader of Direction, which is what keeps the bare tmux `-h`/`-v`
// flag from ever being RENDERED outside this package. The two phrasings are the ones
// the degraded-probe warning is documented with, so they are asserted rather than
// left to the cobra layer.
func TestSplitPlacementDescribe(t *testing.T) {
	carve := SplitPlacement{Target: "%1", Direction: SplitRight, SizePercent: 35}.Describe()
	if carve != "carving a new worker column off pane %1" {
		t.Errorf("carve description = %q, want the column-carving wording", carve)
	}
	stack := SplitPlacement{Target: "%2", Direction: SplitBelow}.Describe()
	if stack != "stacking the worker under pane %2" {
		t.Errorf("stack description = %q, want the stacking wording", stack)
	}
}

// stubTmux writes a fake `tmux` executable (a POSIX sh body) into a temp dir
// prepended to $PATH, so the pane creators' argv handling can be exercised without a
// tmux server — the stubBatchNewBinaries precedent in cmd/fab.
func stubTmux(t *testing.T, body string) {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestOpenSplitPane_RejectedSizeRetriesUnsized covers the size-degradation branch:
// a tmux that refuses `-l <n>%` — every tmux before 3.1, which had no percentage
// size — must not fail a dispatch that would otherwise launch. The split is retried
// with the size dropped and the refusal comes back as a WARNING, not an error.
//
// The refusal is stubbed rather than provoked: modern tmux CLAMPS an extreme
// percentage instead of rejecting it, so the only real-world trigger is a tmux too
// old to run this suite's other pane tests at all.
func TestOpenSplitPane_RejectedSizeRetriesUnsized(t *testing.T) {
	// Refuses any argv containing -l; otherwise prints a pane id (split-window) or
	// succeeds silently (select-pane). The argv of every call is appended to $ARGLOG.
	argLog := filepath.Join(t.TempDir(), "argv.log")
	t.Setenv("ARGLOG", argLog)
	stubTmux(t, `echo "$@" >> "$ARGLOG"
for a in "$@"; do
  if [ "$a" = "-l" ]; then echo "usage: split-window" >&2; exit 1; fi
done
case "$1" in split-window) echo "%42" ;; esac
exit 0`)

	place := SplitPlacement{Target: "%1", Direction: SplitRight, SizePercent: 35}
	paneID, warnings, err := OpenSplitPane("", place, "fab-abcd-apply", "/repo", "cmd")
	if err != nil {
		t.Fatalf("a rejected size must degrade, not fail the dispatch: %v", err)
	}
	if paneID != "%42" {
		t.Errorf("pane id = %q, want the retried split's %q", paneID, "%42")
	}
	if len(warnings) != 1 {
		t.Fatalf("want exactly one warning (the rejected size), got %d: %v", len(warnings), warnings)
	}
	for _, want := range []string{SizeFlag, "35%", "retrying unsized"} {
		if !strings.Contains(warnings[0].Error(), want) {
			t.Errorf("warning %q must name %q so the fallback is explainable from output", warnings[0], want)
		}
	}

	log, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatal(err)
	}
	calls := strings.Split(strings.TrimSpace(string(log)), "\n")
	if len(calls) != 3 {
		t.Fatalf("want 3 tmux calls (sized split, unsized retry, select-pane), got %d: %v", len(calls), calls)
	}
	if !strings.Contains(calls[0], "-l 35%") {
		t.Errorf("first call = %q, want the SIZED split attempted first", calls[0])
	}
	if strings.Contains(calls[1], "-l") {
		t.Errorf("retry = %q, want the size dropped", calls[1])
	}
	if !strings.Contains(calls[2], "select-pane") || !strings.Contains(calls[2], "fab-abcd-apply") {
		t.Errorf("third call = %q, want the identity title still set on the retried pane", calls[2])
	}
}

// TestOpenSplitPane_UnsizedSplitIsNotRetried: a stacking split carries no size, so a
// tmux failure there is a genuine placement failure — reported as an error with no
// second attempt (a blind retry would double-launch a worker).
func TestOpenSplitPane_UnsizedSplitIsNotRetried(t *testing.T) {
	argLog := filepath.Join(t.TempDir(), "argv.log")
	t.Setenv("ARGLOG", argLog)
	stubTmux(t, `echo "$@" >> "$ARGLOG"
echo "can't find pane" >&2
exit 1`)

	_, _, err := OpenSplitPane("", SplitPlacement{Target: "%2", Direction: SplitBelow}, "fab-abcd-apply", "/repo", "cmd")
	if err == nil {
		t.Fatal("a failing unsized split must be an error, not a silent degrade")
	}
	log, _ := os.ReadFile(argLog)
	if n := len(strings.Split(strings.TrimSpace(string(log)), "\n")); n != 1 {
		t.Errorf("tmux was called %d times, want exactly 1 (no retry without a size to drop)", n)
	}
}

// TestSplitFlagsAreDistinct pins the stacked-column rule's flags as the tmux flags
// they must be: the FIRST worker carves the column to the right of the dispatcher,
// later workers stack BELOW the previous worker, and the size rides tmux's `-l`.
// Swap the directions and every dispatch would shrink the dispatcher's own pane.
func TestSplitFlagsAreDistinct(t *testing.T) {
	if SplitRight != "-h" {
		t.Errorf("SplitRight = %q, want tmux's horizontal split flag -h", SplitRight)
	}
	if SplitBelow != "-v" {
		t.Errorf("SplitBelow = %q, want tmux's vertical split flag -v", SplitBelow)
	}
	if SizeFlag != "-l" {
		t.Errorf("SizeFlag = %q, want tmux's split size flag -l", SizeFlag)
	}
}

// TestPointerPromptNamesThePromptFile pins the DELIVERED pointer's shape. It is
// typed into the pane by `fab pane deliver` / `fab dispatch deliver`, not embedded
// at spawn, so the two properties that matter are that it names the prompt path and
// that it is a single line — a newline would submit the pointer half-typed.
func TestPointerPromptNamesThePromptFile(t *testing.T) {
	got := PointerPrompt(".fab-dispatch/abcd/apply-prompt.md")
	if !strings.Contains(got, ".fab-dispatch/abcd/apply-prompt.md") {
		t.Errorf("PointerPrompt = %q, want it to name the prompt path", got)
	}
	if strings.Contains(got, "\n") {
		t.Errorf("PointerPrompt = %q, want a single line", got)
	}
}

// TestPlainPaneArgs pins the provider-generic spawn's argv shapes: a plain
// split carries no target, size, or title (placement is dispatch policy, and
// tmux resolves the current pane from the invoking client), and the window
// fallback is UNNAMED. Both print the pane id so no follow-up lookup can race
// a fast-exiting command.
func TestPlainPaneArgs(t *testing.T) {
	split := plainPaneArgs(true, "/repo", "kimi --agent")
	wantSplit := []string{"split-window", "-P", "-F", "#{pane_id}", "-c", "/repo", "kimi --agent"}
	if !reflect.DeepEqual(split, wantSplit) {
		t.Errorf("split argv = %q, want %q", split, wantSplit)
	}

	window := plainPaneArgs(false, "/repo", "kimi --agent")
	wantWindow := []string{"new-window", "-P", "-F", "#{pane_id}", "-c", "/repo", "kimi --agent"}
	if !reflect.DeepEqual(window, wantWindow) {
		t.Errorf("window argv = %q, want %q", window, wantWindow)
	}
	for _, arg := range window {
		if arg == "-n" {
			t.Errorf("the window fallback must be unnamed, got %q", window)
		}
	}
}
