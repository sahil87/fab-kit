package change

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wvrdz/fab-kit/src/go/fab/internal/log"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/resolve"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/status"
	sf "github.com/wvrdz/fab-kit/src/go/fab/internal/statusfile"
)

var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
var idRegex = regexp.MustCompile(`^[a-z0-9]{4}$`)
var idChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// New creates a new change directory with initialized .status.yaml.
func New(fabRoot, slug, changeID, logArgs string) (string, error) {
	if slug == "" {
		return "", fmt.Errorf("--slug is required")
	}
	if !slugRegex.MatchString(slug) {
		return "", fmt.Errorf("Invalid slug format '%s' (expected alphanumeric and hyphens, no leading/trailing hyphen)", slug)
	}

	idProvided := changeID != ""
	if idProvided {
		if !idRegex.MatchString(changeID) {
			return "", fmt.Errorf("Invalid change-id '%s' (expected 4 lowercase alphanumeric chars)", changeID)
		}
	}

	changesDir := filepath.Join(fabRoot, "changes")
	datePrefix := time.Now().UTC().Format("060102")

	if idProvided {
		if hasIDCollision(changesDir, changeID) {
			existing := findCollision(changesDir, changeID)
			return "", fmt.Errorf("Change ID '%s' already in use (%s)", changeID, existing)
		}
	} else {
		var err error
		changeID, err = generateUniqueID(changesDir, 10)
		if err != nil {
			return "", err
		}
	}

	folderName := fmt.Sprintf("%s-%s-%s", datePrefix, changeID, slug)
	changeDir := filepath.Join(changesDir, folderName)

	if err := os.Mkdir(changeDir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	createdBy := detectCreatedBy()
	now := time.Now().UTC().Format(time.RFC3339)

	// Initialize .status.yaml from template
	templatePath := filepath.Join(fabRoot, ".kit", "templates", "status.yaml")
	tmplData, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("read template: %w", err)
	}

	content := string(tmplData)
	content = strings.ReplaceAll(content, "{ID}", changeID)
	content = strings.ReplaceAll(content, "{NAME}", folderName)
	content = strings.ReplaceAll(content, "{CREATED}", now)
	content = strings.ReplaceAll(content, "{CREATED_BY}", createdBy)

	statusPath := filepath.Join(changeDir, ".status.yaml")
	if err := os.WriteFile(statusPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write .status.yaml: %w", err)
	}

	// Start intake stage
	statusFile, err := sf.Load(statusPath)
	if err != nil {
		return "", fmt.Errorf("load new status: %w", err)
	}
	if err := status.Start(statusFile, statusPath, fabRoot, "intake", "fab-new", "", ""); err != nil {
		return "", fmt.Errorf("start intake: %w", err)
	}

	if logArgs != "" {
		_ = log.Command(fabRoot, "fab-new", folderName, logArgs)
	}

	return folderName, nil
}

// Rename renames a change folder's slug.
func Rename(fabRoot, currentFolder, newSlug string) (string, error) {
	if currentFolder == "" {
		return "", fmt.Errorf("--folder is required")
	}
	if newSlug == "" {
		return "", fmt.Errorf("--slug is required")
	}
	if !slugRegex.MatchString(newSlug) {
		return "", fmt.Errorf("Invalid slug format '%s' (expected alphanumeric and hyphens, no leading/trailing hyphen)", newSlug)
	}

	changesDir := filepath.Join(fabRoot, "changes")
	oldPath := filepath.Join(changesDir, currentFolder)
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return "", fmt.Errorf("Change folder '%s' not found", currentFolder)
	}

	// Extract YYMMDD-XXXX prefix
	parts := strings.SplitN(currentFolder, "-", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid folder name format")
	}
	prefix := parts[0] + "-" + parts[1]
	newName := prefix + "-" + newSlug

	if newName == currentFolder {
		return "", fmt.Errorf("New name is the same as current name")
	}

	newPath := filepath.Join(changesDir, newName)
	if _, err := os.Stat(newPath); err == nil {
		return "", fmt.Errorf("Folder '%s' already exists", newName)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return "", fmt.Errorf("rename: %w", err)
	}

	// Update .status.yaml name field
	statusPath := filepath.Join(newPath, ".status.yaml")
	statusFile, err := sf.Load(statusPath)
	if err == nil {
		statusFile.Name = newName
		_ = statusFile.Save(statusPath)
	}

	// Update .fab-status.yaml symlink if it points to old folder
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	if target, err := os.Readlink(symlinkPath); err == nil {
		expectedOld := fmt.Sprintf("fab/changes/%s/.status.yaml", currentFolder)
		if target == expectedOld {
			newTarget := fmt.Sprintf("fab/changes/%s/.status.yaml", newName)
			os.Remove(symlinkPath)
			_ = os.Symlink(newTarget, symlinkPath)
		}
	}

	_ = log.Command(fabRoot, "changeman-rename", newName, "--folder "+currentFolder+" --slug "+newSlug)

	return newName, nil
}

