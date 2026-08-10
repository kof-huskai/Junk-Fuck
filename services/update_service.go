package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

// UpdateState is a snapshot of the updater exposed to the frontend.
type UpdateState struct {
	State           string  `json:"state"`
	CurrentVersion  string  `json:"currentVersion"`
	Available       string  `json:"available,omitempty"`
	ReleaseNotes    string  `json:"releaseNotes,omitempty"`
	ReleaseURL      string  `json:"releaseUrl,omitempty"`
	LastChecked     string  `json:"lastChecked,omitempty"`
	ProgressPercent float64 `json:"progressPercent,omitempty"`
}

// UpdateService drives the official Wails v3 updater. Update installation is
// refused while a scan or cleanup is in progress (installation restarts the
// app, which must never interrupt filesystem work).
type UpdateService struct {
	core  *Core
	mu    sync.Mutex
	state UpdateState
	up    *updater.Updater
	busy  bool
	last  time.Time
}

// SetUpdater wires the app's official updater instance (called from main).
func (s *UpdateService) SetUpdater(u *updater.Updater) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.up = u
	s.state.CurrentVersion = u.CurrentVersion()
}

// CheckForUpdates runs a manual update check. It never installs.
func (s *UpdateService) CheckForUpdates() (UpdateState, error) {
	s.mu.Lock()
	if s.up == nil {
		s.mu.Unlock()
		return s.state, errors.New("updater is not configured")
	}
	if s.busy {
		s.mu.Unlock()
		return s.state, errors.New("update operation already in progress")
	}
	s.busy = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.busy = false
		s.last = time.Now()
		s.state.LastChecked = s.last.Format("2006-01-02 15:04:05")
		s.mu.Unlock()
	}()

	rel, err := s.up.Check(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.state.State = "error"
		return s.state, err
	}
	if rel == nil {
		s.state.State = "up-to-date"
		s.state.Available = ""
		s.state.ReleaseNotes = ""
		return s.state, nil
	}
	s.state.State = "available"
	s.state.Available = rel.Version
	s.state.ReleaseNotes = rel.Notes
	return s.state, nil
}

// InstallUpdate downloads, verifies and installs the available update.
// Refused while a scan or cleanup is running.
func (s *UpdateService) InstallUpdate() (UpdateState, error) {
	s.mu.Lock()
	if s.up == nil {
		s.mu.Unlock()
		return s.state, errors.New("updater is not configured")
	}
	if s.busy {
		s.mu.Unlock()
		return s.state, errors.New("update operation already in progress")
	}
	if s.coreActive() {
		s.mu.Unlock()
		return s.state, errors.New("a scan or cleanup is running; finish it before updating")
	}
	s.busy = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.busy = false
		s.mu.Unlock()
	}()

	err := s.up.DownloadAndInstall(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.state.State = "error"
		return s.state, err
	}
	s.state.State = "installed"
	return s.state, nil
}

// GetUpdateState returns the current updater snapshot.
func (s *UpdateService) GetUpdateState() UpdateState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// coreActive reports whether a scan session is running or in flight.
func (s *UpdateService) coreActive() bool {
	s.core.mu.Lock()
	defer s.core.mu.Unlock()
	for _, cancel := range s.core.cancelFns {
		if cancel != nil {
			return true
		}
	}
	return false
}
