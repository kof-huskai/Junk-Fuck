package whitelist

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/kof-huskai/Junk-Fuck/internal/filesystem"
	rulesdata "github.com/kof-huskai/Junk-Fuck/rules"
)

// updatedMarker is the cache file holding the RFC3339 timestamp of the last
// successful remote update (its mtime also drives the update TTL).
const updatedMarker = ".updated"

// resolvedRule is a rule whose path has been resolved for this environment.
type resolvedRule struct {
	key   string // canonical comparison key
	exact bool
}

// ruleset is one validated rule set with its resolved matchers.
type ruleset struct {
	version      string
	source       string // bundled | cache | remote
	rules        []Rule
	resolved     []resolvedRule
	manifest     Manifest
	filePayloads map[string][]byte
}

// Whitelist is the active protection-whitelist set. It is safe for
// concurrent use: scans read via Protects/ProtectsAncestor while a
// background update may swap the active set.
type Whitelist struct {
	mu             sync.RWMutex
	active         ruleset
	bundledVersion string
	cachedVersion  string
	lastChecked    time.Time
	lastUpdated    time.Time
}

// NewWhitelist returns an empty whitelist. Call LoadBundled and LoadCache
// before use; the hard protection layer never depends on this store.
func NewWhitelist() *Whitelist {
	return &Whitelist{}
}

// LoadBundled parses, validates and activates the whitelist bundled into the
// binary. It is best-effort: on failure the store stays empty and the
// built-in hard protection remains fully active.
func (w *Whitelist) LoadBundled() error {
	manifest, files, err := readBundledSet()
	if err != nil {
		return err
	}
	rs, err := w.buildRuleset(manifest, files, "bundled")
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.bundledVersion = rs.version
	if w.active.source == "" || compareVersions(rs.version, w.active.version) > 0 {
		w.active = rs
	}
	return nil
}

// LoadCache loads a previously validated ruleset from dir (optional). A
// valid cache newer than the bundled set wins; anything invalid is ignored
// and the bundled (or previously loaded) set stays active.
func (w *Whitelist) LoadCache(dir string) error {
	if dir == "" {
		return nil
	}
	manifestData, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil // no cache yet
	}
	manifest, err := parseManifest(manifestData)
	if err != nil {
		return nil
	}
	files, err := readFilesFromDir(dir, manifest)
	if err != nil {
		return nil
	}
	rs, err := w.buildRuleset(manifest, files, "cache")
	if err != nil {
		return nil
	}
	if w.bundledVersion != "" && compareVersions(rs.version, w.bundledVersion) < 0 {
		return nil // cache is older than the bundled set; keep bundled
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.cachedVersion = rs.version
	if w.active.source == "" || compareVersions(rs.version, w.active.version) >= 0 {
		w.active = rs
	}
	if st, err := os.Stat(filepath.Join(dir, updatedMarker)); err == nil {
		w.lastUpdated = st.ModTime()
	}
	return nil
}

// Adopt activates a newly validated remote ruleset and persists it to the
// cache atomically. Called only after the whole set passed validation.
func (w *Whitelist) Adopt(rs ruleset, cacheDir string) error {
	w.mu.Lock()
	w.active = rs
	w.cachedVersion = rs.version
	w.lastChecked = time.Now()
	w.lastUpdated = time.Now()
	w.mu.Unlock()

	if cacheDir != "" {
		if err := persistRuleset(cacheDir, rs.manifest, rs.filePayloads); err != nil {
			// The in-memory ruleset is already active; a failed persist only
			// means the cache won't survive a restart (bundled remains).
			return fmt.Errorf("rules activated in memory but cache write failed: %w", err)
		}
	}
	return nil
}

// Protects reports whether path is exactly a protected rule or lies under a
// prefix rule.
func (w *Whitelist) Protects(path string) bool {
	key := filesystem.CompareKey(path)
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, r := range w.active.resolved {
		if r.exact {
			if key == r.key {
				return true
			}
		} else if filesystem.SameOrUnder(key, r.key) {
			return true
		}
	}
	return false
}

// ProtectsAncestor reports whether path is a strict ancestor of a protected
// rule — used to refuse deleting a directory that would swallow a
// whitelisted path.
func (w *Whitelist) ProtectsAncestor(path string) bool {
	key := filesystem.CompareKey(path)
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, r := range w.active.resolved {
		if key != r.key && filesystem.SameOrUnder(r.key, key) {
			return true
		}
	}
	return false
}

// Status returns a snapshot for the Settings UI.
func (w *Whitelist) Status() (activeVersion, bundledVersion, cachedVersion, source string, ruleCount int, lastUpdated time.Time) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.active.version, w.bundledVersion, w.cachedVersion, w.active.source, len(w.active.resolved), w.lastUpdated
}

// LastUpdated returns when the active ruleset was last refreshed remotely
// (zero time if it only came from the bundle).
func (w *Whitelist) LastUpdated() time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastUpdated
}

// buildRuleset validates a manifest + file payloads and resolves every rule
// against the process environment.
func (w *Whitelist) buildRuleset(manifest Manifest, files map[string][]byte, source string) (ruleset, error) {
	return w.buildRulesetEnv(manifest, files, source, osEnv())
}

