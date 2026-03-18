package output

import (
	"fmt"
	"os"
)

// IsTTY returns true if stdout is a terminal.
func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Info prints an informational message to stdout.
func Info(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Error prints an error message to stderr.
func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// Progress prints a progress message only if stdout is a TTY.
func Progress(format string, args ...any) {
	if IsTTY() {
		fmt.Fprintf(os.Stdout, format, args...)
	}
}
