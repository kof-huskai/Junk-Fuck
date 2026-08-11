package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/kof-huskai/Junk-Fuck/internal/classifier"
	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/protection"
	"golang.org/x/sys/windows"
)

func newTestScanner(pr *protection.Protection) *Scanner {
	return New(classifier.New(), pr)
}

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsOnlyJunk(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "cache", "inner.tmp"), 100)
	writeFile(t, filepath.Join(dir, "test.tmp"), 50)
	writeFile(t, filepath.Join(dir, "app.log"), 20)
	writeFile(t, filepath.Join(dir, "normal.txt"), 999)

	pr := protection.New(protection.Rules{Env: protection.Env{}})
	s := newTestScanner(pr)
	res := s.Scan(context.Background(), "t1", []string{dir}, nil)

	byName := map[string]model.Candidate{}
	for _, c := range res.Candidates {
		byName[c.Name] = c
	}
	if _, ok := byName["test.tmp"]; !ok {
		t.Error("test.tmp should be a candidate")
	}
	if _, ok := byName["app.log"]; !ok {
		t.Error("app.log should be a candidate")
	}
	if _, ok := byName["normal.txt"]; ok {
		t.Error("normal.txt must not be a candidate")
	}
	dirCand, ok := byName["cache"]
	if !ok {
		t.Fatal("cache dir should be a candidate")
	}
	if dirCand.Size != 100 {
		t.Errorf("cache dir size = %d, want 100", dirCand.Size)
	}
	if dirCand.IsDir != true {
		t.Error("cache should be flagged as directory")
	}
	// inner.tmp must not be a separate candidate (covered by its parent).
	if _, ok := byName["inner.tmp"]; ok {
		t.Error("inner.tmp must be covered by the cache dir candidate")
	}
	if res.Cancelled {
		t.Error("scan should not be cancelled")
	}
}

func TestScanIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.tmp")
	writeFile(t, f, 42)
	s := newTestScanner(protection.New(protection.Rules{Env: protection.Env{}}))
	s.Scan(context.Background(), "t2", []string{dir}, nil)
	info, err := os.Stat(f)
	if err != nil {
		t.Fatalf("file must still exist after scan: %v", err)
	}
	if info.Size() != 42 {
		t.Errorf("file size changed during scan: %d", info.Size())
	}
}

func TestScanProtectedTargetIsWalkedReadOnly(t *testing.T) {
	// A protected target (e.g. C:\ or C:\Windows) must still be scanned:
	// scanning is read-only, protection only forbids deletion. Candidates
	// found there are simply marked protected and never deletable.
	root := t.TempDir()
	pr := protection.New(protection.Rules{Env: protection.Env{}, Paths: []string{root}})
	if !pr.IsProtected(root) {
		t.Fatal("test root should be protected")
	}
	writeFile(t, filepath.Join(root, "junk.tmp"), 10)

	s := newTestScanner(pr)
	res := s.Scan(context.Background(), "t3", []string{root}, nil)
	if len(res.Errors) != 0 {
		t.Errorf("scanning a protected target must not error, got %v", res.Errors)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate from protected target, got %d", len(res.Candidates))
	}
	if !res.Candidates[0].Protected {
		t.Error("candidate under a protected target must be marked protected")
	}
}

func TestScanProtectedInsideTargetIsPruned(t *testing.T) {
	root := t.TempDir()
	// Simulate an app directory nested inside the scanned target.
	appDir := filepath.Join(root, "cache", "discord", "Cache")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(appDir, "junk.tmp"), 10)
	pr := protection.New(protection.Rules{
		Env:  protection.Env{UserProfile: root, LocalAppData: filepath.Join(root, "cache")},
		Apps: []string{"discord"},
	})
	s := newTestScanner(pr)
	res := s.Scan(context.Background(), "t4", []string{root}, nil)
	// The junk dir "cache" is an ancestor of a protected app dir -> protected.
	for _, c := range res.Candidates {
		if c.Name == "cache" {
			if !c.Protected {
				t.Error("cache dir contains a protected app dir and must be marked protected")
			}
		}
	}
}

