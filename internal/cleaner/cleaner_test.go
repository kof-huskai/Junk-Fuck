package cleaner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kof-huskai/Junk-Fuck/internal/filesystem"
	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/protection"
	"github.com/kof-huskai/Junk-Fuck/internal/report"
	"golang.org/x/sys/windows"
)

func makeSession(t *testing.T, cands ...model.Candidate) *model.ScanSession {
	t.Helper()
	s := &model.ScanSession{ID: "test", Candidates: map[string]model.Candidate{}}
	for _, c := range cands {
		s.Candidates[filesystem.CompareKey(c.Path)] = c
	}
	return s
}

func fileCandidate(path string, size int64) model.Candidate {
	return model.Candidate{Path: path, Name: filepath.Base(path), IsDir: false, Category: model.CategoryTempFiles, Size: size}
}

func dirCandidate(path string) model.Candidate {
	return model.Candidate{Path: path, Name: filepath.Base(path), IsDir: true, Category: model.CategoryCache, Size: 10}
}

func TestDeleteSelectedFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.tmp")
	b := filepath.Join(dir, "b.tmp")
	c := filepath.Join(dir, "keep.txt")
	for _, f := range []string{a, b, c} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t, fileCandidate(a, 1), fileCandidate(b, 2)), []string{a, b})

	if rep.DeletedCount != 2 || rep.FailedCount != 0 || rep.SkippedCount != 0 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	if rep.BytesFreed != 3 {
		t.Errorf("bytes freed = %d, want 3", rep.BytesFreed)
	}
	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Error("a.tmp should be deleted")
	}
	if _, err := os.Stat(b); !os.IsNotExist(err) {
		t.Error("b.tmp should be deleted")
	}
	if _, err := os.Stat(c); err != nil {
		t.Error("keep.txt must remain untouched")
	}
}

func TestRejectsPathOutsideSession(t *testing.T) {
	dir := t.TempDir()
	rogue := filepath.Join(dir, "rogue.tmp")
	if err := os.WriteFile(rogue, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t), []string{rogue})
	if rep.FailedCount != 1 {
		t.Fatalf("expected 1 failed item, got %+v", rep)
	}
	if _, err := os.Stat(rogue); err != nil {
		t.Error("rogue file must not be deleted")
	}
}

// fakeProtector mirrors the whitelist engine's DynamicProtector contract
// (exact matches; the real engine adds prefix semantics — irrelevant here).
type fakeProtector struct {
	protected []string
}

func (f fakeProtector) Protects(path string) bool {
	for _, p := range f.protected {
		if p == path {
			return true
		}
	}
	return false
}

func (f fakeProtector) ProtectsAncestor(string) bool { return false }

// The whitelist is protection-only: a rule matching a candidate's path must
// make that candidate non-deletable — remote data can never mark anything
// for cleanup.
func TestWhitelistProtectsExistingCandidate(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "v2rayN")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(app, "guiNConfig.json")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	pr := protection.New(protection.Rules{Env: protection.Env{}})
	pr.SetDynamic(fakeProtector{protected: []string{f}})
	if !pr.IsProtected(f) {
		t.Fatal("setup: whitelist rule must protect the candidate path")
	}

	cleaner := New(pr)
	rep := cleaner.Clean(false, makeSession(t, fileCandidate(f, 1)), []string{f})
	if rep.SkippedCount != 1 {
		t.Fatalf("whitelisted candidate must be skipped, got %+v", rep)
	}
	if _, err := os.Stat(f); err != nil {
		t.Error("whitelisted file must not be deleted")
	}
}

func TestSkipsProtectedCandidate(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "protected.tmp")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cand := fileCandidate(f, 1)
	cand.Protected = true
	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t, cand), []string{f})
	if rep.SkippedCount != 1 {
		t.Fatalf("expected 1 skipped item, got %+v", rep)
	}
	if _, err := os.Stat(f); err != nil {
		t.Error("protected file must not be deleted")
	}
}

func TestDryRunNeverDeletes(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.tmp")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(true, makeSession(t, fileCandidate(f, 7)), []string{f})
	if !rep.DryRun {
		t.Error("report should be marked as dry run")
	}
	if rep.DeletedCount != 1 || rep.BytesFreed != 7 {
		t.Errorf("dry run should report would-delete stats: %+v", rep)
	}
	if _, err := os.Stat(f); err != nil {
		t.Error("dry run must never delete")
	}
}

