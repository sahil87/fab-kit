package pane

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWithServer(t *testing.T) {
	t.Run("empty server returns args verbatim", func(t *testing.T) {
		got := WithServer("", "list-panes", "-a")
		want := []string{"list-panes", "-a"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("WithServer(\"\", ...) = %v, want %v", got, want)
		}
	})

	t.Run("non-empty server prepends -L", func(t *testing.T) {
		got := WithServer("runKit", "list-panes", "-a")
		want := []string{"-L", "runKit", "list-panes", "-a"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("WithServer(\"runKit\", ...) = %v, want %v", got, want)
		}
	})

	t.Run("no args with non-empty server returns just -L and server", func(t *testing.T) {
		got := WithServer("runKit")
		want := []string{"-L", "runKit"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("WithServer(\"runKit\") = %v, want %v", got, want)
		}
	})

	t.Run("no args with empty server returns empty slice", func(t *testing.T) {
		got := WithServer("")
		if len(got) != 0 {
			t.Errorf("WithServer(\"\") = %v, want empty slice", got)
		}
	})

	t.Run("input args slice is not mutated across calls", func(t *testing.T) {
		original := []string{"list-panes", "-a", "-F", "#{pane_id}"}
		snapshot := make([]string, len(original))
		copy(snapshot, original)

		_ = WithServer("runKit", original...)
		_ = WithServer("runKit", original...)

		if !reflect.DeepEqual(original, snapshot) {
			t.Errorf("input slice mutated: got %v, want %v", original, snapshot)
		}
	})

	t.Run("special characters in server name passed verbatim", func(t *testing.T) {
		got := WithServer("my-socket", "list-panes")
		want := []string{"-L", "my-socket", "list-panes"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("WithServer(\"my-socket\", ...) = %v, want %v", got, want)
		}
		got2 := WithServer("socket_1", "list-panes")
		want2 := []string{"-L", "socket_1", "list-panes"}
		if !reflect.DeepEqual(got2, want2) {
			t.Errorf("WithServer(\"socket_1\", ...) = %v, want %v", got2, want2)
		}
	})
}

func TestRunCmd(t *testing.T) {
	t.Run("captures stdout and stderr separately", func(t *testing.T) {
		out, stderr, err := RunCmd("sh", "-c", "printf 'out\\n'; printf 'diag\\n' 1>&2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != "out\n" {
			t.Errorf("stdout = %q, want %q (raw, untrimmed)", out, "out\n")
		}
		if string(stderr) != "diag\n" {
			t.Errorf("stderr = %q, want %q", string(stderr), "diag\n")
		}
	})

	t.Run("returns exec error with captured stderr on failure", func(t *testing.T) {
		out, stderr, err := RunCmd("sh", "-c", "printf 'partial' ; printf 'boom\\n' 1>&2; exit 3")
		if err == nil {
			t.Fatal("expected error for exit 3")
		}
		if out != "partial" {
			t.Errorf("stdout = %q, want %q", out, "partial")
		}
		if string(stderr) != "boom\n" {
			t.Errorf("stderr = %q, want %q", string(stderr), "boom\n")
		}
	})
}

func TestStderrError(t *testing.T) {
	t.Run("appends trimmed stderr to the error", func(t *testing.T) {
		base := os.ErrNotExist
		got := StderrError(base, []byte("  can't find pane: %99\n"))
		if got.Error() != base.Error()+": can't find pane: %99" {
			t.Errorf("StderrError = %q, want %q", got.Error(), base.Error()+": can't find pane: %99")
		}
		// Wrapping preserved for errors.Is.
		if !errors.Is(got, base) {
			t.Error("StderrError must wrap the original error (errors.Is failed)")
		}
	})

	t.Run("empty stderr returns the error unchanged", func(t *testing.T) {
		base := os.ErrPermission
		if got := StderrError(base, nil); got != base {
			t.Errorf("StderrError with nil stderr = %v, want the original error", got)
		}
		if got := StderrError(base, []byte("  \n")); got != base {
			t.Errorf("StderrError with whitespace stderr = %v, want the original error", got)
		}
	})
}