func TestScanCancellation(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 50; i++ {
		writeFile(t, filepath.Join(dir, "junk.tmp"), 1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	s := newTestScanner(protection.New(protection.Rules{Env: protection.Env{}}))
	res := s.Scan(ctx, "t5", []string{dir}, nil)
	if !res.Cancelled {
		t.Error("expected cancelled result")
	}
}

func TestScanProgressReported(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.tmp"), 1)
	writeFile(t, filepath.Join(dir, "b.tmp"), 1)
	var progress []model.Progress
	s := newTestScanner(protection.New(protection.Rules{Env: protection.Env{}}))
	s.Scan(context.Background(), "t6", []string{dir}, func(p model.Progress) {
		progress = append(progress, p)
	})
	if len(progress) == 0 {
		t.Fatal("expected progress callbacks")
	}
	last := progress[len(progress)-1]
	if !last.Done {
		t.Error("final progress must be marked done")
	}
	if last.Candidates != 2 {
		t.Errorf("expected 2 candidates in final progress, got %d", last.Candidates)
	}
}

func TestScanSkipsReparsePointDirectories(t *testing.T) {
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "junk.tmp"), 10)

	dir := t.TempDir()
	link := filepath.Join(dir, "linkdir")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("cannot create symlinks on this machine: %v", err)
	}

	s := newTestScanner(protection.New(protection.Rules{Env: protection.Env{}}))
	res := s.Scan(context.Background(), "t8", []string{dir}, nil)
	for _, c := range res.Candidates {
		if strings.Contains(strings.ToLower(c.Path), "linkdir") {
			t.Errorf("candidate %s was found through a symlink and must be skipped", c.Path)
		}
	}
}

// setHidden applies the real Windows hidden attribute (FILE_ATTRIBUTE_HIDDEN)
// to a file or directory. It is a no-op on other platforms, where hidden
// state is exercised through dot-prefixed names instead. The directory flag
// is preserved because SetFileAttributes replaces the attribute set.
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

// Hidden-state guarantees: hidden is metadata, never junk. The walk must
// discover hidden content (files AND folders), the classifier must stay
// attribute-agnostic (a hidden normal file is not junk), and protection must
// still prune protected subtrees even when they live inside hidden folders.
func TestScanDiscoversHiddenAttributeJunk(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "hidden.tmp")
	writeFile(t, f, 30)
	setHidden(t, f)

	pr := protection.New(protection.Rules{Env: protection.Env{}})
	res := newTestScanner(pr).Scan(context.Background(), "h1", []string{dir}, nil)

	byName := map[string]model.Candidate{}
	for _, c := range res.Candidates {
		byName[c.Name] = c
	}
	if _, ok := byName["hidden.tmp"]; !ok {
		t.Error("hidden-attribute .tmp file should be discovered as a candidate")
	}
	if _, err := os.Stat(f); err != nil {
		t.Error("the scan must not modify hidden files")
	}
	if len(res.Errors) != 0 {
		t.Errorf("hidden files must not produce errors: %v", res.Errors)
	}
}

func TestScanDiscoversJunkInsideHiddenDirectory(t *testing.T) {
	dir := t.TempDir()
	hidden := filepath.Join(dir, "Private") // not a junk name
	if err := os.MkdirAll(hidden, 0o755); err != nil {
		t.Fatal(err)
	}
	setHidden(t, hidden)
	writeFile(t, filepath.Join(hidden, "app.tmp"), 40)

	pr := protection.New(protection.Rules{Env: protection.Env{}})
	res := newTestScanner(pr).Scan(context.Background(), "h2", []string{dir}, nil)

	byName := map[string]model.Candidate{}
	for _, c := range res.Candidates {
		byName[c.Name] = c
	}
	if _, ok := byName["app.tmp"]; !ok {
		t.Error("junk inside a hidden directory should be discovered")
	}
	if _, ok := byName["Private"]; ok {
		t.Error("a hidden directory must not be junk simply because it is hidden")
	}
}

