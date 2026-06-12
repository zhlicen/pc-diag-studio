package collector

import "golang.org/x/sys/windows"

// IsAdmin reports whether the current process token is elevated.
func IsAdmin() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
