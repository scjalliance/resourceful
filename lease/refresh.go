package lease

import (
	"encoding/json"
	"time"
)

// MinimumRefresh is the minimum amount of time that should pass between
// lease refresh attempts. It is a hard limit intended to avoid hammering
// the server when policies are misconfigured.
const MinimumRefresh = time.Second

// Refresh defines the active and queued refresh rates for a lease.
type Refresh struct {
	Active time.Duration `json:"active,omitempty"` // Active lease refresh interval
	Queued time.Duration `json:"queued,omitempty"` // Queued lease refresh interval
}

// MarshalJSON will encode the refresh intervals as JSON.
func (r *Refresh) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Active string `json:"active,omitempty"`
		Queued string `json:"queued,omitempty"`
	}{
		Active: r.Active.String(),
		Queued: r.Queued.String(),
	})
}

// UnmarshalJSON will decode JSON refresh interval data.
func (r *Refresh) UnmarshalJSON(data []byte) error {
	aux := &struct {
		Active string `json:"active"`
		Queued string `json:"queued"`
	}{}
	var err error
	if err = json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.Active != "" {
		if r.Active, err = time.ParseDuration(aux.Active); err != nil {
			return err
		}
	}
	if aux.Queued != "" {
		if r.Queued, err = time.ParseDuration(aux.Queued); err != nil {
			return err
		}
	}
	return nil
}
