package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prompts the user for a yes/no confirmation.
// Returns true if the user answers yes, false otherwise.
// Default is no (pressing enter without input returns false).
func Confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [y/N]: ", prompt)

	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// ConfirmWithDefault prompts the user with a custom default.
func ConfirmWithDefault(prompt string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)

	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}

	fmt.Printf("%s %s: ", prompt, hint)

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}
