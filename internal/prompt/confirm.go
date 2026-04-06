// Package prompt provides user interaction utilities for CLI commands.
package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm asks the user a yes/no question and returns their response.
// Returns true if the user answers "y" or "yes" (case-insensitive).
// If skip is true, returns true without prompting (used for --force and --plain modes).
func Confirm(message string, skip bool) bool {
	if skip {
		return true
	}

	fmt.Fprintf(os.Stderr, "%s [y/N]: ", message)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
}

// ConfirmDelete asks the user to confirm a resource deletion.
// Displays resource type and identifier, returns true if confirmed.
// If skip is true, returns true without prompting.
func ConfirmDelete(resourceType, identifier string, skip bool) bool {
	if skip {
		return true
	}

	fmt.Fprintf(os.Stderr, "\nYou are about to delete the following %s:\n", resourceType)
	fmt.Fprintf(os.Stderr, "  Identifier: %s\n\n", identifier)
	return Confirm("Are you sure you want to delete this resource?", false)
}
