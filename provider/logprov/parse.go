package logprov

import (
	"fmt"
	"strconv"
	"strings"
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
		switch {
		case strings.HasSuffix(item, OpsSuffix):
			item = strings.TrimSuffix(item, OpsSuffix)
			interval, err := strconv.ParseUint(item, 10, 64)
			if err != nil {
				return nil, err
			}
			schedule = append(schedule, Schedule{
				ops: interval,
			})
		default:
			// TODO: Add support for simple time-based duration intervals
			// TODO: Add support for cron-style scheduling?
			return nil, fmt.Errorf("invalid checkpoint interval \"%s\": value must end in ops, as in 1000ops", item)
		}
	}

	return
}
