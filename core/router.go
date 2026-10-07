package core

import (
	"net/http"
	"strings"
)

type Handler func(ctx *Context) error

type Route struct {
	Method string
	Path   string

	Handler Handler

	name        string
	middlewares []Middleware
}

type RouteBuilder struct {
	route      *Route
	namePrefix string
}

type RouteGroup struct {
	router *Router

	prefix      string
	namePrefix  string
	middlewares []Middleware
}

type Router struct {
	routes      []*Route
	middlewares []Middleware
}

func NewRouter() *Router {
	return &Router{
		routes:      make([]*Route, 0),
		middlewares: make([]Middleware, 0),
	}
}

func (router *Router) Use(middlewares ...Middleware) {
	router.middlewares = append(router.middlewares, middlewares...)
}

func (router *Router) Routes() []*Route {
	return append([]*Route{}, router.routes...)
}

func (route *Route) NameValue() string { return route.name }

// ----------------------------------------------------
// ROUTE BUILDER
// ----------------------------------------------------

func (builder *RouteBuilder) Name(name string) *RouteBuilder {
	builder.route.name = builder.namePrefix + name
	return builder
}

func (builder *RouteBuilder) Middleware(
	middlewares ...Middleware,
) *RouteBuilder {

	builder.route.middlewares = append(
		builder.route.middlewares,
		middlewares...,
	)

	return builder
}

// ----------------------------------------------------
// REGISTER ROUTE
// ----------------------------------------------------

func (router *Router) Add(
	method string,
	path string,
	handler Handler,
) *RouteBuilder {

	route := &Route{
		Method:  strings.ToUpper(method),
		Path:    normalizePath(path),
		Handler: handler,

		middlewares: make([]Middleware, 0),
	}

	router.routes = append(router.routes, route)

	return &RouteBuilder{
		route: route,
	}
}

func (router *Router) Get(
	path string,
	handler Handler,
) *RouteBuilder {

	return router.Add(
		http.MethodGet,
		path,
		handler,
	)
}

func (router *Router) Post(
	path string,
	handler Handler,
) *RouteBuilder {

	return router.Add(
		http.MethodPost,
		path,
		handler,
	)
}

func (router *Router) Put(
	path string,
	handler Handler,
) *RouteBuilder {

	return router.Add(
		http.MethodPut,
		path,
		handler,
	)
}

func (router *Router) Patch(
	path string,
	handler Handler,
) *RouteBuilder {

	return router.Add(
		http.MethodPatch,
		path,
		handler,
	)
}

func (router *Router) Delete(
	path string,
	handler Handler,
) *RouteBuilder {

	return router.Add(
		http.MethodDelete,
		path,
		handler,
	)
}

// ----------------------------------------------------
// ROUTE GROUP
// ----------------------------------------------------

func (router *Router) Group(
	prefix string,
) *RouteGroup {

	return &RouteGroup{
		router: router,
		prefix: normalizePath(prefix),

		middlewares: make([]Middleware, 0),
	}
}

func (group *RouteGroup) Middleware(
	middlewares ...Middleware,
) *RouteGroup {

	group.middlewares = append(
		group.middlewares,
		middlewares...,
	)

	return group
}

func (group *RouteGroup) Name(prefix string) *RouteGroup {
	group.namePrefix += prefix
	return group
}

func (group *RouteGroup) Group(prefix string) *RouteGroup {
	return &RouteGroup{
		router:      group.router,
		prefix:      joinPaths(group.prefix, prefix),
		namePrefix:  group.namePrefix,
		middlewares: append([]Middleware{}, group.middlewares...),
	}
}

func (group *RouteGroup) Routes(
	callback func(route *RouteGroup),
) *RouteGroup {
	callback(group)

	return group
}

func (group *RouteGroup) add(
	method string,
	path string,
	handler Handler,
) *RouteBuilder {

	fullPath := joinPaths(
		group.prefix,
		path,
	)

	builder := group.router.Add(
		method,
		fullPath,
		handler,
	)

	// Important:
	// gunakan slice baru supaya route tidak berbagi
	// backing array middleware dengan group.
	builder.route.middlewares = append(
		[]Middleware{},
		group.middlewares...,
	)

	builder.namePrefix = group.namePrefix
	return builder
}

func (group *RouteGroup) Get(
	path string,
	handler Handler,
) *RouteBuilder {

	return group.add(
		http.MethodGet,
		path,
		handler,
	)
}

func (group *RouteGroup) Post(
	path string,
	handler Handler,
) *RouteBuilder {

	return group.add(
		http.MethodPost,
		path,
		handler,
	)
}

func (group *RouteGroup) Put(
	path string,
	handler Handler,
) *RouteBuilder {

	return group.add(
		http.MethodPut,
		path,
		handler,
	)
}

func (group *RouteGroup) Patch(
	path string,
	handler Handler,
) *RouteBuilder {

	return group.add(
		http.MethodPatch,
		path,
		handler,
	)
}

