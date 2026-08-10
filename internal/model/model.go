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

// ScanError records a path that could not be read during a scan.
type ScanError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
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
