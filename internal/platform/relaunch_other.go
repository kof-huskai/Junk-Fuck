//go:build !windows

package platform

import "errors"

// RelaunchElevated is not supported off Windows: Junk-Fuck is a Windows
// utility and the UAC elevation flow is Windows-specific.
func RelaunchElevated() (bool, error) {
	return false, errors.New("elevated relaunch is only supported on Windows")
}
