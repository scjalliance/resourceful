//go:build windows
// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procFreeConsole   = modkernel32.NewProc("FreeConsole")
	procAttachConsole = modkernel32.NewProc("AttachConsole")
	procAllocConsole  = modkernel32.NewProc("AllocConsole")
)

// prepareConsole ensures that the stdard outputs are bound to a console.
// When an application is built with "-ldflags -H=windowsgui" this is necessary
// to connect the console so that the standard outputs can be displayed.
//
// If attach is true it will first try to attach to the parent process.
func prepareConsole(attach bool) (err error) {
	err = attachConsole()
	if err == nil {
		bindOutput()
		fmt.Println() // Start on a new line when attaching to an existing console
		return
	}

	err = allocConsole()
	if err == nil {
		bindOutput()
	}

	return
}

func freeConsole() (err error) {
	r0, _, e0 := syscall.Syscall(procFreeConsole.Addr(), 0, 0, 0, 0)
	if r0 == 0 {
		err = fmt.Errorf("could not free console: %s", e0)
	}
	return
}

func attachConsole() (err error) {
	const attachParentProcess = ^uintptr(0) // -1
	r0, _, e0 := syscall.Syscall(procAttachConsole.Addr(), 1, attachParentProcess, 0, 0)
	if r0 == 0 {
		// The process might already have a console
		err = fmt.Errorf("could not attach console: %s", e0)
	}
	return
}

func allocConsole() (err error) {
	r0, _, e0 := syscall.Syscall(procAllocConsole.Addr(), 0, 0, 0, 0)
	if r0 == 0 {
		// The process might already have a console
		err = fmt.Errorf("could not allocate console: %s", e0)
	}
	return
}

func bindOutput() error {
	hout, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return err
	}
	hin, err := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	if err != nil {
		return err
	}
	herr, err := syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)
	if err != nil {
		return err
	}

	os.Stdout = os.NewFile(uintptr(hout), "/dev/stdout")
	os.Stdin = os.NewFile(uintptr(hin), "/dev/stdin")
	os.Stderr = os.NewFile(uintptr(herr), "/dev/stderr")
	log.SetOutput(os.Stdout)

	return nil
}
