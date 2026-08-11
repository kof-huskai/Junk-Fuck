package whitelist

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultRemoteBaseURL is the raw GitHub path of the rules directory in
// THIS repository (no separate repository). The same files that are bundled
// into the binary are fetched from here; the updater derives rule-file URLs
// only from validated manifest entries under this base, so remote content
// can never point outside the repository's /rules/ path.
const DefaultRemoteBaseURL = "https://raw.githubusercontent.com/kof-huskai/Junk-Fuck/main/rules"

// DefaultRulesTTL is how often the app re-checks the remote whitelist. A
// check happens at most once per TTL per launch; manual "Check for updates"
// always forces a fresh check.
const DefaultRulesTTL = 24 * time.Hour

// ErrNoRemote is returned when the remote repository is unreachable or
// lacks a manifest (e.g. it has not been published yet). Callers keep the
// last valid ruleset.
var ErrNoRemote = errors.New("rules repository is not reachable")

// ErrIncompatible is returned when the remote manifest requires a newer app
// version than the running build. The current ruleset is kept.
var ErrIncompatible = errors.New("remote rules require a newer application version")

// Updater fetches, validates and atomically activates remote whitelist
// rules. It never blocks application startup (callers run it in the
// background) and never replaces a valid ruleset with invalid content.
type Updater struct {
	BaseURL    string
	CacheDir   string
	AppVersion string
	TTL        time.Duration
	Client     *http.Client

	mu        sync.Mutex
	lastError error
}

// NewUpdater builds an updater for the given cache directory. The HTTP
// timeouts are short so a manual "Check for updates" never hangs the UI on
// an unreachable repository (which today falls back to the bundled rules).
func NewUpdater(cacheDir string) *Updater {
	return &Updater{
		BaseURL:  DefaultRemoteBaseURL,
		CacheDir: cacheDir,
		TTL:      DefaultRulesTTL,
		Client:   &http.Client{Timeout: 8 * time.Second},
	}
}

// NeedsUpdate reports whether a remote check should run now: the first time
// (never updated) or when the TTL since the last successful update elapsed.
// Manual refreshes bypass this by calling CheckAndUpdate directly.
func (u *Updater) NeedsUpdate(w *Whitelist) bool {
	last := w.LastUpdated()
	if last.IsZero() {
		return true
	}
	return time.Since(last) >= u.ttl()
}

func (u *Updater) ttl() time.Duration {
	if u.TTL <= 0 {
		return DefaultRulesTTL
	}
	return u.TTL
}

// CheckAndUpdate fetches ONLY the remote manifest first and compares its
// rulesVersion against the active set. When the remote version is not
// newer, nothing is downloaded and the active ruleset stays unchanged. For a
// newer version it downloads the listed rule files, validates everything
// (schema, app compatibility, hashes, rule structure), and only then
// activates the new set and persists it atomically. Any failure keeps the
// previously valid ruleset active and returns the cause.
func (u *Updater) CheckAndUpdate(w *Whitelist) error {
	manifest, files, newer, err := u.fetchRemote(w)
	if err != nil {
		u.setError(err)
		return err
	}
	if !newer {
		// Remote rulesVersion <= active: up to date, nothing to do (and no
		// rule files were downloaded).
		u.setError(nil)
		return nil
	}

	// The ruleset must be compatible with this application build; remote
	// content can never skip a version requirement.
	if manifest.MinimumAppVersion != "" && compareVersions(u.AppVersion, manifest.MinimumAppVersion) < 0 {
		err := fmt.Errorf("%w: app %s < required %s", ErrIncompatible, u.AppVersion, manifest.MinimumAppVersion)
		u.setError(err)
		return err
	}

	rs, err := w.buildRuleset(manifest, files, "remote")
	if err != nil {
		u.setError(err)
		return err
	}

	if err := w.Adopt(rs, u.CacheDir); err != nil {
		// Adopt already activated the newer in-memory set; a cache-write
		// failure is surfaced as a warning (see LastError/snapshot), not a
		// failed update — the app keeps the new rules for this session and
		// falls back to bundled on the next launch.
		u.setError(err)
		return nil
	}
	u.setError(nil)
	return nil
}

// LastError returns the last update failure (nil when the last check
// succeeded or never ran). Used by the service to explain the status.
func (u *Updater) LastError() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.lastError
}

func (u *Updater) setError(err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.lastError = err
}

// fetchRemote downloads ONLY the remote manifest, validates it, and decides
// whether a download is needed by comparing the remote rulesVersion against
// the active ruleset. When the remote version is not newer it returns
// newer=false with no rule files downloaded. Rule-file URLs are derived only
// from validated manifest entries joined onto the configured base, so
// remote content cannot escape the repository's /rules/ path.
func (u *Updater) fetchRemote(w *Whitelist) (Manifest, map[string][]byte, bool, error) {
	base := strings.TrimRight(u.BaseURL, "/")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	manifestData, err := u.get(ctx, base+"/manifest.json")
	if err != nil {
		if isNotFound(err) {
			return Manifest{}, nil, false, ErrNoRemote
		}
		return Manifest{}, nil, false, fmt.Errorf("%w: %v", ErrNoRemote, err)
	}
	manifest, err := parseManifest(manifestData)
	if err != nil {
		return Manifest{}, nil, false, fmt.Errorf("remote manifest rejected: %w", err)
	}

	activeVersion, _, _, _, _, _ := w.Status()
	if compareVersions(manifest.RulesVersion, activeVersion) <= 0 {
		return manifest, nil, false, nil // not newer: skip the download
	}

	files := make(map[string][]byte, len(manifest.Files))
	for _, mf := range manifest.Files {
		data, err := u.get(ctx, base+"/"+mf.Path)
		if err != nil {
			return Manifest{}, nil, false, fmt.Errorf("cannot download %q: %w", mf.Path, err)
		}
		sum := sha256.Sum256(data)
		if !hexEqual(sum[:], mf.SHA256) {
			return Manifest{}, nil, false, fmt.Errorf("checksum mismatch for %q: download rejected", mf.Path)
		}
		files[mf.Path] = data
	}
	return manifest, files, true, nil
}

func (u *Updater) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

var errNotFound = errors.New("not found")

func isNotFound(err error) bool {
	return errors.Is(err, errNotFound)
}