func TestScanHiddenJunkDirectorySized(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, ".cache") // junk folder by name rule, dot-prefixed
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	setHidden(t, cache)
	writeFile(t, filepath.Join(cache, "inner.tmp"), 25)

	pr := protection.New(protection.Rules{Env: protection.Env{}})
	res := newTestScanner(pr).Scan(context.Background(), "h3", []string{dir}, nil)

	var dirCand *model.Candidate
	for i := range res.Candidates {
		if res.Candidates[i].Name == ".cache" {
			dirCand = &res.Candidates[i]
			break
		}
	}
	if dirCand == nil {
		t.Fatal("hidden .cache junk folder should be a candidate")
	}
	if dirCand.Size != 25 {
		t.Errorf(".cache size = %d, want 25 (contents of a hidden junk dir must be counted)", dirCand.Size)
	}
}

func TestScanHiddenNormalFileIsNotJunk(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".keep.me"), 10)
	hiddenNormal := filepath.Join(dir, "notes.txt")
	writeFile(t, hiddenNormal, 20)
	setHidden(t, hiddenNormal)

	pr := protection.New(protection.Rules{Env: protection.Env{}})
	res := newTestScanner(pr).Scan(context.Background(), "h4", []string{dir}, nil)

	if len(res.Candidates) != 0 {
		t.Errorf("hidden state must never turn normal files into junk, got %v", res.Candidates)
	}
}

func TestScanHiddenProtectedContentStaysProtected(t *testing.T) {
	root := t.TempDir()
	hidden := filepath.Join(root, ".private")
	protected := filepath.Join(hidden, "system")
	if err := os.MkdirAll(protected, 0o755); err != nil {
		t.Fatal(err)
	}
	setHidden(t, hidden)
	writeFile(t, filepath.Join(hidden, "junk.tmp"), 10)
	writeFile(t, filepath.Join(protected, "junk.tmp"), 10)

	pr := protection.New(protection.Rules{Env: protection.Env{}, Paths: []string{protected}})
	s := newTestScanner(pr)
	if !pr.IsProtected(protected) {
		t.Fatal("test setup: protected dir should be protected")
	}
	res := s.Scan(context.Background(), "h5", []string{root}, nil)

	byName := map[string]model.Candidate{}
	for _, c := range res.Candidates {
		byName[c.Name] = c
	}
	if _, ok := byName["junk.tmp"]; !ok {
		t.Error("junk file outside the protected dir (inside the hidden dir) should be discovered")
	}
	for _, c := range res.Candidates {
		if strings.Contains(strings.ToLower(c.Path), strings.ToLower(protected)) {
			t.Errorf("protected content inside a hidden dir must be pruned, got %v", c.Path)
		}
	}
}

// isPermissionError must classify ONLY permission/access denials — the UI
// shows its one-time admin hint based on this flag, so unrelated errors must
// never be mislabelled as a permission problem.
func TestIsPermissionError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"os.ErrPermission", os.ErrPermission, true},
		{"windows access denied", syscall.Errno(5), runtime.GOOS == "windows"},
		{"elevation required", syscall.Errno(740), runtime.GOOS == "windows"},
		{"privilege not held", syscall.Errno(1314), runtime.GOOS == "windows"},
		{"wrapped permission", fmt.Errorf("walk %w", os.ErrPermission), true},
		{"unrelated error", errors.New("some random failure"), false},
		{"file not found", syscall.Errno(2), false},
	}
	for _, c := range cases {
		if got := isPermissionError(c.err); got != c.want {
			t.Errorf("%s: isPermissionError(%v) = %v, want %v", c.name, c.err, got, c.want)
		}
	}
}

func TestScanErrorsDoNotAbort(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ok.tmp"), 1)
	s := newTestScanner(protection.New(protection.Rules{Env: protection.Env{}}))
	res := s.Scan(context.Background(), "t7", []string{dir, filepath.Join(dir, "missing-target")}, nil)
	if len(res.Candidates) != 1 {
		t.Errorf("expected 1 candidate despite a bad target, got %d", len(res.Candidates))
	}
	if len(res.Errors) == 0 {
		t.Error("expected an error for the missing target")
	}
}
