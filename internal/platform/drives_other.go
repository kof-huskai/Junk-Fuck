//go:build !windows

package platform

import "github.com/kof-huskai/Junk-Fuck/internal/model"

// ListDrives is only meaningful on Windows. Other platforms report none.
func ListDrives() []model.DriveInfo {
	return nil
}
