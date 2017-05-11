package fsprov

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/scjalliance/resourceful/policy"
)

// Provider is a filesystem-based source of policy data. The policies are read
// from *.pol files in a given directory, each of which are JSON-encoded.
//
// Policies that omit a duration will use the policy.DefaultDuration value.
type Provider struct {
	path string
}

// New returns a new provider that serves policies from the filesystem.
//
// The provided path must point to a directory that contains zero or more *.pol
// files.
func New(path string) *Provider {
	return &Provider{path: path}
}

// Policies will return a complete set of resource policies.
func (p *Provider) Policies() (policies policy.Set, err error) {
	files, err := ioutil.ReadDir(p.path)
	if err != nil {
		return
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		matched, matchErr := filepath.Match("*.pol", file.Name())
		if matchErr != nil {
			err = matchErr
			return
		}
		if !matched {
			continue
		}
		contents, fileErr := ioutil.ReadFile(file.Name())
		if fileErr != nil {
			// TODO: Log the error? Return the error?
			continue
		}
		pol := policy.Policy{}
		// TODO: Use json.Decoder and stream the file into it instead of slurping.
		dataErr := json.Unmarshal(contents, &pol)
		if dataErr != nil {
			// TODO: Log the error? Return the error?
			//log.Printf("Policy decoding error while reading %s: %v\n", file.Name(), dataErr)
			continue
		}

		if pol.Duration == 0 {
			pol.Duration = policy.DefaultDuration
		}

		//log.Printf("Policy loaded from %s: %+v\n", file.Name(), pol)
		policies = append(policies, pol)
	}

	return
}
