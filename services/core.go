// Package services is the Wails v3 application layer. It adapts the pure Go
// Core (internal/) to the desktop: services expose controlled, typed methods
// to React and emit progress events. No filesystem-safety logic lives here.
package services

import (
	"context"
	"sync"

	"github.com/kof-huskai/Junk-Fuck/internal/classifier"
	"github.com/kof-huskai/Junk-Fuck/internal/cleaner"
	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/protection"
	"github.com/kof-huskai/Junk-Fuck/internal/report"
	"github.com/kof-huskai/Junk-Fuck/internal/scanner"
)

// Core owns the shared state of the application layer: scan sessions,
// cancellation handles, the protection ruleset and the last cleanup report.
type Core struct {
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

// NewCore builds the application layer with production protection rules.
func NewCore() *Core {
	pr := protection.FromOS()
	return &Core{
		sessions:   map[string]*session{},
		cancelFns:  map[string]context.CancelFunc{},
		protection: pr,
		scanner:    scanner.New(classifier.New(), pr),
		cleaner:    cleaner.New(pr),
	}
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

// Close cancels any in-flight scans.
func (c *Core) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, cancel := range c.cancelFns {
		cancel()
	}
	c.cancelFns = map[string]context.CancelFunc{}
}
