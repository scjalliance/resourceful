// +build !windows

package main

import "github.com/scjalliance/resourceful/guardian/transport"

func msgBox(title, msg string) {}

func leaseRejectedDlg(program string, response transport.AcquireResponse) {}
