package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gabrielluong/create-bug/internal/config"
)

const (
	moduleURL = "github.com/gabrielluong/create-bug"
	tagsURL   = "https://api.github.com/repos/gabrielluong/create-bug/tags"
	cacheTTL  = 24 * time.Hour
)

type cacheEntry struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

func cachePath() string {
	dir := config.ConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "update_check.json")
}

// LatestVersion fetches the latest tag from GitHub (makes an HTTP call).
func LatestVersion() (string, error) {
	req, err := http.NewRequest("GET", tagsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &tags); err != nil {
		return "", err
	}
	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found")
	}
	return tags[0].Name, nil
}

// CheckCached reads the local cache to determine if an update is available.
// Returns the latest version and whether it is newer than current. No HTTP call.
func CheckCached(current string) (latest string, outdated bool) {
	path := cachePath()
	if path == "" {
		return "", false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", false
	}

	return entry.Latest, IsNewer(current, entry.Latest)
}

// RefreshCache fetches the latest version from GitHub and writes it to the cache.
// Skips the network call if the cache is still fresh. Safe to call in a goroutine.
func RefreshCache() {
	path := cachePath()
	if path == "" {
		return
	}

	// Skip if cache is still fresh.
	data, err := os.ReadFile(path)
	if err == nil {
		var entry cacheEntry
		if json.Unmarshal(data, &entry) == nil && time.Since(entry.CheckedAt) < cacheTTL {
			return
		}
	}

	latest, err := LatestVersion()
	if err != nil {
		return
	}

	entry := cacheEntry{CheckedAt: time.Now(), Latest: latest}
	out, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, out, 0o644)
}

// Run installs the latest version via `go install`. Connects stdio to the terminal.
func Run() error {
	cmd := exec.Command("go", "install", moduleURL+"@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IsNewer reports whether latest is a higher semver than current.
// Both strings may optionally have a leading "v".
func IsNewer(current, latest string) bool {
	cv := parseVersion(strings.TrimPrefix(current, "v"))
	lv := parseVersion(strings.TrimPrefix(latest, "v"))
	for i := range 3 {
		if lv[i] > cv[i] {
			return true
		}
		if lv[i] < cv[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		result[i], _ = strconv.Atoi(p)
	}
	return result
}
