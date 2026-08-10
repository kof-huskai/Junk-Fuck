package protection

import (
	"path/filepath"
	"testing"
)

func emptyRules() Rules {
	return Rules{Env: Env{}}
}

func TestSystemPathsAreProtected(t *testing.T) {
	p := New(emptyRules())
	protected := []string{
		`C:\Windows`,
		`C:\Windows\System32`,
		`C:\Windows\System32\drivers\etc\hosts`,
		`C:\Windows\SysWOW64`,
		`C:\Program Files`,
		`C:\Program Files (x86)`,
		`C:\ProgramData`,
		`C:\Users\All Users`,
		`C:\System Volume Information`,
		`C:\$Recycle.Bin`,
		`C:\Recovery`,
		`C:\Boot`,
		`C:\Documents and Settings`,
	}
	for _, path := range protected {
		if !p.IsProtected(path) {
			t.Errorf("expected %s to be protected", path)
		}
	}
}

func TestDriveRootsAreProtected(t *testing.T) {
	p := New(emptyRules())
	for _, root := range []string{`C:\`, `D:\`, `E:\`, `F:\`} {
		if !p.IsProtected(root) {
			t.Errorf("expected drive root %s to be protected", root)
		}
	}
}

func TestCaseInsensitiveContainment(t *testing.T) {
	p := New(emptyRules())
	if !p.IsProtected(`c:\windows\system32\drivers`) {
		t.Error("lowercased windows path should be protected")
	}
	if !p.IsProtected(`C:\PROGRAM FILES\Something\App`) {
		t.Error("uppercase program files path should be protected")
	}
}

func TestTempDirIsNotProtected(t *testing.T) {
	p := New(emptyRules())
	tmp := filepath.Join(t.TempDir(), "anything", "cache")
	if p.IsProtected(tmp) {
		t.Errorf("temp fixture %s must not be protected", tmp)
	}
}

func TestEnvSystemPathsAreProtected(t *testing.T) {
	p := New(Rules{
		Env: Env{
			ProgramData: `C:\ProgramData`,
			SystemRoot:  `C:\Windows`,
		},
	})
	protected := []string{
		`C:\ProgramData\anything`,
		`C:\Windows\Temp`,
	}
	for _, path := range protected {
		if !p.IsProtected(path) {
			t.Errorf("expected %s to be protected", path)
		}
	}
}

func TestAppDirectoriesAreProtected(t *testing.T) {
	p := New(Rules{
		Env: Env{
			UserProfile:  `C:\Users\demo`,
			LocalAppData: `C:\Users\demo\AppData\Local`,
			AppData:      `C:\Users\demo\AppData\Roaming`,
		},
		Apps: []string{"discord", "chrome"},
	})
	protected := []string{
		`C:\Users\demo\AppData\Local\Discord`,
		`C:\Users\demo\AppData\Local\discord`,
		`C:\Users\demo\AppData\Roaming\chrome`,
		`C:\Users\demo\AppData\Local\Chrome\User Data\Cache`,
	}
	for _, path := range protected {
		if !p.IsProtected(path) {
			t.Errorf("expected %s to be protected", path)
		}
	}
}

func TestAncestorOfProtected(t *testing.T) {
	p := New(emptyRules())
	// The drive root contains protected paths, so it must be refused as a
	// deletion target.
	if !p.IsAncestorOfProtected(`C:\`) {
		t.Error("drive root is an ancestor of protected paths")
	}
	// C:\Windows contains C:\Windows\System32 (protected) -> true.
	if !p.IsAncestorOfProtected(`C:\Windows`) {
		t.Error("C:\\Windows contains protected System32 and must be refused")
	}
	// C:\Windows\System32 is protected itself but contains nothing else
	// protected -> false.
	if p.IsAncestorOfProtected(`C:\Windows\System32`) {
		t.Error("C:\\Windows\\System32 is not a strict ancestor of another protected path")
	}
	if p.IsAncestorOfProtected(`C:\Win`) {
		t.Error("C:\\Win is NOT an ancestor of C:\\Windows (separator boundary)")
	}
	if p.IsAncestorOfProtected(`C:\Users\demo`) {
		t.Error("unrelated path must not be an ancestor of a protected path")
	}
}

func TestUserProfileContainmentIsNotBlanket(t *testing.T) {
	// The Python version protected the entire USERPROFILE. The desktop app
	// intentionally protects app directories instead (see Decisions Log),
	// so unrelated profile paths must NOT be protected.
	p := New(Rules{
		Env:  Env{UserProfile: `C:\Users\demo`},
		Apps: []string{"discord"},
	})
	if p.IsProtected(`C:\Users\demo\Downloads\setup.exe`) {
		t.Error("unrelated user file should not be protected")
	}
	if !p.IsProtected(`C:\Users\demo\AppData\Local\Discord`) {
		t.Error("app dir should be protected")
	}
}

func TestListIsStable(t *testing.T) {
	p := New(emptyRules())
	if len(p.List()) == 0 {
		t.Fatal("expected at least the builtin paths")
	}
}
