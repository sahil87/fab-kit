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
func tmuxSocketDir(t *testing.T, name string) string {
	t.Helper()
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
// tmux server, simulating run-kit's rk agent-setup writer via
// `tmux set-option -p @rk_agent_state "<state>:<epoch>"` (the writer
// simulation directed by the intake — the actual writer does not exist yet).
// It covers the codex-pane scenario (a pane the old Claude-only _agents
// pipeline could never see) and the unknown case (option unset). Skipped when
// tmux is unavailable.
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
}
