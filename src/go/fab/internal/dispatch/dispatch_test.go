package dispatch

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDeriveState(t *testing.T) {
	tests := []struct {
		name          string
		exitPresent   bool
		exitCode      int
		resultPresent bool
		alive         bool
		want          State
	}{
		{"running: no exit, alive", false, 0, false, true, StateRunning},
		{"running ignores result while alive", false, 0, true, true, StateRunning},
		{"orphaned: no exit, dead", false, 0, false, false, StateOrphaned},
		{"orphaned ignores result when dead+no-exit", false, 0, true, false, StateOrphaned},
		{"done: exit 0 + result", true, 0, true, false, StateDone},
		{"done: exit 0 + result even if alive races", true, 0, true, true, StateDone},
		{"failed no-result: exit 0, no result", true, 0, false, false, StateFailedNoResult},
		{"failed: non-zero exit", true, 1, false, false, StateFailed},
		{"failed: non-zero exit ignores result", true, 1, true, false, StateFailed},
		{"failed: timeout 124", true, 124, false, false, StateFailed},
		{"failed: timeout 124 ignores result", true, 124, true, false, StateFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveState(tt.exitPresent, tt.exitCode, tt.resultPresent, tt.alive)
			if got != tt.want {
				t.Errorf("DeriveState(%v,%d,%v,%v) = %q, want %q",
					tt.exitPresent, tt.exitCode, tt.resultPresent, tt.alive, got, tt.want)
			}
		})
	}
}

// TestDerivePaneState exhausts the pane-mode three-state subset. Two properties
// are asserted deliberately: result presence WINS over pane liveness (an
// interactive worker sits at its prompt after finishing, so liveness-first would
// read `running` forever), and no input combination can produce the two
// exit-code-derived states — pane mode has no exit-code channel.
func TestDerivePaneState(t *testing.T) {
	tests := []struct {
		name          string
		resultPresent bool
		paneAlive     bool
		want          State
	}{
		{"running: alive, no result", false, true, StateRunning},
		{"orphaned: dead, no result", false, false, StateOrphaned},
		{"done: result present, pane dead", true, false, StateDone},
		{"done wins while the pane still lives", true, true, StateDone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DerivePaneState(tt.resultPresent, tt.paneAlive)
			if got != tt.want {
				t.Errorf("DerivePaneState(%v,%v) = %q, want %q",
					tt.resultPresent, tt.paneAlive, got, tt.want)
			}
			if got == StateFailed || got == StateFailedNoResult {
				t.Errorf("DerivePaneState produced %q, which is unreachable on the pane path", got)
			}
		})
	}
}

