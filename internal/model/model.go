// Package model defines the shared data types of the JunkFuck core.
// It must stay dependency-free so every internal package can use it.
package model

// Category is the structured classification of a junk candidate.
type Category string

const (
	CategoryTempFiles        Category = "temporary-files"
	CategoryCache            Category = "cache"
	CategoryLogs             Category = "logs"
	CategoryCrashDumps       Category = "crash-dumps"
	CategoryBackups          Category = "backups"
	CategoryPartialDownloads Category = "partial-downloads"
	CategoryBuildArtifacts   Category = "build-artifacts"
	CategoryEditorTemp       Category = "editor-temp"
	CategoryOtherJunk        Category = "other-junk"
)

// Candidate is a file or directory that matched a junk rule during a scan.
type Candidate struct {
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	IsDir      bool     `json:"isDir"`
	Category   Category `json:"category"`
	Size       int64    `json:"size"`
	Reason     string   `json:"reason"`
	Protected  bool     `json:"protected"`
	ScanSource string   `json:"scanSource"`
}

// Progress is the live state of an ongoing scan.
type Progress struct {
	ScanID       string `json:"scanId"`
	ScannedFiles int64  `json:"scannedFiles"`
	Candidates   int64  `json:"candidates"`
	Errors       int64  `json:"errors"`
	CurrentPath  string `json:"currentPath"`
	Done         bool   `json:"done"`
}

// ScanSummary is the canonical record of a successfully completed scan: it
// is the single source of truth for the Dashboard's last-scan summary and
// survives application restarts (persisted by the services layer). It is
// written only when a scan reaches a real successful terminal state —
// cancelled or failed scans never replace the previous successful summary.
// CompletedAt is a real timestamp (RFC3339 UTC, nanosecond precision, so
// successive scans never collide); the UI renders relative time from it,
// never a pre-rendered string.
type ScanSummary struct {
	CompletedAt      string `json:"completedAt"` // RFC3339 UTC timestamp (nano)
	Target           string `json:"target"`
	FilesScanned     int64  `json:"filesScanned"`
	JunkItems        int    `json:"junkItems"`
	ReclaimableBytes int64  `json:"reclaimableBytes"`
	ErrorCount       int    `json:"errorCount"`
}

// ScanError records a path that could not be read during a scan.
// Permission is true when the failure is an access/permission denial
// (used by the UI to offer a one-time "run as administrator" hint).
type ScanError struct {
	Path       string `json:"path"`
	Error      string `json:"error"`
	Permission bool   `json:"permission"`
}

// Result is the outcome of a scan operation.
type Result struct {
	ScannedFiles int64       `json:"scannedFiles"`
	Candidates   []Candidate `json:"candidates"`
	Errors       []ScanError `json:"errors"`
	Cancelled    bool        `json:"cancelled"`
	DurationMS   int64       `json:"durationMs"`
}

// ScanSession is an immutable snapshot of the candidates produced by one
// scan. The cleaner only ever deletes against this snapshot (safety SR-2).
type ScanSession struct {
	ID         string
	Targets    []string
	Candidates map[string]Candidate // key: canonical comparison key
	Done       bool
}

// DriveInfo describes one detected Windows logical drive. The frontend only
// ever selects scan roots from this backend-provided list; it never invents
// paths.
type DriveInfo struct {
	Root       string `json:"root"`       // e.g. "C:\\"
	Label      string `json:"label"`      // volume label, may be empty
	Type       string `json:"type"`       // fixed | removable | network | optical | ram | unknown
	TotalBytes int64  `json:"totalBytes"` // 0 when the drive is not ready
	FreeBytes  int64  `json:"freeBytes"`  // 0 when the drive is not ready
	Ready      bool   `json:"ready"`      // false for empty optical drives etc.
}
