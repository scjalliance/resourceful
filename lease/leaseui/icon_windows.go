//go:build windows
// +build windows

package leaseui

import "github.com/lxn/walk"

// Icon is an icon that can be used with the lease user interface
type Icon walk.Icon

// DefaultIcon returns the default icon for the lease user interface.
func DefaultIcon() *Icon {
	return (*Icon)(walk.IconInformation())
}
