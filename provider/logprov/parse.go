package logprov

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// OpsSuffix is the suffix used to identify operations in a schedule.
const OpsSuffix = "ops"

// ParseSchedule will parse the schedule in the provided string.
func ParseSchedule(s string) (schedule []Schedule, err error) {
	if s == "" {
		return
	}

	items := strings.Split(s, " ")

	for _, item := range items {
		// TODO: Add support for cron-style scheduling?

		switch {
		case strings.HasSuffix(item, OpsSuffix):
			value := strings.TrimSuffix(item, OpsSuffix)
			ops, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid checkpoint interval \"%s\": unable to parse \"%s\" as an integer", item, value)
			}
			if ops == 0 {
				return nil, fmt.Errorf("invalid checkpoint interval \"%s\": ops must be greater than zero", item)
			}
			schedule = append(schedule, OpsSchedule(ops))
		default:
			duration, err := time.ParseDuration(item)
			if err != nil {
				return nil, fmt.Errorf("invalid checkpoint interval \"%s\": unable to parse value as a duration of time", item)
			}
			if duration < MinimumDuration {
				return nil, fmt.Errorf("invalid checkpoint interval \"%s\": duration must not be less than %s", item, MinimumDuration.String())
			}
			schedule = append(schedule, DurationSchedule(duration))
		}
	}

	return
}
