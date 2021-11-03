//go:build windows
// +build windows

package main

import (
	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/lease/leaseui"
)

// IconResourceID is the resource ID of the icon embedded within the
// executable.
const IconResourceID = 2

func programIcon() *leaseui.Icon {
	icon, err := walk.NewIconFromResourceId(IconResourceID)
	if err != nil {
		return leaseui.DefaultIcon()
	}
	return (*leaseui.Icon)(icon)
}
