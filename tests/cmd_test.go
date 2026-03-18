package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/null3000/cws-cli/cmd"
)

// --- FormatState tests ---

func TestFormatState(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"PUBLISHED", "Published"},
		{"PENDING_REVIEW", "Pending Review"},
		{"DRAFT", "Draft"},
		{"DEFERRED", "Staged (Deferred)"},
		{"STATE_UNSPECIFIED", "Unknown"},
		{"", ""},
		{"SOME_NEW_STATE", "SOME_NEW_STATE"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cmd.FormatState(tt.input)
			if got != tt.want {
				t.Errorf("FormatState(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Version tests ---

func TestVersionOutput(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.Version = "1.2.3"
	cmd.VersionCmd.Run(cmd.VersionCmd, []string{})

	w.Close()
	os.Stdout = origStdout

	var buf [256]byte
	n, _ := r.Read(buf[:])
	got := string(buf[:n])

	if !strings.Contains(got, "cws 1.2.3") {
		t.Errorf("version output = %q, want to contain %q", got, "cws 1.2.3")
	}
}
