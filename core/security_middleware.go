package core

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

func SecurityHeaders() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			h := ctx.Response.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			return next(ctx)
		}
	}
}
func BodyLimit(maxBytes int64) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			ctx.Request.Body = http.MaxBytesReader(ctx.Response, ctx.Request.Body, maxBytes)
			return next(ctx)
		}
	}
}
func CORS(origins ...string) Middleware {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[o] = true
	}
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			origin := ctx.Request.Header.Get("Origin")
			if allowed["*"] || allowed[origin] {
				ctx.Response.Header().Set("Access-Control-Allow-Origin", origin)
				if allowed["*"] {
					ctx.Response.Header().Set("Access-Control-Allow-Origin", "*")
				}
				ctx.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
				ctx.Response.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			}
			if ctx.Request.Method == http.MethodOptions {
				ctx.Response.WriteHeader(http.StatusNoContent)
				return nil
			}
			return next(ctx)
		}
	}
}
func CSRF(cookieName string) Middleware {
	if cookieName == "" {
		cookieName = "ctg_csrf"
	}
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			cookie, err := ctx.Request.Cookie(cookieName)
			if err != nil || cookie.Value == "" {
				raw := make([]byte, 32)
				_, _ = rand.Read(raw)
				token := base64.RawURLEncoding.EncodeToString(raw)
				http.SetCookie(ctx.Response, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: false, SameSite: http.SameSiteLaxMode, Secure: ctx.Request.TLS != nil, Expires: time.Now().Add(12 * time.Hour)})
				cookie = &http.Cookie{Value: token}
			}
			if ctx.Request.Method != http.MethodGet && ctx.Request.Method != http.MethodHead && ctx.Request.Method != http.MethodOptions {
				sent := ctx.Request.Header.Get("X-CSRF-Token")
				if sent == "" {
					sent = ctx.Request.FormValue("_token")
				}
				if !strings.EqualFold(sent, cookie.Value) {
					return NewHTTPError(http.StatusForbidden, "Invalid CSRF token")
				}
			}
			return next(ctx)
		}
	}
}
