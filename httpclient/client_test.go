package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arfajhf/copytygo/v4/retry"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestGetRetriesNetworkErrors(t *testing.T) {
	var attempts atomic.Int32
	client := New("http://example.test")
	client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		if attempts.Add(1) < 3 {
			return nil, errors.New("temporary network failure")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader("ok")),
		}, nil
	})
	client.Retry(retry.Policy{
		Attempts: 3,
		Initial: time.Millisecond,
		Multiplier: 1,
	})

	response, err := client.Get(context.Background(), "/")
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != http.StatusOK || attempts.Load() != 3 {
		t.Fatalf("unexpected response=%#v attempts=%d", response, attempts.Load())
	}
}

func TestPostDoesNotAutoRetry(t *testing.T) {
	var attempts atomic.Int32
	client := New("http://example.test")
	client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts.Add(1)
		return nil, errors.New("network failure")
	})
	client.Retry(retry.Policy{Attempts: 3, Initial: time.Millisecond})

	_, err := client.PostJSON(context.Background(), "/", map[string]any{"ok":true})
	if err == nil {
		t.Fatal("expected network error")
	}
	if attempts.Load() != 1 {
		t.Fatalf("POST must not auto-retry, attempts=%d", attempts.Load())
	}
}
