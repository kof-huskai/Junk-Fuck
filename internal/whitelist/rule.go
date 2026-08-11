// Package rules implements the PROTECTION whitelist engine. This is a
// safety-only system: remote rule data may ADD protection, never deletion
// authority. The schema has no field capable of marking anything for
// cleanup — protection ("deny-delete") is the only allowed value.
//
// Hard-coded application safety (internal/protection) always wins; the
// whitelist is additive on top of it.
package whitelist

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kof-huskai/Junk-Fuck/internal/filesystem"
)

// Allowed categories for bundled and remote rules. Keep this list small and
// explicit; unknown categories are rejected.
var allowedCategories = map[string]bool{
	"windows":         true,
	"games":           true,
	"launchers":       true,
	"browsers":        true,
	"developer-tools": true,
	"vpn-proxies":     true,
}

// allowedEnvVars is the allowlist of Windows environment variables rules may
// reference. Arbitrary environment expansion from remote content is never
// allowed — an unknown variable makes the rule unresolvable (skipped).
var allowedEnvVars = map[string]bool{
	"USERPROFILE":      true,
	"APPDATA":          true,
	"LOCALAPPDATA":     true,
	"PROGRAMDATA":      true,
	"PROGRAMFILES":     true,
	"PROGRAMFILES_X86": true,
	"WINDIR":           true,
	"SYSTEMROOT":       true,
}

// RuleProtection is the only allowed protection value. The schema is
// deliberately incapable of expressing deletion authority.
const RuleProtection = "deny-delete"

// Match semantics. Rules are declarative path matches only — no code, no
// shell, no scripts, no arbitrary regexes.
const (
	MatchTypeExact  = "exact"
	MatchTypePrefix = "prefix"
)

var (
	ruleIDRe     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	windowsAbsRe = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
	driveRootRe  = regexp.MustCompile(`^[A-Za-z]:[\\/]*$`)
)

// Rule is one declarative protection rule.
type Rule struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Application string `json:"application"`
	Description string `json:"description"`
	Match       Match  `json:"match"`
	Protection  string `json:"protection"`
}

// Match describes the protected location.
type Match struct {
	// Type is "exact" (that path only) or "prefix" (the path and everything
	// under it).
	Type string `json:"type"`
	// Env is an optional allowlisted environment variable; when set, Path is
	// resolved relative to it.
	Env string `json:"env,omitempty"`
	// Path is a Windows path (backslashes or forward slashes).
	Path string `json:"path,omitempty"`
}

// Env provides the resolved Windows environment values used to resolve
// rules. Empty values make env-based rules unresolvable (skipped silently).
type Env struct {
	UserProfile     string
	AppData         string
	LocalAppData    string
	ProgramData     string
	ProgramFiles    string
	ProgramFilesX86 string
	WinDir          string
	SystemRoot      string
}

func (e Env) get(name string) string {
	switch name {
	case "USERPROFILE":
		return e.UserProfile
	case "APPDATA":
		return e.AppData
	case "LOCALAPPDATA":
		return e.LocalAppData
	case "PROGRAMDATA":
		return e.ProgramData
	case "PROGRAMFILES":
		return e.ProgramFiles
	case "PROGRAMFILES_X86":
		return e.ProgramFilesX86
	case "WINDIR":
		return e.WinDir
	case "SYSTEMROOT":
		return e.SystemRoot
	}
	return ""
}

// validate checks a single rule's structure. Unknown or malformed rules are
// rejected, never silently accepted.
func (r Rule) validate() error {
	if !ruleIDRe.MatchString(r.ID) {
		return fmt.Errorf("rule id %q is invalid (use lowercase letters, digits and dashes)", r.ID)
	}
	if !allowedCategories[r.Category] {
		return fmt.Errorf("rule %q: unknown category %q", r.ID, r.Category)
	}
	if strings.TrimSpace(r.Application) == "" {
		return fmt.Errorf("rule %q: application must not be empty", r.ID)
	}
	if strings.TrimSpace(r.Description) == "" {
		return fmt.Errorf("rule %q: description must not be empty", r.ID)
	}
	if r.Protection != RuleProtection {
		return fmt.Errorf("rule %q: protection %q is not allowed (only %q exists — rules can never add deletion authority)", r.ID, r.Protection, RuleProtection)
	}
	switch r.Match.Type {
	case MatchTypeExact, MatchTypePrefix:
	default:
		return fmt.Errorf("rule %q: unsupported matcher %q", r.ID, r.Match.Type)
	}
	if r.Match.Env != "" && !allowedEnvVars[r.Match.Env] {
		return fmt.Errorf("rule %q: unknown environment variable %q (allowlist: %v)", r.ID, r.Match.Env, envVarNames())
	}
	if strings.ContainsAny(r.Match.Path, "*?") {
		return fmt.Errorf("rule %q: wildcards are not allowed in protected paths (%q)", r.ID, r.Match.Path)
	}
	if r.Match.Env == "" {
		if !windowsAbsRe.MatchString(r.Match.Path) {
			return fmt.Errorf("rule %q: path %q must be an absolute Windows path (or use an allowlisted %%VAR%%)", r.ID, r.Match.Path)
		}
	} else if strings.TrimSpace(r.Match.Path) == "" {
		return fmt.Errorf("rule %q: env-based rules need a path below the variable (bare roots are too broad)", r.ID)
	}
	// Drive-root rules would disable most cleaning on a whole volume.
	if driveRootRe.MatchString(filepath.FromSlash(r.Match.Path)) {
		return fmt.Errorf("rule %q: drive-root paths are too broad (%q)", r.ID, r.Match.Path)
	}
	// Env-relative paths must not escape the variable's root with "..".
	if r.Match.Env != "" {
		for _, part := range strings.FieldsFunc(r.Match.Path, func(c rune) bool { return c == '/' || c == '\\' }) {
			if part == ".." {
				return fmt.Errorf("rule %q: env-based paths must not contain '..' (%q)", r.ID, r.Match.Path)
			}
		}
	}
	return nil
}

// resolvePath turns the rule into a canonical comparison key for the given
// environment. ok=false means the rule cannot be resolved in this
// environment (e.g. the variable is unset) and it must simply not match.
func (r Rule) resolvePath(env Env) (key string, ok bool) {
	raw := r.Match.Path
	if r.Match.Env != "" {
		base := env.get(r.Match.Env)
		if base == "" {
			return "", false
		}
		raw = filepath.Join(base, filepath.FromSlash(r.Match.Path))
	} else {
		raw = filepath.FromSlash(raw)
	}

	cleaned := filepath.Clean(raw)
	// Drive-root-only rules (e.g. C:\) are rejected: they would disable
	// most cleaning on that volume.
	if driveRootRe.MatchString(cleaned) {
		return "", false
	}
	if !windowsAbsRe.MatchString(cleaned) {
		return "", false
	}
	return filesystem.CompareKey(cleaned), true
}

func envVarNames() []string {
	out := make([]string, 0, len(allowedEnvVars))
	for k := range allowedEnvVars {
		out = append(out, k)
	}
	return out
}

// parseRules parses and validates a rules file payload.
func parseRules(data []byte) ([]Rule, error) {
	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("invalid rules JSON: %w", err)
	}
	seen := map[string]bool{}
	for _, r := range rules {
		if err := r.validate(); err != nil {
			return nil, err
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("duplicate rule id %q", r.ID)
		}
		seen[r.ID] = true
	}
	return rules, nil
}
