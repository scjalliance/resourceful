// +build windows

package main

import (
	"golang.org/x/sys/windows/svc"
)

func isInteractive() (bool, error) {
	return svc.IsAnInteractiveSession()
}
