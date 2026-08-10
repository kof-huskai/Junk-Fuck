// Package cleaner performs validated cleanup. It is the only place that
// deletes files, and every deletion goes through a full revalidation
// against the scan session and the protection rules.
package cleaner

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kof-huskai/Junk-Fuck/internal/filesystem"
	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/protection"
	"github.com/kof-huskai/Junk-Fuck/internal/report"
)

// Cleaner deletes scan-session candidates after revalidation.
type Cleaner struct {
	protection *protection.Protection
}

// New creates a Cleaner.
func New(pr *protection.Protection) *Cleaner {
	return &Cleaner{protection: pr}
}

// Clean validates and (unless dryRun) deletes the selected paths against
// the given scan session. Paths outside the session are rejected, as are
// protected, reparse-point and changed-identity candidates.
func (c *Cleaner) Clean(dryRun bool, session *model.ScanSession, selected []string) report.Report {
	rep := report.Report{DryRun: dryRun}
	if session == nil {
		rep.Add(report.Item{Path: "", Name: "", Status: report.StatusFailed, Reason: "no scan session"})
		return rep
	}

	seen := map[string]bool{}
	for _, raw := range selected {
		key := filesystem.CompareKey(raw)
		if seen[key] {
			continue
		}
		seen[key] = true

		cand, ok := session.Candidates[key]
		if !ok {
			rep.Add(report.Item{
				Path:   raw,
				Name:   filepath.Base(raw),
				Status: report.StatusFailed,
				Reason: "not part of the current scan; refusing to delete an unknown path",
			})
			continue
		}

		status, reason := c.validate(cand)
		if status != report.StatusDeleted {
			rep.Add(report.Item{Path: cand.Path, Name: cand.Name, Status: status, Reason: reason})
			continue
		}

		if dryRun {
			rep.Add(report.Item{Path: cand.Path, Name: cand.Name, Status: report.StatusDeleted, Reason: "dry run: would delete"})
			rep.BytesFreed += cand.Size
			continue
		}

		if err := c.delete(cand); err != nil {
			rep.Add(report.Item{Path: cand.Path, Name: cand.Name, Status: report.StatusFailed, Reason: err.Error()})
			continue
		}
		rep.Add(report.Item{Path: cand.Path, Name: cand.Name, Status: report.StatusDeleted, Reason: ""})
		rep.BytesFreed += cand.Size
	}
	return rep
}

// validate runs the full safety revalidation for one candidate (SR-1, SR-3,
// SR-10, SR-11, SR-13). It returns the resulting status: deleted (ok),
// skipped (refused for safety) or failed (could not be processed).
func (c *Cleaner) validate(cand model.Candidate) (report.Status, string) {
	if cand.Protected {
		return report.StatusSkipped, "protected path"
	}
	if filesystem.IsDriveRoot(cand.Path) {
		return report.StatusSkipped, "drive root"
	}

	info, err := os.Stat(cand.Path)
	if err != nil {
		return report.StatusFailed, "no longer exists or is unreadable"
	}
	if cand.IsDir && !info.IsDir() {
		return report.StatusSkipped, "path is no longer a directory (identity changed)"
	}
	if !cand.IsDir && info.IsDir() {
		return report.StatusSkipped, "path is now a directory (identity changed)"
	}

	reparse, err := filesystem.IsReparsePoint(cand.Path)
	if err == nil && reparse {
		return report.StatusSkipped, "symbolic link / reparse point (refused)"
	}

	if c.protection.IsProtected(cand.Path) {
		return report.StatusSkipped, "protected path"
	}
	if cand.IsDir && c.protection.IsAncestorOfProtected(cand.Path) {
		return report.StatusSkipped, "directory contains a protected path"
	}
	return report.StatusDeleted, ""
}

// delete removes a single validated candidate.
func (c *Cleaner) delete(cand model.Candidate) error {
	if cand.IsDir {
		return forceRemoveAll(cand.Path)
	}
	return removeFile(cand.Path)
}

// removeFile removes a file, clearing the read-only attribute first if
// needed (Windows).
func removeFile(path string) error {
	info, err := os.Lstat(path)
	if err == nil && filesystem.IsReadOnly(info) {
		_ = filesystem.MakeWritable(path)
	}
	return os.Remove(path)
}

// forceRemoveAll removes a directory tree, clearing read-only attributes on
// files first so locked read-only files do not abort the whole operation.
func forceRemoveAll(root string) error {
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, ierr := d.Info()
			if ierr == nil && filesystem.IsReadOnly(info) {
				_ = filesystem.MakeWritable(path)
			}
		}
		return nil
	})
	return os.RemoveAll(root)
}
