package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

type Client struct {
	Handler http.Handler
	Headers http.Header
}

type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

func NewClient(handler http.Handler) *Client {
	return &Client{
		Handler: handler,
		Headers: make(http.Header),
	}
}

func (c *Client) WithHeader(key, value string) *Client {
	c.Headers.Set(key, value)
	return c
}

func (c *Client) Request(method, path string, body io.Reader) (*Response, error) {
	if c == nil || c.Handler == nil {
		return nil, fmt.Errorf("copytygo testing: HTTP handler is required")
	}

	req := httptest.NewRequest(method, path, body)
	for key, values := range c.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	rec := httptest.NewRecorder()
	c.Handler.ServeHTTP(rec, req)

	return &Response{
		Status: rec.Code,
		Header: rec.Header().Clone(),
		Body:   append([]byte(nil), rec.Body.Bytes()...),
	}, nil
}

func (c *Client) Get(path string) (*Response, error) {
	return c.Request(http.MethodGet, path, nil)
}

func (c *Client) Delete(path string) (*Response, error) {
	return c.Request(http.MethodDelete, path, nil)
}

func (c *Client) PostJSON(path string, payload any) (*Response, error) {
	return c.jsonRequest(http.MethodPost, path, payload)
}

func (c *Client) PutJSON(path string, payload any) (*Response, error) {
	return c.jsonRequest(http.MethodPut, path, payload)
}

func (c *Client) PatchJSON(path string, payload any) (*Response, error) {
	return c.jsonRequest(http.MethodPatch, path, payload)
}

func (c *Client) jsonRequest(method, path string, payload any) (*Response, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("copytygo testing: encode JSON: %w", err)
	}

	previous := c.Headers.Get("Content-Type")
	c.Headers.Set("Content-Type", "application/json")
	response, requestErr := c.Request(method, path, bytes.NewReader(raw))
	if previous == "" {
		c.Headers.Del("Content-Type")
	} else {
		c.Headers.Set("Content-Type", previous)
	}
	return response, requestErr
}

func (r *Response) Text() string {
	if r == nil {
		return ""
	}
	return string(r.Body)
}

func (r *Response) JSON(target any) error {
	if r == nil {
		return fmt.Errorf("copytygo testing: response is nil")
	}
	if err := json.Unmarshal(r.Body, target); err != nil {
		return fmt.Errorf("copytygo testing: decode response JSON: %w", err)
	}
	return nil
}

func (r *Response) OK() bool {
	return r != nil && r.Status >= 200 && r.Status < 300
}
