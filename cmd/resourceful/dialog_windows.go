// +build windows

package main

import "github.com/lxn/walk"

func msgBox(title, msg string) {
	walk.MsgBox(nil, title, msg, walk.MsgBoxIconInformation)
}
