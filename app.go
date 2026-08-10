package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/kof-huskai/Junk-Fuck/internal/classifier"
	"github.com/kof-huskai/Junk-Fuck/internal/cleaner"
	"github.com/kof-huskai/Junk-Fuck/internal/filesystem"
	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/platform"
	"github.com/kof-huskai/Junk-Fuck/internal/protection"
	"github.com/kof-huskai/Junk-Fuck/internal/report"
	"github.com/kof-huskai/Junk-Fuck/internal/scanner"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version is the application version; overridable at build time with
// -ldflags "-X main.Version=...".
var Version = "4.0.0"

// App is the Wails desktop layer. It owns no filesystem logic itself: it
// only wires the Core packages to the UI and keeps scan sessions in memory.
type App struct {
	ctx        context.Context
	mu         sync.Mutex
	sessions   map[string]*session
	cancelFns  map[string]context.CancelFunc
	protection *protection.Protection
	scanner    *scanner.Scanner
	cleaner    *cleaner.Cleaner
	lastReport *lastReportEntry
}

type session struct {
	candidates map[string]model.Candidate
	errors     []model.ScanError
	status     model.Progress
	targets    []string
}

type lastReportEntry struct {
	Report report.Report `json:"report"`
	At     string        `json:"at"`
}

// NewApp builds the desktop layer with production protection rules.
func NewApp() *App {
	pr := protection.FromOS()
	return &App{
		sessions:   map[string]*session{},
		cancelFns:  map[string]context.CancelFunc{},
		protection: pr,
		scanner:    scanner.New(classifier.New(), pr),
		cleaner:    cleaner.New(pr),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, cancel := range a.cancelFns {
		cancel()
	}
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// StartScan launches an asynchronous scan and returns its scan id. Progress
// is delivered through the "scan:progress" event; completion through
// "scan:done".
func (a *App) StartScan(targets []string) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("no scan targets provided")
	}
	scanID := newID()
	ctx, cancel := context.WithCancel(context.Background())

	a.mu.Lock()
	a.sessions[scanID] = &session{
		candidates: map[string]model.Candidate{},
		targets:    targets,
		status:     model.Progress{ScanID: scanID},
	}
	a.cancelFns[scanID] = cancel
	a.mu.Unlock()

	go func() {
		res := a.scanner.Scan(ctx, scanID, targets, func(p model.Progress) {
			a.mu.Lock()
			if s, ok := a.sessions[scanID]; ok {
				s.status = p
			}
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "scan:progress", p)
		})
		cancel()

		a.mu.Lock()
		if s, ok := a.sessions[scanID]; ok {
			for _, c := range res.Candidates {
				s.candidates[filesystem.CompareKey(c.Path)] = c
			}
			s.errors = res.Errors
			s.status = model.Progress{ScanID: scanID, ScannedFiles: res.ScannedFiles, Candidates: int64(len(res.Candidates)), Errors: int64(len(res.Errors)), Done: true}
		}
		delete(a.cancelFns, scanID)
		a.mu.Unlock()

		runtime.EventsEmit(a.ctx, "scan:done", map[string]any{
			"scanId":    scanID,
			"cancelled": res.Cancelled,
			"error":     "",
		})
	}()

	return scanID, nil
}

// CancelScan stops an in-progress scan by id.
func (a *App) CancelScan(scanID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if cancel, ok := a.cancelFns[scanID]; ok {
		cancel()
	}
}

// GetScanState returns the current progress snapshot for a scan.
func (a *App) GetScanState(scanID string) (model.Progress, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.sessions[scanID]; ok {
		return s.status, nil
	}
	return model.Progress{}, fmt.Errorf("unknown scan id %q", scanID)
}

// GetCandidates returns the scan's candidates sorted by size (descending),
// with protected candidates first flagged for the UI.
func (a *App) GetCandidates(scanID string) ([]model.Candidate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sessions[scanID]
	if !ok {
		return nil, fmt.Errorf("unknown scan id %q", scanID)
	}
	out := make([]model.Candidate, 0, len(s.candidates))
	for _, c := range s.candidates {
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

// Cleanup validates and deletes the selected paths against a scan session.
// dryRun performs the full validation without touching the filesystem.
func (a *App) Cleanup(dryRun bool, scanID string, selected []string) (report.Report, error) {
	a.mu.Lock()
	s, ok := a.sessions[scanID]
	a.mu.Unlock()
	if !ok {
		return report.Report{}, fmt.Errorf("unknown scan id %q", scanID)
	}

	rep := a.cleaner.Clean(dryRun, &model.ScanSession{
		ID:         scanID,
		Targets:    s.targets,
		Candidates: s.candidates,
		Done:       s.status.Done,
	}, selected)

	a.mu.Lock()
	a.lastReport = &lastReportEntry{Report: rep, At: timestamp()}
	a.mu.Unlock()
	return rep, nil
}

// GetLastReport returns the most recent cleanup report (history page).
func (a *App) GetLastReport() (*lastReportEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lastReport == nil {
		return nil, fmt.Errorf("no cleanup has been run yet")
	}
	return a.lastReport, nil
}

// GetScanErrors returns the per-path errors recorded during a scan.
func (a *App) GetScanErrors(scanID string) ([]model.ScanError, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sessions[scanID]
	if !ok {
		return nil, fmt.Errorf("unknown scan id %q", scanID)
	}
	return s.errors, nil
}

// GetProtectedPaths returns the canonical protected roots (informational).
func (a *App) GetProtectedPaths() []string {
	return a.protection.List()
}

// GetSystemInfo reports OS version and elevation status.
func (a *App) GetSystemInfo() platform.Info {
	return platform.GetInfo()
}

// GetAppInfo returns static application metadata.
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{"name": "JunkFuck", "version": Version}
}

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
