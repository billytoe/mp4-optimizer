package bridge

import (
	"context"
	"fmt"
	"time"

	"mp4-optimizer/internal/analyzer"
	"mp4-optimizer/internal/optimizer"
	"mp4-optimizer/internal/updater"

	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ProgressEvent represents the progress update event data
type ProgressEvent struct {
	Path     string  `json:"path"`
	Progress float64 `json:"progress"`
	Message  string  `json:"message"`
}

// App struct
type App struct {
	ctx              context.Context
	version          string
	optimizingCount  int
	optimizingMu     sync.Mutex
	shouldClose      bool
	forceClose       bool
	visitedFolders   map[string]bool
	visitedFoldersMu sync.Mutex
}

// NewApp creates a new App application struct
func NewApp(version string) *App {
	return &App{
		version:        version,
		visitedFolders: make(map[string]bool),
	}
}

// startOptimizing increments the optimizing count
func (a *App) startOptimizing() {
	a.optimizingMu.Lock()
	a.optimizingCount++
	a.optimizingMu.Unlock()
}

// stopOptimizing decrements the optimizing count
func (a *App) stopOptimizing() {
	a.optimizingMu.Lock()
	if a.optimizingCount > 0 {
		a.optimizingCount--
	}
	a.optimizingMu.Unlock()
}

// isOurTempFile checks if a file is our temporary file
// Our temp file pattern is: {name}_tmp_{random}.mp4
func isOurTempFile(path string) bool {
	name := filepath.Base(path)
	if !strings.HasSuffix(name, ".mp4") {
		return false
	}
	nameWithoutExt := strings.TrimSuffix(name, ".mp4")
	parts := strings.Split(nameWithoutExt, "_tmp_")
	return len(parts) == 2
}

// cleanupTempFilesInDir removes our temporary files from the given directory
func (a *App) cleanupTempFilesInDir(dir string) {
	logToFile(fmt.Sprintf("[Cleanup] Scanning for temp files in: %s", dir))

	entries, err := os.ReadDir(dir)
	if err != nil {
		logToFile(fmt.Sprintf("[Cleanup] Failed to read dir %s: %v", dir, err))
		return
	}

	cleanedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		if isOurTempFile(fullPath) {
			logToFile(fmt.Sprintf("[Cleanup] Removing temp file: %s", fullPath))
			if err := os.Remove(fullPath); err != nil {
				logToFile(fmt.Sprintf("[Cleanup] Failed to remove %s: %v", fullPath, err))
			} else {
				cleanedCount++
			}
		}
	}

	if cleanedCount > 0 {
		logToFile(fmt.Sprintf("[Cleanup] Cleaned %d temp files from %s", cleanedCount, dir))
	}
}

// cleanupAllVisitedFolders removes temp files from all visited folders
func (a *App) cleanupAllVisitedFolders() {
	a.visitedFoldersMu.Lock()
	defer a.visitedFoldersMu.Unlock()

	logToFile("[Cleanup] Cleaning up all visited folders...")
	for folder := range a.visitedFolders {
		a.cleanupTempFilesInDir(folder)
	}
}

// trackFolder adds a folder to the visited list and cleans it up
func (a *App) trackFolder(dir string) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		absPath = dir
	}

	a.visitedFoldersMu.Lock()
	a.visitedFolders[absPath] = true
	a.visitedFoldersMu.Unlock()

	// Always cleanup, even if already visited
	a.cleanupTempFilesInDir(absPath)
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	logToFile("App Startup")

	// Apply Windows UIPI fix (在 Windows 上启用拖拽权限)
	FixWindowsDropPermissions()

	// 调用平台特定的拖拽设置 (Windows 需要延迟注册)
	a.setupFileDrop(ctx)
}

// CheckFile checks if the MP4 file is fast-start optimized.
// Returns true if optimized, false otherwise.
func (a *App) CheckFile(path string) (bool, error) {
	return analyzer.CheckFastStart(path)
}

// ValidateFile checks if the MP4 file is complete and not truncated.
// Returns true if the file appears to be complete, false if truncated.
func (a *App) ValidateFile(path string) (bool, error) {
	return analyzer.ValidateFile(path)
}

// OptimizeFile performs the fast-start optimization on the file.
func (a *App) OptimizeFile(path string) error {
	a.startOptimizing()
	defer a.stopOptimizing()

	// Track the folder of this file
	parentDir := filepath.Dir(path)
	a.trackFolder(parentDir)

	callback := func(progress float64, message string) {
		event := ProgressEvent{
			Path:     path,
			Progress: progress,
			Message:  message,
		}
		runtime.EventsEmit(a.ctx, "optimize-progress", event)
	}

	return optimizer.Optimize(path, callback)
}

// IsOptimizing returns whether there's an optimization in progress
func (a *App) IsOptimizing() bool {
	a.optimizingMu.Lock()
	defer a.optimizingMu.Unlock()
	return a.optimizingCount > 0
}

