package protection

import (
	"path/filepath"
	"strings"
	"testing"
)

// fakeProtector is a minimal DynamicProtector for contract tests. It
// protects exactly the listed paths (no prefix semantics — the whitelist
// engine implements those; here we only test the protection integration).
type fakeProtector struct {
	protected []string
	ancestors []string
}

func (f fakeProtector) Protects(path string) bool {
	for _, p := range f.protected {
		if strings.EqualFold(p, path) {
			return true
		}
	}
	return false
}

func (f fakeProtector) ProtectsAncestor(path string) bool {
	for _, p := range f.ancestors {
		if strings.EqualFold(p, path) {
			return true
		}
	}
	return false
}

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

// The whitelist engine is an ADDITIVE protection source: it can only make
// more paths protected. Hard-coded safety always wins — a dynamic source
// can never disable it, and clearing the dynamic source removes only the
// dynamic protections.
func TestDynamicProtectorIsAdditive(t *testing.T) {
	p := New(emptyRules())
	whitelisted := `C:\Users\demo\AppData\Roaming\v2rayN`
	if p.IsProtected(whitelisted) {
		t.Fatal("setup: dynamic path must not be protected before wiring")
	}

	p.SetDynamic(fakeProtector{protected: []string{whitelisted}, ancestors: []string{`C:\Users\demo`}})

	// Dynamic rule adds protection.
	if !p.IsProtected(whitelisted) {
		t.Error("whitelisted path must become protected")
	}
	// Hard-coded safety is unaffected (even though the dynamic source does
	// not cover it).
	if !p.IsProtected(`C:\Windows\System32\drivers\etc\hosts`) {
		t.Error("hard-coded protection must stay active")
	}
	// Dynamic ancestor check participates in the same refusal.
	if !p.IsAncestorOfProtected(`C:\Users\demo`) {
		t.Error("ancestor of a whitelisted path must be refused for deletion")
	}
	// Unrelated paths stay unprotected.
	if p.IsProtected(`C:\Users\demo\Downloads\setup.exe`) {
		t.Error("unrelated path must stay unprotected")
	}

	// Clearing the dynamic source removes only dynamic protections.
	p.SetDynamic(nil)
	if p.IsProtected(whitelisted) {
		t.Error("clearing the dynamic source must remove dynamic protection")
	}
	if !p.IsProtected(`C:\Windows\System32`) {
		t.Error("hard protection must survive SetDynamic(nil)")
	}
}

// A dynamic source with no rules cannot affect anything, and the contract
// is strictly one-way: nothing in Protection can ever mark a path deletable.
func TestDynamicSourceCannotWeakenProtection(t *testing.T) {
	p := New(emptyRules())
	p.SetDynamic(fakeProtector{})
	for _, path := range []string{`C:\Windows`, `C:\Program Files`, `C:\`} {
		if !p.IsProtected(path) {
			t.Errorf("%s must stay protected with an empty dynamic source", path)
		}
	}
}
