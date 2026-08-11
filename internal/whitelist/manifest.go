package whitelist

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SchemaVersion is the only accepted manifest schema. Unknown future schemas
// are rejected (fail closed) rather than partially applied.
const SchemaVersion = 1

// ManifestFile describes one rules file in a manifest.
type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Manifest is the versioned index of a ruleset (bundled or remote).
type Manifest struct {
	SchemaVersion     int            `json:"schemaVersion"`
	RulesVersion      string         `json:"rulesVersion"`
	MinimumAppVersion string         `json:"minimumAppVersion,omitempty"`
	UpdatedAt         string         `json:"updatedAt,omitempty"`
	Files             []ManifestFile `json:"files"`
}

var (
	hex256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// validate checks the manifest structure. Malformed manifests are rejected.
func (m Manifest) validate() error {
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported manifest schema version %d (expected %d)", m.SchemaVersion, SchemaVersion)
	}
	if strings.TrimSpace(m.RulesVersion) == "" {
		return fmt.Errorf("manifest has no rulesVersion")
	}
	if versionParts(m.RulesVersion) == nil {
		return fmt.Errorf("manifest rulesVersion %q is not a valid version", m.RulesVersion)
	}
	if m.MinimumAppVersion != "" && versionParts(m.MinimumAppVersion) == nil {
		return fmt.Errorf("manifest minimumAppVersion %q is not a valid version", m.MinimumAppVersion)
	}
	if m.UpdatedAt != "" {
		if _, err := time.Parse(time.RFC3339, m.UpdatedAt); err != nil {
			return fmt.Errorf("manifest updatedAt %q is not an RFC3339 timestamp", m.UpdatedAt)
		}
	}
	if len(m.Files) == 0 {
		return fmt.Errorf("manifest declares no rule files")
	}
	for _, f := range m.Files {
		if filepath.Base(f.Path) != f.Path || filepath.Ext(f.Path) != ".json" {
			return fmt.Errorf("manifest file %q has an invalid path (must be a flat *.json name)", f.Path)
		}
		if !hex256Re.MatchString(f.SHA256) {
			return fmt.Errorf("manifest file %q has an invalid sha256 (want 64 lowercase hex chars)", f.Path)
		}
	}
	return nil
}

// versionParts parses a dotted numeric version ("1", "1.2", "4.2.0", or a
// date-style "2026.08.11.1", optionally v-prefixed) into its components.
// Missing components compare as 0. Returns nil for malformed versions.
func versionParts(v string) []int {
	s := strings.TrimPrefix(strings.TrimSpace(v), "v")
	s = strings.SplitN(s, "-", 2)[0] // drop pre-release suffix
	var out []int
	for _, p := range strings.Split(s, ".") {
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// compareVersions returns -1, 0 or 1 comparing a and b component-wise
// (missing components count as 0). Handles semver (4.2.0) and date-style
// rules versions (2026.08.11.1).
func compareVersions(a, b string) int {
	ap, bp := versionParts(a), versionParts(b)
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

// parseManifest parses and validates a manifest payload.
func parseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	if err := m.validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// hexEqual reports whether the lowercased hex digest matches the expected
// lowercase hex string.
func hexEqual(digest []byte, expected string) bool {
	return hex.EncodeToString(digest) == expected
}
