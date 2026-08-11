package whitelist

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	rulesdata "github.com/kof-huskai/Junk-Fuck/rules"
)

// testEnv is a hermetic Windows-like environment for resolution tests.
func testEnv() Env {
	return Env{
		UserProfile:     `C:\Users\test`,
		AppData:         `C:\Users\test\AppData\Roaming`,
		LocalAppData:    `C:\Users\test\AppData\Local`,
		ProgramData:     `C:\ProgramData`,
		ProgramFiles:    `C:\Program Files`,
		ProgramFilesX86: `C:\Program Files (x86)`,
		WinDir:          `C:\Windows`,
		SystemRoot:      `C:\Windows`,
	}
}

func hashOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// makeSet builds a valid manifest + files for the given rules grouped by
// file name. Returns (manifest, files, rulesVersion).
func makeSet(files map[string][]Rule, version string) (Manifest, map[string][]byte) {
	payloads := make(map[string][]byte, len(files))
	manifest := Manifest{SchemaVersion: 1, RulesVersion: version}
	// Deterministic file order for stable manifest files.
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sortStrings(names)
	for _, name := range names {
		data, _ := json.Marshal(files[name])
		payloads[name] = data
		manifest.Files = append(manifest.Files, ManifestFile{Path: name, SHA256: hashOf(data)})
	}
	return manifest, payloads
}

func sortStrings(s []string) {
	sort.Strings(s)
}

func rule(id, category, app, desc, typ, env, path string) Rule {
	return Rule{
		ID:          id,
		Category:    category,
		Application: app,
		Description: desc,
		Match:       Match{Type: typ, Env: env, Path: path},
		Protection:  RuleProtection,
	}
}

// ---------- Bundled ruleset ----------

func TestBundledRulesLoad(t *testing.T) {
	w := NewWhitelist()
	if err := w.LoadBundled(); err != nil {
		t.Fatalf("bundled rules must load: %v", err)
	}
	active, bundled, _, source, count, _ := w.Status()
	if source != "bundled" {
		t.Fatalf("source = %q, want bundled", source)
	}
	if bundled != "2026.08.11.1" {
		t.Fatalf("bundledVersion = %q, want 2026.08.11.1", bundled)
	}
	if active != bundled {
		t.Fatalf("active %q != bundled %q", active, bundled)
	}
	// All 23 conservative rules must resolve in a real Windows-like env.
	if count < 20 {
		t.Fatalf("expected ~23 resolvable bundled rules, got %d", count)
	}
}

// ---------- Parsing / validation ----------

func TestInvalidSchemaRejected(t *testing.T) {
	m := Manifest{SchemaVersion: 99, RulesVersion: "1.0.0", Files: []ManifestFile{{Path: "a.json", SHA256: strings.Repeat("0", 64)}}}
	if err := m.validate(); err == nil {
		t.Fatal("schema version 99 must be rejected")
	}
}

func TestDuplicateIDsRejected(t *testing.T) {
	dupe := []Rule{
		rule("dup", "windows", "A", "d", MatchTypePrefix, "USERPROFILE", ".ssh"),
		rule("dup", "windows", "A", "d", MatchTypePrefix, "USERPROFILE", ".gnupg"),
	}
	if _, err := parseRules(mustJSON(t, dupe)); err == nil {
		t.Fatal("duplicate rule ids in one file must be rejected")
	}
	// Across files within one set too.
	manifest, files := makeSet(map[string][]Rule{
		"a.json": {rule("x", "windows", "A", "d", MatchTypePrefix, "USERPROFILE", ".ssh")},
		"b.json": {rule("x", "windows", "A", "d", MatchTypePrefix, "USERPROFILE", ".gnupg")},
	}, "1.0.0")
	w := NewWhitelist()
	if _, err := w.buildRulesetEnv(manifest, files, "remote", testEnv()); err == nil {
		t.Fatal("duplicate ids across files must be rejected")
	}
}

