package core

import "net/http"

func (ctx *Context) Created(data any) error  { return ctx.Status(http.StatusCreated).JSON(data) }
func (ctx *Context) Accepted(data any) error { return ctx.Status(http.StatusAccepted).JSON(data) }
func (ctx *Context) IP() string              { return ctx.Request.RemoteAddr }
func (ctx *Context) BearerToken() string {
	const prefix = "Bearer "
	value := ctx.Request.Header.Get("Authorization")
	if len(value) > len(prefix) && value[:len(prefix)] == prefix {
		return value[len(prefix):]
	}
	return ""
}
