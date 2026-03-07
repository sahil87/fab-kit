package log

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wvrdz/fab-kit/src/fab-go/internal/resolve"
)

// Command logs a skill invocation.
func Command(fabRoot, cmd, changeArg, args string) error {
	var changeDir string
	var err error

	if changeArg != "" {
		changeDir, err = resolve.ToAbsDir(fabRoot, changeArg)
		if err != nil {
			return err
		}
	} else {
		// No change arg: resolve from .fab-status.yaml symlink, graceful degradation
		repoRoot := filepath.Dir(fabRoot)
		symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
		if _, statErr := os.Lstat(symlinkPath); os.IsNotExist(statErr) {
			return nil // silent exit
		}
		changeDir, err = resolve.ToAbsDir(fabRoot, "")
		if err != nil {
			return nil // silent exit
		}
		if _, statErr := os.Stat(changeDir); os.IsNotExist(statErr) {
			return nil // silent exit
		}
	}

	entry := map[string]interface{}{
		"ts":    nowISO(),
		"event": "command",
		"cmd":   cmd,
	}
	if args != "" {
		entry["args"] = args
	}

	return appendJSON(changeDir, entry)
}

// ConfidenceLog logs a confidence score change.
func ConfidenceLog(fabRoot, changeArg string, score float64, delta, trigger string) error {
	changeDir, err := resolve.ToAbsDir(fabRoot, changeArg)
	if err != nil {
		return err
	}

	entry := map[string]interface{}{
		"ts":      nowISO(),
		"event":   "confidence",
		"score":   score,
		"delta":   delta,
		"trigger": trigger,
	}

	return appendJSON(changeDir, entry)
}

// Review logs a review outcome.
func Review(fabRoot, changeArg, result, rework string) error {
	changeDir, err := resolve.ToAbsDir(fabRoot, changeArg)
	if err != nil {
		return err
	}

	entry := map[string]interface{}{
		"ts":     nowISO(),
		"event":  "review",
		"result": result,
	}
	if rework != "" {
		entry["rework"] = rework
	}

	return appendJSON(changeDir, entry)
}

// Transition logs a stage transition.
func Transition(fabRoot, changeArg, stage, action, from, reason, driver string) error {
	changeDir, err := resolve.ToAbsDir(fabRoot, changeArg)
	if err != nil {
		return err
	}

	entry := map[string]interface{}{
		"ts":     nowISO(),
		"event":  "stage-transition",
		"stage":  stage,
		"action": action,
	}
	if from != "" {
		entry["from"] = from
	}
	if reason != "" {
		entry["reason"] = reason
	}
	if driver != "" {
		entry["driver"] = driver
	}

	return appendJSON(changeDir, entry)
}

func appendJSON(changeDir string, entry map[string]interface{}) error {
	historyFile := filepath.Join(changeDir, ".history.jsonl")

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	f, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open history file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

func nowISO() string {
	return time.Now().Format(time.RFC3339)
}
