// Package classifier implements the explicit junk rules. It is a pure
// rule engine: it never touches the filesystem, which keeps it trivially
// testable. Rules are explicit and curated — no broad substring matching
// (see docs/MODERNIZATION-SPEC.md "Decisions Log").
package classifier

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/kof-huskai/Junk-Fuck/internal/model"
)

// Match is the classification result for one path.
type Match struct {
	Matched  bool
	Category model.Category
	Reason   string
}

// RuleSet holds the explicit classification rules.
type RuleSet struct {
	fileExt map[string]model.Category
	fileTok map[string]model.Category
	folders map[string]model.Category
}

// New returns a RuleSet with the curated rules.
func New() *RuleSet {
	rs := &RuleSet{
		fileExt: map[string]model.Category{},
		fileTok: map[string]model.Category{},
		folders: map[string]model.Category{},
	}

	// --- File extension rules (lowercase, exact match) ---
	temp := []string{".tmp", ".temp", ".cache", ".tmp_", ".~", ".~~~", ".swm", ".exe~", ".dll~", ".so~", ".dylib~"}
	for _, e := range temp {
		rs.fileExt[e] = model.CategoryTempFiles
	}
	for _, e := range []string{".log", ".etl", ".evtx"} {
		rs.fileExt[e] = model.CategoryLogs
	}
	for _, e := range []string{".dmp", ".dump", ".hdmp", ".mdmp", ".wer", ".stackdump"} {
		rs.fileExt[e] = model.CategoryCrashDumps
	}
	for _, e := range []string{".bak", ".backup", ".old"} {
		rs.fileExt[e] = model.CategoryBackups
	}
	for _, e := range []string{".crdownload", ".download", ".part"} {
		rs.fileExt[e] = model.CategoryPartialDownloads
	}
	for _, e := range []string{".pyc", ".pyo", ".class", ".o", ".obj"} {
		rs.fileExt[e] = model.CategoryBuildArtifacts
	}
	for _, e := range []string{".swp", ".swo"} {
		rs.fileExt[e] = model.CategoryEditorTemp
	}
	for _, e := range []string{".thumb", ".thumbcache"} {
		rs.fileExt[e] = model.CategoryCache
	}
	for _, e := range []string{".chk", ".fts", ".gid", ".pft", ".error", ".trace", ".sess", ".session", ".lock"} {
		rs.fileExt[e] = model.CategoryOtherJunk
	}

	// --- Conservative filename token rules (files only) ---
	// The stem is split on non-alphanumeric characters and matched against
	// explicit keywords. Deliberately excludes broad words like "old",
	// "copy" or "~" that the Python version matched (see Decisions Log).
	rs.fileTok["temp"] = model.CategoryTempFiles
	rs.fileTok["tmp"] = model.CategoryTempFiles
	rs.fileTok["cache"] = model.CategoryCache
	rs.fileTok["thumbcache"] = model.CategoryCache
	rs.fileTok["backup"] = model.CategoryBackups

	// --- Exact folder name rules (looked up case-insensitively) ---
	folder := func(cat model.Category, names ...string) {
		for _, n := range names {
			rs.folders[strings.ToLower(n)] = cat
		}
	}
	folder(model.CategoryTempFiles, "temp", "tmp", "Temporary")
	folder(model.CategoryCache, "cache", "caches", "cached", ".cache", ".thumbnails", "thumbnails", "DeliveryOptimization")
	folder(model.CategoryLogs, "logs")
	folder(model.CategoryBackups, "backup", ".Trash-1000")
	folder(model.CategoryBuildArtifacts, "__pycache__")
	folder(model.CategoryOtherJunk, "trash", ".trash", "prefetch", "downloaded installations")

	return rs
}

// Classify returns the match for a path. isDir must reflect the real
// filesystem type of the entry.
func (rs *RuleSet) Classify(path string, isDir bool) Match {
	base := filepath.Base(path)
	name := strings.ToLower(base)

	if isDir {
		if cat, ok := rs.folders[name]; ok {
			return Match{Matched: true, Category: cat, Reason: fmt.Sprintf("matches junk folder name %q", base)}
		}
		return Match{}
	}

	ext := strings.ToLower(filepath.Ext(base))
	if cat, ok := rs.fileExt[ext]; ok {
		return Match{Matched: true, Category: cat, Reason: fmt.Sprintf("matches junk extension %q", ext)}
	}

	// Editor/temporary backup suffixes like "notes.txt~" carry the marker
	// at the very end, which filepath.Ext does not surface on its own.
	if strings.HasSuffix(name, "~") {
		return Match{Matched: true, Category: model.CategoryTempFiles, Reason: "temporary backup suffix (~)"}
	}

	stem := strings.TrimSuffix(base, filepath.Ext(base))
	for _, tok := range tokens(stem) {
		if cat, ok := rs.fileTok[tok]; ok {
			return Match{Matched: true, Category: cat, Reason: fmt.Sprintf("filename token %q is a known junk keyword", tok)}
		}
	}
	return Match{}
}

// tokens splits a filename stem into lowercase word tokens.
func tokens(stem string) []string {
	parts := strings.FieldsFunc(stem, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.ToLower(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
