package internal

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// requiredTools lists prerequisites that must be available on PATH.
var requiredTools = []string{"git", "bash", "yq", "direnv"}

// checkPrerequisites validates that all required tools are available.
func checkPrerequisites() error {
	var missing []string
	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s. Install with: brew install %s",
			strings.Join(missing, ", "), strings.Join(missing, " "))
	}

	// Validate yq is v4+
	out, err := exec.Command("yq", "--version").Output()
	if err == nil {
		verStr := string(out)
		// Extract version number — yq output is like "yq (https://...) version v4.x.y"
		// or "yq version 4.x.y"
		parts := strings.Fields(verStr)
		for _, p := range parts {
			p = strings.TrimPrefix(p, "v")
			if len(p) > 0 && p[0] >= '0' && p[0] <= '9' {
				// Numeric compare — a lexicographic string compare would
				// misorder multi-digit majors (e.g. "10" < "4").
				major, convErr := strconv.Atoi(strings.Split(p, ".")[0])
				if convErr == nil && major < 4 {
					return fmt.Errorf("yq version 4+ required (found %s). Install the Go version: brew install yq", p)
				}
				break
			}
		}
	}

	return nil
}
