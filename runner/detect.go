package runner

import (
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"time"

	"github.com/scjalliance/resourceful/lease"
)

// DetectEnvironment queries the execution context to determine the subject
// and properties of a lease for a resourceful run.
//
// The returned instance will be a random string.
func DetectEnvironment(c Config) (lease.Instance, lease.Properties, error) {
	host, err := os.Hostname()
	if err != nil {
		return lease.Instance{}, nil, fmt.Errorf("unable to query hostname: %v", err)
	}

	u, err := user.Current()
	if err != nil {
		return lease.Instance{}, nil, fmt.Errorf("unable to determine current user: %v", err)
	}

	instance := lease.Instance{
		Host: host,
		User: u.Username,
		ID:   randomID(12),
	}
	props := Properties(c, host, u)

	return instance, props, nil
}

// randomID generates a random string identifier of length n.
//
// Code provided by icza on Stack Overflow.
// See: https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func randomID(n int) string {
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
