// Package protection implements backend-enforced safety rules. Every
// filesystem deletion and scan pruning decision consults this package.
//
// The environment (user profile, AppData, ...) is injected so tests can
// run hermetic safety checks without touching the developer's machine.
package protection

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Rules configures a Protection instance. Paths are additional protected
// roots; Apps are protected application directory names.
type Rules struct {
	// Paths are extra protected roots (beyond the built-in system paths).
	Paths []string
	// Apps are protected application names.
	Apps []string
	// Env mirrors the relevant environment variables. Empty values are
	// simply ignored, so tests can pass an empty Env to isolate behavior.
	Env Env
}

// Env provides the environment values used to derive protected paths.
type Env struct {
	SystemRoot      string
	WinDir          string
	ProgramFiles    string
	ProgramFilesX86 string
	ProgramData     string
	AppData         string
	LocalAppData    string
	UserProfile     string
}

// FromOS builds the default rules from the process environment.
func FromOS() *Protection {
	return New(Rules{
		Env: Env{
			SystemRoot:      os.Getenv("SYSTEMROOT"),
			WinDir:          os.Getenv("WINDIR"),
			ProgramFiles:    os.Getenv("ProgramFiles"),
			ProgramFilesX86: os.Getenv("ProgramFiles(x86)"),
			ProgramData:     os.Getenv("ProgramData"),
			AppData:         os.Getenv("APPDATA"),
			LocalAppData:    os.Getenv("LOCALAPPDATA"),
			UserProfile:     os.Getenv("USERPROFILE"),
		},
		Apps: defaultApps(),
	})
}

// defaultApps is the curated list of protected applications (kept from the
// Python implementation and extended).
func defaultApps() []string {
	return []string{
		"discord", "discord ptb", "discord canary", "discord development",
		"slack", "teams", "zoom", "skype", "telegram", "whatsapp",
		"spotify", "steam", "epicgames", "battle.net", "origin", "uplay",
		"chrome", "firefox", "edge", "brave", "opera", "vivaldi",
		"vscode", "code", "pycharm", "intellij", "webstorm", "rider",
		"postman", "insomnia", "docker", "vmware", "virtualbox",
		"obs", "streamlabs", "xsplit", "nvidia", "amd", "intel",
		"logitech", "razer", "steelseries", "corsair",
	}
}

// builtinSystemPaths are always protected, regardless of environment.
func builtinSystemPaths() []string {
	return []string{
		`C:\Windows`,
		`C:\Windows\System32`,
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
}

var driveRootRe = regexp.MustCompile(`^[A-Za-z]:[\\/]*$`)

// Protection holds the canonical protected path set.
type Protection struct {
	paths []string // canonical comparison keys
}

// New builds a Protection from explicit rules.
func New(rules Rules) *Protection {
	seen := map[string]bool{}
	var paths []string
	add := func(p string) {
		if p == "" {
			return
		}
		k := compareKey(p)
		if !seen[k] {
			seen[k] = true
			paths = append(paths, k)
		}
	}

	for _, p := range builtinSystemPaths() {
		add(p)
	}
	e := rules.Env
	for _, p := range []string{e.SystemRoot, e.WinDir, e.ProgramFiles, e.ProgramFilesX86, e.ProgramData} {
		add(p)
	}
	for _, p := range rules.Paths {
		add(p)
	}

	// Protected application directories under the user profile.
	appBases := []string{
		joinEnv(e.LocalAppData, ""),
		joinEnv(e.AppData, ""),
		joinEnv(e.UserProfile, "AppData", "Local"),
		joinEnv(e.UserProfile, "AppData", "Roaming"),
	}
	for _, base := range appBases {
		if base == "" {
			continue
		}
		for _, app := range rules.Apps {
			add(filepath.Join(base, app))
		}
	}

	return &Protection{paths: paths}
}

// joinEnv joins path elements that may be empty.
func joinEnv(base string, elems ...string) string {
	if base == "" {
		return ""
	}
	return filepath.Join(append([]string{base}, elems...)...)
}

// IsProtected reports whether path equals or is under a protected root,
// or is a drive root.
func (p *Protection) IsProtected(path string) bool {
	key := compareKey(path)
	if driveRootRe.MatchString(key) {
		return true
	}
	for _, base := range p.paths {
		if sameOrUnder(key, base) {
			return true
		}
	}
	return false
}

// IsAncestorOfProtected reports whether path is a strict ancestor of a
// protected root. Used to refuse deleting a directory that would swallow a
// protected path (safety SR-13).
func (p *Protection) IsAncestorOfProtected(path string) bool {
	key := compareKey(path)
	for _, base := range p.paths {
		if key != base && sameOrUnder(base, key) {
			return true
		}
	}
	return false
}

// List returns the protected roots (informational, for the UI).
func (p *Protection) List() []string {
	out := make([]string, 0, len(p.paths))
	for _, k := range p.paths {
		out = append(out, k)
	}
	return out
}

// compareKey canonicalizes a path for comparisons: absolute, cleaned, and
// lowercased on Windows.
func compareKey(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	if runtime.GOOS == "windows" {
		return strings.ToLower(abs)
	}
	return abs
}

// sameOrUnder reports whether a is equal to or inside base. The separator
// boundary is handled so drive roots ("c:\\") correctly match children
// without producing a doubled separator.
func sameOrUnder(a, base string) bool {
	if a == base {
		return true
	}
	sep := string(filepath.Separator)
	prefix := base
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	return strings.HasPrefix(a, prefix)
}
