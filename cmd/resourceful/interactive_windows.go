//go:build windows
// +build windows

package main

import (
	"golang.org/x/sys/windows/svc"
)

func isWindowsService() (bool, error) {
	return svc.IsWindowsService()
}