// TestSelectMode exhausts the configured descent ladder plus explicit precedence.
func TestSelectMode(t *testing.T) {
	tests := []struct {
		name                   string
		paneFlag, headlessFlag bool
		timeoutSet, serverSet  bool
		preference             string
		native, session        bool
		dispatch               bool
		tmux                   TmuxAvailability
		wantMode               Mode
		wantReason             AutoReason
		wantErr                bool
	}{
		{"pane preferred and possible", false, false, false, false, "pane", true, true, true, TmuxAvailable, ModePane, "mode: pane (preferred)", false},
		{"pane to native: no tmux", false, false, false, false, "pane", true, true, true, TmuxAbsent, ModeNative, "mode: native (descended: pane unavailable: no tmux)", false},
		{"pane to native: unreachable", false, false, false, false, "pane", true, true, true, TmuxUnreachable, ModeNative, "mode: native (descended: pane unavailable: tmux unreachable)", false},
		{"pane to native: no session", false, false, false, false, "pane", true, false, true, TmuxAvailable, ModeNative, "mode: native (descended: pane unavailable: no interactive_command)", false},
		{"pane to headless across two rungs", false, false, false, false, "pane", false, true, true, TmuxAbsent, ModeHeadless, "mode: headless (descended: pane unavailable: no tmux; native unavailable)", false},
		{"native preferred and possible", false, false, false, false, "native", true, true, true, TmuxAvailable, ModeNative, "mode: native (preferred)", false},
		{"native to headless", false, false, false, false, "native", false, true, true, TmuxAvailable, ModeHeadless, "mode: headless (descended: native unavailable)", false},
		{"headless preferred", false, false, false, false, "headless", true, true, true, TmuxAvailable, ModeHeadless, "mode: headless (preferred)", false},
		{"no rung from pane", false, false, false, false, "pane", false, false, false, TmuxAbsent, "", "", true},
		{"no rung from native", false, false, false, false, "native", false, true, false, TmuxAvailable, "", "", true},
		{"headless never ascends", false, false, false, false, "headless", true, true, false, TmuxAvailable, "", "", true},

		{"explicit pane ignores capabilities", true, false, false, false, "headless", false, false, false, TmuxAbsent, ModePane, ReasonExplicit, false},
		{"explicit headless beats pane preference", false, true, false, false, "pane", true, true, true, TmuxAvailable, ModeHeadless, ReasonExplicit, false},
		{"timeout beats server", false, false, true, true, "pane", true, true, true, TmuxAvailable, ModeHeadless, ReasonExplicit, false},
		{"server implies pane", false, false, false, true, "headless", false, false, false, TmuxAbsent, ModePane, ReasonExplicit, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, reason, err := SelectMode(tt.paneFlag, tt.headlessFlag, tt.timeoutSet, tt.serverSet,
				tt.preference, tt.native, tt.session, tt.dispatch, tt.tmux)
			if tt.wantErr {
				if !errors.Is(err, ErrNoMode) {
					t.Fatalf("error = %v, want ErrNoMode", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("SelectMode: %v", err)
			}
			if mode != tt.wantMode || reason != tt.wantReason {
				t.Errorf("SelectMode(pane=%v,headless=%v,timeout=%v,server=%v,preference=%q) = (%q,%q), want (%q,%q)",
					tt.paneFlag, tt.headlessFlag, tt.timeoutSet, tt.serverSet, tt.preference,
					mode, reason, tt.wantMode, tt.wantReason)
			}
			// Any explicit signal must report ReasonExplicit, so the dispatched
			// output line carries no automatic suffix.
			explicit := tt.paneFlag || tt.headlessFlag || tt.timeoutSet || tt.serverSet
			if explicit && reason != ReasonExplicit {
				t.Errorf("explicit selection reported reason %q, want the empty ReasonExplicit", reason)
			}
			if !explicit && reason == ReasonExplicit {
				t.Error("auto selection must report its source, got ReasonExplicit")
			}
		})
	}
}

// TestSelectModeAutomaticMatrix exhausts every preference × capability × tmux
// combination. TestSelectMode above pins the exact reason vocabulary for the
// representative paths; this matrix guards the global first-possible-rung and
// never-ascend invariants.
func TestSelectModeAutomaticMatrix(t *testing.T) {
	preferences := []Mode{ModePane, ModeNative, ModeHeadless}
	tmuxStates := []TmuxAvailability{TmuxAbsent, TmuxAvailable, TmuxUnreachable}
	rank := map[Mode]int{ModePane: 0, ModeNative: 1, ModeHeadless: 2}

	for _, preference := range preferences {
		for bits := 0; bits < 8; bits++ {
			native := bits&1 != 0
			session := bits&2 != 0
			headless := bits&4 != 0
			for _, tmux := range tmuxStates {
				name := fmt.Sprintf("preference=%s/native=%t/session=%t/headless=%t/tmux=%s",
					preference, native, session, headless, tmux)
				t.Run(name, func(t *testing.T) {
					want := Mode("")
					if preference == ModePane && tmux == TmuxAvailable && session {
						want = ModePane
					} else if preference != ModeHeadless && native {
						want = ModeNative
					} else if headless {
						want = ModeHeadless
					}

					got, reason, err := SelectMode(false, false, false, false,
						string(preference), native, session, headless, tmux)
					if want == "" {
						if !errors.Is(err, ErrNoMode) {
							t.Fatalf("SelectMode = (%q, %q, %v), want ErrNoMode", got, reason, err)
						}
						return
					}
					if err != nil {
						t.Fatalf("SelectMode: %v", err)
					}
					if got != want {
						t.Fatalf("mode = %q, want first available rung %q", got, want)
					}
					if rank[got] < rank[preference] {
						t.Fatalf("mode ascended from %q to %q", preference, got)
					}
					if !strings.HasPrefix(string(reason), "mode: "+string(got)+" (") {
						t.Errorf("reason = %q, want selected rung %q", reason, got)
					}
					if got == preference && reason != preferredReason(got) {
						t.Errorf("direct selection reason = %q, want %q", reason, preferredReason(got))
					}
					if got != preference && !strings.Contains(string(reason), "(descended:") {
						t.Errorf("descent reason = %q, want descended marker", reason)
					}
				})
			}
		}
	}
}

// TestSelectPaneShape exhausts the pane-PLACEMENT decision: every combination of
// the three inputs. The properties worth pinning are that only a dispatcher which
// IS a tmux pane on the TARGET server gets the split shape, and that both
// window-shape rungs (--server named, $TMUX_PANE unset) reproduce the pre-split
// behavior — which is what makes the change additive rather than a behavior swap.
func TestSelectPaneShape(t *testing.T) {
	tests := []struct {
		name      string
		paneMode  bool
		serverSet bool
		tmuxPane  string
		want      PaneShape
	}{
		// The new shape: a pane-mode dispatch from a tmux pane on the same server.
		{"pane mode from a tmux pane ⇒ split", true, false, "%7", ShapeSplit},

		// --server may name ANOTHER socket, where the caller's pane id means nothing.
		{"--server wins over $TMUX_PANE ⇒ window", true, true, "%7", ShapeWindow},
		{"--server with no $TMUX_PANE ⇒ window", true, true, "", ShapeWindow},

		// A headless orchestrator passing explicit --pane has no pane to split.
		{"no $TMUX_PANE ⇒ window", true, false, "", ShapeWindow},

		// Not pane mode at all: no worker pane is opened, so the shape is vacuous
		// and must read as the pre-split default rather than an accidental split.
		{"headless mode ⇒ window (vacuous)", false, false, "%7", ShapeWindow},
		{"headless mode with --server ⇒ window (vacuous)", false, true, "%7", ShapeWindow},
		{"headless mode, no pane ⇒ window (vacuous)", false, false, "", ShapeWindow},
		{"headless mode, --server, no pane ⇒ window (vacuous)", false, true, "", ShapeWindow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SelectPaneShape(tt.paneMode, tt.serverSet, tt.tmuxPane); got != tt.want {
				t.Errorf("SelectPaneShape(pane=%v, serverSet=%v, TMUX_PANE=%q) = %q, want %q",
					tt.paneMode, tt.serverSet, tt.tmuxPane, got, tt.want)
			}
		})
	}
}

