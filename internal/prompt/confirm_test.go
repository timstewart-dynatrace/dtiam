package prompt

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestConfirm_ForceSkips(t *testing.T) {
	// When skip=true, Confirm should return true without reading stdin
	result := Confirm("Delete everything?", true)
	if !result {
		t.Error("expected Confirm to return true when skip=true")
	}
}

func TestConfirm_AcceptsYes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"lowercase y", "y\n", true},
		{"lowercase yes", "yes\n", true},
		{"uppercase Y", "Y\n", true},
		{"uppercase YES", "YES\n", true},
		{"mixed case Yes", "Yes\n", true},
		{"no", "n\n", false},
		{"empty", "\n", false},
		{"random text", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original stdin and stderr
			origStdin := os.Stdin
			origStderr := os.Stderr
			defer func() {
				os.Stdin = origStdin
				os.Stderr = origStderr
			}()

			// Create a pipe to simulate stdin
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}

			// Suppress stderr output during test
			devNull, err := os.Open(os.DevNull)
			if err != nil {
				t.Fatalf("failed to open devnull: %v", err)
			}
			defer devNull.Close()
			os.Stderr = devNull

			os.Stdin = r
			go func() {
				defer w.Close()
				_, _ = w.Write([]byte(tt.input))
			}()

			got := Confirm("Test question?", false)
			if got != tt.want {
				t.Errorf("Confirm() with input %q = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConfirmDelete_ForceSkips(t *testing.T) {
	// When skip=true, ConfirmDelete should return true without prompting
	result := ConfirmDelete("group", "my-group-uuid", true)
	if !result {
		t.Error("expected ConfirmDelete to return true when skip=true")
	}
}

func TestConfirmDelete_MessageFormat(t *testing.T) {
	// Capture stderr to verify the prompt message format
	origStdin := os.Stdin
	origStderr := os.Stderr
	defer func() {
		os.Stdin = origStdin
		os.Stderr = origStderr
	}()

	// Create a pipe for stdin (answer "n" to not delete)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}

	// Create a pipe for stderr to capture output
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdin = r
	os.Stderr = stderrW

	go func() {
		defer w.Close()
		_, _ = w.Write([]byte("n\n"))
	}()

	resourceType := "policy"
	identifier := "my-policy-abc123"

	_ = ConfirmDelete(resourceType, identifier, false)

	// Close stderr writer so we can read from it
	stderrW.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(stderrR)
	output := buf.String()

	// Verify the output contains the resource type and identifier
	if !strings.Contains(output, resourceType) {
		t.Errorf("expected stderr to contain resource type %q, got: %s", resourceType, output)
	}
	if !strings.Contains(output, identifier) {
		t.Errorf("expected stderr to contain identifier %q, got: %s", identifier, output)
	}
	if !strings.Contains(output, "delete") {
		t.Errorf("expected stderr to contain 'delete', got: %s", output)
	}

	// Suppress unused import warning
	_ = fmt.Sprintf
}
