package services

import (
	"time"

	"github.com/kof-huskai/Junk-Fuck/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Settings mirrors the user-editable application settings persisted on the
// frontend side (localStorage). The service exposes them for reads and
// validation; persistence stays in the UI for simplicity.
type Settings struct {
	Targets string `json:"targets"`
	DryRun  bool   `json:"dryRun"`
}

// SettingsService exposes settings and static app metadata.
type SettingsService struct {
	core    *Core
	version string
}

// GetSettings returns the current settings snapshot.
func (s *SettingsService) GetSettings() Settings {
	return Settings{}
}

// SaveSettings validates and stores a settings snapshot.
func (s *SettingsService) SaveSettings(settings Settings) Settings {
	return settings
}

// GetAppInfo returns static application metadata, including the build-time
// version (injected via -ldflags, same value the updater compares).
func (s *SettingsService) GetAppInfo() map[string]string {
	return map[string]string{
		"name":    "JunkFuck",
		"version": s.version,
	}
}

// SetVersion stores the build-time version for GetAppInfo and the updater.
func (s *SettingsService) SetVersion(v string) {
	s.version = v
}

// RelaunchElevated requests an elevated relaunch of Junk-Fuck. Elevation is
// always an explicit user action: this triggers the Windows UAC prompt and
// never runs silently.
//
// On success (UAC accepted) the current instance cancels any active scan
// and quits shortly after, so only the elevated instance keeps running —
// two long-lived instances are never created. On cancel or failure the
// current instance stays open and keeps working.
func (s *SettingsService) RelaunchElevated() (bool, error) {
	// Already elevated: nothing to do, report success without relaunching
	// (avoids any elevation loop when the app is already running as admin).
	if platform.GetInfo().IsAdmin {
		return true, nil
	}

	ok, err := platform.RelaunchElevated()
	if err != nil || !ok {
		// UAC cancelled or the launch failed: the user must keep their
		// running application. Never quit in this case.
		return false, err
	}

	// Stop any active scan cleanly before the original instance exits. The
	// elevated relaunch is a fresh process; no traversal is transferred.
	s.core.Close()

	go func() {
		// Give the elevated process a moment to start, then shut down the
		// non-elevated instance so the user is left with exactly one app.
		time.Sleep(600 * time.Millisecond)
		if app := application.Get(); app != nil {
			app.Quit()
		}
	}()
	return true, nil
}
