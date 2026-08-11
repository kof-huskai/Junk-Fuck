//go:build windows

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var shellExecute = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")

// RelaunchElevated relaunches the current executable with the "runas" verb,
// which always shows the Windows UAC prompt (elevation is an explicit user
// action; it never happens silently).
//
// Return values:
//   - ok=true  — the elevated process was launched; the caller may now shut
//     down this (non-elevated) instance.
//   - ok=false — the user cancelled the UAC prompt; the current instance
//     must keep running.
//   - err      — the launch failed for a real reason (e.g. the executable
//     path is unresolvable); the current instance must keep running.
func RelaunchElevated() (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("cannot resolve the executable path: %w", err)
	}
	exePath, err := filepath.Abs(exe)
	if err != nil {
		return false, fmt.Errorf("cannot resolve the executable path: %w", err)
	}

	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return false, fmt.Errorf("cannot build the elevation request: %w", err)
	}
	file, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return false, fmt.Errorf("cannot build the elevation request: %w", err)
	}

	// ShellExecuteW(hwnd, verb, file, args, cwd, showCmd). A return value
	// > 32 means success; <= 32 means failure (ERROR_CANCELLED = 1223 when
	// the user declines the UAC prompt).
	const swShownormal = 1
	r1, _, _ := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0, // no arguments
		0, // inherit the working directory
		swShownormal,
	)
	if int32(r1) <= 32 {
		return false, fmt.Errorf("elevation failed (ShellExecute returned %d)", int32(r1))
	}
	return true, nil
}
