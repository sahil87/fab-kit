package worktree

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ShowMenu displays a numbered menu with a Cancel option (0), reads user input,
// validates numeric input, and returns the selected index.
// defaultIdx is used when the user presses Enter without input (-1 means no default).
// Returns the selected option number (0 = Cancel, 1..N = option).
func ShowMenu(prompt string, options []string, defaultIdx int) (int, error) {
	fmt.Println(prompt)

	for i, opt := range options {
		defaultMarker := ""
		if defaultIdx == i+1 {
			defaultMarker = " " + ColorGreen + "(default)" + ColorReset
		}
		fmt.Printf("  %s%d)%s %s%s\n", ColorBold, i+1, ColorReset, opt, defaultMarker)
	}

	cancelMarker := ""
	if defaultIdx == 0 {
		cancelMarker = " " + ColorGreen + "(default)" + ColorReset
	}
	fmt.Printf("  %s0)%s Cancel%s\n", ColorBold, ColorReset, cancelMarker)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		if defaultIdx >= 0 {
			fmt.Printf("Choice [%d]: ", defaultIdx)
		} else {
			fmt.Print("Choice: ")
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, fmt.Errorf("reading input: %w", err)
		}
		line = strings.TrimSpace(line)

		// Handle empty input
		if line == "" {
			if defaultIdx >= 0 {
				return defaultIdx, nil
			}
			return 0, nil
		}

		// Validate numeric input
		choice, err := strconv.Atoi(line)
		if err != nil {
			fmt.Println("Invalid choice. Please enter a number.")
			continue
		}

		if choice < 0 || choice > len(options) {
			fmt.Printf("Invalid choice. Please enter a number between 0 and %d.\n", len(options))
			continue
		}

		return choice, nil
	}
}

// ConfirmYesNo prompts for a Y/n confirmation. Returns true if yes (default).
func ConfirmYesNo(prompt string) bool {
	fmt.Printf("%s [Y/n] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(line), "y")
}

// PromptWithDefault prompts for input with a default value.
func PromptWithDefault(prompt, defaultValue string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue
	}
	return line
}
