//go:build !windows

package privatefile

import "os"

func replaceFile(source, target string) error {
	return os.Rename(source, target)
}
