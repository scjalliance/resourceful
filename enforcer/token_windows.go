//go:build windows
// +build windows

package enforcer

import (
	"errors"
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

func validateTokenForUser(token syscall.Token, expectedUsername, expectedDomain string) error {
	t := windows.Token(token)

	user, err := t.GetTokenUser()
	if err != nil {
		return fmt.Errorf("unable to get user: %v", err)
	}

	username, domain, accType, err := user.User.Sid.LookupAccount("")
	if err != nil {
		return fmt.Errorf("unable to look up account: %v", err)
	}

	if accType != windows.SidTypeUser {
		return errors.New("token is not for a standard user")
	}

	if username != expectedUsername || domain != expectedDomain {
		return errors.New("token does not match expected user")
	}

	return nil
}
