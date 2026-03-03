package commands

import (
	"fmt"
	"strings"
	"time"
)

const (
	rateLimitMinDelay = 400 * time.Millisecond
	rateLimitMaxDelay = 60 * time.Second
	rateLimitMaxRetry = 10
)

// adaptiveRateLimiter adjusts inter-request delay based on API responses.
// On success it slowly decreases delay toward the minimum.
// On 429 it doubles the delay and retries the same request.
type adaptiveRateLimiter struct {
	current time.Duration
}

func newRateLimiter() *adaptiveRateLimiter {
	return &adaptiveRateLimiter{current: 800 * time.Millisecond}
}

func (r *adaptiveRateLimiter) wait() {
	if r.current > 0 {
		time.Sleep(r.current)
	}
}

func (r *adaptiveRateLimiter) success() {
	r.current = time.Duration(float64(r.current) / 1.3)
	if r.current < rateLimitMinDelay {
		r.current = rateLimitMinDelay
	}
}

func (r *adaptiveRateLimiter) backoff() {
	r.current *= 2
	if r.current > rateLimitMaxDelay {
		r.current = rateLimitMaxDelay
	}
}

// do calls fn, retrying on 429 with exponential backoff.
// On success, decreases the current delay.
func (r *adaptiveRateLimiter) do(fn func() error) error {
	for attempt := 0; attempt <= rateLimitMaxRetry; attempt++ {
		if attempt > 0 {
			r.backoff()
			fmt.Printf("  rate limit: retry %d/%d after %v...\n", attempt, rateLimitMaxRetry, r.current.Round(time.Millisecond))
			time.Sleep(r.current)
		}
		err := fn()
		if err == nil {
			r.success()
			return nil
		}
		if !isRateLimit(err) {
			return err
		}
		// 429 — will retry
		if attempt == rateLimitMaxRetry {
			return err
		}
	}
	return nil
}

func isRateLimit(err error) bool {
	return err != nil && strings.Contains(err.Error(), "429")
}
