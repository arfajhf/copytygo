// Package frontend connects the TypeScript frontend to the Go HTTP server.
package frontend

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/core"
)

// Page serves the auth entry point. In ctg dev it proxies Vite so the UI and
// session APIs share one origin. Production reads the compiled frontend files.
func Page() core.Handler {
	return func(ctx *core.Context) error {
		ctx.Header("Cache-Control", "no-store")
		if proxy := devProxy(); proxy != nil {
			if ctx.Request.Header.Get("Upgrade") == "websocket" {
				proxy.ServeHTTP(ctx.Response, ctx.Request)
				return nil
			}
			request := ctx.Request.Clone(ctx.Request.Context())
			request.URL.Path = "/auth.html"
			proxy.ServeHTTP(ctx.Response, request)
			return nil
		}
		path := filepath.Join(distDirectory(), "auth.html")
		if _, err := os.Stat(path); err != nil {
			return core.NewHTTPError(503, "Frontend is not built. Run ctg build, or use ctg dev for development.")
		}
		http.ServeFile(ctx.Response, ctx.Request, path)
		return nil
	}
}

// Assets serves Vite modules in development and all public files in both
// development and production. Application/API routes always take precedence.
func Assets(app *core.Application) {
	const marker = "copytygo.frontend.assets"
	for _, route := range app.Routes() {
		if route.NameValue() == marker {
			return
		}
	}
	for index, prefix := range []string{"/src/*path", "/@vite/*path", "/@id/*path", "/@fs/*path", "/node_modules/*path"} {
		route := app.Get(prefix, func(ctx *core.Context) error {
			if proxy := devProxy(); proxy != nil {
				proxy.ServeHTTP(ctx.Response, ctx.Request)
				return nil
			}
			return core.NewHTTPError(404, "Not Found")
		})
		if index == 0 {
			route.Name(marker)
		}
	}
	options := core.StaticOptions{Exclude: []string{"/api", "/__copytygo"}}
	if !strings.EqualFold(config.Get("APP_ENV", "local"), "production") {
		app.Static("/", filepath.Join("frontend", "public"), options)
		if devProxy() != nil {
			return
		}
	}
	app.Static("/", distDirectory(), options)
}
func distDirectory() string {
	if dir := config.Get("COPYTYGO_FRONTEND_DIST"); dir != "" {
		return dir
	}
	return filepath.Join("frontend", "dist")
}
func devProxy() *httputil.ReverseProxy {
	if strings.EqualFold(config.Get("APP_ENV", "local"), "production") {
		return nil
	}
	origin := os.Getenv("COPYTYGO_VITE_ORIGIN")
	if origin == "" {
		return nil
	}
	target, err := url.Parse(origin)
	if err != nil || target.Scheme != "http" || target.Hostname() != "127.0.0.1" {
		return nil
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprint("TypeScript frontend unavailable. Restart ctg dev."), http.StatusServiceUnavailable)
	}
	return proxy
}
