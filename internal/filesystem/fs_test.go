package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonical(t *testing.T) {
	dir := t.TempDir()
	got, err := Canonical(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(dir) {
		t.Errorf("Canonical(%s) = %s", dir, got)
	}
}

func TestIsDriveRoot(t *testing.T) {
	for _, root := range []string{`C:\`, `C:\Windows\..`, `D:\`} {
		if !IsDriveRoot(root) {
			t.Errorf("expected %q to be a drive root", root)
		}
	}
	if IsDriveRoot(filepath.Join(t.TempDir(), "x")) {
		t.Error("subdirectory should not be a drive root")
	}
}

func TestCompareKeyAndContainment(t *testing.T) {
	base := t.TempDir()
	child := filepath.Join(base, "cache", "inner.tmp")
	if !SameOrUnder(child, base) {
		t.Error("child should be under base")
	}
	sibling := filepath.Join(filepath.Dir(base), "other")
	if SameOrUnder(sibling, base) {
		t.Error("sibling should not be under base")
	}
	if !SameOrUnder(base, base) {
		t.Error("base is under itself")
	}
}

func TestIsReparsePointRegularFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "normal.tmp")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	reparse, err := IsReparsePoint(f)
	if err != nil {
		t.Fatal(err)
	}
	if reparse {
		t.Error("regular file must not be a reparse point")
	}
}

func TestReadOnlyDetection(t *testing.T) {
	f := filepath.Join(t.TempDir(), "ro.tmp")
	if err := os.WriteFile(f, []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(f)
	if err != nil {
		t.Fatal(err)
	}
	if !IsReadOnly(info) {
		t.Error("0444 file should be detected as read-only")
	}
}
