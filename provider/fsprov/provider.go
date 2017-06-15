package fsprov

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/scjalliance/resourceful/policy"
	"github.com/scjalliance/resourceful/strategy"
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

// Close releases any resources consumed by the provider.
func (p *Provider) Close() error {
	return nil
}

// ProviderName returns the name of the provider.
func (p *Provider) ProviderName() string {
	return "Filesystem"
}

// Policies will return a complete set of resource policies.
func (p *Provider) Policies() (policies policy.Set, err error) {
	files, dirErr := ioutil.ReadDir(p.path)
	if dirErr != nil {
		return nil, fmt.Errorf("unable to access policy directory \"%s\": %v", p.path, dirErr)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matched, matchErr := filepath.Match("*.pol", file.Name())
		if matchErr != nil {
			return nil, fmt.Errorf("unable to perform policy filename match for file \"%s\": %v", file.Name(), matchErr)
		}
		if !matched {
			continue
		}

		path := filepath.Join(p.path, file.Name())
		contents, fileErr := ioutil.ReadFile(path)
		if fileErr != nil {
			return nil, fmt.Errorf("unable to read policy file \"%s\": %v", path, fileErr)
		}

		// TODO: Use json.Decoder and stream the file into it instead of slurping?
		pol := policy.Policy{}
		dataErr := json.Unmarshal(contents, &pol)
		if dataErr != nil {
			err = fmt.Errorf("decoding error while parsing policy file \"%s\": %v", path, dataErr)
			return
		}

		if !strategy.Valid(pol.Strategy) {
			err = fmt.Errorf("invalid policy strategy in \"%s\": \"%s\"", path, pol.Strategy)
			return
		}

		if pol.Duration == 0 {
			pol.Duration = policy.DefaultDuration
		}

		policies = append(policies, pol)
	}

	return
}
