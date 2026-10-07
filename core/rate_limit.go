package core

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateBucket struct {
	count int
	reset time.Time
}

type RateLimitOptions struct {
	Max       int
	Window    time.Duration
	Key       func(*Context) string
	Skip      func(*Context) bool
	CleanupAt time.Duration
}

func RateLimit(max int, window time.Duration) Middleware {
	return RateLimitWith(RateLimitOptions{
		Max:    max,
		Window: window,
	})
}

func RateLimitWith(options RateLimitOptions) Middleware {
	if options.Max < 1 {
		options.Max = 60
	}
	if options.Window <= 0 {
		options.Window = time.Minute
	}
	if options.CleanupAt <= 0 {
		options.CleanupAt = options.Window * 2
	}
	if options.Key == nil {
		options.Key = func(ctx *Context) string {
			host, _, _ := net.SplitHostPort(ctx.Request.RemoteAddr)
			if host == "" {
				host = ctx.Request.RemoteAddr
			}
			if host == "" {
				host = "unknown"
			}
			return host
		}
	}

	var mu sync.Mutex
	buckets := map[string]rateBucket{}
	lastCleanup := time.Now()

	return func(next Handler) Handler {
		return func(ctx *Context) error {
			if options.Skip != nil && options.Skip(ctx) {
				return next(ctx)
			}

			now := time.Now()
			key := options.Key(ctx)

			mu.Lock()
			if now.Sub(lastCleanup) >= options.CleanupAt {
				for bucketKey, bucket := range buckets {
					if now.After(bucket.reset) {
						delete(buckets, bucketKey)
					}
				}
				lastCleanup = now
			}

			bucket := buckets[key]
			if bucket.reset.IsZero() || !now.Before(bucket.reset) {
				bucket = rateBucket{reset: now.Add(options.Window)}
			}
			bucket.count++
			buckets[key] = bucket

			remaining := options.Max - bucket.count
			if remaining < 0 {
				remaining = 0
			}
			resetUnix := bucket.reset.Unix()
			mu.Unlock()

			ctx.Response.Header().Set("X-RateLimit-Limit", strconv.Itoa(options.Max))
			ctx.Response.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			ctx.Response.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetUnix, 10))

			if bucket.count > options.Max {
				ctx.Response.Header().Set("Retry-After", strconv.Itoa(int(time.Until(bucket.reset).Seconds())+1))
				return NewHTTPError(http.StatusTooManyRequests, "Too many requests")
			}

			return next(ctx)
		}
	}
}
