// +build windows

package main

import (
	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/lease/leaseui"
)

func programIcon() *leaseui.Icon {
	icon, err := walk.NewIconFromResourceId(5)
	if err != nil {
		return leaseui.DefaultIcon()
	}
	return (*leaseui.Icon)(icon)
}