func TestUnknownEnvVarRejected(t *testing.T) {
	r := rule("bad-env", "windows", "A", "d", MatchTypePrefix, "SOME_SECRET_VAR", "x")
	if err := r.validate(); err == nil {
		t.Fatal("unknown env var must be rejected")
	}
}

func TestWildcardRejected(t *testing.T) {
	r := rule("wild", "windows", "A", "d", MatchTypePrefix, "", `C:\Users\*`)
	if err := r.validate(); err == nil {
		t.Fatal("wildcards must be rejected")
	}
}

func TestDriveRootRejected(t *testing.T) {
	r := rule("root", "windows", "A", "d", MatchTypePrefix, "", `C:\`)
	if err := r.validate(); err == nil {
		t.Fatal("drive-root rule must be rejected")
	}
}

func TestEnvTraversalRejected(t *testing.T) {
	r := rule("escape", "windows", "A", "d", MatchTypePrefix, "USERPROFILE", `..\..\Users\evil`)
	if err := r.validate(); err == nil {
		t.Fatal("env-based path traversal must be rejected")
	}
}

func TestBadProtectionValueRejected(t *testing.T) {
	// The security invariant: the schema cannot express deletion authority.
	var payload = []byte(`[
	  {"id":"evil","category":"windows","application":"A","description":"d",
	   "match":{"type":"prefix","env":"USERPROFILE","path":".ssh"},
	   "protection":"delete"}]`)
	if _, err := parseRules(payload); err == nil {
		t.Fatal("a rule with protection=delete must be rejected")
	}
	// Even an unknown extra field attempting to add authority is ignored by
	// the struct (no such field) and the rule still requires deny-delete.
	var payload2 = []byte(`[
	  {"id":"evil2","category":"windows","application":"A","description":"d",
	   "match":{"type":"prefix","env":"USERPROFILE","path":".ssh"},
	   "protection":"deny-delete","delete":true,"junk":true}]`)
	rules, err := parseRules(payload2)
	if err != nil {
		t.Fatalf("extra unknown fields with valid protection must parse: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestBadChecksumRejected(t *testing.T) {
	manifest, files := makeSet(map[string][]Rule{
		"a.json": {rule("ok", "windows", "A", "d", MatchTypePrefix, "USERPROFILE", ".ssh")},
	}, "1.0.0")
	// Tamper the payload after hashing.
	files["a.json"] = []byte(`[{"id":"tampered"}]`)
	w := NewWhitelist()
	if _, err := w.buildRulesetEnv(manifest, files, "remote", testEnv()); err == nil {
		t.Fatal("tampered payload must fail checksum validation")
	}
}

// ---------- Protection semantics ----------

func TestProtectsExactAndPrefix(t *testing.T) {
	manifest, files := makeSet(map[string][]Rule{
		"a.json": {
			rule("exact", "developer-tools", "npm", "npmrc", MatchTypeExact, "USERPROFILE", ".npmrc"),
			rule("prefix", "vpn-proxies", "v2rayN", "config", MatchTypePrefix, "APPDATA", "v2rayN"),
		},
	}, "1.0.0")
	w := NewWhitelist()
	rs, err := w.buildRulesetEnv(manifest, files, "remote", testEnv())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	w.mu.Lock()
	w.active = rs
	w.mu.Unlock()

	if !w.Protects(`C:\Users\test\.npmrc`) {
		t.Fatal("exact rule must protect the file")
	}
	if w.Protects(`C:\Users\test\.npmrc.bak`) {
		t.Fatal("exact rule must not protect a different file")
	}
	if !w.Protects(`C:\Users\test\AppData\Roaming\v2rayN\guiNConfig.json`) {
		t.Fatal("prefix rule must protect anything under the dir")
	}
	if w.Protects(`C:\Users\test\AppData\Roaming\v2rayNx\other`) {
		t.Fatal("prefix rule must respect the separator boundary")
	}
	if !w.ProtectsAncestor(`C:\Users\test\AppData\Roaming`) {
		t.Fatal("deleting an ancestor of a protected dir must be refused")
	}
	if w.ProtectsAncestor(`C:\Users\test\AppData\Roaming\v2rayN`) {
		t.Fatal("the protected dir itself is not its own ancestor")
	}
}

// Remote rules can protect an existing candidate (the additive invariant):
// a path the classifier might call junk becomes non-deletable.
func TestRemoteRulesCanProtectCandidate(t *testing.T) {
	manifest, files := makeSet(map[string][]Rule{
		"a.json": {
			rule("game-cache-guard", "games", "Game X", "protect save-adjacent data", MatchTypePrefix, "USERPROFILE", "Saved Games"),
		},
	}, "1.0.0")
	w := NewWhitelist()
	rs, err := w.buildRulesetEnv(manifest, files, "remote", testEnv())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	w.mu.Lock()
	w.active = rs
	w.mu.Unlock()

	candidate := `C:\Users\test\Saved Games\Game X\cache`
	if !w.Protects(candidate) {
		t.Fatal("a whitelisted candidate must be reported as protected")
	}
}

// The security invariant test: a "malicious" remote payload can never turn
// a rule into deletion authority — the schema has no such field, and
// non-whitelist protection values are rejected outright.
func TestSecurityInvariantRemoteCannotDelete(t *testing.T) {
	var evil = []byte(`[
	  {"id":"steal","category":"windows","application":"A","description":"d",
	   "match":{"type":"prefix","env":"USERPROFILE","path":"Desktop"},
	   "protection":"delete","delete":true,"force":true,"junk":true}]`)
	if _, err := parseRules(evil); err == nil {
		t.Fatal("remote payload attempting deletion authority must be rejected")
	}
}

// Game / browser / VPN whitelist rules actually protect their targets.
func TestCategoryWhitelistsWork(t *testing.T) {
	manifest, files := makeSet(map[string][]Rule{
		"games.json": {
			rule("g1", "games", "GOG Galaxy", "config", MatchTypePrefix, "LOCALAPPDATA", `GOG.com\Galaxy\config`),
		},
		"browsers.json": {
			rule("b1", "browsers", "Firefox", "profiles", MatchTypePrefix, "APPDATA", `Mozilla\Firefox\Profiles`),
		},
		"vpn-proxies.json": {
			rule("v1", "vpn-proxies", "Clash for Windows", "config", MatchTypePrefix, "APPDATA", "clash"),
		},
	}, "1.0.0")
	w := NewWhitelist()
	rs, err := w.buildRulesetEnv(manifest, files, "remote", testEnv())
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	w.mu.Lock()
	w.active = rs
	w.mu.Unlock()

	if !w.Protects(`C:\Users\test\AppData\Local\GOG.com\Galaxy\config\some.json`) {
		t.Fatal("GOG config rule must protect its target")
	}
	if !w.Protects(`C:\Users\test\AppData\Roaming\Mozilla\Firefox\Profiles\abc.default\places.sqlite`) {
		t.Fatal("Firefox profile rule must protect the profile database")
	}
	if !w.Protects(`C:\Users\test\AppData\Roaming\clash\config.yaml`) {
		t.Fatal("Clash config rule must protect its target")
	}
}

// ---------- Cache behavior ----------

func TestCorruptedCacheIgnored(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{not json`), 0o644)

	w := NewWhitelist()
	if err := w.LoadBundled(); err != nil {
		t.Fatalf("bundled load: %v", err)
	}
	before, _, _, src, _, _ := w.Status()
	if err := w.LoadCache(dir); err != nil {
		t.Fatalf("corrupted cache must be ignored, got error: %v", err)
	}
	after, _, _, src2, _, _ := w.Status()
	if after != before || src2 != src {
		t.Fatal("corrupted cache must not change the active ruleset")
	}
}

