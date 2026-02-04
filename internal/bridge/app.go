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

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx     context.Context
	version string
}

// NewApp creates a new App application struct
func NewApp(version string) *App {
	return &App{
		version: version,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	logToFile("App Startup")
	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		logToFile(fmt.Sprintf("Dropped files: %v", paths))
		fmt.Printf("[Go Debug] Dropped files: %v\n", paths)
		runtime.EventsEmit(ctx, "files-dropped", paths)
	})
}

func logToFile(msg string) {
	f, err := os.OpenFile("debug_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(time.Now().Format("15:04:05") + " " + msg + "\n")
}

// CheckFile checks if the MP4 file is fast-start optimized.
// Returns true if optimized, false otherwise.
func (a *App) CheckFile(path string) (bool, error) {
	return analyzer.CheckFastStart(path)
}

// OptimizeFile performs the fast-start optimization on the file.
func (a *App) OptimizeFile(path string) error {
	return optimizer.Optimize(path)
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

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue // Skip invalid paths
		}

		if info.IsDir() {
			// Walk directory
			err := filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil // Skip errors accessing files
				}
				if !d.IsDir() {
					if isMP4(path) {
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
				fmt.Printf("Error walking dir %s: %v\n", p, err)
			}
		} else {
			// It's a file
			if isMP4(p) {
				absPath, err := filepath.Abs(p)
				if err == nil {
					if !uniquePaths[absPath] {
						uniquePaths[absPath] = true
						result = append(result, absPath)
					}
				}
			}
		}
	}
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
