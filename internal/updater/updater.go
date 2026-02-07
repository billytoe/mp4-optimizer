package updater

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	// macOS requires special handling: downloads are .zip files containing .app bundles
	if runtime.GOOS == "darwin" && strings.HasSuffix(downloadURL, ".zip") {
		return applyMacOSUpdate(downloadURL)
	}

	// Windows: use selfupdate for single executable
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Apply update (Windows .exe)
	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		return err
	}

	return nil
}

// applyMacOSUpdate handles macOS .app bundle updates
func applyMacOSUpdate(downloadURL string) error {
	// 1. Download zip to temp file
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create temp file for zip
	tmpZip, err := os.CreateTemp("", "update-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpZip.Name())

	// Copy download to temp file
	_, err = io.Copy(tmpZip, resp.Body)
	if err != nil {
		tmpZip.Close()
		return fmt.Errorf("failed to save download: %w", err)
	}
	tmpZip.Close()

	// 2. Find current .app bundle path
	// os.Executable() returns something like /path/to/App.app/Contents/MacOS/binary
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Navigate up to find .app directory
	// /path/to/App.app/Contents/MacOS/binary -> /path/to/App.app
	appPath := exePath
	for i := 0; i < 3; i++ {
		appPath = filepath.Dir(appPath)
	}

	if !strings.HasSuffix(appPath, ".app") {
		return fmt.Errorf("could not locate .app bundle from executable path: %s", exePath)
	}

	// 3. Create temp directory for extraction
	tmpDir, err := os.MkdirTemp("", "update-extract-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 4. Extract zip
	if err := unzip(tmpZip.Name(), tmpDir); err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}

	// 5. Find extracted .app (should be the only .app in tmpDir)
	var newAppPath string
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return fmt.Errorf("failed to read temp dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), ".app") {
			newAppPath = filepath.Join(tmpDir, entry.Name())
			break
		}
	}
	if newAppPath == "" {
		return fmt.Errorf("no .app found in downloaded zip")
	}

	// 6. Backup old .app and replace with new
	backupPath := appPath + ".backup"
	os.RemoveAll(backupPath) // Remove old backup if exists

	// Rename old to backup
	if err := os.Rename(appPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup old app: %w", err)
	}

	// Move new app to original location
	if err := os.Rename(newAppPath, appPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, appPath)
		return fmt.Errorf("failed to install new app: %w", err)
	}

	// Remove backup
	os.RemoveAll(backupPath)

	return nil
}

// unzip extracts a zip file to the destination directory
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// Create file
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
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
