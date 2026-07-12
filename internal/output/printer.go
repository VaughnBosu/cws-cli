package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// jsonMode suppresses human-oriented output so commands can emit a single
// machine-readable JSON object instead.
var jsonMode bool

// SetJSONMode toggles machine-readable output.
func SetJSONMode(on bool) {
	jsonMode = on
}

// JSONMode reports whether machine-readable output is enabled.
func JSONMode() bool {
	return jsonMode
}

// EmitJSON prints v as indented JSON to stdout.
func EmitJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// IsTTY returns true if stdout is a terminal.
func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Info prints an informational message to stdout (suppressed in JSON mode).
func Info(format string, args ...any) {
	if jsonMode {
		return
	}
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Warn prints a warning message to stderr.
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Warning: "+format+"\n", args...)
}

// Error prints an error message to stderr.
func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// Hint prints an actionable hint to stderr, indented under the preceding error.
func Hint(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "  Hint: "+format+"\n", args...)
}

// Progress prints a progress message only if stdout is a TTY (and not JSON mode).
func Progress(format string, args ...any) {
	if jsonMode {
		return
	}
	if IsTTY() {
		fmt.Fprintf(os.Stdout, format, args...)
	}
}
