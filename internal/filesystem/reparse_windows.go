//go:build windows

package filesystem

import "golang.org/x/sys/windows"

// IsReparsePoint reports whether path has the FILE_ATTRIBUTE_REPARSE_POINT
// attribute (symlinks, junctions, mounted volumes). The cleaner refuses to
// delete such entries (safety SR-11).
func IsReparsePoint(path string) (bool, error) {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	attrs, err := windows.GetFileAttributes(ptr)
	if err != nil {
		return false, err
	}
	return attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}
