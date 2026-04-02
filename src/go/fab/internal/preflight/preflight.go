package preflight

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"

	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
	"gopkg.in/yaml.v3"
)

// Result holds the structured preflight output.
type Result struct {
	ID           string
	Name         string
	ChangeDir    string
	Stage        string
	DisplayStage string
	DisplayState string
	Progress     []sf.StageState
	Checklist    sf.Checklist
	Confidence   sf.Confidence
}

// Run performs preflight validation and returns structured result.
func Run(fabRoot, changeOverride string) (*Result, error) {
	// 1. Check project initialization
	configPath := filepath.Join(fabRoot, "project", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Project not initialized — fab/project/config.yaml not found. Run /fab-setup.")
	}
	constPath := filepath.Join(fabRoot, "project", "constitution.md")
	if _, err := os.Stat(constPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Project not initialized — fab/project/constitution.md not found. Run /fab-setup.")
	}

	// 2. Sync staleness warning (non-blocking)
	checkSyncStaleness(fabRoot)

	// 3. Resolve change
	folder, err := resolve.ToFolder(fabRoot, changeOverride)
	if err != nil {
		return nil, err
	}

	// 4. Check change directory
	changeDir := filepath.Join(fabRoot, "changes", folder)
	if _, err := os.Stat(changeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Change directory not found: fab/changes/%s", folder)
	}

	// 5. Check .status.yaml
	statusPath := filepath.Join(changeDir, ".status.yaml")
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		return nil, fmt.Errorf(".status.yaml not found in fab/changes/%s", folder)
	}

	// 6. Load and validate
	statusFile, err := sf.Load(statusPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to load .status.yaml: %w", err)
	}

	if err := status.Validate(statusFile); err != nil {
		return nil, fmt.Errorf("Invalid .status.yaml: %w", err)
	}

	// Build result
	id := resolve.ExtractID(folder)
	currentStage := status.CurrentStage(statusFile)
	displayStage, displayState := status.DisplayStage(statusFile)

	relChangeDir := "fab/changes/" + folder

	return &Result{
		ID:           id,
		Name:         folder,
		ChangeDir:    relChangeDir,
		Stage:        currentStage,
		DisplayStage: displayStage,
		DisplayState: displayState,
		Progress:     statusFile.GetProgressMap(),
		Checklist:    statusFile.Checklist,
		Confidence:   statusFile.Confidence,
	}, nil
}

// FormatYAML produces the YAML output matching preflight.sh format.
func FormatYAML(r *Result) string {
	var b strings.Builder

	fmt.Fprintf(&b, "id: %s\n", r.ID)
	fmt.Fprintf(&b, "name: %s\n", r.Name)
	fmt.Fprintf(&b, "change_dir: %s\n", r.ChangeDir)
	fmt.Fprintf(&b, "stage: %s\n", r.Stage)
	fmt.Fprintf(&b, "display_stage: %s\n", r.DisplayStage)
	fmt.Fprintf(&b, "display_state: %s\n", r.DisplayState)
	b.WriteString("progress:\n")
	for _, ss := range r.Progress {
		fmt.Fprintf(&b, "  %s: %s\n", ss.Stage, ss.State)
	}
	b.WriteString("checklist:\n")
	fmt.Fprintf(&b, "  generated: %v\n", r.Checklist.Generated)
	fmt.Fprintf(&b, "  completed: %d\n", r.Checklist.Completed)
	fmt.Fprintf(&b, "  total: %d\n", r.Checklist.Total)
	b.WriteString("confidence:\n")
	fmt.Fprintf(&b, "  certain: %d\n", r.Confidence.Certain)
	fmt.Fprintf(&b, "  confident: %d\n", r.Confidence.Confident)
	fmt.Fprintf(&b, "  tentative: %d\n", r.Confidence.Tentative)
	fmt.Fprintf(&b, "  unresolved: %d\n", r.Confidence.Unresolved)
	fmt.Fprintf(&b, "  score: %.1f\n", r.Confidence.Score)

	if r.Confidence.Indicative != nil && *r.Confidence.Indicative {
		b.WriteString("  indicative: true\n")
	}

	return b.String()
}

func checkSyncStaleness(fabRoot string) {
	kitVersion := ""
	versionFile := filepath.Join(fabRoot, ".kit", "VERSION")
	if data, err := os.ReadFile(versionFile); err == nil {
		kitVersion = strings.TrimSpace(string(data))
	}
	if kitVersion == "" {
		return
	}

	configPath := filepath.Join(fabRoot, "project", "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	var cfg struct {
		FabVersion string `yaml:"fab_version"`
	}
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		return
	}
	configVersion := strings.TrimSpace(cfg.FabVersion)
	if configVersion == "" {
		return
	}

	if kitVersion != configVersion {
		fmt.Fprintf(os.Stderr, "⚠ Skills may be out of sync — run fab sync to refresh (engine %s, project %s)\n", kitVersion, configVersion)
	}
}