func TestIsPaneMissing(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{"can't find pane", "can't find pane: %99\n", true},
		{"no such pane", "no such pane: %99", true},
		{"pane + not found", "pane %99 not found", true},
		{"case insensitive", "Can't Find Pane: %99", true},
		{"dead server", "error connecting to /tmp/tmux-1001/x (No such file or directory)", false},
		{"no server running", "no server running on /tmp/tmux-1001/default", false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsPaneMissing([]byte(tc.stderr)); got != tc.want {
				t.Errorf("IsPaneMissing(%q) = %t, want %t", tc.stderr, got, tc.want)
			}
		})
	}
}

// TestCurrentCommandArgs pins the argv the foreground-command read builds —
// server-first with the -L prefix, and the format literal carried verbatim.
func TestCurrentCommandArgs(t *testing.T) {
	t.Run("no server", func(t *testing.T) {
		got := CurrentCommandArgs("", "%17")
		want := []string{"display-message", "-p", "-t", "%17", "#{pane_current_command}"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("CurrentCommandArgs = %v, want %v", got, want)
		}
	})

	t.Run("server prefixed", func(t *testing.T) {
		got := CurrentCommandArgs("runKit", "%17")
		want := []string{"-L", "runKit", "display-message", "-p", "-t", "%17", "#{pane_current_command}"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("CurrentCommandArgs = %v, want %v", got, want)
		}
	})
}

// TestIsShellCommand pins the shell-name predicate both pane-foreground
// consumers share: the same nine basenames, a case-sensitive basename match
// (a full path like /usr/bin/fish still matches), and an empty command never
// matching.
func TestIsShellCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"sh", true},
		{"bash", true},
		{"zsh", true},
		{"fish", true},
		{"dash", true},
		{"ksh", true},
		{"tcsh", true},
		{"csh", true},
		{"nu", true},
		{"/usr/bin/fish", true}, // basename match — a full path still resolves
		{"zshrc-lint", false},   // a shell-NAMED binary is not a shell
		{"Bash", false},         // case-sensitive
		{"kimi", false},
		{"claude", false},
		{"node", false},
		{"cat", false},   // the integration fixture's ready stand-in
		{"sleep", false}, // the parked/booting fixtures' foreground
		{"", false},      // legacy enumeration line — never a shell
	}
	for _, tc := range tests {
		t.Run(tc.cmd, func(t *testing.T) {
			if got := IsShellCommand(tc.cmd); got != tc.want {
				t.Errorf("IsShellCommand(%q) = %t, want %t", tc.cmd, got, tc.want)
			}
		})
	}
}

// TestValidatePaneResult exercises the pure decision half of the targeted
// display-message probe — every branch verified against real tmux behavior:
// tmux ≥3.6 exits 0 with EMPTY output for a missing pane (comparison branch);
// older tmux errors with "can't find pane" stderr (mapping branch); a dead
// server fails with a connection diagnostic; a window-name target resolves
// to a real pane ID that differs from the argument (ID-exactness).
func TestValidatePaneResult(t *testing.T) {
	mkErr := errors.New("exit status 1")

	t.Run("exact match passes", func(t *testing.T) {
		if err := validatePaneResult("%5", "%5\n", nil, nil); err != nil {
			t.Errorf("expected nil for exact match, got %v", err)
		}
	})

	t.Run("missing pane on tmux>=3.6: exit 0, empty output", func(t *testing.T) {
		err := validatePaneResult("%99", "\n", nil, nil)
		if err == nil || err.Error() != "pane %99 not found" {
			t.Errorf("expected 'pane %%99 not found', got %v", err)
		}
	})

	t.Run("missing pane on older tmux: can't-find-pane stderr", func(t *testing.T) {
		err := validatePaneResult("%99", "", []byte("can't find pane: %99\n"), mkErr)
		if err == nil || err.Error() != "pane %99 not found" {
			t.Errorf("expected 'pane %%99 not found', got %v", err)
		}
	})

	t.Run("window-name target rejected (ID-exactness)", func(t *testing.T) {
		// `-t mywindow` resolves to that window's active pane — the probe
		// output is a real pane ID that differs from the argument.
		err := validatePaneResult("mywindow", "%0\n", nil, nil)
		if err == nil || err.Error() != "pane mywindow not found" {
			t.Errorf("expected 'pane mywindow not found', got %v", err)
		}
	})

	t.Run("dead server surfaces the tmux diagnostic", func(t *testing.T) {
		stderr := []byte("error connecting to /tmp/tmux-1001/x (No such file or directory)\n")
		err := validatePaneResult("%5", "", stderr, mkErr)
		if err == nil {
			t.Fatal("expected error for dead server")
		}
		if !strings.Contains(err.Error(), "tmux display-message") || !strings.Contains(err.Error(), "error connecting") {
			t.Errorf("dead-server error should name display-message and carry the diagnostic, got %q", err.Error())
		}
		if !errors.Is(err, mkErr) {
			t.Error("dead-server error must wrap the exec error")
		}
	})
}

