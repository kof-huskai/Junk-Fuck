package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kof-huskai/Junk-Fuck/internal/classifier"
	"github.com/kof-huskai/Junk-Fuck/internal/model"
	"github.com/kof-huskai/Junk-Fuck/internal/protection"
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