// buildRulesetEnv validates a manifest + file payloads and resolves every
// rule against an explicit environment (injected so tests run hermetic).
// Duplicate IDs across the set are rejected; rules that cannot resolve
// (missing env var) are dropped.
func (w *Whitelist) buildRulesetEnv(manifest Manifest, files map[string][]byte, source string, env Env) (ruleset, error) {
	if err := manifest.validate(); err != nil {
		return ruleset{}, err
	}
	seen := map[string]bool{}
	var all []Rule
	for _, mf := range manifest.Files {
		payload, ok := files[mf.Path]
		if !ok {
			return ruleset{}, fmt.Errorf("manifest references %q but the payload is missing", mf.Path)
		}
		sum := sha256.Sum256(payload)
		if !hexEqual(sum[:], mf.SHA256) {
			return ruleset{}, fmt.Errorf("checksum mismatch for %q", mf.Path)
		}
		rules, err := parseRules(payload)
		if err != nil {
			return ruleset{}, fmt.Errorf("%s: %w", mf.Path, err)
		}
		for _, r := range rules {
			if seen[r.ID] {
				return ruleset{}, fmt.Errorf("duplicate rule id %q across the ruleset", r.ID)
			}
			seen[r.ID] = true
			all = append(all, r)
		}
	}

	resolved := make([]resolvedRule, 0, len(all))
	for _, r := range all {
		key, ok := r.resolvePath(env)
		if !ok {
			continue // unresolvable in this environment (e.g. var unset)
		}
		resolved = append(resolved, resolvedRule{key: key, exact: r.Match.Type == MatchTypeExact})
	}
	sort.Slice(resolved, func(i, j int) bool { return resolved[i].key < resolved[j].key })

	return ruleset{
		version:      manifest.RulesVersion,
		source:       source,
		rules:        all,
		resolved:     resolved,
		manifest:     manifest,
		filePayloads: files,
	}, nil
}

// readBundledSet loads the embedded manifest and its referenced files from
// the root rules/ package (the single source of truth for the bundled
// ruleset and the remote update path). buildRuleset re-verifies every
// sha256 against the manifest.
func readBundledSet() (Manifest, map[string][]byte, error) {
	all, err := rulesdata.Bundled()
	if err != nil {
		return Manifest{}, nil, err
	}
	manifestData, ok := all["manifest.json"]
	if !ok {
		return Manifest{}, nil, fmt.Errorf("bundled ruleset is missing manifest.json")
	}
	delete(all, "manifest.json")
	manifest, err := parseManifest(manifestData)
	if err != nil {
		return Manifest{}, nil, err
	}
	return manifest, all, nil
}

func readFilesFromDir(dir string, manifest Manifest) (map[string][]byte, error) {
	out := make(map[string][]byte, len(manifest.Files))
	for _, mf := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(dir, mf.Path))
		if err != nil {
			return nil, err
		}
		out[mf.Path] = data
	}
	return out, nil
}

// persistRuleset writes the set to dir atomically: every file is written to
// a temp name first; only after all temp files are written are they renamed
// into place, with manifest.json renamed last as the commit point. A partial
// download can never become the active cache.
func persistRuleset(dir string, manifest Manifest, files map[string][]byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	type entry struct {
		name string
		data []byte
	}
	var entries []entry
	for _, mf := range manifest.Files {
		entries = append(entries, entry{name: mf.Path, data: files[mf.Path]})
	}
	manifestData, err := jsonMarshal(manifest)
	if err != nil {
		return err
	}
	entries = append(entries, entry{name: "manifest.json", data: manifestData})

	// Write all temp files first.
	for _, e := range entries {
		tmp := filepath.Join(dir, e.name+".tmp")
		if err := os.WriteFile(tmp, e.data, 0o644); err != nil {
			cleanupTemps(dir, manifest)
			return err
		}
	}
	// Then rename them into place (manifest last = commit point).
	for _, e := range entries {
		if e.name == "manifest.json" {
			continue
		}
		if err := os.Rename(filepath.Join(dir, e.name+".tmp"), filepath.Join(dir, e.name)); err != nil {
			cleanupTemps(dir, manifest)
			return err
		}
	}
	if err := os.Rename(filepath.Join(dir, "manifest.json.tmp"), filepath.Join(dir, "manifest.json")); err != nil {
		cleanupTemps(dir, manifest)
		return err
	}
	// Touch the update marker last.
	return os.WriteFile(filepath.Join(dir, updatedMarker), []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)
}

func cleanupTemps(dir string, manifest Manifest) {
	for _, mf := range manifest.Files {
		_ = os.Remove(filepath.Join(dir, mf.Path+".tmp"))
	}
	_ = os.Remove(filepath.Join(dir, "manifest.json.tmp"))
}

// jsonMarshal is a small helper so the persisted manifest stays pretty-printed.
func jsonMarshal(m Manifest) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// osEnv builds the resolution environment from the process environment.
func osEnv() Env {
	return Env{
		UserProfile:     os.Getenv("USERPROFILE"),
		AppData:         os.Getenv("APPDATA"),
		LocalAppData:    os.Getenv("LOCALAPPDATA"),
		ProgramData:     os.Getenv("ProgramData"),
		ProgramFiles:    os.Getenv("ProgramFiles"),
		ProgramFilesX86: os.Getenv("ProgramFiles(x86)"),
		WinDir:          os.Getenv("WINDIR"),
		SystemRoot:      os.Getenv("SYSTEMROOT"),
	}
}
