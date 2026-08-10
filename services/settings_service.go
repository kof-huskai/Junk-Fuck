package services

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
