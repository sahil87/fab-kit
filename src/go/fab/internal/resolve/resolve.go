package resolve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Typed resolution sentinels. Callers use errors.Is to distinguish
// "this reference matches no active change" (ErrNotFound — may be already
// archived, an idempotent soft skip) from "this reference is ambiguous"
// (ErrAmbiguous — a real user error that must surface). Without these,
// callers re-resolve against the archive to guess, which itself cannot tell
// not-found from ambiguous (jznd (d); precedent: archive.ErrAlreadyArchived).
var (
	ErrNotFound  = errors.New("no matching change")
	ErrAmbiguous = errors.New("ambiguous change reference")
)

// classifiedError pairs a sentinel kind with a user-facing message. Its
// Error() returns the message verbatim — the documented Common Error Messages
// strings are preserved unchanged — while errors.Is(err, kind) matches the
// sentinel, so callers can branch on not-found vs ambiguous without parsing
// the text.
type classifiedError struct {
	kind error
	msg  string
}

func (e *classifiedError) Error() string { return e.msg }
func (e *classifiedError) Unwrap() error { return e.kind }

// notFoundf / ambiguousf build a classifiedError whose surfaced text is the
// formatted message and whose sentinel is ErrNotFound / ErrAmbiguous.
func notFoundf(format string, a ...any) error {
	return &classifiedError{kind: ErrNotFound, msg: fmt.Sprintf(format, a...)}
}

func ambiguousf(format string, a ...any) error {
	return &classifiedError{kind: ErrAmbiguous, msg: fmt.Sprintf(format, a...)}
}

// FabRoot returns the fab/ directory path by searching upward from cwd.
func FabRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "fab")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("fab/ directory not found")
		}
		dir = parent
	}
}

// ToFolder resolves a change reference to a full folder name.
// If override is empty, reads .fab-status.yaml symlink at repo root.
func ToFolder(fabRoot, override string) (string, error) {
	changesDir := filepath.Join(fabRoot, "changes")

	if override != "" {
		return resolveOverride(changesDir, override)
	}
	return resolveFromCurrent(fabRoot, changesDir)
}

// ExtractID extracts the 4-char change ID from a YYMMDD-XXXX-slug folder name.
func ExtractID(folder string) string {
	parts := strings.SplitN(folder, "-", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// ToAbsDir returns the absolute directory path.
func ToAbsDir(fabRoot, override string) (string, error) {
	folder, err := ToFolder(fabRoot, override)
	if err != nil {
		return "", err
	}
	return filepath.Join(fabRoot, "changes", folder), nil
}

// ToAbsStatus returns the absolute .status.yaml path.
func ToAbsStatus(fabRoot, override string) (string, error) {
	folder, err := ToFolder(fabRoot, override)
	if err != nil {
		return "", err
	}
	return filepath.Join(fabRoot, "changes", folder, ".status.yaml"), nil
}

func resolveOverride(changesDir, override string) (string, error) {
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return "", fmt.Errorf("fab/changes/ not found.")
	}

	folders, err := listChangeFolders(changesDir)
	if err != nil {
		return "", err
	}
	if len(folders) == 0 {
		return "", notFoundf("No active changes found.")
	}

	overrideLower := strings.ToLower(override)

	// Exact match
	for _, f := range folders {
		if strings.ToLower(f) == overrideLower {
			return f, nil
		}
	}

	// Substring match
	var partials []string
	for _, f := range folders {
		if strings.Contains(strings.ToLower(f), overrideLower) {
			partials = append(partials, f)
		}
	}

	if len(partials) == 1 {
		return partials[0], nil
	}
	if len(partials) > 1 {
		return "", ambiguousf("Multiple changes match \"%s\": %s.", override, strings.Join(partials, ", "))
	}

	return "", notFoundf("No change matches \"%s\".", override)
}

func resolveFromCurrent(fabRoot, changesDir string) (string, error) {
	// Read .fab-status.yaml symlink at repo root
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	if target, err := os.Readlink(symlinkPath); err == nil {
		// target is "fab/changes/{name}/.status.yaml"
		name := ExtractFolderFromSymlink(target)
		if name != "" {
			// Trust the pointer only when its target still exists: a dangling
			// pointer (change archived/deleted underneath the gitignored
			// symlink) must fall through to the no-active-change /
			// single-change logic below so callers get the actionable
			// /fab-switch guidance instead of a silently stale folder (mz4q
			// F08). The link is left in place — resolve is a pure query with
			// no side effects. Only genuine absence falls through: any other
			// stat failure (permission, I/O) surfaces with its cause instead
			// of masquerading as "no active change" (mz4q F06 posture).
			statusPath := filepath.Join(changesDir, name, ".status.yaml")
			_, statErr := os.Stat(statusPath)
			if statErr == nil {
				return name, nil
			}
			if !os.IsNotExist(statErr) {
				return "", fmt.Errorf("stat active change target %s: %w", statusPath, statErr)
			}
		}
	}

	// Fallback: single-change guess
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return "", notFoundf("No active change. Run /fab-new <description> to start one, or /fab-switch to activate an existing one.")
	}

	var candidates []string
	entries, _ := os.ReadDir(changesDir)
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		statusPath := filepath.Join(changesDir, e.Name(), ".status.yaml")
		if _, err := os.Stat(statusPath); err == nil {
			candidates = append(candidates, e.Name())
		}
	}

	if len(candidates) == 1 {
		fmt.Fprintf(os.Stderr, "(resolved from single active change)\n")
		return candidates[0], nil
	}
	if len(candidates) == 0 {
		return "", notFoundf("No active change. Run /fab-new <description> to start one, or /fab-switch to activate an existing one.")
	}
	return "", ambiguousf("No active change (multiple changes exist — use /fab-switch).")
}

// ExtractFolderFromSymlink extracts the change folder name from a symlink target path.
// Expected format: "fab/changes/{name}/.status.yaml"
func ExtractFolderFromSymlink(target string) string {
	// Normalize separators for cross-platform
	target = filepath.ToSlash(target)
	const prefix = "fab/changes/"
	const suffix = "/.status.yaml"
	if strings.HasPrefix(target, prefix) && strings.HasSuffix(target, suffix) {
		name := target[len(prefix) : len(target)-len(suffix)]
		if name != "" && !strings.Contains(name, "/") {
			return name
		}
	}
	return ""
}

func listChangeFolders(changesDir string) ([]string, error) {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil, err
	}
	var folders []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "archive" {
			folders = append(folders, e.Name())
		}
	}
	return folders, nil
}
