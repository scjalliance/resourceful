// +build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gentlemanautomaton/filework"
	"github.com/gentlemanautomaton/filework/fwos"
	"github.com/gentlemanautomaton/winservice"
	"github.com/scjalliance/resourceful/enforcer"
)

func install(ctx context.Context, program string, conf EnforceConfig) {
	// Determine the source path
	sourcePath, err := filepath.Abs(program)
	if err != nil {
		fmt.Printf("Failed to determine the absolute path of %s: %v\n", program, err)
		os.Exit(1)
	}

	// Determine the installation directory
	dest, err := installDir(Version)
	if err != nil {
		fmt.Printf("Failed to locate installation directory: %v\n", err)
		os.Exit(1)
	}

	// TODO: Determine the version by using the PE package: https://golang.org/pkg/debug/pe/

	// Determine the source directory
	source, exe := filepath.Split(sourcePath)
	if !strings.HasSuffix(exe, ".exe") {
		exe += ".exe"
	}
	fmt.Printf("Installing %s to: %s\n", exe, dest)

	// Ensure that we can open the source file data
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		fmt.Printf("Failed to install %s: %v\n", exe, err)
		os.Exit(1)
	}
	defer sourceFile.Close()

	// Check to see if there's an existing file with the expected content
	diff, err := filework.CompareFileContent(sourceFile, fwos.Dir(dest), exe)
	if err != nil {
		fmt.Printf("Failed to examine existing %s file: %v\n", exe, err)
		os.Exit(1)
	}
	switch diff {
	case filework.Same:
		fmt.Printf("Existing %s file is up to date.\n", exe)
		return
	case filework.Different:
		fmt.Printf("Existing %s file is out of date.\n", exe)
	}

	// Create the installation directory
	if err = os.MkdirAll(dest, os.ModePerm); err != nil {
		fmt.Printf("Failed to create installation directory \"%s\": %v\n", dest, err)
		os.Exit(1)
	}

	// Check for an existing enforcement service
	exists, err := winservice.Exists(enforcer.ServiceName)
	if err != nil {
		fmt.Printf("Failed to check for existing enforcement service: %v\n", err)
		os.Exit(1)
	}
	if exists {
		fmt.Printf("Existing %s service found.\n", enforcer.ServiceName)
	}

	// Stop and remove any existing service
	if exists {
		if err := winservice.Delete(context.Background(), enforcer.ServiceName); err != nil {
			fmt.Printf("Removal of existing service failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Existing %s service stopped and removed.\n", enforcer.ServiceName)
	}

	// Copy the service
	result := filework.CopyFile(fwos.Dir(source), exe, sourceFile, fwos.Dir(dest), exe)
	if result.Err != nil {
		fmt.Printf("Failed to copy %s service executable: %v\n", enforcer.ServiceName, err)
		os.Exit(1)
	}
	fmt.Printf("%s copied to %s\n", exe, dest)

	// TODO: Create a symlink from the root install directory to this version?

	// Determine the service arguments
	args := []string{"enforce"}
	if conf.Passive {
		args = append(args, "-p")
	}
	if conf.Server != "" {
		args = append(args, "-s", conf.Server)
	}

	// Install the service
	if err := winservice.Install(enforcer.ServiceName, winservice.Path(filepath.Join(dest, exe)), winservice.Args(args...), winservice.AutoStart); err != nil {
		fmt.Printf("Failed to install %s service: %v\n", enforcer.ServiceName, err)
		os.Exit(1)
	}
	fmt.Printf("\"%s\" service installed successfully.\n", enforcer.ServiceName)

	// Start the service
	fmt.Printf("Starting service.\n")
	if err := winservice.Start(ctx, enforcer.ServiceName); err != nil {
		switch err {
		case context.Canceled, context.DeadlineExceeded:
			fmt.Printf("Stopped waiting for service startup.\n")
			os.Exit(1)
		}
	}
	fmt.Printf("Service started.\n")
}

func installDir(version string) (dir string, err error) {
	dir = os.Getenv("PROGRAMFILES")
	if dir == "" {
		return "", errors.New("unable to determine ProgramFiles location")
	}

	return filepath.Join(dir, "SCJ", "resourceful", version), nil
}