func TestTailLines(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"trailing blank padding stripped", "a\nb\nc\n\n\n\n", 5, "a\nb\nc\n"},
		{"tails to last n", "a\nb\nc\nd\ne\n", 2, "d\ne\n"},
		{"padding stripped then tailed", "a\nb\nc\nd\n\n\n\n\n", 2, "c\nd\n"},
		{"interior blank lines preserved", "a\n\nb\n\n\n", 5, "a\n\nb\n"},
		{"fewer lines than n returns all", "a\nb\n", 50, "a\nb\n"},
		{"exactly n lines", "a\nb\nc\n", 3, "a\nb\nc\n"},
		{"whitespace-only trailing rows are padding", "a\nb\n   \n\t\n", 5, "a\nb\n"},
		{"bytes within window untouched", "a  \nb\t \n", 2, "a  \nb\t \n"},
		{"all blank returns empty", "\n\n  \n", 3, ""},
		{"empty input", "", 3, ""},
		{"no trailing newline gains one", "a\nb", 2, "a\nb\n"},
		{"n of one", "a\nb\nc\n", 1, "c\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TailLines(tc.s, tc.n)
			if got != tc.want {
				t.Errorf("TailLines(%q, %d) = %q, want %q", tc.s, tc.n, got, tc.want)
			}
		})
	}
}

func TestFormatIdleDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int64
		expected string
	}{
		{"zero seconds", 0, "0s"},
		{"30 seconds", 30, "30s"},
		{"45 seconds", 45, "45s"},
		{"59 seconds", 59, "59s"},
		{"exactly 60 seconds", 60, "1m"},
		{"125 seconds", 125, "2m"},
		{"300 seconds (5m)", 300, "5m"},
		{"3599 seconds", 3599, "59m"},
		{"exactly 3600 seconds", 3600, "1h"},
		{"7500 seconds (2h)", 7500, "2h"},
		{"7200 seconds (2h exact)", 7200, "2h"},
		{"86400 seconds (24h)", 86400, "24h"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatIdleDuration(tc.seconds)
			if result != tc.expected {
				t.Errorf("FormatIdleDuration(%d) = %q, want %q", tc.seconds, result, tc.expected)
			}
		})
	}
}

func TestWorktreeDisplayPath(t *testing.T) {
	tests := []struct {
		name     string
		wtRoot   string
		mainRoot string
		expected string
	}{
		{
			"main worktree",
			"/home/user/myrepo",
			"/home/user/myrepo",
			"(main)",
		},
		{
			"child worktree",
			"/home/user/myrepo.worktrees/alpha",
			"/home/user/myrepo",
			"myrepo.worktrees/alpha/",
		},
		{
			"another child worktree",
			"/home/user/myrepo.worktrees/bravo",
			"/home/user/myrepo",
			"myrepo.worktrees/bravo/",
		},
		{
			"no main root fallback",
			"/home/user/some-repo",
			"",
			"some-repo/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := WorktreeDisplayPath(tc.wtRoot, tc.mainRoot)
			if result != tc.expected {
				t.Errorf("WorktreeDisplayPath(%q, %q) = %q, want %q", tc.wtRoot, tc.mainRoot, result, tc.expected)
			}
		})
	}
}

