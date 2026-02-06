//go:build !windows

package bridge

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// setupFileDrop 在非 Windows 平台上设置文件拖放处理
func (a *App) setupFileDrop(ctx context.Context) {
	logToFile("[Others] Registering OnFileDrop handler...")

	runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
		logToFile(fmt.Sprintf("[Others] Dropped files: %v", paths))
		fmt.Printf("[Go Debug] Dropped files: %v\n", paths)
		runtime.EventsEmit(ctx, "files-dropped", paths)
	})

	logToFile("[Others] OnFileDrop handler registered successfully")
}
