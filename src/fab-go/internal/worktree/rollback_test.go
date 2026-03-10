package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRollback_LIFOOrder(t *testing.T) {
	rb := NewRollback()

	// Register commands that create files containing a sequence number.
	// Because Execute runs in reverse order, the last-registered command
	// writes first, overwriting the shared order file. The final value
	// in the file tells us which command ran last.
	dir := t.TempDir()
	orderFile := filepath.Join(dir, "order.txt")

	// "first" is registered first → executed last in LIFO
	rb.Register("bash", "-c", "echo first > "+orderFile)
	// "second" is registered second → executed first in LIFO
	rb.Register("bash", "-c", "echo second > "+orderFile)

	rb.Execute()

	// In LIFO order: "second" runs first, then "first" overwrites the file.
	// So the file should contain "first" if LIFO is correct.
	data, err := os.ReadFile(orderFile)
	if err != nil {
		t.Fatalf("order file not created: %v", err)
	}
	got := string(data)
	if got != "first\n" {
		t.Errorf("expected LIFO to end with 'first' command, got %q", got)
	}

	// Verify Commands() returns entries in registration order
	rb2 := NewRollback()
	rb2.Register("git", "cmd-A")
	rb2.Register("git", "cmd-B")
	rb2.Register("git", "cmd-C")

	cmds := rb2.Commands()
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}
	if cmds[0][1] != "cmd-A" || cmds[1][1] != "cmd-B" || cmds[2][1] != "cmd-C" {
		t.Errorf("unexpected command order: %v", cmds)
	}
}

func TestRollback_Disarm(t *testing.T) {
	rb := NewRollback()
	dir := t.TempDir()

	marker := filepath.Join(dir, "should-not-exist")
	rb.Register("touch", marker)
	rb.Disarm()
	rb.Execute()

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Error("disarmed rollback should not have created the file")
	}
}

func TestRollback_Empty(t *testing.T) {
	rb := NewRollback()
	// Execute on empty stack should not panic
	rb.Execute()
}

func TestRollback_ContinuesOnFailure(t *testing.T) {
	rb := NewRollback()
	dir := t.TempDir()

	marker := filepath.Join(dir, "exists")
	rb.Register("touch", marker)
	rb.Register("nonexistent-command-that-should-fail")

	// Execute should still run the first command even though second fails
	rb.Execute()

	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error("rollback should continue executing after a command failure")
	}
}
