//go:build !windows
// +build !windows

package leaseui

// Notify will send a notification message to the current user.
func Notify(title, msg string) {
}
