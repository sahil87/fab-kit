package runtime

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func setupFabRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	if err := os.MkdirAll(fabRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	return fabRoot
}

// intPtr and int64Ptr are test helpers for building AgentEntry fixtures with
// optional numeric fields.
func intPtr(v int) *int        { return &v }
func int64Ptr(v int64) *int64  { return &v }

func TestFilePath(t *testing.T) {
	fabRoot := "/tmp/repo/fab"
	got := FilePath(fabRoot)
	want := "/tmp/repo/.fab-runtime.yaml"
	if got != want {
		t.Errorf("FilePath(%q) = %q, want %q", fabRoot, got, want)
	}
}

func TestLoadFile_NonExistent(t *testing.T) {
	m, err := LoadFile("/tmp/nonexistent-runtime.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestLoadFile_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fab-runtime.yaml")
	content := `_agents:
  uuid-1:
    idle_since: 1741193400
    tmux_pane: "%15"
last_run_gc: 1741193000
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agents, ok := m["_agents"].(map[string]interface{})
	if !ok {
		t.Fatal("expected _agents map")
	}
	entry, ok := agents["uuid-1"].(map[string]interface{})
	if !ok {
		t.Fatal("expected uuid-1 entry as map")
	}
	if entry["idle_since"] == nil {
		t.Error("expected idle_since to be set")
	}
	if entry["tmux_pane"] != "%15" {
		t.Errorf("tmux_pane = %v, want \"%%15\"", entry["tmux_pane"])
	}
	if m["last_run_gc"] == nil {
		t.Error("expected last_run_gc to be set")
	}
}

func TestLoadFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fab-runtime.yaml")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestSaveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fab-runtime.yaml")

	m := map[string]interface{}{
		"_agents": map[string]interface{}{
			"uuid-1": map[string]interface{}{
				"idle_since": 1741193400,
				"tmux_pane":  "%5",
			},
		},
	}

	if err := SaveFile(path, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	var loaded map[string]interface{}
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse saved file: %v", err)
	}

	agents, ok := loaded["_agents"].(map[string]interface{})
	if !ok {
		t.Fatal("expected _agents map")
	}
	entry, ok := agents["uuid-1"].(map[string]interface{})
	if !ok {
		t.Fatal("expected uuid-1 entry")
	}
	if entry["idle_since"] == nil {
		t.Error("expected idle_since")
	}
	if entry["tmux_pane"] != "%5" {
		t.Errorf("tmux_pane = %v, want \"%%5\"", entry["tmux_pane"])
	}
}

func TestWriteAgent_CreatesFile(t *testing.T) {
	fabRoot := setupFabRoot(t)
	sessionID := "uuid-new"

	entry := AgentEntry{
		Change:         "260417-2fbb-pane-flag",
		IdleSince:      int64Ptr(1741193400),
		PID:            intPtr(2356168),
		TmuxServer:     "fabKit",
		TmuxPane:       "%15",
		TranscriptPath: "/home/user/.claude/uuid-new.jsonl",
	}

	if err := WriteAgent(fabRoot, sessionID, entry); err != nil {
		t.Fatalf("WriteAgent failed: %v", err)
	}

	m, err := LoadFile(FilePath(fabRoot))
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	agents, ok := m["_agents"].(map[string]interface{})
	if !ok {
		t.Fatal("expected _agents map")
	}
	got, ok := agents[sessionID].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry for sessionID")
	}
	if got["change"] != "260417-2fbb-pane-flag" {
		t.Errorf("change = %v, want 260417-2fbb-pane-flag", got["change"])
	}
	if got["tmux_pane"] != "%15" {
		t.Errorf("tmux_pane = %v, want \"%%15\"", got["tmux_pane"])
	}
	if got["tmux_server"] != "fabKit" {
		t.Errorf("tmux_server = %v, want fabKit", got["tmux_server"])
	}
	if got["idle_since"] == nil {
		t.Error("expected idle_since to be present")
	}
	if got["pid"] == nil {
		t.Error("expected pid to be present")
	}
	if got["transcript_path"] != "/home/user/.claude/uuid-new.jsonl" {
		t.Errorf("transcript_path = %v", got["transcript_path"])
	}
}

func TestWriteAgent_OmitsEmptyFields(t *testing.T) {
	fabRoot := setupFabRoot(t)
	sessionID := "uuid-minimal"

	entry := AgentEntry{
		IdleSince: int64Ptr(1741193400),
	}

	if err := WriteAgent(fabRoot, sessionID, entry); err != nil {
		t.Fatalf("WriteAgent failed: %v", err)
	}

	m, err := LoadFile(FilePath(fabRoot))
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	agents := m["_agents"].(map[string]interface{})
	got := agents[sessionID].(map[string]interface{})

	for _, key := range []string{"change", "pid", "tmux_server", "tmux_pane", "transcript_path"} {
		if _, present := got[key]; present {
			t.Errorf("expected %s to be omitted, got %v", key, got[key])
		}
	}
	if got["idle_since"] == nil {
		t.Error("expected idle_since to be present")
	}
}

func TestWriteAgent_DiscussionMode(t *testing.T) {
	fabRoot := setupFabRoot(t)
	sessionID := "uuid-discussion"

	// No change — discussion mode.
	entry := AgentEntry{
		IdleSince: int64Ptr(1741193400),
		TmuxPane:  "%7",
	}

	if err := WriteAgent(fabRoot, sessionID, entry); err != nil {
		t.Fatalf("WriteAgent failed: %v", err)
	}

	m, err := LoadFile(FilePath(fabRoot))
	if err != nil {
		t.Fatal(err)
	}
	agents := m["_agents"].(map[string]interface{})
	got := agents[sessionID].(map[string]interface{})

	if _, present := got["change"]; present {
		t.Errorf("expected change to be absent in discussion mode, got %v", got["change"])
	}
	if got["tmux_pane"] != "%7" {
		t.Errorf("tmux_pane = %v, want \"%%7\"", got["tmux_pane"])
	}
}

func TestWriteAgent_Overwrites(t *testing.T) {
	fabRoot := setupFabRoot(t)
	sessionID := "uuid-over"

	first := AgentEntry{IdleSince: int64Ptr(1000), TmuxPane: "%5"}
	if err := WriteAgent(fabRoot, sessionID, first); err != nil {
		t.Fatal(err)
	}

	second := AgentEntry{IdleSince: int64Ptr(2000), TmuxPane: "%6"}
	if err := WriteAgent(fabRoot, sessionID, second); err != nil {
		t.Fatal(err)
	}

	m, _ := LoadFile(FilePath(fabRoot))
	got := m["_agents"].(map[string]interface{})[sessionID].(map[string]interface{})
	if got["tmux_pane"] != "%6" {
		t.Errorf("expected overwrite to win, got tmux_pane=%v", got["tmux_pane"])
	}
}

func TestClearAgent_RemovesEntry(t *testing.T) {
	fabRoot := setupFabRoot(t)
	sessionID := "uuid-1"

	if err := WriteAgent(fabRoot, sessionID, AgentEntry{IdleSince: int64Ptr(1000)}); err != nil {
		t.Fatal(err)
	}
	if err := ClearAgent(fabRoot, sessionID); err != nil {
		t.Fatalf("ClearAgent failed: %v", err)
	}

	m, _ := LoadFile(FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	if _, present := agents[sessionID]; present {
		t.Error("expected entry to be removed")
	}
}

func TestClearAgent_FileNotExists(t *testing.T) {
	fabRoot := setupFabRoot(t)

	if err := ClearAgent(fabRoot, "uuid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not have created the file as a side effect.
	if _, err := os.Stat(FilePath(fabRoot)); !os.IsNotExist(err) {
		t.Error("expected runtime file to not exist after ClearAgent on missing file")
	}
}

func TestClearAgent_Idempotent(t *testing.T) {
	fabRoot := setupFabRoot(t)
	sessionID := "uuid-1"

	if err := WriteAgent(fabRoot, sessionID, AgentEntry{IdleSince: int64Ptr(1000)}); err != nil {
		t.Fatal(err)
	}
	if err := ClearAgent(fabRoot, sessionID); err != nil {
		t.Fatalf("first ClearAgent failed: %v", err)
	}
	if err := ClearAgent(fabRoot, sessionID); err != nil {
		t.Fatalf("second ClearAgent failed: %v", err)
	}
}

func TestClearAgentIdle_PreservesFields(t *testing.T) {
	fabRoot := setupFabRoot(t)
	sessionID := "uuid-1"

	entry := AgentEntry{
		Change:         "260417-2fbb",
		IdleSince:      int64Ptr(1000),
		PID:            intPtr(12345),
		TmuxServer:     "fabKit",
		TmuxPane:       "%15",
		TranscriptPath: "/tmp/t.jsonl",
	}
	if err := WriteAgent(fabRoot, sessionID, entry); err != nil {
		t.Fatal(err)
	}

	if err := ClearAgentIdle(fabRoot, sessionID); err != nil {
		t.Fatalf("ClearAgentIdle failed: %v", err)
	}

	m, _ := LoadFile(FilePath(fabRoot))
	got := m["_agents"].(map[string]interface{})[sessionID].(map[string]interface{})

	if _, present := got["idle_since"]; present {
		t.Error("expected idle_since to be removed")
	}
	if got["change"] != "260417-2fbb" {
		t.Errorf("change lost: %v", got["change"])
	}
	if got["tmux_pane"] != "%15" {
		t.Errorf("tmux_pane lost: %v", got["tmux_pane"])
	}
	if got["tmux_server"] != "fabKit" {
		t.Errorf("tmux_server lost: %v", got["tmux_server"])
	}
	if got["pid"] == nil {
		t.Error("pid lost")
	}
	if got["transcript_path"] != "/tmp/t.jsonl" {
		t.Errorf("transcript_path lost: %v", got["transcript_path"])
	}
}

func TestClearAgentIdle_MissingEntry(t *testing.T) {
	fabRoot := setupFabRoot(t)

	// File does not exist — no-op.
	if err := ClearAgentIdle(fabRoot, "uuid-nope"); err != nil {
		t.Fatalf("unexpected error on missing file: %v", err)
	}

	// File exists but entry missing — no-op.
	if err := WriteAgent(fabRoot, "uuid-other", AgentEntry{IdleSince: int64Ptr(1000)}); err != nil {
		t.Fatal(err)
	}
	if err := ClearAgentIdle(fabRoot, "uuid-nope"); err != nil {
		t.Fatalf("unexpected error on missing entry: %v", err)
	}
}

func TestGCIfDue_NoFile(t *testing.T) {
	fabRoot := setupFabRoot(t)
	if err := GCIfDue(fabRoot, 180*time.Second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(FilePath(fabRoot)); !os.IsNotExist(err) {
		t.Error("expected runtime file to not be created by GCIfDue on missing file")
	}
}

func TestGCIfDue_Throttled(t *testing.T) {
	fabRoot := setupFabRoot(t)

	// Seed a file with recent last_run_gc and an entry with a dead pid.
	rtPath := FilePath(fabRoot)
	now := time.Now().Unix()
	seed := map[string]interface{}{
		"_agents": map[string]interface{}{
			"uuid-dead": map[string]interface{}{
				"pid":        999999999, // essentially guaranteed not to exist
				"idle_since": now - 100,
			},
		},
		"last_run_gc": now - 60, // 60s ago, within 180s window
	}
	if err := SaveFile(rtPath, seed); err != nil {
		t.Fatal(err)
	}

	if err := GCIfDue(fabRoot, 180*time.Second); err != nil {
		t.Fatalf("GCIfDue failed: %v", err)
	}

	m, _ := LoadFile(rtPath)
	agents := m["_agents"].(map[string]interface{})
	if _, present := agents["uuid-dead"]; !present {
		t.Error("expected dead entry to be preserved when GC is throttled")
	}

	// last_run_gc should be unchanged.
	lastRun, ok := asInt64(m["last_run_gc"])
	if !ok || lastRun != now-60 {
		t.Errorf("expected last_run_gc unchanged, got %v", m["last_run_gc"])
	}
}

func TestGCIfDue_PrunesDeadPID(t *testing.T) {
	fabRoot := setupFabRoot(t)

	rtPath := FilePath(fabRoot)
	now := time.Now().Unix()
	seed := map[string]interface{}{
		"_agents": map[string]interface{}{
			"uuid-dead": map[string]interface{}{
				"pid":        999999999,
				"idle_since": now - 100,
			},
		},
		"last_run_gc": now - 300, // past 180s window
	}
	if err := SaveFile(rtPath, seed); err != nil {
		t.Fatal(err)
	}

	if err := GCIfDue(fabRoot, 180*time.Second); err != nil {
		t.Fatalf("GCIfDue failed: %v", err)
	}

	m, _ := LoadFile(rtPath)
	agents := m["_agents"].(map[string]interface{})
	if _, present := agents["uuid-dead"]; present {
		t.Error("expected dead entry to be pruned")
	}

	lastRun, _ := asInt64(m["last_run_gc"])
	if lastRun < now {
		t.Errorf("expected last_run_gc updated to ~%d, got %d", now, lastRun)
	}
}

func TestGCIfDue_PreservesLivePID(t *testing.T) {
	fabRoot := setupFabRoot(t)

	// Use our own PID — guaranteed alive during test.
	mypid := os.Getpid()

	rtPath := FilePath(fabRoot)
	now := time.Now().Unix()
	seed := map[string]interface{}{
		"_agents": map[string]interface{}{
			"uuid-live": map[string]interface{}{
				"pid":        mypid,
				"idle_since": now - 100,
			},
		},
		"last_run_gc": now - 300,
	}
	if err := SaveFile(rtPath, seed); err != nil {
		t.Fatal(err)
	}

	if err := GCIfDue(fabRoot, 180*time.Second); err != nil {
		t.Fatalf("GCIfDue failed: %v", err)
	}

	m, _ := LoadFile(rtPath)
	agents := m["_agents"].(map[string]interface{})
	if _, present := agents["uuid-live"]; !present {
		t.Error("expected live entry to be preserved")
	}
}

func TestGCIfDue_PreservesPidless(t *testing.T) {
	fabRoot := setupFabRoot(t)

	rtPath := FilePath(fabRoot)
	now := time.Now().Unix()
	seed := map[string]interface{}{
		"_agents": map[string]interface{}{
			"uuid-nopid": map[string]interface{}{
				"idle_since": now - 100,
				"tmux_pane":  "%7",
			},
		},
		"last_run_gc": now - 300,
	}
	if err := SaveFile(rtPath, seed); err != nil {
		t.Fatal(err)
	}

	if err := GCIfDue(fabRoot, 180*time.Second); err != nil {
		t.Fatalf("GCIfDue failed: %v", err)
	}

	m, _ := LoadFile(rtPath)
	agents := m["_agents"].(map[string]interface{})
	if _, present := agents["uuid-nopid"]; !present {
		t.Error("expected pid-less entry to be preserved regardless of liveness")
	}
}

func TestPidAlive(t *testing.T) {
	// Our own PID is alive.
	if !pidAlive(os.Getpid()) {
		t.Error("pidAlive(self) should be true")
	}
	// Zero and negative PIDs are not valid liveness probes.
	if pidAlive(0) {
		t.Error("pidAlive(0) should be false")
	}
	if pidAlive(-1) {
		t.Error("pidAlive(-1) should be false")
	}
	// A clearly dead PID.
	if pidAlive(999999999) {
		// EPERM is possible in exotic environments; accept skip in that case.
		if err := syscall.Kill(999999999, 0); err != syscall.ESRCH {
			t.Skipf("999999999 is not ESRCH in this environment: err=%v", err)
		} else {
			t.Error("pidAlive(999999999) should be false")
		}
	}
}
