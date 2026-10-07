package core

import (
	"fmt"
	"sync"
	"time"
)

type ErrorRecord struct {
	RequestID string    `json:"request_id"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Message   string    `json:"message"`
	At        time.Time `json:"at"`
	Panic     bool      `json:"panic"`
}

type ErrorInspector struct {
	mu       sync.RWMutex
	capacity int
	records  []ErrorRecord
}

func NewErrorInspector(capacity int) *ErrorInspector {
	if capacity < 1 {
		capacity = 100
	}
	return &ErrorInspector{capacity: capacity}
}

func (i *ErrorInspector) Middleware() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					i.add(ErrorRecord{
						RequestID: RequestIDValue(ctx),
						Method:    ctx.Request.Method,
						Path:      ctx.Request.URL.Path,
						Message:   fmt.Sprint(recovered),
						At:        time.Now().UTC(),
						Panic:     true,
					})
					panic(recovered)
				}
			}()

			err = next(ctx)
			if err != nil {
				i.add(ErrorRecord{
					RequestID: RequestIDValue(ctx),
					Method:    ctx.Request.Method,
					Path:      ctx.Request.URL.Path,
					Message:   err.Error(),
					At:        time.Now().UTC(),
				})
			}
			return err
		}
	}
}

func (i *ErrorInspector) Records() []ErrorRecord {
	i.mu.RLock()
	defer i.mu.RUnlock()

	out := make([]ErrorRecord, len(i.records))
	for index := range i.records {
		out[len(i.records)-1-index] = i.records[index]
	}
	return out
}

func (i *ErrorInspector) Clear() {
	i.mu.Lock()
	i.records = nil
	i.mu.Unlock()
}

func (i *ErrorInspector) add(record ErrorRecord) {
	i.mu.Lock()
	i.records = append(i.records, record)
	if len(i.records) > i.capacity {
		i.records = append([]ErrorRecord(nil), i.records[len(i.records)-i.capacity:]...)
	}
	i.mu.Unlock()
}
