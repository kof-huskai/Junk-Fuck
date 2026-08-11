package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/kof-huskai/Junk-Fuck/internal/whitelist"
)

// RulesStatus is a snapshot of the protection-whitelist engine for the
// Settings UI. It is informational only — it exposes no deletion authority.
type RulesStatus struct {
	SchemaVersion  int    `json:"schemaVersion"`
	ActiveVersion  string `json:"activeVersion"`
	BundledVersion string `json:"bundledVersion"`
	CachedVersion  string `json:"cachedVersion,omitempty"`
	// Source: bundled | cache | remote.
	Source string `json:"source"`
	// Status: up-to-date | error | not-checked.
	Status      string `json:"status"`
	LastChecked string `json:"lastChecked,omitempty"`
	LastUpdated string `json:"lastUpdated,omitempty"`
	RuleCount   int    `json:"ruleCount"`
	Error       string `json:"error,omitempty"`
}

// RulesService exposes the whitelist (protection-only) engine. It is
// distinct from the application updater: rules updates add protection, they
// are never application binary updates.
type RulesService struct {
	core       *Core
	whitelist  *whitelist.Whitelist
	updater    *whitelist.Updater
	appVersion string

	mu     sync.Mutex
	status string
	last   time.Time
}

// NewRulesService wires the whitelist and updater into a service.
func NewRulesService(core *Core) *RulesService {
	w := core.whitelist
	u := whitelist.NewUpdater(core.rulesCacheDir)
	return &RulesService{
		core:      core,
		whitelist: w,
		updater:   u,
		status:    "not-checked",
	}
}

// SetAppVersion stores the build version for compatibility checks against
// remote manifests. Called from main before the background check starts.
func (s *RulesService) SetAppVersion(v string) {
	s.appVersion = v
	s.updater.AppVersion = v
}

// GetRulesStatus returns the current whitelist snapshot.
func (s *RulesService) GetRulesStatus() RulesStatus {
	return s.snapshot()
}

// RefreshRules forces a fresh remote whitelist check (bypassing the TTL)
// and returns the updated snapshot. Any remote failure keeps the last valid
// ruleset active and is reported in the status.
func (s *RulesService) RefreshRules() (RulesStatus, error) {
	s.mu.Lock()
	if s.whitelist == nil {
		s.mu.Unlock()
		return s.snapshot(), fmt.Errorf("rules engine is not initialized")
	}
	s.mu.Unlock()

	s.setStatus("checking")
	err := s.updater.CheckAndUpdate(s.whitelist)
	if err != nil {
		s.setStatus("error")
		return s.snapshot(), err
	}
	s.setStatus("up-to-date")
	return s.snapshot(), nil
}

// StartBackgroundCheck runs one non-blocking TTL-driven whitelist check in
// the background. It never blocks application startup and failures are
// quiet (the last valid ruleset stays active).
func (s *RulesService) StartBackgroundCheck() {
	go func() {
		s.mu.Lock()
		should := s.whitelist != nil && s.updater.NeedsUpdate(s.whitelist)
		s.mu.Unlock()
		if !should {
			return
		}
		s.setStatus("checking")
		err := s.updater.CheckAndUpdate(s.whitelist)
		if err != nil {
			s.setStatus("error")
			return
		}
		s.setStatus("up-to-date")
	}()
}

func (s *RulesService) setStatus(st string) {
	s.mu.Lock()
	s.status = st
	s.last = time.Now()
	s.mu.Unlock()
}

func (s *RulesService) snapshot() RulesStatus {
	s.mu.Lock()
	status, lastChecked := s.status, s.last
	s.mu.Unlock()

	activeVer, bundledVer, cachedVer, source, count, lastUpdated := s.whitelist.Status()
	st := RulesStatus{
		SchemaVersion:  whitelist.SchemaVersion,
		ActiveVersion:  activeVer,
		BundledVersion: bundledVer,
		CachedVersion:  cachedVer,
		Source:         source,
		Status:         status,
		RuleCount:      count,
	}
	if !lastChecked.IsZero() {
		st.LastChecked = lastChecked.Format("2006-01-02 15:04:05")
	}
	if !lastUpdated.IsZero() {
		st.LastUpdated = lastUpdated.Format("2006-01-02 15:04:05")
	}
	if err := s.updater.LastError(); err != nil {
		st.Error = err.Error()
	}
	return st
}
