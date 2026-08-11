package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kof-huskai/Junk-Fuck/internal/model"
)

// newTestCore builds a Core with an isolated temp last-scan file so the
// tests are hermetic (never touch the real %LOCALAPPDATA% path).
func newTestCore(t *testing.T) *Core {
	t.Helper()
	return &Core{
		sessions:     map[string]*session{},
		cancelFns:    map[string]context.CancelFunc{},
		lastScanFile: filepath.Join(t.TempDir(), "last-scan.json"),
	}
}

// sampleResult builds a scan result with n junk candidates of known sizes.
func sampleResult(n int) model.Result {
	res := model.Result{ScannedFiles: 1000}
	for i := 0; i < n; i++ {
		res.Candidates = append(res.Candidates, model.Candidate{
			Path: "C:\\tmp\\junk-" + string(rune('a'+i)),
			Size: int64(100 + i*10),
		})
	}
	return res
}

func reclaimable(res model.Result) int64 {
	var sum int64
	for _, c := range res.Candidates {
		sum += c.Size
	}
	return sum
}

func lastScan(t *testing.T, c *Core) *model.ScanSummary {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastScan
}

// A successful scan records the canonical summary with a real timestamp.
func TestRecordScanSummarySuccess(t *testing.T) {
	c := newTestCore(t)
	res := sampleResult(3)
	c.recordScanResult([]string{`C:\`}, res, false)

	s := lastScan(t, c)
	if s == nil {
		t.Fatal("successful scan must record a last-scan summary")
	}
	at, err := time.Parse(time.RFC3339, s.CompletedAt)
	if err != nil {
		t.Fatalf("CompletedAt is not a parseable RFC3339 timestamp: %v", err)
	}
	if time.Since(at) > 5*time.Second || time.Since(at) < 0 {
		t.Fatalf("CompletedAt must be the completion moment, got %s", s.CompletedAt)
	}
	if s.Target != `C:\` {
		t.Errorf("target = %q, want C:\\", s.Target)
	}
	if s.JunkItems != 3 || s.ReclaimableBytes != reclaimable(res) {
		t.Errorf("summary counts mismatch: items=%d bytes=%d", s.JunkItems, s.ReclaimableBytes)
	}
	if s.FilesScanned != 1000 || s.ErrorCount != 0 {
		t.Errorf("files=%d errors=%d", s.FilesScanned, s.ErrorCount)
	}
}

// A successful scan with zero junk still records the last scan (items 0).
func TestRecordScanSummaryZeroJunk(t *testing.T) {
	c := newTestCore(t)
	c.recordScanResult([]string{`C:\`}, sampleResult(0), false)

	s := lastScan(t, c)
	if s == nil {
		t.Fatal("successful zero-result scan must still record a last-scan summary")
	}
	if s.JunkItems != 0 || s.ReclaimableBytes != 0 {
		t.Errorf("zero-junk scan must record 0 items/bytes, got %d/%d", s.JunkItems, s.ReclaimableBytes)
	}
	if s.CompletedAt == "" {
		t.Error("zero-junk scan must record a timestamp")
	}
}

// A cancelled scan never overwrites the previous successful summary.
func TestRecordScanSummaryCancelledPreservesPrevious(t *testing.T) {
	c := newTestCore(t)
	c.recordScanResult([]string{`C:\`}, sampleResult(2), false)
	before := lastScan(t, c)

	// Simulate cancellation the way the scanner reports it: the result is
	// partial (9 candidates) but marked cancelled — it must be ignored.
	res := sampleResult(9)
	res.Cancelled = true
	c.recordScanResult([]string{`C:\`}, res, false)

	s := lastScan(t, c)
	if s == nil {
		t.Fatal("previous summary must be preserved")
	}
	if s.CompletedAt != before.CompletedAt {
		t.Errorf("cancelled scan overwrote the previous summary (was %s, got %s)", before.CompletedAt, s.CompletedAt)
	}
	if s.JunkItems != 2 {
		t.Errorf("cancelled scan changed the item count, got %d", s.JunkItems)
	}
}

// A failed scan never overwrites the previous successful summary.
func TestRecordScanSummaryFailedPreservesPrevious(t *testing.T) {
	c := newTestCore(t)
	c.recordScanResult([]string{`C:\`}, sampleResult(2), false)
	before := lastScan(t, c)

	c.recordScanResult([]string{`C:\`}, sampleResult(5), true) // failed terminal state

	s := lastScan(t, c)
	if s == nil || s.CompletedAt != before.CompletedAt {
		t.Fatal("failed scan must preserve the previous successful summary")
	}
	if s.JunkItems != 2 {
		t.Errorf("failed scan changed the item count, got %d", s.JunkItems)
	}
}

// A NEWER successful scan replaces the previous summary (latest wins); the
// timestamp is nanosecond-precision so successive scans never collide.
func TestRecordScanSummaryNewerScanReplaces(t *testing.T) {
	c := newTestCore(t)
	c.recordScanResult([]string{`C:\`}, sampleResult(1), false)
	first := lastScan(t, c)

	c.recordScanResult([]string{`D:\`}, sampleResult(4), false)
	s := lastScan(t, c)
	if s == nil {
		t.Fatal("expected a summary after the second scan")
	}
	if s.CompletedAt == first.CompletedAt {
		t.Fatalf("a newer successful scan must update the timestamp (both %s)", s.CompletedAt)
	}
	if s.JunkItems != 4 || s.Target != `D:\` {
		t.Errorf("latest summary must reflect the newest scan, got %+v", s)
	}
}

// The persistence creates its parent directory on demand: on a fresh
// install %LOCALAPPDATA%\JunkFuck may not exist yet (the rules cache is
// only written after a successful remote rules update).
func TestLastScanPersistenceCreatesParentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing", "nested") // does not exist
	c := &Core{
		sessions:     map[string]*session{},
		cancelFns:    map[string]context.CancelFunc{},
		lastScanFile: filepath.Join(dir, "last-scan.json"),
	}
	c.recordScanResult([]string{`C:\`}, sampleResult(1), false)

	if _, err := os.Stat(filepath.Join(dir, "last-scan.json")); err != nil {
		t.Fatalf("summary must be persisted even when the parent dir did not exist: %v", err)
	}
}

// The summary survives persistence + reload (application restart).
func TestLastScanSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	c1 := &Core{sessions: map[string]*session{}, cancelFns: map[string]context.CancelFunc{}, lastScanFile: filepath.Join(dir, "last-scan.json")}
	c1.recordScanResult([]string{`C:\`}, sampleResult(2), false)

	// A fresh Core (as after an app restart) loads the persisted summary.
	c2 := &Core{sessions: map[string]*session{}, cancelFns: map[string]context.CancelFunc{}, lastScanFile: filepath.Join(dir, "last-scan.json")}
	c2.loadLastScan()

	s := lastScan(t, c2)
	if s == nil {
		t.Fatal("persisted summary must load on restart")
	}
	if s.JunkItems != 2 || s.ReclaimableBytes != reclaimable(sampleResult(2)) {
		t.Errorf("reloaded summary counts mismatch: %+v", s)
	}
	if s.CompletedAt == "" {
		t.Error("reloaded summary must keep its timestamp")
	}
}

// The summary counts always agree with the scan result's candidates
// (Dashboard must match Results).
func TestSummaryCountsMatchResult(t *testing.T) {
	c := newTestCore(t)
	res := sampleResult(7)
	c.recordScanResult([]string{`C:\`}, res, false)

	s := lastScan(t, c)
	if s == nil {
		t.Fatal("expected a summary")
	}
	if s.JunkItems != len(res.Candidates) {
		t.Errorf("junk items = %d, want %d (Results count)", s.JunkItems, len(res.Candidates))
	}
	if s.ReclaimableBytes != reclaimable(res) {
		t.Errorf("reclaimable = %d, want %d (Results sum)", s.ReclaimableBytes, reclaimable(res))
	}
	if s.ErrorCount != len(res.Errors) {
		t.Errorf("errors = %d, want %d", s.ErrorCount, len(res.Errors))
	}
}

// GetLastScanSummary exposes the summary; before the first successful scan
// it reports a clean error (nil record, not stale data).
func TestGetLastScanSummary(t *testing.T) {
	svc := &ScannerService{core: newTestCore(t)}
	if _, err := svc.GetLastScanSummary(); err == nil {
		t.Fatal("before any scan the summary must not exist")
	}

	svc.core.recordScanResult([]string{`C:\`}, sampleResult(1), false)
	s, err := svc.GetLastScanSummary()
	if err != nil {
		t.Fatalf("after a successful scan the summary must be available: %v", err)
	}
	if s.JunkItems != 1 {
		t.Errorf("junk items = %d, want 1", s.JunkItems)
	}
}
