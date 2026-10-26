package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const repoAPI = "https://api.github.com/repos/wkerekes-godaddy/betterpd/releases/latest"

type Release struct {
	Version  string
	AssetURL string
	Newer    bool
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func CheckLatest(currentVersion string) (*Release, error) {
	if currentVersion == "" || currentVersion == "dev" {
		return nil, nil
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", repoAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}

	if !isNewer(currentVersion, rel.TagName) {
		return &Release{Version: rel.TagName, Newer: false}, nil
	}

	assetURL := findAssetURL(rel.Assets)
	return &Release{
		Version:  rel.TagName,
		AssetURL: assetURL,
		Newer:    true,
	}, nil
}

func SelfUpdate(assetURL string) error {
	if assetURL == "" {
		return fmt.Errorf("no download URL for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(assetURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find current executable: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("cannot resolve executable path: %w", err)
	}

	info, err := os.Stat(exe)
	if err != nil {
		return err
	}

	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".betterpd-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("download write failed: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, info.Mode()); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, exe); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot replace binary: %w", err)
	}

	return nil
}

func findAssetURL(assets []ghAsset) string {
	target := fmt.Sprintf("betterpd-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		target += ".exe"
	}
	for _, a := range assets {
		if a.Name == target {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

// isNewer returns true if remote is a higher semver than current.
func isNewer(current, remote string) bool {
	cur := parseSemver(current)
	rem := parseSemver(remote)
	if cur == nil || rem == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if rem[i] > cur[i] {
			return true
		}
		if rem[i] < cur[i] {
			return false
		}
	}
	return false
}

func parseSemver(s string) []int {
	s = strings.TrimPrefix(s, "v")
	// Strip git-describe suffixes like "-dirty" or "-3-gabcdef"
	if i := strings.Index(s, "-"); i >= 0 {
		s = s[:i]
	}
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	result := make([]int, 3)
	for i, p := range parts {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil
			}
			n = n*10 + int(c-'0')
		}
		result[i] = n
	}
	return result
}
