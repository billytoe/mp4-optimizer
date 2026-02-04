package bridge

import (
	"fmt"
	"syscall"
)

// WM_DROPFILES is the message we need to allow
const WM_DROPFILES = 0x0233
const WM_COPYDATA = 0x004A
const WM_COPYGLOBALDATA = 0x0049
const MSGFLT_ADD = 1

var (
	user32                        = syscall.NewLazyDLL("user32.dll")
	procChangeWindowMessageFilter = user32.NewProc("ChangeWindowMessageFilter")
)

// FixWindowsDropPermissions attempts to allow drag/drop messages through UIPI
func FixWindowsDropPermissions() {
	// Attempt to allow WM_DROPFILES
	ret, _, err := procChangeWindowMessageFilter.Call(uintptr(WM_DROPFILES), uintptr(MSGFLT_ADD))
	if ret == 0 {
		fmt.Printf("[Windows UIPI Fix] Failed to add WM_DROPFILES: %v\n", err)
	} else {
		fmt.Println("[Windows UIPI Fix] Successfully allowed WM_DROPFILES")
	}

	// Also allow COPYDATA which is sometimes used
	procChangeWindowMessageFilter.Call(uintptr(WM_COPYDATA), uintptr(MSGFLT_ADD))
	procChangeWindowMessageFilter.Call(uintptr(WM_COPYGLOBALDATA), uintptr(MSGFLT_ADD))
}
