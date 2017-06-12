// +build !windows

package main

func prepareConsole(attach bool) {}

func freeConsole() (err error) { return nil }