func TestReadFabCurrent(t *testing.T) {
	t.Run("symlink present", func(t *testing.T) {
		tmp := t.TempDir()
		target := "fab/changes/260306-ab12-some-change/.status.yaml"
		if err := os.Symlink(target, tmp+"/.fab-status.yaml"); err != nil {
			t.Fatal(err)
		}

		display, folder := ReadFabCurrent(tmp)
		if display != "260306-ab12-some-change" {
			t.Errorf("display = %q, want %q", display, "260306-ab12-some-change")
		}
		if folder != "260306-ab12-some-change" {
			t.Errorf("folder = %q, want %q", folder, "260306-ab12-some-change")
		}
	})

	t.Run("broken symlink", func(t *testing.T) {
		tmp := t.TempDir()
		target := "fab/changes/260306-ab12-deleted-change/.status.yaml"
		if err := os.Symlink(target, tmp+"/.fab-status.yaml"); err != nil {
			t.Fatal(err)
		}

		display, folder := ReadFabCurrent(tmp)
		if display != "260306-ab12-deleted-change" {
			t.Errorf("display = %q, want %q", display, "260306-ab12-deleted-change")
		}
		if folder != "260306-ab12-deleted-change" {
			t.Errorf("folder = %q, want %q", folder, "260306-ab12-deleted-change")
		}
	})

	t.Run("no symlink", func(t *testing.T) {
		tmp := t.TempDir()

		display, folder := ReadFabCurrent(tmp)
		if display != "(no change)" {
			t.Errorf("display = %q, want %q", display, "(no change)")
		}
		if folder != "" {
			t.Errorf("folder = %q, want empty", folder)
		}
	})
}

func TestParseAgentState(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantState string
		wantEpoch int64
		wantOK    bool
	}{
		{"idle with epoch", "idle:1751800000", "idle", 1751800000, true},
		{"active with epoch", "active:1751800000", "active", 1751800000, true},
		{"waiting with epoch", "waiting:1751800000", "waiting", 1751800000, true},
		{"surrounding whitespace trimmed", "  idle:1751800000\n", "idle", 1751800000, true},
		{"empty value", "", "", 0, false},
		{"no epoch suffix", "idle", "", 0, false},
		{"non-integer epoch", "idle:notanum", "", 0, false},
		{"unknown state token", "bogus:1751800000", "", 0, false},
		{"empty state token", ":1751800000", "", 0, false},
		{"trailing empty epoch", "idle:", "", 0, false},
		// run-kit's three-segment form: "<state>:<epoch>:<pid>" — pid validated, ignored
		{"waiting with epoch and pid", "waiting:1751790000:48213", "waiting", 1751790000, true},
		{"idle with epoch and pid", "idle:1751800000:1", "idle", 1751800000, true},
		{"zero pid is malformed", "idle:1751800000:0", "", 0, false},
		{"negative pid is malformed", "idle:1751800000:-5", "", 0, false},
		{"non-integer pid is malformed", "idle:1751800000:abc", "", 0, false},
		{"empty pid segment is malformed", "idle:1751800000:", "", 0, false},
		{"four segments is malformed", "idle:1751800000:1:2", "", 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st, ep, ok := parseAgentState(tc.raw)
			if ok != tc.wantOK {
				t.Fatalf("parseAgentState(%q) ok = %t, want %t", tc.raw, ok, tc.wantOK)
			}
			if ok && (st != tc.wantState || ep != tc.wantEpoch) {
				t.Errorf("parseAgentState(%q) = (%q, %d), want (%q, %d)", tc.raw, st, ep, tc.wantState, tc.wantEpoch)
			}
		})
	}
}

