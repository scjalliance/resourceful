// +build windows

package enforcer

// Blacklisted returns true if p is a blacklisted process that must not
// be managed.
func Blacklisted(p Process) bool {
	return p.Protected()

	// TODO: Skip anything running as something other than the session's user

	// TODO: Skip anything with the CREATE_PROTECTED_PROCESS flag
}
