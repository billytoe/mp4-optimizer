//go:build windows

package bridge

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WM_DROPFILES is the message we need to allow
const WM_DROPFILES = 0x0233
const WM_COPYDATA = 0x004A
const WM_COPYGLOBALDATA = 0x0049
const MSGFLT_ADD = 1

var (
	user32                        = syscall.NewLazyDLL("user32.dll")
	procChangeWindowMessageFilter = user32.NewProc("ChangeWindowMessageFilter")
	procFindWindowW               = user32.NewProc("FindWindowW")
)

var (
	shell32             = syscall.NewLazyDLL("shell32.dll")
	procDragAcceptFiles = shell32.NewProc("DragAcceptFiles")
)

// FixWindowsDropPermissions attempts to allow drag/drop messages through UIPI and force DragAcceptFiles
func FixWindowsDropPermissions() {
	go func() {
		// Wait for WebView2 to fully initialize and settle
		time.Sleep(3 * time.Second)
		logToFile("[Windows Fix] Running delayed permission fix...")

		// 1. Global Filter
		procChangeWindowMessageFilter.Call(uintptr(WM_DROPFILES), uintptr(MSGFLT_ADD))
		procChangeWindowMessageFilter.Call(uintptr(WM_COPYDATA), uintptr(MSGFLT_ADD))
		procChangeWindowMessageFilter.Call(uintptr(WM_COPYGLOBALDATA), uintptr(MSGFLT_ADD))
		logToFile("[Windows UIPI Fix] Applied global message filters")

		// 2. Find Main Window
		titlePtr, _ := syscall.UTF16PtrFromString("MP4 FastStart Inspector")
		hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))

		if hwnd == 0 {
			logToFile("[Windows Fix] Could not find 'MP4 FastStart Inspector' window.")
			return
		}

		logToFile(fmt.Sprintf("[Windows Fix] Found Main Window HWND: %x", hwnd))

		// Force parent to ACCEPT drops
		forceDrag(windows.HWND(hwnd), true)

		// 3. Enumerate Children (WebView2 is a child)
		// Force enable on all children to ensure SOMEBODY catches it
		cb := syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
			child := windows.HWND(hwnd)
			logToFile(fmt.Sprintf("[Windows Fix] Found Child HWND: %x (Enabling drops)", child))
			forceDrag(child, true) // <-- Changed to TRUE
			return 1               // Continue enumeration
		})

		// Fix: Pass nil as lParam
		windows.EnumChildWindows(windows.HWND(hwnd), cb, nil)
		logToFile("[Windows Fix] Completed delayed fix.")
	}()
}

func forceDrag(hwnd windows.HWND, enable bool) {
	val := uintptr(0)
	if enable {
		val = 1
	}
	_, _, err := procDragAcceptFiles.Call(uintptr(hwnd), val)
	if err != nil && err.Error() != "The operation completed successfully." {
		// Usually returns void, ignore trivial errors
	}
}
