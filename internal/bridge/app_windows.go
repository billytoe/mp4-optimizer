//go:build windows

package bridge

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// setupFileDrop 在 Windows 上设置文件拖放处理
func (a *App) setupFileDrop(ctx context.Context) {
	// Windows 修复在单独的 goroutine 中运行
	// 等待修复完成（FixWindowsDropPermissions 内部有 2 秒延迟）
	go func() {
		// 等待窗口修复完成
		time.Sleep(3 * time.Second)

		logToFile("[Windows] Registering OnFileDrop handler...")

		// 在 Windows 上，OnFileDrop 必须在窗口完全准备好之后注册
		// 使用 runtime.OnFileDrop 注册拖拽处理器
		runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
			logToFile(fmt.Sprintf("[Windows] OnFileDrop triggered! Position: (%d, %d), Files: %v", x, y, paths))
			fmt.Printf("[Go Debug Windows] Dropped files at (%d, %d): %v\n", x, y, paths)

			if len(paths) > 0 {
				// 发送事件到前端
				runtime.EventsEmit(ctx, "files-dropped", paths)
				logToFile("[Windows] Event 'files-dropped' emitted")
			} else {
				logToFile("[Windows] WARNING: No paths received in OnFileDrop")
			}
		})

		logToFile("[Windows] OnFileDrop handler registered successfully")
	}()
}
