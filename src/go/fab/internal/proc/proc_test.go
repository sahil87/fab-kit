package proc

import "testing"

// TestClaudePID_SmokeTest verifies ClaudePID returns a positive integer when
// invoked from a normal test process. The test process's parent is `go test`
// and its grandparent is the shell that invoked it — both exist and have
// well-defined PIDs.
func TestClaudePID_SmokeTest(t *testing.T) {
	gp, err := ClaudePID()
	if err != nil {
		t.Fatalf("ClaudePID() returned error: %v", err)
	}
	if gp <= 0 {
		t.Errorf("ClaudePID() = %d, want positive integer", gp)
	}
}
