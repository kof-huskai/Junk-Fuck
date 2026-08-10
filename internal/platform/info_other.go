//go:build !windows

package platform

import "runtime"

// GetInfo returns basic host info on non-Windows platforms.
func GetInfo() Info {
	return Info{OS: runtime.GOOS, Version: "n/a", IsAdmin: false, Arch: runtime.GOARCH}
}
