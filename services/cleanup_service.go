package services

import (
	"fmt"

	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/report"
)

// CleanupService handles deletion of scan candidates. It never accepts an
// arbitrary path: every selection is validated against the scan session
// snapshot by the cleaner (re-stat, protection, drive-root, reparse-point,
// identity checks) before anything is removed.
type CleanupService struct {
	core *Core
}

// Cleanup validates and deletes the selected paths against a scan session.
// dryRun performs the full validation without touching the filesystem.
func (s *CleanupService) Cleanup(dryRun bool, scanID string, selected []string) (report.Report, error) {
	s.core.mu.Lock()
	ss, ok := s.core.sessions[scanID]
	s.core.mu.Unlock()
	if !ok {
		return report.Report{}, fmt.Errorf("unknown scan id %q", scanID)
	}

	rep := s.core.cleaner.Clean(dryRun, &model.ScanSession{
		ID:         scanID,
		Targets:    ss.targets,
		Candidates: ss.candidates,
		Done:       ss.status.Done,
	}, selected)

	s.core.mu.Lock()
	s.core.lastReport = &lastReportEntry{Report: rep, At: timestamp()}
	s.core.mu.Unlock()
	return rep, nil
}

// GetLastReport returns the most recent cleanup report (history page).
func (s *CleanupService) GetLastReport() (*lastReportEntry, error) {
	s.core.mu.Lock()
	defer s.core.mu.Unlock()
	if s.core.lastReport == nil {
		return nil, fmt.Errorf("no cleanup has been run yet")
	}
	return s.core.lastReport, nil
}