func TestValidCachePreferredOverOlderBundled(t *testing.T) {
	dir := t.TempDir()
	// A cache ruleset with a NEWER version than the bundled one.
	manifest, files := makeSet(map[string][]Rule{
		"windows.json": {rule("ssh", "windows", "OpenSSH", "d", MatchTypePrefix, "USERPROFILE", ".ssh")},
	}, "2099.01.01.1")
	writeSetToDir(t, dir, manifest, files)
	_ = os.WriteFile(filepath.Join(dir, updatedMarker), []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)

	w := NewWhitelist()
	if err := w.LoadBundled(); err != nil {
		t.Fatalf("bundled load: %v", err)
	}
	if err := w.LoadCache(dir); err != nil {
		t.Fatalf("valid cache must load: %v", err)
	}
	_, _, cached, source, _, _ := w.Status()
	if cached != "2099.01.01.1" {
		t.Fatalf("cachedVersion = %q", cached)
	}
	if source != "cache" {
		t.Fatalf("source = %q, want cache", source)
	}
}

func writeSetToDir(t *testing.T, dir string, manifest Manifest, files map[string][]byte) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, mf := range manifest.Files {
		if err := os.WriteFile(filepath.Join(dir, mf.Path), files[mf.Path], 0o644); err != nil {
			t.Fatal(err)
		}
	}
	md, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), md, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Older cache must not replace the newer bundled set.
