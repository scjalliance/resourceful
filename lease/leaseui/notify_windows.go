// +build windows

package leaseui

import "github.com/lxn/walk"

// Notify will send a notification message to the current user.
func Notify(title, msg string) {
	walk.MsgBox(nil, title, msg, walk.MsgBoxIconInformation)
}
