// Package rules is the bundled protection-whitelist DATA SOURCE. The JSON
// files in this directory are the single source of truth for the bundled
// ruleset AND for remote updates: the updater fetches the same files from
// the raw GitHub path `rules/` in THIS repository (no separate repository).
//
// The files are embedded into the Junk-Fuck binary, so a known-safe ruleset
// is always available offline. This package only exposes raw bytes; the
// parsing/validation engine lives in internal/whitelist (importing this
// package would create a cycle the other way).
//
// Versioning convention: when the bundled rules change, bump
// `manifest.json` → `rulesVersion` (e.g. 2026.08.15.2). Never rely on the
// git commit SHA alone — the app compares explicit rules versions.
package rules

import (
	"embed"
	"fmt"
	"sort"
)

//go:embed *.json
var bundled embed.FS

// Bundled returns every embedded ruleset file keyed by file name
// (manifest.json plus each category file). The caller validates and parses
// them; a missing manifest is a packaging error.
func Bundled() (map[string][]byte, error) {
	entries, err := bundled.ReadDir(".")
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := bundled.ReadFile(e.Name())
		if err != nil {
			return nil, fmt.Errorf("cannot read embedded rules file %q: %w", e.Name(), err)
		}
		out[e.Name()] = data
	}
	if _, ok := out["manifest.json"]; !ok {
		return nil, fmt.Errorf("bundled ruleset is missing manifest.json")
	}
	return out, nil
}

// FileNames returns the sorted embedded file names (informational, used by
// tests and tooling).
func FileNames() []string {
	entries, err := bundled.ReadDir(".")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}
