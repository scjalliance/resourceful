package logprov

// Schedule is a transaction log checkpointing schedule.
type Schedule struct {
	ops uint64 // The number of operations between each checkpoint.
}

// OpsSchedule creates a checkpointing schedule that will cause a checkpoint to
// occur after the specified number of ops have been written since the last
// checkpoint.
func OpsSchedule(ops uint64) Schedule {
	return Schedule{
		ops: ops,
	}
}
