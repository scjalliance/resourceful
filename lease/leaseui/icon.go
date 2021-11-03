//go:build !windows
// +build !windows

package leaseui

// Icon is an icon that can be used with the lease user interface.
//
// In linux the Icon type is not used and is just a placeholder.
type Icon struct{}

// DefaultIcon returns the default icon for the lease user interface.
func DefaultIcon() *Icon {
	return &Icon{}
}
