// Package services is the Wails v3 application layer. It adapts the pure Go
// Core (internal/) to the desktop: services expose controlled, typed methods
// to React and emit progress events. No filesystem-safety logic lives here.
package services

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kof-huskai/Junk-Fuck/internal/classifier"
	"github.com/kof-huskai/Junk-Fuck/internal/cleaner"
	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/protection"
	"github.com/kof-huskai/Junk-Fuck/internal/report"
	"github.com/kof-huskai/Junk-Fuck/internal/scanner"
	"github.com/kof-huskai/Junk-Fuck/internal/whitelist"
)

// Core owns the shared state of the application layer: scan sessions,
// cancellation handles, the protection ruleset, the whitelist rules engine,
// the last cleanup report and the canonical last-scan summary.
type Core struct {
	mu            sync.Mutex
	sessions      map[string]*session
	cancelFns     map[string]context.CancelFunc
	protection    *protection.Protection
	scanner       *scanner.Scanner
	cleaner       *cleaner.Cleaner
	lastReport    *lastReportEntry
	lastScan      *model.ScanSummary // canonical last successful scan (Dashboard)
	lastScanFile  string             // persistence path; empty disables persistence (tests)
	whitelist     *whitelist.Whitelist
	rulesCacheDir string
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

// NewCore builds the application layer with production protection rules.
// The bundled whitelist is loaded best-effort (a failure only means fewer
// dynamic protections; the hard-coded safety layer never depends on it),
// and a previously validated cached whitelist is preferred when newer.
func NewCore() *Core {
	pr := protection.FromOS()
	wl := whitelist.NewWhitelist()
	cacheDir := rulesCacheDir()
	_ = wl.LoadBundled()
	_ = wl.LoadCache(cacheDir)
	pr.SetDynamic(wl)
	c := &Core{
		sessions:      map[string]*session{},
		cancelFns:     map[string]context.CancelFunc{},
		protection:    pr,
		scanner:       scanner.New(classifier.New(), pr),
		cleaner:       cleaner.New(pr),
		whitelist:     wl,
		rulesCacheDir: cacheDir,
	}
	c.lastScanFile = lastScanPath()
	c.loadLastScan()
	return c
}

// rulesCacheDir returns the per-user cache directory for the validated
// whitelist cache (%LOCALAPPDATA%\JunkFuck\rules on Windows).
func rulesCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "JunkFuck", "rules")
}

// lastScanPath returns the per-user file where the canonical last-scan
// summary is persisted (%LOCALAPPDATA%\JunkFuck\last-scan.json on Windows).
func lastScanPath() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "JunkFuck", "last-scan.json")
}

// recordScanResult persists the canonical last-scan summary when a scan
// reaches a real successful terminal state. Cancelled or failed scans never
// overwrite the previous successful summary; a newer successful scan
// replaces the older one. Permission errors encountered during an otherwise
// completed scan are ordinary partial results: they are counted in the
// summary (ErrorCount), not treated as scan failure.
func (c *Core) recordScanResult(targets []string, res model.Result, failed bool) {
	if res.Cancelled || failed {
		return
	}
	var reclaimable int64
	for _, cand := range res.Candidates {
		reclaimable += cand.Size
	}
	summary := &model.ScanSummary{
		CompletedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		Target:           strings.Join(targets, ", "),
		FilesScanned:     res.ScannedFiles,
		JunkItems:        len(res.Candidates),
		ReclaimableBytes: reclaimable,
		ErrorCount:       len(res.Errors),
	}
	c.mu.Lock()
	c.lastScan = summary
	c.mu.Unlock()
	if err := c.persistLastScan(summary); err != nil {
		// The in-session summary is already active; a failed persist only
		// means it won't survive a restart. Log it rather than fail the
		// scan silently (the Dashboard reads the same in-memory value).
		log.Printf("last-scan: failed to persist summary: %v", err)
	}
}

// persistLastScan writes the summary atomically (temp file + rename) so a
// crash mid-write can never corrupt the stored record. The parent directory
// is created on demand: on a fresh install %LOCALAPPDATA%\JunkFuck may not
// exist yet (the rules cache is only written after a successful remote
// rules update).
func (c *Core) persistLastScan(summary *model.ScanSummary) error {
	if c.lastScanFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.lastScanFile), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.lastScanFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.lastScanFile)
}

// loadLastScan reads a previously persisted summary at startup (best-effort:
// a missing or unreadable file simply means "no successful scan yet").
func (c *Core) loadLastScan() {
	if c.lastScanFile == "" {
		return
	}
	data, err := os.ReadFile(c.lastScanFile)
	if err != nil {
		return
	}
	var summary model.ScanSummary
	if err := json.Unmarshal(data, &summary); err != nil || summary.CompletedAt == "" {
		return
	}
	c.mu.Lock()
	c.lastScan = &summary
	c.mu.Unlock()
}

// ScannerService returns a new scanner service bound to this core.
func (c *Core) ScannerService() *ScannerService {
	return &ScannerService{core: c}
}

// CleanupService returns a new cleanup service bound to this core.
func (c *Core) CleanupService() *CleanupService {
	return &CleanupService{core: c}
}

// SettingsService returns a new settings service bound to this core.
func (c *Core) SettingsService() *SettingsService {
	return &SettingsService{core: c, version: "dev"}
}

// UpdateService returns a new update service bound to this core.
func (c *Core) UpdateService() *UpdateService {
	return &UpdateService{core: c}
}

// RulesService returns a new protection-rules service bound to this core.
func (c *Core) RulesService() *RulesService {
	return NewRulesService(c)
}

// Close cancels any in-flight scans.
func (c *Core) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cancel := range c.cancelFns {
		cancel()
	}
	c.cancelFns = map[string]context.CancelFunc{}
}
