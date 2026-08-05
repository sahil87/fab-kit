package dispatch

import (
	"os"
	"os/exec"
	"path/filepath"
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
