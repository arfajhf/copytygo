package retry

import (
	"context"
	"fmt"
	"time"
)

type Policy struct {
	Attempts   int
	Initial    time.Duration
	Max        time.Duration
	Multiplier float64
}

func Default() Policy {
	return Policy{
		Attempts:   3,
		Initial:    100 * time.Millisecond,
		Max:        2 * time.Second,
		Multiplier: 2,
	}
}

func Do(ctx context.Context, policy Policy, operation func(attempt int) error) error {
	if operation == nil {
		return fmt.Errorf("copytygo retry: operation is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	policy = normalize(policy)

	var last error
	delay := policy.Initial

	for attempt := 1; attempt <= policy.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		last = operation(attempt)
		if last == nil {
			return nil
		}
		if attempt == policy.Attempts {
			break
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		delay = nextDelay(delay, policy)
	}

	return last
}

func DoValue[T any](ctx context.Context, policy Policy, operation func(attempt int) (T, error)) (T, error) {
	var zero T
	if operation == nil {
		return zero, fmt.Errorf("copytygo retry: operation is required")
	}

	var value T
	err := Do(ctx, policy, func(attempt int) error {
		var inner error
		value, inner = operation(attempt)
		return inner
	})
	if err != nil {
		return zero, err
	}
	return value, nil
}

func normalize(policy Policy) Policy {
	if policy.Attempts < 1 {
		policy.Attempts = 1
	}
	if policy.Initial < 0 {
		policy.Initial = 0
	}
	if policy.Multiplier <= 0 {
		policy.Multiplier = 1
	}
	if policy.Max < 0 {
		policy.Max = 0
	}
	return policy
}

func nextDelay(current time.Duration, policy Policy) time.Duration {
	if current <= 0 {
		return 0
	}
	next := time.Duration(float64(current) * policy.Multiplier)
	if policy.Max > 0 && next > policy.Max {
		return policy.Max
	}
	return next
}
