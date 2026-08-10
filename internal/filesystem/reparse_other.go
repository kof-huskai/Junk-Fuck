//go:build !windows

package filesystem

import "os"

// IsReparsePoint reports whether path is a symbolic link on non-Windows
// platforms. The cleaner refuses to delete such entries.
func IsReparsePoint(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}
