package core

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/arfajhf/copytygo/v4/logging"
)

const RequestIDContextKey = "copytygo.request_id"

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}

func RequestID() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			id := strings.TrimSpace(ctx.Request.Header.Get("X-Request-ID"))
			if id == "" || len(id) > 128 {
				id = newRequestID()
			}

			ctx.Set(RequestIDContextKey, id)
			ctx.Response.Header().Set("X-Request-ID", id)
			return next(ctx)
		}
	}
}

func RequestIDValue(ctx *Context) string {
	value, _ := ContextValue[string](ctx, RequestIDContextKey)
	return value
}

func RequestLogger() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			start := time.Now()
			writer := &statusResponseWriter{ResponseWriter: ctx.Response}
			ctx.Response = writer

			err := next(ctx)

			status := writer.status
			if status == 0 && err != nil {
				switch typed := err.(type) {
				case *ValidationError:
					status = http.StatusUnprocessableEntity
				case *HTTPError:
					status = typed.Status
				default:
					status = http.StatusInternalServerError
				}
			}
			if status == 0 {
				status = ctx.statusCode
			}
			if status == 0 {
				status = http.StatusOK
			}

			logging.Default.Info("http.request", map[string]any{
				"request_id":  RequestIDValue(ctx),
				"method":      ctx.Request.Method,
				"path":        ctx.Request.URL.Path,
				"status":      status,
				"duration_ms": time.Since(start).Milliseconds(),
				"remote_addr": ctx.Request.RemoteAddr,
			})

			return err
		}
	}
}

func newRequestID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(raw)
}
