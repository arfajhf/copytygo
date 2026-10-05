package core

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type rateBucket struct {
	count int
	reset time.Time
}

func RateLimit(max int, window time.Duration) Middleware {
	var mu sync.Mutex
	buckets := map[string]rateBucket{}
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			host, _, _ := net.SplitHostPort(ctx.Request.RemoteAddr)
			if host == "" {
				host = ctx.Request.RemoteAddr
			}
			now := time.Now()
			mu.Lock()
			b := buckets[host]
			if now.After(b.reset) {
				b = rateBucket{reset: now.Add(window)}
			}
			b.count++
			buckets[host] = b
			mu.Unlock()
			ctx.Response.Header().Set("X-RateLimit-Limit", fmtInt(max))
			if b.count > max {
				return NewHTTPError(http.StatusTooManyRequests, "Too many requests")
			}
			return next(ctx)
		}
	}
}
func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
