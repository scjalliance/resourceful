// +build windows

package main

import (
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// UninstallKeyName is the name of the Windows uninstall registry key.
const UninstallKeyName = "Resourceful"

// Uninstall entry registry values.
const (
	ProgramName = "Resourceful"
	CompanyName = "SCJ Alliance"
)

// UninstallData holds a set of values for the Windows uninstall registry key.
type UninstallData struct {
	DisplayName      string // Required
	DisplayVersion   string
	DisplayIcon      string
	UninstallCommand string // Required, must be quoted
	InstallLocation  string
	InstallSource    string
	Publisher        string
	EstimatedSize    uint32 // In KB
}

// WriteValues writes uninstall values to a Windows registry key.
func (u *UninstallData) WriteValues(key registry.Key) error {
	// Required Fields
	if err := key.SetStringValue("DisplayName", u.DisplayName); err != nil {
		return err
	}
	if err := key.SetStringValue("UninstallString", u.UninstallCommand); err != nil {
		return err
	}

	// Optional Fields
	if u.DisplayVersion != "" {
		key.SetStringValue("DisplayVersion", u.DisplayVersion)
	}
	if u.DisplayIcon != "" {
		key.SetStringValue("DisplayIcon", u.DisplayIcon)
	}
	if u.InstallLocation != "" {
		key.SetStringValue("InstallLocation", u.InstallLocation)
	}
	if u.InstallSource != "" {
		key.SetStringValue("InstallSource", u.InstallSource)
	}
	if u.Publisher != "" {
		key.SetStringValue("Publisher", u.Publisher)
	}
	if u.EstimatedSize != 0 {
		key.SetDWordValue("EstimatedSize", u.EstimatedSize)
	}

	return nil
}

// ReadValues reads uninstall values from a Windows registry key.
func (u *UninstallData) ReadValues(key registry.Key) error {
	// Required Fields
	var err error
	if u.DisplayName, _, err = key.GetStringValue("DisplayName"); err != nil {
		return err
	}
	if u.UninstallCommand, _, err = key.GetStringValue("UninstallString"); err != nil {
		return err
	}

	// Optional Fields
	u.DisplayVersion, _, _ = key.GetStringValue("DisplayVersion")
	u.DisplayIcon, _, _ = key.GetStringValue("DisplayIcon")
	u.InstallLocation, _, _ = key.GetStringValue("InstallLocation")
	u.InstallSource, _, _ = key.GetStringValue("InstallSource")
	u.Publisher, _, _ = key.GetStringValue("Publisher")
	if val, _, err := key.GetIntegerValue("EstimatedSize"); err == nil {
		u.EstimatedSize = uint32(val)
	}

	return nil
}

// getUninstallRegKeys returns existing uninstall data from the windows
// registry.
func getUninstallRegKeys() (data UninstallData, err error) {
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return data, err
	}
	defer root.Close()

	entry, err := registry.OpenKey(root, UninstallKeyName, registry.QUERY_VALUE)
	if err != nil {
		return data, err
	}
	defer entry.Close()

	err = data.ReadValues(entry)

	return data, err
}

// addUninstallRegKeys adds information to the Windows registry.
func addUninstallRegKeys(sourceDir, destDir, exe string, sizeInBytes int64) error {
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.CREATE_SUB_KEY)
	if err != nil {
		return err
	}
	defer root.Close()

	entry, _, err := registry.CreateKey(root, UninstallKeyName, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer entry.Close()

	data := UninstallData{
		DisplayName:      ProgramName,
		DisplayVersion:   Version,
		DisplayIcon:      filepath.Join(destDir, exe),
		UninstallCommand: uninstallCommand(destDir, exe),
		InstallLocation:  destDir,
		InstallSource:    filepath.Join(sourceDir, exe),
		Publisher:        CompanyName,
		EstimatedSize:    uint32(sizeInBytes / 1024), // Convert to KB
	}

	return data.WriteValues(entry)
}

// delUninstallRegKeys adds information to the Windows registry.
func delUninstallRegKeys() error {
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.CREATE_SUB_KEY)
	if err != nil {
		return err
	}
	defer root.Close()

	return registry.DeleteKey(root, UninstallKeyName)
}