func (group *RouteGroup) Delete(
	path string,
	handler Handler,
) *RouteBuilder {

	return group.add(
		http.MethodDelete,
		path,
		handler,
	)
}

func (group *RouteGroup) Resource(path string, controller ResourceController) *RouteGroup {
	base := strings.TrimRight(path, "/")
	if base == "" {
		base = "/"
	}
	member := base
	if member == "/" {
		member = ""
	}
	member += "/:id"

	group.Get(base, controller.Index)
	group.Get(member, controller.Show)
	group.Post(base, controller.Store)
	group.Put(member, controller.Update)
	group.Delete(member, controller.Destroy)
	return group
}

// ----------------------------------------------------
// HTTP
// ----------------------------------------------------

func (router *Router) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {

	defer router.recoverPanic(w)

	requestPath := normalizePath(r.URL.Path)

	pathMatched := false

	for _, route := range router.routes {

		params, matched := matchPath(
			route.Path,
			requestPath,
		)

		if !matched {
			continue
		}

		pathMatched = true

		if route.Method != r.Method {
			continue
		}

		ctx := newContext(w, r)
		ctx.router = router

		ctx.params = params

		middlewares := append([]Middleware{}, router.middlewares...)
		middlewares = append(middlewares, route.middlewares...)

		handler := applyMiddleware(
			route.Handler,
			middlewares,
		)

		if err := handler(ctx); err != nil {
			router.handleError(ctx, err)
		}

		return
	}

	if pathMatched {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error":{"status":405,"message":"Method Not Allowed"}}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":{"status":404,"message":"Not Found"}}`))
}

// ----------------------------------------------------
// ERROR
// ----------------------------------------------------

func (router *Router) handleError(
	ctx *Context,
	err error,
) {

	if validationError, ok := err.(*ValidationError); ok {
		ctx.Status(http.StatusUnprocessableEntity)
		_ = ctx.JSON(Map{
			"error": Map{
				"status":  http.StatusUnprocessableEntity,
				"message": "Validation failed",
				"fields":  validationError.Fields,
			},
		})
		return
	}

	if httpError, ok := err.(*HTTPError); ok {

		ctx.Status(httpError.Status)

		_ = ctx.JSON(Map{
			"error": Map{
				"status":  httpError.Status,
				"message": httpError.Message,
			},
		})

		return
	}

	ctx.Status(
		http.StatusInternalServerError,
	)

	_ = ctx.JSON(Map{
		"error": Map{
			"status": http.StatusInternalServerError,

			"message": "Internal Server Error",
		},
	})
}

func (router *Router) recoverPanic(
	w http.ResponseWriter,
) {

	if recover() == nil {
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.WriteHeader(
		http.StatusInternalServerError,
	)

	_, _ = w.Write([]byte(
		`{"error":{"status":500,"message":"Internal Server Error"}}`,
	))
}

// ----------------------------------------------------
// PATH
// ----------------------------------------------------

func matchPath(
	routePath string,
	requestPath string,
) (map[string]string, bool) {

	routeParts := splitPath(routePath)
	requestParts := splitPath(requestPath)

	wildcard := len(routeParts) > 0 && strings.HasPrefix(routeParts[len(routeParts)-1], "*")
	if !wildcard && len(routeParts) != len(requestParts) || wildcard && len(requestParts) < len(routeParts)-1 {
		return nil, false
	}

	params := make(map[string]string)

	for index := range routeParts {

		routePart := routeParts[index]
		if wildcard && index == len(routeParts)-1 {
			name := strings.TrimPrefix(routePart, "*")
			if name == "" {
				return nil, false
			}
			params[name] = strings.Join(requestParts[index:], "/")
			return params, true
		}
		requestPart := requestParts[index]

		if strings.HasPrefix(
			routePart,
			":",
		) {

			paramName := strings.TrimPrefix(
				routePart,
				":",
			)

			if paramName == "" ||
				requestPart == "" {

				return nil, false
			}

			params[paramName] = requestPart

			continue
		}

		if routePart != requestPart {
			return nil, false
		}
	}

	return params, true
}

func normalizePath(path string) string {

	if path == "" {
		return "/"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if len(path) > 1 {
		path = strings.TrimSuffix(
			path,
			"/",
		)
	}

	return path
}

func splitPath(path string) []string {

	path = normalizePath(path)

	if path == "/" {
		return []string{}
	}

	return strings.Split(
		strings.Trim(path, "/"),
		"/",
	)
}

func joinPaths(
	prefix string,
	path string,
) string {

	if path == "/" {
		return normalizePath(prefix)
	}

	return normalizePath(
		strings.TrimSuffix(prefix, "/") +
			"/" +
			strings.TrimPrefix(path, "/"),
	)
}

func (router *Router) URL(name string, params map[string]string) (string, bool) {
	for _, route := range router.routes {
		if route.name != name {
			continue
		}
		path := route.Path
		for key, value := range params {
			path = strings.ReplaceAll(path, ":"+key, value)
		}
		if strings.Contains(path, ":") {
			return "", false
		}
		return path, true
	}
	return "", false
}
