//go:build windows

package bridge

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows message constants
const WM_DROPFILES = 0x0233
const WM_COPYDATA = 0x004A
const WM_COPYGLOBALDATA = 0x0049

// UIPI filter action constants
const MSGFLT_ALLOW = 1 // ChangeWindowMessageFilterEx action
const MSGFLT_ADD = 1   // ChangeWindowMessageFilter (legacy fallback)

var (
	user32                          = syscall.NewLazyDLL("user32.dll")
	procChangeWindowMessageFilterEx = user32.NewProc("ChangeWindowMessageFilterEx")
	procChangeWindowMessageFilter   = user32.NewProc("ChangeWindowMessageFilter")
	procFindWindowW                 = user32.NewProc("FindWindowW")
)

var (
	shell32             = syscall.NewLazyDLL("shell32.dll")
	procDragAcceptFiles = shell32.NewProc("DragAcceptFiles")
)

// allowMessageForWindow applies ChangeWindowMessageFilterEx (per-window, not deprecated)
func allowMessageForWindow(hwnd uintptr, message uint32) {
	ret, _, err := procChangeWindowMessageFilterEx.Call(
		hwnd, uintptr(message), uintptr(MSGFLT_ALLOW), 0,
	)
	if ret == 0 {
		logToFile(fmt.Sprintf("[Windows UIPI] FilterEx failed HWND %x, msg 0x%04X: %v", hwnd, message, err))
	}
}

// enableDropForWindow applies per-window UIPI filter + DragAcceptFiles
func enableDropForWindow(hwnd uintptr) {
	allowMessageForWindow(hwnd, WM_DROPFILES)
	allowMessageForWindow(hwnd, WM_COPYDATA)
	allowMessageForWindow(hwnd, WM_COPYGLOBALDATA)
	procDragAcceptFiles.Call(hwnd, 1)
}

// FixWindowsDropPermissions sets up UIPI message filters for drag/drop.
func FixWindowsDropPermissions() {
	go func() {
		time.Sleep(3 * time.Second)
		logToFile("[Windows Fix] Running delayed permission fix...")

		// 1. Global filter as fallback (deprecated but harmless)
		procChangeWindowMessageFilter.Call(uintptr(WM_DROPFILES), uintptr(MSGFLT_ADD))
		procChangeWindowMessageFilter.Call(uintptr(WM_COPYDATA), uintptr(MSGFLT_ADD))
		procChangeWindowMessageFilter.Call(uintptr(WM_COPYGLOBALDATA), uintptr(MSGFLT_ADD))
		logToFile("[Windows UIPI Fix] Applied global message filters (fallback)")

		// 2. Find Main Window
		titlePtr, _ := syscall.UTF16PtrFromString("MP4 FastStart Inspector")
		hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
		if hwnd == 0 {
			logToFile("[Windows Fix] Could not find main window.")
			return
		}
		logToFile(fmt.Sprintf("[Windows Fix] Found Main Window HWND: %x", hwnd))

		// 3. Per-window filter on main window
		enableDropForWindow(hwnd)
		logToFile(fmt.Sprintf("[Windows Fix] Applied per-window filters on HWND: %x", hwnd))

		// 4. Enumerate children
		cb := syscall.NewCallback(func(childHwnd uintptr, lParam uintptr) uintptr {
			logToFile(fmt.Sprintf("[Windows Fix] Found Child HWND: %x (Enabling drops)", childHwnd))
			enableDropForWindow(childHwnd)
			return 1
		})
		windows.EnumChildWindows(windows.HWND(hwnd), cb, nil)

		// 5. Second pass for late-created WebView2 child windows
		time.Sleep(2 * time.Second)
		logToFile("[Windows Fix] Second pass for late child windows...")
		windows.EnumChildWindows(windows.HWND(hwnd), cb, nil)

		logToFile("[Windows Fix] Completed delayed fix.")
	}()
}