func TestDeleteDirectoryTree(t *testing.T) {
	dir := t.TempDir()
	folder := filepath.Join(dir, "cache")
	if err := os.MkdirAll(filepath.Join(folder, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "nested", "junk.tmp"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t, dirCandidate(folder)), []string{folder})
	if rep.DeletedCount != 1 {
		t.Fatalf("expected 1 deletion, got %+v", rep)
	}
	if _, err := os.Stat(folder); !os.IsNotExist(err) {
		t.Error("cache dir should be deleted")
	}
}

func TestDeleteReadOnlyFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "ro.tmp")
	if err := os.WriteFile(f, []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t, fileCandidate(f, 1)), []string{f})
	if rep.DeletedCount != 1 {
		t.Fatalf("expected read-only file to be deleted, got %+v", rep)
	}
}

func TestIdentityChangedIsRejected(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "wasfile.tmp")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Candidate claims it is a directory, but it is now a file.
	cand := dirCandidate(f)
	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t, cand), []string{f})
	if rep.SkippedCount != 1 {
		t.Fatalf("expected identity change to be skipped, got %+v", rep)
	}
	if _, err := os.Stat(f); err != nil {
		t.Error("file must not be deleted on identity mismatch")
	}
}

func TestMissingFileIsFailed(t *testing.T) {
	dir := t.TempDir()
	gone := filepath.Join(dir, "gone.tmp")
	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t, fileCandidate(gone, 1)), []string{gone})
	if rep.FailedCount != 1 {
		t.Fatalf("expected 1 failed item, got %+v", rep)
	}
}

func TestRefusesDirContainingProtectedPath(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep")
	if err := os.MkdirAll(keep, 0o755); err != nil {
		t.Fatal(err)
	}
	// The junk dir is an ancestor of a protected path.
	pr := protection.New(protection.Rules{Env: protection.Env{}, Paths: []string{keep}})
	cleaner := New(pr)
	rep := cleaner.Clean(false, makeSession(t, dirCandidate(dir)), []string{dir})
	if rep.SkippedCount != 1 {
		t.Fatalf("expected ancestor-of-protected to be skipped, got %+v", rep)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Error("protected path must survive")
	}
}

func TestRefusesSymlinkedCandidate(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target.tmp")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.tmp")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlinks on this machine: %v", err)
	}

	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t, fileCandidate(link, 1)), []string{link})
	if rep.SkippedCount != 1 {
		t.Fatalf("expected symlink candidate to be refused, got %+v", rep)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Error("symlink must not be deleted")
	}
}

func TestDuplicateSelectionIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.tmp")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t, fileCandidate(f, 1)), []string{f, f})
	if rep.DeletedCount != 1 {
		t.Fatalf("duplicates should be handled once, got %+v", rep)
	}
}

// setHidden applies the real Windows hidden attribute (FILE_ATTRIBUTE_HIDDEN);
// a no-op elsewhere. The directory flag is preserved on directories because
// SetFileAttributes replaces the attribute set.
func setHidden(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		return
	}
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	attrs := windows.FILE_ATTRIBUTE_HIDDEN
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		attrs |= windows.FILE_ATTRIBUTE_DIRECTORY
	}
	if err := windows.SetFileAttributes(ptr, uint32(attrs)); err != nil {
		t.Fatal(err)
	}
}

// Hidden state never relaxes deletion safety: dry runs stay non-destructive
// and arbitrary (even hidden) paths outside the scan session are refused.
func TestDryRunNeverDeletesHiddenCandidate(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "hidden.tmp")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	setHidden(t, f)

	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(true, makeSession(t, fileCandidate(f, 7)), []string{f})
	if !rep.DryRun {
		t.Error("report should be marked as dry run")
	}
	if rep.DeletedCount != 1 || rep.BytesFreed != 7 {
		t.Errorf("dry run should report would-delete stats: %+v", rep)
	}
	if _, err := os.Stat(f); err != nil {
		t.Error("dry run must never delete hidden files")
	}
}

func TestRejectsHiddenPathOutsideSession(t *testing.T) {
	dir := t.TempDir()
	rogue := filepath.Join(dir, "rogue.tmp")
	if err := os.WriteFile(rogue, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	setHidden(t, rogue)

	cleaner := New(protection.New(protection.Rules{Env: protection.Env{}}))
	rep := cleaner.Clean(false, makeSession(t), []string{rogue})
	if rep.FailedCount != 1 {
		t.Fatalf("expected 1 failed item, got %+v", rep)
	}
	if _, err := os.Stat(rogue); err != nil {
		t.Error("hidden arbitrary path must never be deleted")
	}
}

var _ = report.StatusDeleted
