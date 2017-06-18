package leaseui

import "time"

// sleepRound will sleep until the current time is near a whole second.
func sleepRound() {
	now := time.Now()
	start := now.Round(time.Second)
	if start.Before(now) {
		// We rounded down
		start.Add(time.Second)
	}
	duration := start.Sub(now)
	time.Sleep(duration)
}
