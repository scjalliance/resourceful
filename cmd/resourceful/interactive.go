// +build !windows

package main

func isInteractive() (bool, error) {
	return true, nil
}
