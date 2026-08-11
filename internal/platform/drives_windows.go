//go:build windows

package platform

import (
	"sort"
	"strings"

	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"golang.org/x/sys/windows"
)

// driveTypeString maps Win32 drive types to stable, frontend-friendly ids.
func driveTypeString(t uint32) string {
	switch t {
	case windows.DRIVE_FIXED:
		return "fixed"
	case windows.DRIVE_REMOVABLE:
		return "removable"
	case windows.DRIVE_REMOTE:
		return "network"
	case windows.DRIVE_CDROM:
		return "optical"
	case windows.DRIVE_RAMDISK:
		return "ram"
	default:
		return "unknown"
	}
}

// driveRank orders drives for display: the system drive first, then fixed
// disks, removable media, and finally network/optical/unknown drives.
func driveRank(root, driveType, sysRoot string) int {
	if strings.EqualFold(root, sysRoot) {
		return 0
	}
	switch driveType {
	case "fixed":
		return 1
	case "removable":
		return 2
	case "ram":
		return 3
	case "network":
		return 4
	case "optical":
		return 5
	default:
		return 6
	}
}

// ListDrives enumerates the currently available Windows logical drives using
// the native Win32 API (no hardcoded letters). Fixed/local disks come first,
// with the system drive (where Windows is installed) first among them.
func ListDrives() []model.DriveInfo {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}

	// Locate the system drive (not assumed to be C:) via GetWindowsDirectory.
	sysRoot := ""
	if dir, err := windows.GetWindowsDirectory(); err == nil && len(dir) >= 3 && dir[1] == ':' {
		sysRoot = strings.ToUpper(dir[:2]) + `\`
	}

	var drives []model.DriveInfo
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		driveType := driveTypeString(windows.GetDriveType(utf16Ptr(root)))
		di := model.DriveInfo{
			Root:  root,
			Label: volumeLabel(root),
			Type:  driveType,
		}
		// A drive is "ready" when free-space enumeration succeeds (empty
		// optical drives, ejected USB media and disconnected network drives
		// all fail here and are reported as not ready).
		if driveType != "unknown" && driveType != "optical" {
			di.Ready = queryFreeSpace(root, &di)
		}
		drives = append(drives, di)
	}

	sort.SliceStable(drives, func(i, j int) bool {
		return driveRank(drives[i].Root, drives[i].Type, sysRoot) < driveRank(drives[j].Root, drives[j].Type, sysRoot)
	})
	return drives
}

func utf16Ptr(s string) *uint16 {
	p, _ := windows.UTF16PtrFromString(s)
	return p
}

func volumeLabel(root string) string {
	buf := make([]uint16, 261)
	err := windows.GetVolumeInformation(utf16Ptr(root), &buf[0], uint32(len(buf)), nil, nil, nil, nil, 0)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(windows.UTF16ToString(buf))
}

func queryFreeSpace(root string, di *model.DriveInfo) bool {
	var free, total uint64
	if err := windows.GetDiskFreeSpaceEx(utf16Ptr(root), &free, &total, nil); err != nil {
		return false
	}
	di.FreeBytes = int64(free)
	di.TotalBytes = int64(total)
	return true
}