func TestAgentDisplayFromOption(t *testing.T) {
	t.Run("idle carries an epoch-derived duration", func(t *testing.T) {
		epoch := time.Now().Unix() - 125 // ~2m ago
		state, dur := AgentDisplayFromOption("idle:" + strconv.FormatInt(epoch, 10))
		if state != "idle" {
			t.Errorf("state = %q, want idle", state)
		}
		if dur == "" {
			t.Error("expected non-empty idle duration")
		}
	})

	t.Run("active carries no duration", func(t *testing.T) {
		state, dur := AgentDisplayFromOption("active:1751800000")
		if state != "active" || dur != "" {
			t.Errorf("got (%q, %q), want (active, \"\")", state, dur)
		}
	})

	t.Run("waiting carries no duration", func(t *testing.T) {
		state, dur := AgentDisplayFromOption("waiting:1751800000")
		if state != "waiting" || dur != "" {
			t.Errorf("got (%q, %q), want (waiting, \"\")", state, dur)
		}
	})

	t.Run("unknown/unparseable yields empty state (em-dash sentinel)", func(t *testing.T) {
		for _, raw := range []string{"", "idle", "idle:nope", "bogus:1"} {
			state, dur := AgentDisplayFromOption(raw)
			if state != "" || dur != "" {
				t.Errorf("AgentDisplayFromOption(%q) = (%q, %q), want empty", raw, state, dur)
			}
		}
	})

	t.Run("future epoch clamps duration to 0s", func(t *testing.T) {
		future := time.Now().Unix() + 3600
		state, dur := AgentDisplayFromOption("idle:" + strconv.FormatInt(future, 10))
		if state != "idle" || dur != "0s" {
			t.Errorf("got (%q, %q), want (idle, 0s)", state, dur)
		}
	})
}

// TestValidatePaneResult_PaneNotFoundErrorType: the missing-pane branches
// return the typed PaneNotFoundError (detectable via errors.As for the
// pane-family 2-vs-3 exit-code mapping) with the historical message intact.
func TestValidatePaneResult_PaneNotFoundErrorType(t *testing.T) {
	cases := []struct {
		name   string
		out    string
		stderr []byte
		err    error
	}{
		{"missing pane via stderr", "", []byte("can't find pane: %9"), fmt.Errorf("exit status 1")},
		{"missing pane via empty output (tmux >=3.6)", "", nil, nil},
		{"target-grammar mismatch", "%4", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePaneResult("%9", tc.out, tc.stderr, tc.err)
			if err == nil {
				t.Fatal("expected an error")
			}
			var nf *PaneNotFoundError
			if !errors.As(err, &nf) {
				t.Fatalf("expected PaneNotFoundError, got %T: %v", err, err)
			}
			if err.Error() != "pane %9 not found" {
				t.Errorf("message drifted: %q", err.Error())
			}
		})
	}

	// Non-missing tmux failures stay untyped (mapped to exit 3 by callers).
	err := validatePaneResult("%9", "", []byte("no server running on /tmp/x"), fmt.Errorf("exit status 1"))
	var nf *PaneNotFoundError
	if errors.As(err, &nf) {
		t.Errorf("dead-server failure must NOT be PaneNotFoundError, got: %v", err)
	}
}

// tmuxSocketPathBudget is a conservative cap for the full tmux socket path
// ($TMUX_TMPDIR/tmux-$UID/<name>): macOS caps sun_path at 104 bytes
// including the terminating NUL.
const tmuxSocketPathBudget = 103

// tmuxSocketPathLen returns the length of the socket path tmux would bind
// for a server named name under TMUX_TMPDIR dir.
func tmuxSocketPathLen(dir, name string) int {
	return len(filepath.Join(dir, "tmux-"+strconv.Itoa(os.Getuid()), name))
}