func TestOldCacheDoesNotDowngradeBundled(t *testing.T) {
	dir := t.TempDir()
	manifest, files := makeSet(map[string][]Rule{
		"windows.json": {rule("ssh", "windows", "OpenSSH", "d", MatchTypePrefix, "USERPROFILE", ".ssh")},
	}, "1.0.0") // older than bundled 2026.08.11.1
	writeSetToDir(t, dir, manifest, files)

	w := NewWhitelist()
	if err := w.LoadBundled(); err != nil {
		t.Fatalf("bundled load: %v", err)
	}
	if err := w.LoadCache(dir); err != nil {
		t.Fatalf("load cache: %v", err)
	}
	active, _, _, source, _, _ := w.Status()
	if source != "bundled" || active != "2026.08.11.1" {
		t.Fatalf("older cache must not downgrade the bundled set (source=%s active=%s)", source, active)
	}
}

// ---------- Remote update flow ----------

func TestRemoteUpdateRoundTrip(t *testing.T) {
	manifest, files := makeSet(map[string][]Rule{
		"windows.json": {rule("ssh", "windows", "OpenSSH", "d", MatchTypePrefix, "USERPROFILE", ".ssh")},
	}, "2099.02.02.2")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if name == "manifest.json" {
			md, _ := json.Marshal(manifest)
			_, _ = w.Write(md)
			return
		}
		if data, ok := files[name]; ok {
			_, _ = w.Write(data)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cacheDir := filepath.Join(t.TempDir(), "rules")
	u := NewUpdater(cacheDir)
	u.BaseURL = server.URL
	u.AppVersion = "4.2.0"

	w := NewWhitelist()
	if err := w.LoadBundled(); err != nil {
		t.Fatalf("bundled load: %v", err)
	}
	if err := u.CheckAndUpdate(w); err != nil {
		t.Fatalf("remote update failed: %v", err)
	}

	active, _, cached, source, _, _ := w.Status()
	if source != "remote" || active != "2099.02.02.2" {
		t.Fatalf("after update: source=%s active=%s", source, active)
	}
	if cached != active {
		t.Fatalf("cachedVersion %q != active %q", cached, active)
	}

	// The cache must be complete, atomic (no .tmp leftovers) and loadable.
	if _, err := os.Stat(filepath.Join(cacheDir, "manifest.json")); err != nil {
		t.Fatalf("cache manifest missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, updatedMarker)); err != nil {
		t.Fatalf("update marker missing: %v", err)
	}
	leftovers, _ := filepath.Glob(filepath.Join(cacheDir, "*.tmp"))
	if len(leftovers) != 0 {
		t.Fatalf("temp files left in cache: %v", leftovers)
	}

	// A fresh whitelist must pick the cache back up (restart behavior).
	w2 := NewWhitelist()
	if err := w2.LoadBundled(); err != nil {
		t.Fatalf("bundled load 2: %v", err)
	}
	if err := w2.LoadCache(cacheDir); err != nil {
		t.Fatalf("cache reload: %v", err)
	}
	if got, _, _, source2, _, _ := w2.Status(); source2 != "cache" || got != "2099.02.02.2" {
		t.Fatalf("restart cache load: source=%s active=%s", source2, got)
	}
}

func TestRemoteOfflineKeepsLastValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	u := NewUpdater(filepath.Join(t.TempDir(), "rules"))
	u.BaseURL = server.URL
	u.AppVersion = "4.2.0"

	w := NewWhitelist()
	if err := w.LoadBundled(); err != nil {
		t.Fatalf("bundled load: %v", err)
	}
	before, _, _, src, _, _ := w.Status()

	if err := u.CheckAndUpdate(w); err == nil {
		t.Fatal("unreachable remote must return an error")
	} else if !strings.Contains(err.Error(), "not reachable") {
		t.Fatalf("unexpected error: %v", err)
	}
	after, _, _, src2, _, _ := w.Status()
	if after != before || src2 != src {
		t.Fatal("failed update must keep the last valid ruleset")
	}
}

func TestRemoteIncompatibleAppVersionKept(t *testing.T) {
	manifest, files := makeSet(map[string][]Rule{
		"windows.json": {rule("ssh", "windows", "OpenSSH", "d", MatchTypePrefix, "USERPROFILE", ".ssh")},
	}, "2099.03.03.3")
	manifest.MinimumAppVersion = "99.0.0"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if name == "manifest.json" {
			md, _ := json.Marshal(manifest)
			_, _ = w.Write(md)
			return
		}
		if data, ok := files[name]; ok {
			_, _ = w.Write(data)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	u := NewUpdater(filepath.Join(t.TempDir(), "rules"))
	u.BaseURL = server.URL
	u.AppVersion = "4.2.0"

	w := NewWhitelist()
	if err := w.LoadBundled(); err != nil {
		t.Fatalf("bundled load: %v", err)
	}
	before, _, _, _, _, _ := w.Status()
	if err := u.CheckAndUpdate(w); err == nil {
		t.Fatal("incompatible rules must be refused")
	}
	after, _, _, _, _, _ := w.Status()
	if after != before {
		t.Fatal("incompatible remote must not change the active ruleset")
	}
}

func TestTTL(t *testing.T) {
	w := NewWhitelist()
	if !NewUpdater(t.TempDir()).NeedsUpdate(w) {
		t.Fatal("never-updated whitelist must need an update")
	}
	// Set an explicit past timestamp so the test is deterministic (no
	// dependence on clock granularity).
	w.mu.Lock()
	w.lastUpdated = time.Now().Add(-2 * time.Second)
	w.mu.Unlock()
	u := NewUpdater(t.TempDir())
	u.TTL = time.Second
	if !u.NeedsUpdate(w) {
		t.Fatal("expired TTL must force an update")
	}
	u.TTL = 24 * time.Hour
	if u.NeedsUpdate(w) {
		t.Fatal("a recent update with a long TTL must not need an update")
	}
}

// ---------- Version comparison ----------

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"4.2.0", "4.2.0", 0},
		{"4.2.0", "4.1.0", 1},
		{"4.1.9", "4.1.10", -1},
		{"v4.2.0", "4.2.0", 0},
		{"5", "4.9.9", 1},
		{"2026.08.11.1", "2026.08.11.1", 0},
		{"2026.08.11.1", "2026.08.11.2", -1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// ---------- Remote version-gating (same repo, manifest-first) ----------

// A remote manifest whose rulesVersion is not newer must NOT trigger any
// rule-file download — the updater only fetches the manifest.
func TestRemoteSameVersionDoesNothing(t *testing.T) {
	ruleFileHits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if name == "manifest.json" {
			m := Manifest{
				SchemaVersion:     1,
				RulesVersion:      "2026.08.11.1",
				MinimumAppVersion: "4.2.0",
				Files:             []ManifestFile{{Path: "windows.json", SHA256: strings.Repeat("0", 64)}},
			}
			md, _ := json.Marshal(m)
			_, _ = w.Write(md)
			return
		}
		ruleFileHits++
		http.NotFound(w, r)
	}))
	defer server.Close()

	u := NewUpdater(filepath.Join(t.TempDir(), "rules"))
	u.BaseURL = server.URL
	u.AppVersion = "4.2.0"

	w := NewWhitelist()
	if err := w.LoadBundled(); err != nil {
		t.Fatalf("bundled load: %v", err)
	}
	before, _, _, src, _, _ := w.Status()

	if err := u.CheckAndUpdate(w); err != nil {
		t.Fatalf("same-version check must succeed: %v", err)
	}
	if ruleFileHits != 0 {
		t.Fatalf("same/older remote version must not download rule files (got %d hits)", ruleFileHits)
	}
	after, _, _, src2, _, _ := w.Status()
	if after != before || src2 != src {
		t.Fatal("same-version remote must not change the active ruleset")
	}
}