// RequestClose requests the app to close, will prompt user if optimizing
func (a *App) RequestClose() bool {
	if a.IsOptimizing() {
		// Signal frontend to show confirmation dialog
		runtime.EventsEmit(a.ctx, "request-close-confirm")
		return false
	}
	a.shouldClose = true
	runtime.Quit(a.ctx)
	return true
}

// IsForceClosing returns whether the app is in the process of force closing
func (a *App) IsForceClosing() bool {
	return a.forceClose
}

// ForceClose forces the app to close immediately
func (a *App) ForceClose() {
	logToFile("[ForceClose] User requested force close - cleaning up temp files first...")
	a.forceClose = true
	a.cleanupAllVisitedFolders()
	a.shouldClose = true
	runtime.Quit(a.ctx)
}

// GetFileMetadata returns the metadata for the given file
func (a *App) GetFileMetadata(path string) (*analyzer.Metadata, error) {
	return analyzer.GetMetadata(path)
}

// SelectFiles opens a file dialog to select multiple MP4 files.
func (a *App) SelectFiles() ([]string, error) {
	selection, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select MP4 Files",
		Filters: []runtime.FileFilter{
			{DisplayName: "MP4 Video", Pattern: "*.mp4"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dialog error: %w", err)
	}
	return selection, nil
}

// SelectDirectory opens a dialog to select a directory
func (a *App) SelectDirectory() (string, error) {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder",
	})
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	return selection, nil
}

// ExpandPaths takes a list of paths and returns a flat list of all MP4 files found.
// It handles directories recursively.
func (a *App) ExpandPaths(paths []string) ([]string, error) {
	var result []string
	uniquePaths := make(map[string]bool)
	foldersToTrack := make(map[string]bool)

	logToFile(fmt.Sprintf("[ExpandPaths] Start processing %d paths: %v", len(paths), paths))

	for _, p := range paths {
		cleanPath := filepath.Clean(p)
		info, err := os.Stat(cleanPath)
		if err != nil {
			logToFile(fmt.Sprintf("[ExpandPaths] Error accessing path '%s': %v", cleanPath, err))
			continue // Skip invalid paths
		}

		if info.IsDir() {
			logToFile(fmt.Sprintf("[ExpandPaths] processing directory: %s", cleanPath))
			// Track this folder
			absDir, _ := filepath.Abs(cleanPath)
			foldersToTrack[absDir] = true

			// Walk directory
			err := filepath.WalkDir(cleanPath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					logToFile(fmt.Sprintf("[ExpandPaths] Walk error at %s: %v", path, err))
					return nil // Skip errors accessing files
				}
				if d.IsDir() {
					absSubDir, _ := filepath.Abs(path)
					foldersToTrack[absSubDir] = true
				} else {
					if isMP4(path) && !isOurTempFile(path) {
						absPath, err := filepath.Abs(path)
						if err == nil {
							if !uniquePaths[absPath] {
								uniquePaths[absPath] = true
								result = append(result, absPath)
							}
						}
					}
				}
				return nil
			})
			if err != nil {
				logToFile(fmt.Sprintf("[ExpandPaths] Error walking dir %s: %v", cleanPath, err))
				fmt.Printf("Error walking dir %s: %v\n", cleanPath, err)
			}
		} else {
			// It's a file - track its parent folder
			parentDir := filepath.Dir(cleanPath)
			absParentDir, _ := filepath.Abs(parentDir)
			foldersToTrack[absParentDir] = true

			if isMP4(cleanPath) && !isOurTempFile(cleanPath) {
				absPath, err := filepath.Abs(cleanPath)
				if err == nil {
					if !uniquePaths[absPath] {
						uniquePaths[absPath] = true
						result = append(result, absPath)
					}
				}
			} else {
				logToFile(fmt.Sprintf("[ExpandPaths] Skipped non-MP4 or temp file: %s", cleanPath))
			}
		}
	}

	// Cleanup all tracked folders
	for folder := range foldersToTrack {
		a.trackFolder(folder)
	}

	logToFile(fmt.Sprintf("[ExpandPaths] Finished. Found %d MP4 files.", len(result)))
	return result, nil
}

// GetAppVersion returns the current application version
func (a *App) GetAppVersion() string {
	return a.version
}

// CheckForUpdates checks for updates from the given URL
func (a *App) CheckForUpdates(url string) (*updater.CheckResult, error) {
	return updater.CheckUpdate(a.version, url)
}

// InstallUpdate downloads and installs the update from the given URL
func (a *App) InstallUpdate(url string) error {
	return updater.ApplyUpdate(url)
}

func isMP4(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp4"
}

func logToFile(msg string) {
	f, err := os.OpenFile("debug_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(msg) // fallback to console
		return
	}
	defer f.Close()
	f.WriteString(time.Now().Format("15:04:05") + " " + msg + "\n")
}
