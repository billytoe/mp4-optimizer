package bridge

import (
	"fmt"
	"syscall"
	"unsafe"
)

// WM_DROPFILES is the message we need to allow
const WM_DROPFILES = 0x0233
const WM_COPYDATA = 0x004A
const WM_COPYGLOBALDATA = 0x0049
const MSGFLT_ADD = 1

var (
	user32                        = syscall.NewLazyDLL("user32.dll")
	procChangeWindowMessageFilter = user32.NewProc("ChangeWindowMessageFilter")
	procFindWindowW               = user32.NewProc("FindWindowW") // Added
)

var (
	shell32             = syscall.NewLazyDLL("shell32.dll")  // Added
	procDragAcceptFiles = shell32.NewProc("DragAcceptFiles") // Added
)

// FixWindowsDropPermissions attempts to allow drag/drop messages through UIPI and force DragAcceptFiles
func FixWindowsDropPermissions() {
	// 1. Global Filter (Keep this)
	// Attempt to allow WM_DROPFILES
	ret, _, err := procChangeWindowMessageFilter.Call(uintptr(WM_DROPFILES), uintptr(MSGFLT_ADD))
	if ret == 0 {
		logToFile(fmt.Sprintf("[Windows UIPI Fix] Failed to add WM_DROPFILES: %v", err))
	} else {
		logToFile("[Windows UIPI Fix] Successfully allowed WM_DROPFILES")
	}

	// Also allow COPYDATA which is sometimes used
	procChangeWindowMessageFilter.Call(uintptr(WM_COPYDATA), uintptr(MSGFLT_ADD))
	procChangeWindowMessageFilter.Call(uintptr(WM_COPYGLOBALDATA), uintptr(MSGFLT_ADD))
	logToFile("[Windows UIPI Fix] Applied global message filters")

	// 2. Find Window and Force DragAcceptFiles
	// We need to wait a bit or hope the window is ready? OnStartup it should be.
	// Title must match main.go exactly: "MP4 FastStart Inspector"
	titlePtr, _ := syscall.UTF16PtrFromString("MP4 FastStart Inspector")
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))

	if hwnd == 0 {
		logToFile("[Windows Fix] Could not find 'MP4 FastStart Inspector' window to force DragAcceptFiles. (Is the title correct?)")
	} else {
		logToFile(fmt.Sprintf("[Windows Fix] Found Window HWND: %x", hwnd))
		// Force DragAcceptFiles
		_, _, err := procDragAcceptFiles.Call(hwnd, 1)
		if err != nil && err.Error() != "The operation completed successfully." {
			// DragAcceptFiles returns void usually, syscall might report err if something weird happens
			logToFile(fmt.Sprintf("[Windows Fix] DragAcceptFiles syscall result: %v", err))
		} else {
			logToFile("[Windows Fix] DragAcceptFiles(hwnd, TRUE) called")
		}
	}
}
