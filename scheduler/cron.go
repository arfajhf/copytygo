package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type cronExpression struct {
	minute     map[int]bool
	hour       map[int]bool
	day        map[int]bool
	month      map[int]bool
	weekday    map[int]bool
	expression string
}

func parseCron(expression string) (cronExpression, error) {
	fields := strings.Fields(strings.TrimSpace(expression))
	if len(fields) != 5 {
		return cronExpression{}, fmt.Errorf("copytygo scheduler: cron requires 5 fields")
	}

	minute, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return cronExpression{}, fmt.Errorf("copytygo scheduler: cron minute: %w", err)
	}
	hour, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return cronExpression{}, fmt.Errorf("copytygo scheduler: cron hour: %w", err)
	}
	day, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return cronExpression{}, fmt.Errorf("copytygo scheduler: cron day: %w", err)
	}
	month, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return cronExpression{}, fmt.Errorf("copytygo scheduler: cron month: %w", err)
	}
	weekday, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return cronExpression{}, fmt.Errorf("copytygo scheduler: cron weekday: %w", err)
	}

	return cronExpression{
		minute: minute, hour: hour, day: day, month: month, weekday: weekday,
		expression: strings.Join(fields, " "),
	}, nil
}

func parseCronField(field string, min, max int) (map[int]bool, error) {
	values := make(map[int]bool)

	addRange := func(start, end, step int) error {
		if start < min || end > max || start > end || step < 1 {
			return fmt.Errorf("invalid range %d-%d/%d", start, end, step)
		}
		for value := start; value <= end; value += step {
			values[value] = true
		}
		return nil
	}

	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty field segment")
		}

		step := 1
		base := part
		if strings.Contains(part, "/") {
			pieces := strings.Split(part, "/")
			if len(pieces) != 2 {
				return nil, fmt.Errorf("invalid step %q", part)
			}
			base = pieces[0]
			parsedStep, err := strconv.Atoi(pieces[1])
			if err != nil || parsedStep < 1 {
				return nil, fmt.Errorf("invalid step %q", pieces[1])
			}
			step = parsedStep
		}

		switch {
		case base == "*":
			if err := addRange(min, max, step); err != nil {
				return nil, err
			}

		case strings.Contains(base, "-"):
			pieces := strings.Split(base, "-")
			if len(pieces) != 2 {
				return nil, fmt.Errorf("invalid range %q", base)
			}
			start, err1 := strconv.Atoi(pieces[0])
			end, err2 := strconv.Atoi(pieces[1])
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid range %q", base)
			}
			if err := addRange(start, end, step); err != nil {
				return nil, err
			}

		default:
			value, err := strconv.Atoi(base)
			if err != nil || value < min || value > max {
				return nil, fmt.Errorf("invalid value %q", base)
			}
			values[value] = true
		}
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("field has no values")
	}
	return values, nil
}

func (c cronExpression) matches(at time.Time) bool {
	return c.minute[at.Minute()] &&
		c.hour[at.Hour()] &&
		c.day[at.Day()] &&
		c.month[int(at.Month())] &&
		c.weekday[int(at.Weekday())]
}

func (c cronExpression) nextAfter(after time.Time) time.Time {
	candidate := after.Truncate(time.Minute).Add(time.Minute)
	limit := candidate.AddDate(2, 0, 0)
	for !candidate.After(limit) {
		if c.matches(candidate) {
			return candidate
		}
		candidate = candidate.Add(time.Minute)
	}
	return time.Time{}
}