// TestLastRecordedPane pins the sibling probe's row grammar AND its intersection
// rule: only a pane the dispatch RECORDS claim is a worker, the LAST such pane wins
// (that is the bottom of the stacked column, so the next split lands under the
// newest worker), and every other pane in the window — the dispatcher's own, one the
// user split by hand — is never selected. Nothing here reads a pane TITLE, which is
// the whole point: a harness inside the worker rewrites the title within seconds.
func TestLastRecordedPane(t *testing.T) {
	recorded := map[string]bool{"%2": true, "%3": true}
	tests := []struct {
		name     string
		out      string
		recorded map[string]bool
		want     string
	}{
		{"no panes at all", "", recorded, ""},
		{"only the dispatcher's own pane", "%1\n", recorded, ""},
		{"one recorded worker", "%1\n%2\n", recorded, "%2"},
		{"the LAST recorded worker wins", "%1\n%2\n%3\n", recorded, "%3"},
		{"a hand-split pane after the workers is skipped", "%1\n%2\n%9\n", recorded, "%2"},
		{"a hand-split pane BETWEEN workers does not become the target",
			"%1\n%2\n%9\n%3\n", recorded, "%3"},
		{"no records at all ⇒ first-worker case", "%1\n%2\n%3\n", map[string]bool{}, ""},
		{"a recorded pane absent from THIS window never matches",
			"%1\n%7\n", recorded, ""},
		{"blank and padded rows are ignored", "%1\n\n  %2  \n\n", recorded, "%2"},
		{"a pane id merely CONTAINING a recorded id does not match", "%20\n", recorded, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lastRecordedPane(tt.out, tt.recorded); got != tt.want {
				t.Errorf("lastRecordedPane(%q, %v) = %q, want %q", tt.out, tt.recorded, got, tt.want)
			}
		})
	}
}

