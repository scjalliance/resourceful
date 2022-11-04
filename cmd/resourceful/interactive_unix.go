//go:build !windows
// +build !windows

package main

func isWindowsService() (bool, error) {
	return false, nil
}
