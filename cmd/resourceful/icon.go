// +build !windows

package main

import "github.com/scjalliance/resourceful/lease/leaseui"

func programIcon() *leaseui.Icon {
	return leaseui.DefaultIcon()
}
