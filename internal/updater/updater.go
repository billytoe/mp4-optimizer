package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/minio/selfupdate"
)

// UpdateInfo represents the JSON structure of the update manifest
type UpdateInfo struct {
	Version             string `json:"version"`
	DownloadURLWindows  string `json:"download_url_windows"`
	DownloadURLMac      string `json:"download_url_mac"`
	DownloadURLMacArm64 string `json:"download_url_mac_arm64"` // Optional: detailed architecture support
	ReleaseNotes        string `json:"release_notes"`
}

// CheckResult holds the result of checking for updates
type CheckResult struct {
	Available    bool   `json:"available"`
	Version      string `json:"version"`
	ReleaseNotes string `json:"release_notes"`
	DownloadURL  string `json:"download_url"`
	Error        string `json:"error,omitempty"`
}

// CheckUpdate checks if a newer version is available
func CheckUpdate(currentVersion string, updateURL string) (*CheckResult, error) {
	resp, err := http.Get(updateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch update info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	var info UpdateInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}

	// Simple string comparison for now, or assume semantic versioning
	// "v1.0.1" > "v1.0.0"
	// Let's strip "v" prefix if present for robust comparison if needed, but string compare works for simple cases
	// Ideally use semver lib, but let's keep it simple: if strings are different, it's an update?
	// No, checking if remote > current.
	// For simplicity in this iteration: if remote != current, assume update (or strictly greater)

	if compareVersions(info.Version, currentVersion) > 0 {
		url := ""
		if runtime.GOOS == "windows" {
			url = info.DownloadURLWindows
		} else if runtime.GOOS == "darwin" {
			// Check architecture
			if runtime.GOARCH == "arm64" && info.DownloadURLMacArm64 != "" {
				url = info.DownloadURLMacArm64
			} else {
				url = info.DownloadURLMac
			}
		}

		return &CheckResult{
			Available:    true,
			Version:      info.Version,
			ReleaseNotes: info.ReleaseNotes,
			DownloadURL:  url,
		}, nil
	}

	return &CheckResult{Available: false}, nil
}

// ApplyUpdate downloads and applies the update
func ApplyUpdate(downloadURL string) error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Apply update
	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		// selfupdate might fail if binary is not writable or signature check fails (if configured)
		// Usually works for unsigned binaries if no options passed.
		// On Error, we roll back? selfupdate handles that.
		return err
	}

	return nil
}

// compareVersions returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
// Assumes formats like "1.0.0" or "v1.0.0"
func compareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	if v1 == v2 {
		return 0
	}

	// Very basic splitter check
	p1 := strings.Split(v1, ".")
	p2 := strings.Split(v2, ".")

	len1 := len(p1)
	len2 := len(p2)
	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}

	for i := 0; i < maxLen; i++ {
		n1 := 0
		if i < len1 {
			fmt.Sscanf(p1[i], "%d", &n1)
		}
		n2 := 0
		if i < len2 {
			fmt.Sscanf(p2[i], "%d", &n2)
		}

		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}

	return 0
}
