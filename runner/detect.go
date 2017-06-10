package runner

import (
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"time"

	"github.com/scjalliance/resourceful/environment"
)

// DetectEnvironment queries the execution context to determine the
// consumer, instance and environment for a resourceful run.
//
// The returned instance will be a random string.
func DetectEnvironment() (consumer, instance string, env environment.Environment, err error) {
	hostname, err := os.Hostname()
	if err != nil {
		err = fmt.Errorf("unable to query hostname: %v", err)
		return
	}

	u, err := user.Current()
	if err != nil {
		err = fmt.Errorf("unable to determine current user: %v", err)
		return
	}

	consumer = fmt.Sprintf("%s %s", hostname, u.Username)
	env = make(environment.Environment)
	env["host.name"] = hostname
	env["user.uid"] = u.Uid
	env["user.username"] = u.Username
	env["user.name"] = u.Name

	instance = randomInstance(12)

	return
}

// randomInstance generates a random instance identifier of length n.
//
// Code provided by icza on Stack Overflow.
// See: https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func randomInstance(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)

	src := rand.NewSource(time.Now().UnixNano())

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
