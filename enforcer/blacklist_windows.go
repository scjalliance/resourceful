// +build windows

package enforcer

import "strings"

var blacklist = map[string]bool{
	"applicationframehost":    true,
	"backgroundtaskhost":      true,
	"chrome":                  true,
	"cmd":                     true,
	"csrss":                   true,
	"conhost":                 true,
	"ctfmon":                  true,
	"dllhost":                 true,
	"dwm":                     true,
	"explorer":                true,
	"fontdrvhost":             true,
	"launchtm":                true,
	"lockapp":                 true,
	"lsass":                   true,
	"mmc":                     true,
	"regedit":                 true,
	"registry":                true,
	"resourceful":             true,
	"runtimebroker":           true,
	"rtkauduservice64":        true,
	"searchindexer":           true,
	"searchfilterhost":        true,
	"searchprotocolhost":      true,
	"searchui":                true,
	"securityhealthsystray":   true,
	"services":                true,
	"shellexperiencehost":     true,
	"sihost":                  true,
	"smartscreen":             true,
	"svchost":                 true,
	"systemsettings":          true,
	"systemsettingsbroker":    true,
	"startmenuexperiencehost": true,
	"taskhostw":               true,
	"taskmgr":                 true,
	"thunderbolt":             true,
	"wininit":                 true,
	"winlogon":                true,
	"wireguard":               true,
	"wudfhost":                true,
	"windowsinternal.composableshell.experiences.textinput.inputapp": true,
}

// Blacklisted returns true if p is a blacklisted process that must not
// be managed.
func Blacklisted(p ProcessData) bool {
	if p.Protected() {
		return true
	}

	name := strings.ToLower(p.Name)
	name = strings.TrimSuffix(name, ".exe")

	return blacklist[name]

	// TODO: Skip anything running as something other than the session's user

	// TODO: Skip anything with the CREATE_PROTECTED_PROCESS flag
}