func TestRemoteOlderVersionDoesNothing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if name == "manifest.json" {
			m := Manifest{
				SchemaVersion: 1,
				RulesVersion:  "1.0.0",
				Files:         []ManifestFile{{Path: "windows.json", SHA256: strings.Repeat("0", 64)}},
			}
			md, _ := json.Marshal(m)
			_, _ = w.Write(md)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	u := NewUpdater(filepath.Join(t.TempDir(), "rules"))
	u.BaseURL = server.URL
	u.AppVersion = "4.2.0"

	w := NewWhitelist()
	if err := w.LoadBundled(); err != nil {
		t.Fatalf("bundled load: %v", err)
	}
	if err := u.CheckAndUpdate(w); err != nil {
		t.Fatalf("older-version check must succeed: %v", err)
	}
	_, _, _, src, _, _ := w.Status()
	if src != "bundled" {
		t.Fatalf("older remote must keep the bundled ruleset (source=%s)", src)
	}
}

// A manifest can never point outside the repository's /rules/ path: paths
// with traversal or absolute prefixes are rejected at validation.
func TestManifestCannotEscapeRulesPath(t *testing.T) {
	bad := []string{"../evil.json", "./evil.json", "/etc/passwd.json", "sub/windows.json", "C:\\evil.json"}
	for _, p := range bad {
		m := Manifest{
			SchemaVersion: 1,
			RulesVersion:  "1.0.0",
			Files:         []ManifestFile{{Path: p, SHA256: strings.Repeat("0", 64)}},
		}
		if err := m.validate(); err == nil {
			t.Errorf("manifest file path %q must be rejected", p)
		}
	}
}

// The bundled manifest ships from the repo-root rules/ package and its
// sha256 entries must match the embedded files (self-consistency).
func TestBundledManifestMatchesEmbeddedFiles(t *testing.T) {
	all, err := rulesdata.Bundled()
	if err != nil {
		t.Fatalf("embedded rules: %v", err)
	}
	manifest, err := parseManifest(all["manifest.json"])
	if err != nil {
		t.Fatalf("bundled manifest invalid: %v", err)
	}
	for _, mf := range manifest.Files {
		payload, ok := all[mf.Path]
		if !ok {
			t.Errorf("manifest lists %q but it is not embedded", mf.Path)
			continue
		}
		sum := sha256.Sum256(payload)
		if !hexEqual(sum[:], mf.SHA256) {
			t.Errorf("sha256 mismatch for embedded %q", mf.Path)
		}
	}
}
