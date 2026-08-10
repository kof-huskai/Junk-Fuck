// Package filesystem provides low-level, testable filesystem helpers used
// by the scanner and the cleaner.
package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var driveRootRe = regexp.MustCompile(`^[A-Za-z]:[\\/]*$`)

// Canonical returns the absolute, cleaned form of a path.
func Canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// IsDriveRoot reports whether path is a filesystem root such as C:\.
func IsDriveRoot(path string) bool {
	canon, err := Canonical(path)
	if err != nil {
		return false
	}
	return driveRootRe.MatchString(canon)
}

// CompareKey returns the canonical comparison key for a path
// (absolute, cleaned, lowercased on Windows).
func CompareKey(path string) string {
	canon, err := Canonical(path)
	if err != nil {
		canon = filepath.Clean(path)
	}
	if runtime.GOOS == "windows" {
		return strings.ToLower(canon)
	}
	return canon
}

// SameOrUnder reports whether path is equal to or inside base.
func SameOrUnder(path, base string) bool {
	a, b := CompareKey(path), CompareKey(base)
	if a == b {
		return true
	}
	return strings.HasPrefix(a, b+string(filepath.Separator))
}

// IsReadOnly reports whether the file info has no writable bits.
func IsReadOnly(info fs.FileInfo) bool {
	return info.Mode().Perm()&0200 == 0
}

// MakeWritable clears the read-only attribute so deletion can proceed.
func MakeWritable(path string) error {
	return os.Chmod(path, 0o644)
}
