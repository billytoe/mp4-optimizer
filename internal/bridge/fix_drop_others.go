//go:build !windows

package bridge

// FixWindowsDropPermissions is a no-op on non-Windows platforms
func FixWindowsDropPermissions() {
	// Do nothing
}
