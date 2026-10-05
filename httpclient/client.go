package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	headers http.Header
	client  *http.Client
}
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, headers: make(http.Header), client: &http.Client{Timeout: 15 * time.Second}}
}
func (c *Client) Timeout(d time.Duration) *Client  { c.client.Timeout = d; return c }
func (c *Client) Header(key, value string) *Client { c.headers.Set(key, value); return c }
func (c *Client) Bearer(token string) *Client      { return c.Header("Authorization", "Bearer "+token) }
func (c *Client) Get(ctx context.Context, path string) (*Response, error) {
	return c.Do(ctx, http.MethodGet, path, nil)
}
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return c.Do(ctx, http.MethodDelete, path, nil)
}
func (c *Client) PostJSON(ctx context.Context, path string, body any) (*Response, error) {
	return c.json(ctx, http.MethodPost, path, body)
}
func (c *Client) PutJSON(ctx context.Context, path string, body any) (*Response, error) {
	return c.json(ctx, http.MethodPut, path, body)
}
func (c *Client) json(ctx context.Context, method, path string, body any) (*Response, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, method, path, bytes.NewReader(raw))
}
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header = c.headers.Clone()
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	return &Response{Status: res.StatusCode, Header: res.Header.Clone(), Body: raw}, nil
}
func (r *Response) OK() bool { return r.Status >= 200 && r.Status < 300 }
func (r *Response) JSON(target any) error {
	if err := json.Unmarshal(r.Body, target); err != nil {
		return fmt.Errorf("copytygo: decode response: %w", err)
	}
	return nil
}
func (r *Response) String() string { return string(r.Body) }
