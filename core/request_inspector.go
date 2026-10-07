package core

import (
	"net/http"
	"sync"
	"time"
)

type RequestRecord struct {
	RequestID  string    `json:"request_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	DurationMS int64     `json:"duration_ms"`
	At         time.Time `json:"at"`
}

type RequestInspector struct {
	mu       sync.RWMutex
	capacity int
	records  []RequestRecord
}

func NewRequestInspector(capacity int) *RequestInspector {
	if capacity < 1 {
		capacity = 100
	}
	return &RequestInspector{capacity: capacity}
}

func (i *RequestInspector) Middleware() Middleware {
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

			i.add(RequestRecord{
				RequestID:  RequestIDValue(ctx),
				Method:     ctx.Request.Method,
				Path:       ctx.Request.URL.Path,
				Status:     status,
				DurationMS: time.Since(start).Milliseconds(),
				At:         time.Now().UTC(),
			})
			return err
		}
	}
}

func (i *RequestInspector) add(record RequestRecord) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.records = append(i.records, record)
	if len(i.records) > i.capacity {
		i.records = append([]RequestRecord(nil), i.records[len(i.records)-i.capacity:]...)
	}
}

func (i *RequestInspector) Records() []RequestRecord {
	i.mu.RLock()
	defer i.mu.RUnlock()

	out := make([]RequestRecord, len(i.records))
	for index := range i.records {
		out[len(i.records)-1-index] = i.records[index]
	}
	return out
}

func (i *RequestInspector) Clear() {
	i.mu.Lock()
	i.records = nil
	i.mu.Unlock()
}