// tmuxSocketDir returns a per-test private directory for TMUX_TMPDIR so the
// test's tmux socket dies with the test — tmux never unlinks its socket on
// server exit, so a socket in the shared /tmp/tmux-$UID would leak on every
// run (change 0j0t). Prefers t.TempDir(); when the resulting socket path
// would exceed the sun_path budget (long $TMPDIR bases on macOS), it falls
// back to a short /tmp dir removed via t.Cleanup — never a skip.
//
// It also scrubs $TMUX/$TMUX_PANE: $TMUX outranks TMUX_TMPDIR in tmux's
// socket resolution (-L/-S > $TMUX > TMUX_TMPDIR), so for any test run from
// inside a tmux pane an inherited $TMUX would silently redirect unscoped
// tmux calls onto the HOST server — a destructive cleanup then kills the
// host (change kgam). Empirically (tmux 3.6a) an empty $TMUX is treated as
// unset, so t.Setenv suffices. Tests that need $TMUX set (simulating a
// dispatcher inside a pane) set it themselves after this call.
func tmuxSocketDir(t *testing.T, name string) string {
	t.Helper()
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	dir := t.TempDir()
	if tmuxSocketPathLen(dir, name) > tmuxSocketPathBudget {
		short, err := os.MkdirTemp("/tmp", "fabtest-")
		if err != nil {
			t.Fatalf("create short TMUX_TMPDIR fallback: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(short) })
		dir = short
	}
	return dir
}

// TestReadAgentStateOption_Integration drives the full reader against a real
// tmux server, simulating run-kit's `rk agent setup` hook writer via
// `tmux set-option -p <option> "<state>:<epoch>"`. It covers the unknown case
// (neither option set), the canonical @rk_pane_agent_state read, and the
// dual-read contract: legacy-only panes still resolve, and when both names are
// set the canonical value wins. Skipped when tmux is unavailable.
func TestReadAgentStateOption_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	// Ephemeral private server so the test never touches the user's tmux.
	// The private TMUX_TMPDIR (process env — the code under test shells out
	// to `tmux -L` itself and must resolve the same socket dir) makes the
	// socket die with the test; a short fixed name keeps the socket path
	// inside the sun_path budget.
	server := "fabtest"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	// Start a detached session; tmux creates the server on demand.
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	paneID, err := tmux("display-message", "-p", "-t", "s", "#{pane_id}")
	if err != nil || paneID == "" {
		t.Fatalf("resolve pane id: %v (%q)", err, paneID)
	}

	t.Run("unset option → unknown", func(t *testing.T) {
		if raw := ReadAgentStateOption(paneID, server); raw != "" {
			t.Errorf("expected empty raw option, got %q", raw)
		}
		state, dur := AgentDisplayFromOption(ReadAgentStateOption(paneID, server))
		if state != "" || dur != "" {
			t.Errorf("unset option should be unknown, got (%q, %q)", state, dur)
		}
	})

	cases := []struct {
		state string
		epoch int64
	}{
		{AgentStateActive, 1751800000},
		{AgentStateWaiting, 1751800000},
		{AgentStateIdle, time.Now().Unix() - 125},
	}
	for _, tc := range cases {
		t.Run(tc.state+" set via tmux set-option -p", func(t *testing.T) {
			val := tc.state + ":" + strconv.FormatInt(tc.epoch, 10)
			if out, err := tmux("set-option", "-p", "-t", paneID, AgentStateOption, val); err != nil {
				t.Fatalf("set-option: %v: %s", err, out)
			}
			raw := ReadAgentStateOption(paneID, server)
			if raw != val {
				t.Fatalf("ReadAgentStateOption = %q, want %q", raw, val)
			}
			state, dur := AgentDisplayFromOption(raw)
			if state != tc.state {
				t.Errorf("state = %q, want %q", state, tc.state)
			}
			if tc.state == AgentStateIdle {
				if dur == "" {
					t.Error("idle must carry a duration")
				}
			} else if dur != "" {
				t.Errorf("%s must carry no duration, got %q", tc.state, dur)
			}
		})
	}

	// Dual-read contract. Reset both options first so the sub-tests below
	// start from a clean pane regardless of the loop above.
	unsetBoth := func(t *testing.T) {
		t.Helper()
		_, _ = tmux("set-option", "-pu", "-t", paneID, AgentStateOption)
		_, _ = tmux("set-option", "-pu", "-t", paneID, LegacyAgentStateOption)
	}

	t.Run("legacy-only option falls back", func(t *testing.T) {
		unsetBoth(t)
		if out, err := tmux("set-option", "-p", "-t", paneID, LegacyAgentStateOption, "active:1751800000"); err != nil {
			t.Fatalf("set-option: %v: %s", err, out)
		}
		if raw := ReadAgentStateOption(paneID, server); raw != "active:1751800000" {
			t.Fatalf("ReadAgentStateOption = %q, want legacy value", raw)
		}
	})

	t.Run("canonical wins when both are set", func(t *testing.T) {
		unsetBoth(t)
		if out, err := tmux("set-option", "-p", "-t", paneID, LegacyAgentStateOption, "idle:1600000000"); err != nil {
			t.Fatalf("set-option legacy: %v: %s", err, out)
		}
		if out, err := tmux("set-option", "-p", "-t", paneID, AgentStateOption, "waiting:1751790000:48213"); err != nil {
			t.Fatalf("set-option canonical: %v: %s", err, out)
		}
		raw := ReadAgentStateOption(paneID, server)
		if raw != "waiting:1751790000:48213" {
			t.Fatalf("ReadAgentStateOption = %q, want canonical value", raw)
		}
		if state, _ := AgentDisplayFromOption(raw); state != AgentStateWaiting {
			t.Errorf("three-segment canonical value resolved to %q, want waiting", state)
		}
	})

	t.Run("neither set → unknown", func(t *testing.T) {
		unsetBoth(t)
		if raw := ReadAgentStateOption(paneID, server); raw != "" {
			t.Errorf("expected empty raw option, got %q", raw)
		}
	})
}

// TestCurrentCommand_Integration drives the foreground-command read against a
// real tmux server: a pane running `sleep` reports `sleep`, and its pane's
// default shell reports a name IsShellCommand matches — the two shapes the
// readiness gate's takeover precondition distinguishes. Skipped when tmux is
// unavailable.
func TestCurrentCommand_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	server := "fabtest"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	// The session is pinned to an explicit `sh` rather than the host's login
	// shell: the assertion below is about the predicate recognising a shell
	// foreground, not about which shell this host happens to run (a default
	// outside the nine-name set — xonsh, say — must not fail the suite).
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24", "sh"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	paneID, err := tmux("display-message", "-p", "-t", "s", "#{pane_id}")
	if err != nil || paneID == "" {
		t.Fatalf("resolve pane id: %v (%q)", err, paneID)
	}

	t.Run("sh foreground is a shell", func(t *testing.T) {
		cmd, err := CurrentCommand(server, paneID)
		if err != nil {
			t.Fatalf("CurrentCommand: %v", err)
		}
		if !IsShellCommand(cmd) {
			t.Errorf("CurrentCommand = %q, want a known shell for a pane running sh", cmd)
		}
	})

	t.Run("sleep foreground reports sleep", func(t *testing.T) {
		if out, err := tmux("send-keys", "-t", paneID, "exec sleep 300", "Enter"); err != nil {
			t.Fatalf("start sleep: %v: %s", err, out)
		}
		// The exec is near-instant, but tmux's pane_current_command refresh is
		// not synchronous with send-keys returning — poll briefly.
		deadline := time.Now().Add(5 * time.Second)
		for {
			cmd, err := CurrentCommand(server, paneID)
			if err != nil {
				t.Fatalf("CurrentCommand: %v", err)
			}
			if cmd == "sleep" {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("CurrentCommand = %q, want %q", cmd, "sleep")
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
}
