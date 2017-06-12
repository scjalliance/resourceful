// +build !windows

package main

import "github.com/scjalliance/resourceful/lease/leaseui"

func progamIcon() *leaseui.Icon {
	return leaseui.DefaultIcon()
}
