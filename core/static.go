package core

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// StaticOptions reserves URL prefixes that must never resolve to public files.
type StaticOptions struct{ Exclude []string }
type staticMount struct {
	prefix, directory string
	exclude           []string
}

// Static serves files below directory after all application routes have been
// checked. Global middleware still runs. Directories and hidden files are not
// published; os.Root prevents filesystem traversal and symlinks escaping the root.
func (app *Application) Static(prefix, directory string, options ...StaticOptions) *Application {
	mount := staticMount{prefix: normalizePath(prefix), directory: directory}
	if len(options) > 0 {
		mount.exclude = append([]string{}, options[0].Exclude...)
	}
	app.router.staticFiles = append(app.router.staticFiles, mount)
	return app
}
func (router *Router) serveStatic(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	for _, mount := range router.staticFiles {
		if router.serveStaticMount(w, r, mount) {
			return true
		}
	}
	return false
}
func (router *Router) serveStaticMount(w http.ResponseWriter, r *http.Request, mount staticMount) bool {
	prefix := strings.TrimSuffix(mount.prefix, "/") + "/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return false
	}
	for _, excluded := range mount.exclude {
		excluded = strings.ToLower(normalizePath(excluded))
		requestPath := strings.ToLower(r.URL.Path)
		if requestPath == excluded || strings.HasPrefix(requestPath, strings.TrimSuffix(excluded, "/")+"/") {
			return false
		}
	}
	name := strings.TrimPrefix(r.URL.Path, prefix)
	if !fs.ValidPath(name) {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") || strings.ContainsAny(part, "\\:") {
			return false
		}
	}
	root, err := os.OpenRoot(mount.directory)
	if err != nil {
		return false
	}
	defer root.Close()
	file, err := root.Open(name)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	ctx := newContext(w, r)
	ctx.router = router
	handler := applyMiddleware(func(ctx *Context) error {
		http.ServeContent(ctx.Response, ctx.Request, path.Base(name), info.ModTime(), file)
		return nil
	}, router.middlewares)
	if err := handler(ctx); err != nil {
		router.handleError(ctx, err)
	}
	return true
}
