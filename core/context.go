package core

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type Context struct {
	Response http.ResponseWriter
	Request  *http.Request

	params     map[string]string
	statusCode int
	values     map[string]any
}

type Map map[string]any

func newContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Response:   w,
		Request:    r,
		params:     make(map[string]string),
		statusCode: http.StatusOK,
		values:     make(map[string]any),
	}
}

func (ctx *Context) Param(name string) string {
	return ctx.params[name]
}

func (ctx *Context) Query(name string) string {
	return ctx.Request.URL.Query().Get(name)
}

func (ctx *Context) Body() string {
	if ctx.Request == nil || ctx.Request.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return ""
	}
	ctx.Request.Body = io.NopCloser(strings.NewReader(string(raw)))
	return string(raw)
}

func (ctx *Context) Status(code int) *Context {
	ctx.statusCode = code
	return ctx
}

func (ctx *Context) Text(value string) error {
	ctx.Response.Header().Set(
		"Content-Type",
		"text/plain; charset=utf-8",
	)

	ctx.Response.WriteHeader(ctx.statusCode)

	_, err := ctx.Response.Write([]byte(value))

	return err
}

func (ctx *Context) HTML(value string) error {
	ctx.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.Response.WriteHeader(ctx.statusCode)
	_, err := ctx.Response.Write([]byte(value))
	return err
}

func (ctx *Context) JSON(data any) error {
	ctx.Response.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	ctx.Response.WriteHeader(ctx.statusCode)

	return json.NewEncoder(ctx.Response).Encode(data)
}

func (ctx *Context) Redirect(url string) error {
	http.Redirect(
		ctx.Response,
		ctx.Request,
		url,
		http.StatusFound,
	)

	return nil
}

func (ctx *Context) Abort(status int, message string) error {
	return NewHTTPError(status, message)
}

func (ctx *Context) NotFound(message string) error {
	return NewHTTPError(
		http.StatusNotFound,
		message,
	)
}

func (ctx *Context) BadRequest(message string) error {
	return NewHTTPError(
		http.StatusBadRequest,
		message,
	)
}

func (ctx *Context) Unauthorized(message string) error {
	return NewHTTPError(
		http.StatusUnauthorized,
		message,
	)
}

func (ctx *Context) Forbidden(message string) error {
	return NewHTTPError(
		http.StatusForbidden,
		message,
	)
}

func (ctx *Context) Header(key, value string) *Context {
	ctx.Response.Header().Set(key, value)
	return ctx
}
func (ctx *Context) Cookie(name string) (string, bool) {
	cookie, err := ctx.Request.Cookie(name)
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}
func (ctx *Context) SetCookie(cookie *http.Cookie) { http.SetCookie(ctx.Response, cookie) }
func (ctx *Context) NoContent(status int) error    { ctx.Response.WriteHeader(status); return nil }


func (ctx *Context) Set(key string, value any) {
	if ctx.values == nil {
		ctx.values = make(map[string]any)
	}
	ctx.values[key] = value
}

func (ctx *Context) Get(key string) (any, bool) {
	if ctx.values == nil {
		return nil, false
	}
	value, ok := ctx.values[key]
	return value, ok
}

func ContextValue[T any](ctx *Context, key string) (T, bool) {
	var zero T
	value, ok := ctx.Get(key)
	if !ok {
		return zero, false
	}
	typed, ok := value.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}
