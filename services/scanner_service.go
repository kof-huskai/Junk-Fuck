package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/kof-huskai/Junk-Fuck/internal/filesystem"
	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ScannerService exposes scan control to the frontend. Scans run in a
// goroutine and report progress through the "scan:progress" / "scan:done"
// events so the UI never blocks.
type ScannerService struct {
	core *Core
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// StartScan launches an asynchronous scan and returns its scan id.
func (s *ScannerService) StartScan(targets []string) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("no scan targets provided")
	}
	scanID := newID()
	ctx, cancel := context.WithCancel(context.Background())

	s.core.mu.Lock()
	s.core.sessions[scanID] = &session{
		candidates: map[string]model.Candidate{},
		targets:    targets,
		status:     model.Progress{ScanID: scanID},
	}
	s.core.cancelFns[scanID] = cancel
	s.core.mu.Unlock()

	app := application.Get()

	go func() {
		res := s.core.scanner.Scan(ctx, scanID, targets, func(p model.Progress) {
			s.core.mu.Lock()
			if s, ok := s.core.sessions[scanID]; ok {
				s.status = p
			}
			s.core.mu.Unlock()
			app.Event.Emit("scan:progress", p)
		})
		cancel()

		s.core.mu.Lock()
		if s, ok := s.core.sessions[scanID]; ok {
			for _, c := range res.Candidates {
				s.candidates[filesystem.CompareKey(c.Path)] = c
			}
			s.errors = res.Errors
			s.status = model.Progress{ScanID: scanID, ScannedFiles: res.ScannedFiles, Candidates: int64(len(res.Candidates)), Errors: int64(len(res.Errors)), Done: true}
		}
		delete(s.core.cancelFns, scanID)
		s.core.mu.Unlock()

		app.Event.Emit("scan:done", map[string]any{
			"scanId":    scanID,
			"cancelled": res.Cancelled,
			"error":     "",
		})
	}()

	return scanID, nil
}

// CancelScan stops an in-progress scan by id.
func (s *ScannerService) CancelScan(scanID string) {
	s.core.mu.Lock()
	defer s.core.mu.Unlock()
	if cancel, ok := s.core.cancelFns[scanID]; ok {
		cancel()
	}
}

// GetScanState returns the current progress snapshot for a scan.
func (s *ScannerService) GetScanState(scanID string) (model.Progress, error) {
	s.core.mu.Lock()
	defer s.core.mu.Unlock()
	if ss, ok := s.core.sessions[scanID]; ok {
		return ss.status, nil
	}
	return model.Progress{}, fmt.Errorf("unknown scan id %q", scanID)
}

// GetCandidates returns the scan's candidates sorted by size (descending),
// with directories first.
func (s *ScannerService) GetCandidates(scanID string) ([]model.Candidate, error) {
	s.core.mu.Lock()
	defer s.core.mu.Unlock()
	ss, ok := s.core.sessions[scanID]
	if !ok {
		return nil, fmt.Errorf("unknown scan id %q", scanID)
	}
	out := make([]model.Candidate, 0, len(ss.candidates))
	for _, c := range ss.candidates {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Size > out[j].Size
	})
	return out, nil
}

// GetScanErrors returns the per-path errors recorded during a scan.
func (s *ScannerService) GetScanErrors(scanID string) ([]model.ScanError, error) {
	s.core.mu.Lock()
	defer s.core.mu.Unlock()
	ss, ok := s.core.sessions[scanID]
	if !ok {
		return nil, fmt.Errorf("unknown scan id %q", scanID)
	}
	return ss.errors, nil
}

// GetProtectedPaths returns the canonical protected roots (informational).
func (s *ScannerService) GetProtectedPaths() []string {
	return s.core.protection.List()
}

// GetSystemInfo reports OS version and elevation status.
func (s *ScannerService) GetSystemInfo() platform.Info {
	return platform.GetInfo()
}

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
