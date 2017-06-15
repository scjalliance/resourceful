package cacheprov

import (
	"fmt"
	"sync"

	"github.com/scjalliance/resourceful/policy"
)

// Provider is a cached source of policy data.
type Provider struct {
	Source   policy.Provider
	mutex    sync.RWMutex
	cached   bool
	policies []policy.Policy
}

// New returns a new provider that caches policies for the given source.
func New(source policy.Provider) *Provider {
	return &Provider{Source: source}
}

// Close releases any resources consumed by the provider.
func (p *Provider) Close() error {
	return p.Source.Close()
}

// ProviderName returns the name of the provider.
func (p *Provider) ProviderName() string {
	return fmt.Sprintf("%s (with in-memory caching)", p.Source.ProviderName())
}

// Policies returns a complete set of resource policies.
func (p *Provider) Policies() (policies policy.Set, err error) {
	policies, err = p.pull()
	return
}

func (p *Provider) pull() (policies policy.Set, err error) {
	p.mutex.RLock()
	if !p.cached {
		p.mutex.RUnlock()
		p.mutex.Lock()
		if !p.cached {
			p.policies, err = p.Source.Policies()
			if err == nil {
				p.cached = true
			}
		}
		p.mutex.Unlock()
		p.mutex.RLock()
	}
	policies = p.policies
	p.mutex.RUnlock()
	return
}
