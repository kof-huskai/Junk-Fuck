//go:build windows

package platform

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetInfo returns Windows OS version and elevation status.
func GetInfo() Info {
	return Info{
		OS:      "windows",
		Version: osVersion(),
		IsAdmin: isAdmin(),
		Arch:    runtime.GOARCH,
	}
}

// isAdmin checks whether the process token is elevated (UAC admin).
func isAdmin() bool {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false
	}
	defer token.Close()

	var size uint32
	buf := make([]byte, 4) // TOKEN_ELEVATION is a uint32
	if err := windows.GetTokenInformation(token, windows.TokenElevation, &buf[0], uint32(len(buf)), &size); err != nil {
		return false
	}
	return *(*uint32)(unsafe.Pointer(&buf[0])) != 0
}

// osVersionInfoExW mirrors the Windows RTL_OSVERSIONINFOW structure.
// It is declared locally because x/sys/windows does not export it.
type osVersionInfoExW struct {
	OSVersionInfoSize uint32
	MajorVersion      uint32
	MinorVersion      uint32
	BuildNumber       uint32
	PlatformID        uint32
	CSDVersion        [128]uint16
}

var rtlGetVersion = syscall.NewLazyDLL("ntdll.dll").NewProc("RtlGetVersion")

func osVersion() string {
	v := &osVersionInfoExW{OSVersionInfoSize: uint32(unsafe.Sizeof(osVersionInfoExW{}))}
	r1, _, _ := rtlGetVersion.Call(uintptr(unsafe.Pointer(v)))
	if r1 != 0 {
		return "unknown"
	}
	return fmt.Sprintf("%d.%d.%d", v.MajorVersion, v.MinorVersion, v.BuildNumber)
}