// Switch changes the active change pointer.
func Switch(fabRoot, name string) (string, error) {
	folder, err := resolve.ToFolder(fabRoot, name)
	if err != nil {
		return "", err
	}

	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	target := fmt.Sprintf("fab/changes/%s/.status.yaml", folder)
	os.Remove(symlinkPath)
	if err := os.Symlink(target, symlinkPath); err != nil {
		return "", fmt.Errorf("create .fab-status.yaml symlink: %w", err)
	}

	// Derive display info
	statusPath := filepath.Join(fabRoot, "changes", folder, ".status.yaml")
	displayStage := "unknown"
	displayState := "pending"
	routingStage := "unknown"
	confDisplay := "not yet scored"

	if statusFile, err := sf.Load(statusPath); err == nil {
		ds, dstate := status.DisplayStage(statusFile)
		displayStage = ds
		displayState = dstate
		routingStage = status.CurrentStage(statusFile)

		c := statusFile.Confidence
		totalCounts := c.Certain + c.Confident + c.Tentative + c.Unresolved
		if c.Score == 0 && totalCounts == 0 {
			confDisplay = "not yet scored"
		} else if c.Indicative != nil && *c.Indicative {
			confDisplay = fmt.Sprintf("%.1f of 5.0 (indicative)", c.Score)
		} else {
			confDisplay = fmt.Sprintf("%.1f of 5.0", c.Score)
		}
	}

	dnum := sf.StageNumber(displayStage)

	var output strings.Builder
	fmt.Fprintf(&output, ".fab-status.yaml → %s\n", folder)
	fmt.Fprintln(&output)
	fmt.Fprintf(&output, "Stage:       %s (%d/8) — %s\n", displayStage, dnum, displayState)
	fmt.Fprintf(&output, "Confidence:  %s\n", confDisplay)

	cmd := defaultCommand(routingStage)
	nstage := sf.NextStage(routingStage)
	if nstage != "" {
		fmt.Fprintf(&output, "Next:        %s (via %s)", nstage, cmd)
	} else {
		fmt.Fprintf(&output, "Next:        %s", cmd)
	}

	return output.String(), nil
}

// SwitchBlank deactivates the current change.
func SwitchBlank(fabRoot string) string {
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		return "No active change (already blank)."
	}
	os.Remove(symlinkPath)
	return "No active change."
}

// List lists changes with stage info.
func List(fabRoot string, archive bool) ([]string, error) {
	scanDir := filepath.Join(fabRoot, "changes")
	if archive {
		scanDir = filepath.Join(fabRoot, "changes", "archive")
	}

	if _, err := os.Stat(scanDir); os.IsNotExist(err) {
		if archive {
			return nil, nil // empty is valid for archive
		}
		return nil, fmt.Errorf("fab/changes/ not found.")
	}

	entries, err := os.ReadDir(scanDir)
	if err != nil {
		return nil, err
	}

	var results []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !archive && e.Name() == "archive" {
			continue
		}

		statusPath := filepath.Join(scanDir, e.Name(), ".status.yaml")
		statusFile, loadErr := sf.Load(statusPath)

		if loadErr != nil {
			results = append(results, fmt.Sprintf("%s:unknown:unknown", e.Name()))
			fmt.Fprintf(os.Stderr, "Warning: .status.yaml not found for %s\n", e.Name())
			continue
		}

		ds, dstate := status.DisplayStage(statusFile)
		c := statusFile.Confidence
		indicative := "false"
		if c.Indicative != nil && *c.Indicative {
			indicative = "true"
		}
		results = append(results, fmt.Sprintf("%s:%s:%s:%.1f:%s", e.Name(), ds, dstate, c.Score, indicative))
	}

	return results, nil
}

// Resolve is a passthrough to resolve.ToFolder with --folder mode.
func Resolve(fabRoot, override string) (string, error) {
	return resolve.ToFolder(fabRoot, override)
}

func detectCreatedBy() string {
	// Try gh api user
	out, err := exec.Command("gh", "api", "user", "--jq", ".login").Output()
	if err == nil {
		user := strings.TrimSpace(string(out))
		if user != "" {
			return user
		}
	}

	// Try git config
	out, err = exec.Command("git", "config", "user.name").Output()
	if err == nil {
		user := strings.TrimSpace(string(out))
		if user != "" {
			return user
		}
	}

	return "unknown"
}

func generateRandomID() (string, error) {
	b := make([]byte, 4)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(idChars))))
		if err != nil {
			return "", err
		}
		b[i] = idChars[n.Int64()]
	}
	return string(b), nil
}

func generateUniqueID(changesDir string, maxRetries int) (string, error) {
	for i := 0; i < maxRetries; i++ {
		id, err := generateRandomID()
		if err != nil {
			return "", err
		}
		if !hasIDCollision(changesDir, id) {
			return id, nil
		}
	}
	return "", fmt.Errorf("Failed to generate unique change ID after %d attempts", maxRetries)
}

func hasIDCollision(changesDir, changeID string) bool {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return false
	}
	pattern := fmt.Sprintf("??????-%s-*", changeID)
	for _, e := range entries {
		if e.IsDir() {
			matched, _ := filepath.Match(pattern, e.Name())
			if matched {
				return true
			}
		}
	}
	return false
}

func findCollision(changesDir, changeID string) string {
	entries, _ := os.ReadDir(changesDir)
	pattern := fmt.Sprintf("??????-%s-*", changeID)
	for _, e := range entries {
		if e.IsDir() {
			matched, _ := filepath.Match(pattern, e.Name())
			if matched {
				return e.Name()
			}
		}
	}
	return ""
}

func defaultCommand(stage string) string {
	switch stage {
	case "intake", "spec", "tasks", "apply", "review":
		return "/fab-continue"
	case "hydrate":
		return "/git-pr"
	case "ship":
		return "/git-pr-review"
	case "review-pr":
		return "/fab-archive"
	default:
		return "/fab-status"
	}
}