// TestRecordedPanes covers the record-enumeration half: every PANE dispatch's pane
// id across ALL of the checkout's record dirs is collected, headless records
// contribute nothing, result files are not mistaken for records, and an ABSENT tree
// is the benign first-dispatch case (empty set, NIL error — nothing failed, there is
// simply nothing recorded yet).
func TestRecordedPanes(t *testing.T) {
	repoRoot := t.TempDir()

	// No .fab-dispatch/ at all.
	got, err := recordedPanes(repoRoot, "")
	if err != nil {
		t.Errorf("an absent dispatch tree is not a failure, got error %v", err)
	}
	if len(got) != 0 {
		t.Errorf("an absent dispatch tree must yield an empty set, got %v", got)
	}

	// Two changes' dirs: one pane + one headless record each, plus a result file
	// that also ends in .yaml and must NOT be read as a record.
	mustSave := func(id, stage string, rec *Dispatch) {
		t.Helper()
		if err := Save(DirFor(repoRoot, id), stage, rec); err != nil {
			t.Fatalf("Save %s/%s: %v", id, stage, err)
		}
	}
	mustSave("abcd", "apply", &Dispatch{Pane: "%2", Window: WindowName("abcd", "apply")})
	mustSave("abcd", "review", &Dispatch{PID: 1234, PGID: 1234})
	mustSave("efgh", "hydrate", &Dispatch{Pane: "%5", Window: WindowName("efgh", "hydrate")})
	if err := os.WriteFile(ResultPath(DirFor(repoRoot, "abcd"), "apply"),
		[]byte("stage: apply\nstatus: success\npane: %99\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A stray file in the tree must not derail the walk.
	if err := os.WriteFile(filepath.Join(repoRoot, DirName, "not-a-dir"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err = recordedPanes(repoRoot, "")
	if err != nil {
		t.Fatalf("a well-formed tree must read cleanly, got %v", err)
	}
	want := map[string]bool{"%2": true, "%5": true}
	if len(got) != len(want) {
		t.Fatalf("recordedPanes = %v, want %v", got, want)
	}
	for id := range want {
		if !got[id] {
			t.Errorf("recordedPanes = %v, missing the recorded pane %q", got, id)
		}
	}
	if got["%99"] {
		t.Error("a {stage}-result.yaml must not be read as a dispatch record")
	}
}

// TestRecordedPanesServerFilter pins the per-SOCKET scoping. A tmux pane id is
// server-global, not GLOBALLY global: the `%3` recorded by a `--server work`
// dispatch names a different pane from the `%3` in a default-socket window, so an
// unfiltered set would let a foreign record false-match a live local pane and stack
// a worker onto something unrelated. Equality against the record's Server is the
// exact test in both directions, with the default socket recorded as "".
func TestRecordedPanesServerFilter(t *testing.T) {
	repoRoot := t.TempDir()
	mustSave := func(id, stage string, rec *Dispatch) {
		t.Helper()
		if err := Save(DirFor(repoRoot, id), stage, rec); err != nil {
			t.Fatalf("Save %s/%s: %v", id, stage, err)
		}
	}
	mustSave("abcd", "apply", &Dispatch{Pane: "%2"})                    // default socket
	mustSave("abcd", "review", &Dispatch{Pane: "%3", Server: "work"})   // another socket
	mustSave("efgh", "hydrate", &Dispatch{Pane: "%4", Server: "other"}) // a third

	tests := []struct {
		name   string
		server string
		want   map[string]bool
	}{
		{"the default socket sees only Server:\"\" records", "", map[string]bool{"%2": true}},
		{"a named socket sees only its own records", "work", map[string]bool{"%3": true}},
		{"a socket with no records sees nothing", "nosuch", map[string]bool{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := recordedPanes(repoRoot, tt.server)
			if err != nil {
				t.Fatalf("recordedPanes(%q) errored: %v", tt.server, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("recordedPanes(%q) = %v, want %v", tt.server, got, tt.want)
			}
		})
	}
}

// TestRecordedPanesReadFailure: a REAL read failure (a corrupt record, an unreadable
// dir) is reported rather than swallowed, so launchPane's warn can tell the user the
// placement probe was degraded. Two properties are load-bearing together:
//
//   - the error is non-nil (the silent-degrade defect this closes), and
//   - the PARTIAL set is still returned, since a record that could not be read can
//     only fail to find a sibling, never invent one — discarding the readable records
//     would carve a redundant column for no gain.
func TestRecordedPanesReadFailure(t *testing.T) {
	t.Run("corrupt record", func(t *testing.T) {
		repoRoot := t.TempDir()
		if err := Save(DirFor(repoRoot, "abcd"), "apply", &Dispatch{Pane: "%2"}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(YAMLPath(DirFor(repoRoot, "abcd"), "review"),
			[]byte("pane: [unterminated\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		got, err := recordedPanes(repoRoot, "")
		if err == nil {
			t.Fatal("a corrupt record must surface an error, not degrade silently")
		}
		if !strings.Contains(err.Error(), "review.yaml") {
			t.Errorf("error %q must name the record that failed", err)
		}
		if !got["%2"] {
			t.Errorf("the readable record's pane must survive the partial failure, got %v", got)
		}
	})

	t.Run("unreadable dispatch dir", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}
		repoRoot := t.TempDir()
		if err := Save(DirFor(repoRoot, "abcd"), "apply", &Dispatch{Pane: "%2"}); err != nil {
			t.Fatal(err)
		}
		locked := DirFor(repoRoot, "efgh")
		if err := os.MkdirAll(locked, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(locked, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

		got, err := recordedPanes(repoRoot, "")
		if err == nil {
			t.Fatal("an unreadable dispatch dir must surface an error, not degrade silently")
		}
		if !got["%2"] {
			t.Errorf("the readable record's pane must survive the partial failure, got %v", got)
		}
	})

	t.Run("unreadable tree root", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}
		repoRoot := t.TempDir()
		root := filepath.Join(repoRoot, DirName)
		if err := os.MkdirAll(root, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(root, 0o755) })

		if _, err := recordedPanes(repoRoot, ""); err == nil {
			t.Fatal("an unreadable tree root must surface an error — only an ABSENT tree is benign")
		}
	})
}

// TestSiblingDispatchPaneRecordError proves the record-read error reaches the CALLER:
// SiblingDispatchPane joins it into its own return so SplitTarget hands it to
// launchPane's warn. The placement answer is unaffected — the readable record still
// wins the intersection — which is exactly the degrade-don't-fail contract.
func TestSiblingDispatchPaneRecordError(t *testing.T) {
	repoRoot := t.TempDir()
	if err := Save(DirFor(repoRoot, "abcd"), "apply", &Dispatch{Pane: "%2"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(YAMLPath(DirFor(repoRoot, "abcd"), "review"),
		[]byte("pane: [unterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stubTmux(t, `case "$1" in list-panes) printf '%%1\n%%2\n' ;; esac
exit 0`)

	sibling, err := SiblingDispatchPane("", "%1", repoRoot)
	if err == nil {
		t.Fatal("a record-read failure must reach the caller's warn path, not vanish")
	}
	if sibling != "%2" {
		t.Errorf("sibling = %q, want the readable record's pane %q (placement degrades, it does not fail)", sibling, "%2")
	}
}

// TestSplitPlacement pins the column invariant as a pure decision: the first worker
// (no sibling) CARVES a sized column off the dispatcher, every later worker STACKS
// under the newest sibling unsized. A degraded probe reaches the same first-worker
// branch, so a fallback column is sized too — a fallback that halved the dispatcher
// would reintroduce the squeeze the width exists to prevent.
func TestSplitPlacement(t *testing.T) {
	tests := []struct {
		name        string
		sibling     string
		dispatcher  string
		columnWidth int
		want        SplitPlacement
	}{
		{"no sibling ⇒ carve a sized column off the dispatcher", "", "%1", 35,
			SplitPlacement{Target: "%1", Direction: splitRight, SizePercent: 35}},
		{"probe failed (empty sibling) ⇒ the same sized carve", "", "%1", 20,
			SplitPlacement{Target: "%1", Direction: splitRight, SizePercent: 20}},
		{"a sibling ⇒ stack under it, unsized", "%2", "%1", 35,
			SplitPlacement{Target: "%2", Direction: splitBelow, SizePercent: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitPlacement(tt.sibling, tt.dispatcher, tt.columnWidth); got != tt.want {
				t.Errorf("splitPlacement(%q, %q, %d) = %+v, want %+v",
					tt.sibling, tt.dispatcher, tt.columnWidth, got, tt.want)
			}
		})
	}
}

// TestSplitArgs pins the sized-split argv composition: `-l <n>%` appears only when
// the placement carries a size (i.e. only for a column-carving split), always as a
// PERCENTAGE so the column scales with the window, and the pane-id format request
// is present in both shapes so no follow-up lookup can race a fast-exiting worker.
func TestSplitArgs(t *testing.T) {
	carve := splitArgs(SplitPlacement{Target: "%1", Direction: splitRight, SizePercent: 35},
		"/repo", "claude 'go'")
	wantCarve := []string{"split-window", "-h", "-t", "%1", "-l", "35%",
		"-P", "-F", "#{pane_id}", "-c", "/repo", "claude 'go'"}
	if !reflect.DeepEqual(carve, wantCarve) {
		t.Errorf("carving argv = %q, want %q", carve, wantCarve)
	}

	stack := splitArgs(SplitPlacement{Target: "%2", Direction: splitBelow},
		"/repo", "claude 'go'")
	wantStack := []string{"split-window", "-v", "-t", "%2",
		"-P", "-F", "#{pane_id}", "-c", "/repo", "claude 'go'"}
	if !reflect.DeepEqual(stack, wantStack) {
		t.Errorf("stacking argv = %q, want %q", stack, wantStack)
	}

	// A zero/absent size is the UNSIZED signal in either direction — tmux's own even
	// split — never a literal `-l 0%`, which tmux would reject.
	unsizedCarve := splitArgs(SplitPlacement{Target: "%1", Direction: splitRight}, "/repo", "cmd")
	for _, arg := range unsizedCarve {
		if arg == sizeFlag {
			t.Errorf("a zero-size placement must emit no %s argument, got %q", sizeFlag, unsizedCarve)
		}
	}
}

// TestSplitPlacementDescribe pins the placement's own vocabulary: Describe is the
// only cross-package reader of Direction, which is what keeps the bare tmux `-h`/`-v`
// flag inside this package. The two phrasings are the ones the degraded-probe warning
// is documented with, so they are asserted rather than left to the cobra layer.
func TestSplitPlacementDescribe(t *testing.T) {
	carve := SplitPlacement{Target: "%1", Direction: splitRight, SizePercent: 35}.Describe()
	if carve != "carving a new worker column off pane %1" {
		t.Errorf("carve description = %q, want the column-carving wording", carve)
	}
	stack := SplitPlacement{Target: "%2", Direction: splitBelow}.Describe()
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

	place := SplitPlacement{Target: "%1", Direction: splitRight, SizePercent: 35}
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
	for _, want := range []string{sizeFlag, "35%", "retrying unsized"} {
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

	_, _, err := OpenSplitPane("", SplitPlacement{Target: "%2", Direction: splitBelow}, "fab-abcd-apply", "/repo", "cmd")
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
	if splitRight != "-h" {
		t.Errorf("splitRight = %q, want tmux's horizontal split flag -h", splitRight)
	}
	if splitBelow != "-v" {
		t.Errorf("splitBelow = %q, want tmux's vertical split flag -v", splitBelow)
	}
	if sizeFlag != "-l" {
		t.Errorf("sizeFlag = %q, want tmux's split size flag -l", sizeFlag)
	}
}

func TestModeAccessors(t *testing.T) {
	headless := &Dispatch{PID: 10, PGID: 10, SpawnCmd: "codex exec", StartedAt: "t"}
	if headless.IsPane() {
		t.Error("a record with no pane id must not read as pane mode")
	}
	if got := headless.Mode(); got != ModeHeadless {
		t.Errorf("Mode() = %q, want %q", got, ModeHeadless)
	}

	panedisp := &Dispatch{Pane: "%17", Window: "fab-abcd-apply", SpawnCmd: "claude", StartedAt: "t"}
	if !panedisp.IsPane() {
		t.Error("a record carrying a pane id must read as pane mode")
	}
	if got := panedisp.Mode(); got != ModePane {
		t.Errorf("Mode() = %q, want %q", got, ModePane)
	}
}

func TestWindowNameCarriesNoOperatorMarker(t *testing.T) {
	got := WindowName("abcd", "apply")
	if got != "fab-abcd-apply" {
		t.Errorf("WindowName = %q, want fab-abcd-apply", got)
	}
	// The operator's `»` enrollment prefix and `›` done marker assert operator
	// ownership of a window's lifecycle, which a pipeline dispatch does not have.
	for _, marker := range []string{"\u00bb", "\u203a"} {
		if contains(got, marker) {
			t.Errorf("WindowName = %q, must not carry the operator marker %q", got, marker)
		}
	}
}

func TestPointerPromptNamesThePromptFile(t *testing.T) {
	got := PointerPrompt(".fab-dispatch/abcd/apply-prompt.md")
	if !contains(got, ".fab-dispatch/abcd/apply-prompt.md") {
		t.Errorf("PointerPrompt = %q, want it to name the prompt path", got)
	}
	// One line: the pointer is embedded as a single quoted spawn argument, and a
	// newline in it would break the one-prompt/one-command spawn contract.
	if contains(got, "\n") {
		t.Errorf("PointerPrompt = %q, want a single line", got)
	}
}

// TestWindowCommandShellQuotesThePointer pins that the pointer rides as ONE
// safely-quoted argument. A repo path containing a single quote — the realistic
// case, e.g. a checkout under `/home/me/sahil's-repo/` — must not terminate the
// quoted argument early, which would both break the tmux new-window command and
// hand the remainder of the path to the window's shell.
func TestWindowCommandShellQuotesThePointer(t *testing.T) {
	// The resolved session command is inserted verbatim (its shell expansions are
	// deliberate and must expand inside the new window).
	const resolved = `claude -n "$(basename "$(pwd)")" --model m`
	got := WindowCommand(resolved, PointerPrompt("sahil's-repo/.fab-dispatch/abcd/apply-prompt.md"))

	if !hasPrefix(got, resolved+" ") {
		t.Errorf("WindowCommand = %q, want the resolved command inserted verbatim as a prefix", got)
	}
	// The quote is escaped via the '\'' idiom, so the argument stays a single
	// shell word — the naive "'%s'" form would yield a bare `'` here.
	if contains(got, `'\''`) == false {
		t.Errorf("WindowCommand = %q, want the embedded single quote escaped", got)
	}
	// Prove it by round-tripping through a real shell: the pointer must arrive as
	// exactly one argument with the quote intact.
	arg := shellRoundTrip(t, got, resolved)
	if arg != PointerPrompt("sahil's-repo/.fab-dispatch/abcd/apply-prompt.md") {
		t.Errorf("shell parsed the pointer as %q, want it byte-identical to the pointer", arg)
	}
}

// shellRoundTrip runs `sh -c 'printf ... <quoted-pointer>'` — the windowCmd with
// its verbatim command prefix swapped for a printf that echoes its single
// argument — to prove the quoted tail parses as exactly one shell word.
func shellRoundTrip(t *testing.T, windowCmd, resolvedPrefix string) string {
	t.Helper()
	quotedTail := windowCmd[len(resolvedPrefix)+1:]
	out, err := exec.Command("sh", "-c", `printf '%s' `+quotedTail).Output()
	if err != nil {
		t.Fatalf("shell could not parse the quoted pointer %q: %v", quotedTail, err)
	}
	return string(out)
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// TestSaveOmitsHeadlessFieldsForPaneRecord pins the on-disk shape of both modes:
// a headless record carries pid/pgid and NO pane keys (byte-identical to the
// pre-pane-mode format), and a pane record carries the pane identity and no
// meaningless `pid: 0`.
func TestSaveOmitsHeadlessFieldsForPaneRecord(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".fab-dispatch", "abcd")

	if err := Save(base, "apply", &Dispatch{PID: 7, PGID: 7, SpawnCmd: "codex exec", StartedAt: "t"}); err != nil {
		t.Fatalf("Save headless: %v", err)
	}
	headlessYAML, _ := os.ReadFile(YAMLPath(base, "apply"))
	for _, key := range []string{"pid:", "pgid:"} {
		if !contains(string(headlessYAML), key) {
			t.Errorf("headless record missing %q:\n%s", key, headlessYAML)
		}
	}
	for _, key := range []string{"pane:", "window:", "server:"} {
		if contains(string(headlessYAML), key) {
			t.Errorf("headless record must omit %q:\n%s", key, headlessYAML)
		}
	}

	if err := Save(base, "review", &Dispatch{
		Pane: "%17", Window: "fab-abcd-review", Server: "work", SpawnCmd: "claude", StartedAt: "t",
	}); err != nil {
		t.Fatalf("Save pane: %v", err)
	}
	paneYAML, _ := os.ReadFile(YAMLPath(base, "review"))
	for _, key := range []string{"pane:", "window:", "server:"} {
		if !contains(string(paneYAML), key) {
			t.Errorf("pane record missing %q:\n%s", key, paneYAML)
		}
	}
	for _, key := range []string{"pid:", "pgid:"} {
		if contains(string(paneYAML), key) {
			t.Errorf("pane record must omit %q:\n%s", key, paneYAML)
		}
	}

	// Round-trip: the derived mode survives a save/load cycle.
	got, err := Load(base, "review")
	if err != nil {
		t.Fatalf("Load pane: %v", err)
	}
	if got.Mode() != ModePane || got.Pane != "%17" || got.Window != "fab-abcd-review" || got.Server != "work" {
		t.Errorf("pane round-trip = %+v", *got)
	}
}

func TestWrapperArgv(t *testing.T) {
	tests := []struct {
		name       string
		cmd        string
		timeout    int
		wantScript string
	}{
		{
			name:       "no timeout",
			cmd:        "claude --dangerously-skip-permissions",
			timeout:    0,
			wantScript: "claude --dangerously-skip-permissions < 'p.md' > 'l.log' 2>&1; echo $? > 'e.exit'",
		},
		{
			name:       "with timeout wraps in POSIX timeout",
			cmd:        "codex exec",
			timeout:    600,
			wantScript: "timeout 600 codex exec < 'p.md' > 'l.log' 2>&1; echo $? > 'e.exit'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argv := WrapperArgv(tt.cmd, "p.md", "l.log", "e.exit", tt.timeout)
			// The detach is performed by SysProcAttr.Setsid in Launch, not by a
			// `setsid` binary prefix (which would double-fork and untrack the
			// worker pid) — so the argv is a plain `sh -c <script>`.
			if len(argv) != 3 {
				t.Fatalf("argv = %v, want 3 elements (sh -c <script>)", argv)
			}
			if argv[0] != "sh" || argv[1] != "-c" {
				t.Errorf("argv prefix = %v, want [sh -c ...]", argv[:2])
			}
			if argv[2] != tt.wantScript {
				t.Errorf("script =\n  %q\nwant\n  %q", argv[2], tt.wantScript)
			}
		})
	}
}

func TestWrapperArgvQuotesPathsWithSpaces(t *testing.T) {
	argv := WrapperArgv("cmd", "/a b/p.md", "/a b/l.log", "/a b/e.exit", 0)
	want := "cmd < '/a b/p.md' > '/a b/l.log' 2>&1; echo $? > '/a b/e.exit'"
	if argv[2] != want {
		t.Errorf("script = %q, want %q", argv[2], want)
	}
}

func TestTail(t *testing.T) {
	tests := []struct {
		name string
		data string
		n    int
		want string
	}{
		{"n<=0 returns all", "a\nb\nc\n", 0, "a\nb\nc\n"},
		{"empty", "", 5, ""},
		{"fewer lines than n", "a\nb\n", 5, "a\nb\n"},
		{"last 1 with trailing newline", "a\nb\nc\n", 1, "c\n"},
		{"last 2 with trailing newline", "a\nb\nc\n", 2, "b\nc\n"},
		{"last 1 without trailing newline", "a\nb\nc", 1, "c"},
		{"exact match keeps all", "a\nb\nc\n", 3, "a\nb\nc\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(Tail([]byte(tt.data), tt.n))
			if got != tt.want {
				t.Errorf("Tail(%q, %d) = %q, want %q", tt.data, tt.n, got, tt.want)
			}
		})
	}
}

func TestPathHelpers(t *testing.T) {
	dir := DirFor("/repo", "abcd")
	if dir != filepath.Join("/repo", ".fab-dispatch", "abcd") {
		t.Errorf("DirFor = %q", dir)
	}
	if got := PromptPath(dir, "apply"); got != filepath.Join(dir, "apply-prompt.md") {
		t.Errorf("PromptPath = %q", got)
	}
	if got := YAMLPath(dir, "apply"); got != filepath.Join(dir, "apply.yaml") {
		t.Errorf("YAMLPath = %q", got)
	}
	if got := LogPath(dir, "apply"); got != filepath.Join(dir, "apply.log") {
		t.Errorf("LogPath = %q", got)
	}
	if got := ExitPath(dir, "apply"); got != filepath.Join(dir, "apply.exit") {
		t.Errorf("ExitPath = %q", got)
	}
	if got := ResultPath(dir, "apply"); got != filepath.Join(dir, "apply-result.yaml") {
		t.Errorf("ResultPath = %q", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".fab-dispatch", "abcd")
	rec := &Dispatch{PID: 1234, PGID: 1234, SpawnCmd: "codex exec", StartedAt: "2026-07-02T00:00:00Z", Timeout: 600}

	if err := Save(dir, "apply", rec); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if *got != *rec {
		t.Errorf("round-trip = %+v, want %+v", *got, *rec)
	}
}

func TestSaveOmitsZeroTimeout(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".fab-dispatch", "abcd")
	if err := Save(dir, "apply", &Dispatch{PID: 1, PGID: 1, SpawnCmd: "x", StartedAt: "t"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, _ := os.ReadFile(YAMLPath(dir, "apply"))
	if want := "timeout"; contains(string(data), want) {
		t.Errorf("zero timeout should be omitted, got:\n%s", data)
	}
}

func TestLoadMissingIsNotExist(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".fab-dispatch", "abcd")
	_, err := Load(dir, "apply")
	if !os.IsNotExist(err) {
		t.Errorf("Load of missing = %v, want IsNotExist", err)
	}
}

func TestReadExit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".fab-dispatch", "abcd")
	os.MkdirAll(dir, 0o755)

	// Absent → not present.
	present, code, err := ReadExit(dir, "apply")
	if err != nil || present || code != 0 {
		t.Errorf("absent exit: present=%v code=%d err=%v", present, code, err)
	}

	// Present, code 0.
	os.WriteFile(ExitPath(dir, "apply"), []byte("0\n"), 0o644)
	present, code, err = ReadExit(dir, "apply")
	if err != nil || !present || code != 0 {
		t.Errorf("exit 0: present=%v code=%d err=%v", present, code, err)
	}

	// Present, non-zero.
	os.WriteFile(ExitPath(dir, "apply"), []byte("124\n"), 0o644)
	present, code, err = ReadExit(dir, "apply")
	if err != nil || !present || code != 124 {
		t.Errorf("exit 124: present=%v code=%d err=%v", present, code, err)
	}

	// Empty file → present, code 0 (finished-but-garbage → conservative).
	os.WriteFile(ExitPath(dir, "apply"), []byte(""), 0o644)
	present, code, err = ReadExit(dir, "apply")
	if err != nil || !present || code != 0 {
		t.Errorf("empty exit: present=%v code=%d err=%v", present, code, err)
	}
}

func TestResultPresent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".fab-dispatch", "abcd")
	os.MkdirAll(dir, 0o755)
	if ResultPresent(dir, "apply") {
		t.Error("result should be absent")
	}
	os.WriteFile(ResultPath(dir, "apply"), []byte("ok: true\n"), 0o644)
	if !ResultPresent(dir, "apply") {
		t.Error("result should be present")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
