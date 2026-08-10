package dispatch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
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
		want        pane.SplitPlacement
	}{
		{"no sibling ⇒ carve a sized column off the dispatcher", "", "%1", 35,
			pane.SplitPlacement{Target: "%1", Direction: pane.SplitRight, SizePercent: 35}},
		{"probe failed (empty sibling) ⇒ the same sized carve", "", "%1", 20,
			pane.SplitPlacement{Target: "%1", Direction: pane.SplitRight, SizePercent: 20}},
		{"a sibling ⇒ stack under it, unsized", "%2", "%1", 35,
			pane.SplitPlacement{Target: "%2", Direction: pane.SplitBelow, SizePercent: 0}},
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

// TestDeliveredMarkerIsPaneOnlyAndOmittedWhenFalse pins the record's additive
// shape: `delivered` appears only once it is true, so an `open`ed pane reads as
// undelivered by ABSENCE and every headless record's bytes are unchanged.
//
// The assertions read the MARSHALLED YAML, not the struct field: `omitempty` is
// the whole contract here, and a round-tripped struct reads false either way — it
// cannot tell an omitted key from a `delivered: false` one written into every
// headless record on disk.
func TestDeliveredMarkerIsPaneOnlyAndOmittedWhenFalse(t *testing.T) {
	dir := t.TempDir()

	readRecord := func(stage string) string {
		t.Helper()
		data, err := os.ReadFile(YAMLPath(dir, stage))
		if err != nil {
			t.Fatalf("read %s record: %v", stage, err)
		}
		return string(data)
	}

	if err := Save(dir, "apply", &Dispatch{Pane: "%17", Window: "fab-abcd-apply", SpawnCmd: "claude", StartedAt: "t"}); err != nil {
		t.Fatalf("Save opened: %v", err)
	}
	if got := readRecord("apply"); strings.Contains(got, "delivered") {
		t.Errorf("an `open`ed pane record carries a delivery key:\n%s", got)
	}
	opened, err := Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load opened: %v", err)
	}
	if opened.Delivered {
		t.Error("a pane record written by `open` must read as not delivered")
	}

	opened.Delivered = true
	if err := Save(dir, "apply", opened); err != nil {
		t.Fatalf("Save delivered: %v", err)
	}
	if got := readRecord("apply"); !strings.Contains(got, "delivered: true") {
		t.Errorf("a delivered pane record must carry `delivered: true`:\n%s", got)
	}
	delivered, err := Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load delivered: %v", err)
	}
	if !delivered.Delivered {
		t.Error("the delivery marker must round-trip")
	}

	if err := Save(dir, "review", &Dispatch{PID: 7, PGID: 7, SpawnCmd: "codex exec", StartedAt: "t"}); err != nil {
		t.Fatalf("Save headless: %v", err)
	}
	if got := readRecord("review"); strings.Contains(got, "delivered") {
		t.Errorf("a headless record's bytes changed — it now carries a delivery key:\n%s", got)
	}
	headless, err := Load(dir, "review")
	if err != nil {
		t.Fatalf("Load headless: %v", err)
	}
	if headless.Delivered {
		t.Error("a headless record must never carry a delivery marker")
	}
}

// TestDeliveredChangesNoState pins that the marker is bookkeeping, not a state:
// the pane derivation is a function of the result file and pane liveness alone,
// exactly as before.
func TestDeliveredChangesNoState(t *testing.T) {
	for _, delivered := range []bool{false, true} {
		rec := &Dispatch{Pane: "%17", Delivered: delivered}
		if !rec.IsPane() || rec.Mode() != ModePane {
			t.Errorf("delivered=%v changed the derived mode", delivered)
		}
		if got := DerivePaneState(true, false); got != StateDone {
			t.Errorf("delivered=%v: result-present derivation = %q, want %q", delivered, got, StateDone)
		}
	}
}
