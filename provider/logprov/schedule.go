package logprov

import "time"

const (
	// MinimumDuration is the shortest valid duration for a duration schedule.
	MinimumDuration = time.Second
)

type scheduleType uint

const (
	opsSchedule scheduleType = iota
	durationSchedule
)

// Schedule is a transaction log checkpointing schedule.
type Schedule struct {
	t        scheduleType  // The type of schedule
	ops      uint64        // The number of operations between each checkpoint
	duration time.Duration // The time duration between each checkpoint
}

// OpsSchedule creates a checkpointing schedule that will cause a checkpoint to
// occur after the specified number of ops have been written since the last
// checkpoint.
func OpsSchedule(ops uint64) Schedule {
	return Schedule{
		t:   opsSchedule,
		ops: ops,
	}
}

// DurationSchedule creates a checkpointing schedule that will cause a
// checkpoint to occur after the specified duration of time has passed since
// the last checkpoint.
func DurationSchedule(d time.Duration) Schedule {
	return Schedule{
		t:        durationSchedule,
		duration: d,
	}
}
