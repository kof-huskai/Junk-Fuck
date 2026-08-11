// Package scanner performs read-only junk discovery. It never modifies
// the filesystem; cleanup is exclusively the cleaner's job.
package scanner

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kof-huskai/Junk-Fuck/internal/classifier"
	"github.com/kof-huskai/Junk-Fuck/internal/filesystem"
	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/protection"
)

// isPermissionError reports whether err is an access/permission denial. The
// UI uses this to offer a one-time "run as administrator" hint — it must
// never claim an arbitrary error is a permission problem.
//
// Recognized failures: os.ErrPermission and the Windows error codes for
// access denied (5), privilege not held (1314), elevation required (740),
// cannot access file (1920) and WSAEACCES (10013). The Errno numbers are
// harmless no-ops on non-Windows platforms.
func isPermissionError(err error) bool {
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case 5, 1314, 740, 1920, 10013:
			return true
		}
	}
	return false
}

// Scanner discovers junk candidates within configured targets.
type Scanner struct {
	classifier *classifier.RuleSet
	protection *protection.Protection
}

// New creates a Scanner.
func New(cl *classifier.RuleSet, pr *protection.Protection) *Scanner {
	return &Scanner{classifier: cl, protection: pr}
}

// accumulator tracks the running size of a junk-directory candidate found
// during the walk. Directory sizes are accumulated as files are visited
// beneath them, avoiding a second full walk.
type accumulator struct {
	key  string
	size *int64
}

// Scan walks targets read-only. Progress is reported through progressFn;
// cancellation is honored via ctx. The scan always returns partial results
// rather than failing as a whole.
func (s *Scanner) Scan(ctx context.Context, scanID string, targets []string, progressFn func(model.Progress)) model.Result {
	start := time.Now()
	if progressFn == nil {
		progressFn = func(model.Progress) {}
	}

	var (
		candidates []model.Candidate
		scanErrors []model.ScanError
		scanned    int64
		pending    []accumulator
		lastEmit   time.Time
	)

	emit := func(current string) {
		if time.Since(lastEmit) < 100*time.Millisecond && scanned%512 != 0 {
			return
		}
		lastEmit = time.Now()
		progressFn(model.Progress{
			ScanID:       scanID,
			ScannedFiles: scanned,
			Candidates:   int64(len(candidates)),
			Errors:       int64(len(scanErrors)),
			CurrentPath:  current,
		})
	}

	cancelled := false
	for _, target := range targets {
		if ctx.Err() != nil {
			cancelled = true
			break
		}
		canon, err := filesystem.Canonical(target)
		if err != nil {
			scanErrors = append(scanErrors, model.ScanError{Path: target, Error: err.Error(), Permission: isPermissionError(err)})
			continue
		}

		// Scanning is read-only: protected targets (e.g. C:\ drive root) may
		// still be walked. Protection only forbids *deletion*, and the
		// cleaner re-validates every candidate before touching anything.
		walkErr := filepath.WalkDir(canon, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				if d != nil && d.IsDir() {
					scanErrors = append(scanErrors, model.ScanError{Path: path, Error: err.Error(), Permission: isPermissionError(err)})
					return fs.SkipDir
				}
				scanErrors = append(scanErrors, model.ScanError{Path: path, Error: err.Error(), Permission: isPermissionError(err)})
				return nil
			}

			scanned++
			emit(path)

			key := filesystem.CompareKey(path)

			if d.IsDir() {
				// Skip protected subdirectories (e.g. C:\Windows when scanning
				// C:\) but never skip the scan target itself - otherwise a
				// protected drive root like C:\ would scan nothing.
				if path != canon && s.protection.IsProtected(path) {
					return fs.SkipDir
				}
				// Never traverse symbolic links / junctions: they can redirect
				// the walk into protected areas or cause loops (safety SR-11).
				if reparse, rerr := filesystem.IsReparsePoint(path); rerr == nil && reparse {
					return fs.SkipDir
				}
				if m := s.classifier.Classify(path, true); m.Matched {
					candidates = append(candidates, model.Candidate{
						Path:       path,
						Name:       d.Name(),
						IsDir:      true,
						Category:   m.Category,
						Size:       0,
						Reason:     m.Reason,
						Protected:  s.protection.IsProtected(path) || s.protection.IsAncestorOfProtected(path),
						ScanSource: canon,
					})
					pending = append(pending, accumulator{key: key, size: &candidates[len(candidates)-1].Size})
				}
				return nil
			}

			// Never offer symlinked files for deletion.
			if reparse, rerr := filesystem.IsReparsePoint(path); rerr == nil && reparse {
				return nil
			}

			// File: if it sits under a junk-directory candidate, its size
			// is counted there and it is not added separately.
			if acc := deepestAncestor(pending, key); acc != nil {
				info, ierr := d.Info()
				if ierr == nil {
					*acc.size += info.Size()
				}
				return nil
			}

			if m := s.classifier.Classify(path, false); m.Matched {
				info, ierr := d.Info()
				size := int64(0)
				if ierr == nil {
					size = info.Size()
				}
				candidates = append(candidates, model.Candidate{
					Path:       path,
					Name:       d.Name(),
					IsDir:      false,
					Category:   m.Category,
					Size:       size,
					Reason:     m.Reason,
					Protected:  s.protection.IsProtected(path),
					ScanSource: canon,
				})
			}
			return nil
		})
		if ctx.Err() != nil {
			cancelled = true
		}
		if walkErr != nil && ctx.Err() == nil {
			scanErrors = append(scanErrors, model.ScanError{Path: canon, Error: walkErr.Error(), Permission: isPermissionError(walkErr)})
		}
	}

	progressFn(model.Progress{
		ScanID:       scanID,
		ScannedFiles: scanned,
		Candidates:   int64(len(candidates)),
		Errors:       int64(len(scanErrors)),
		Done:         true,
	})

	return model.Result{
		ScannedFiles: scanned,
		Candidates:   candidates,
		Errors:       scanErrors,
		Cancelled:    cancelled,
		DurationMS:   time.Since(start).Milliseconds(),
	}
}

// deepestAncestor returns the active junk-dir accumulator with the longest
// key that is an ancestor of key, or nil.
func deepestAncestor(pending []accumulator, key string) *accumulator {
	var best *accumulator
	for i := range pending {
		if strings.HasPrefix(key, pending[i].key+string(filepath.Separator)) {
			if best == nil || len(pending[i].key) > len(best.key) {
				best = &pending[i]
			}
		}
	}
	return best
}
