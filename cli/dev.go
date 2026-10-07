package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"net/http"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"strings"

	copyauth "github.com/arfajhf/copytygo/v4/auth"
	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/security"
	"github.com/arfajhf/copytygo/v4/validation"
)

type liteRoute struct {
	Method         string
	Path           string
	ContentType    string
	Status         int
	Body           string
	Validation     map[string]string
	ResourceAction string
	Resource       string
	ResourceID     string
	ResourceData   string
	AuthAction     string
	AuthRole       string
	Searchable     []string
	Filterable     []string
	Sortable       []string
	SoftDeletes    bool
}

type liteMemoryBucket struct {
	NextID int64
	Items  map[string]map[string]any
}

var liteMemoryStore = struct {
	sync.RWMutex
	Buckets map[string]*liteMemoryBucket
}{
	Buckets: make(map[string]*liteMemoryBucket),
}

func Dev(args []string) error {
	forceLite := false
	for _, arg := range args {
		if arg == "--lite" {
			forceLite = true
		}
	}

	if forceLite {
		return runLiteDev()
	}

	fmt.Println("CopyTyGo Dev")
	fmt.Println("-------------")
	fmt.Println("Runtime : native")
	fmt.Println()

	err := runNativeDev()
	if err == nil {
		return nil
	}

	if !looksLikeExecutionPolicyBlock(err) {
		return err
	}

	fmt.Println()
	fmt.Println("Native execution is blocked by the operating system.")
	fmt.Println("Switching automatically to CopyTyGo Lite Runtime...")
	fmt.Println()

	return runLiteDev()
}

func runNativeDev() error {
	cmd := exec.Command("go", "run", "./cmd/app")
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err == nil {
		return nil
	}

	message := strings.TrimSpace(stderr.String())
	if message != "" {
		fmt.Fprintln(os.Stderr, message)
	}

	return fmt.Errorf("copytygo native runtime: %w: %s", err, message)
}

func looksLikeExecutionPolicyBlock(err error) bool {
	if err == nil || runtime.GOOS != "windows" {
		return false
	}
	text := strings.ToLower(err.Error())
	needles := []string{
		"application control policy",
		"blocked this file",
		"access is denied",
		"operation did not complete successfully because the file contains",
	}
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func runLiteDev() error {
	if err := config.LoadEnv(".env"); err != nil {
		return err
	}

	routes, err := discoverLiteRoutes("routes")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if handleLiteStudio(w, req) {
			return
		}
		currentRoutes, reloadErr := discoverLiteRoutes("routes")
		if reloadErr != nil {
			http.Error(w, "CopyTyGo Lite reload error: "+reloadErr.Error(), http.StatusInternalServerError)
			return
		}
		for _, route := range currentRoutes {
			params, matched := matchLitePath(route.Path, req.URL.Path)
			if route.Method == req.Method && matched {
				if fields := liteValidateRequest(req, route.Validation); len(fields) > 0 {
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusUnprocessableEntity)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]any{
							"status":  http.StatusUnprocessableEntity,
							"message": "Validation failed",
							"fields":  fields,
						},
					})
					return
				}
				if route.AuthAction != "" {
					handleLiteAuthRoute(w, req, route)
					return
				}
				if strings.HasPrefix(route.ResourceAction, "db:") {
					handleLiteDatabaseRoute(w, req, route, params)
					return
				}
				if route.ResourceAction != "" {
					handleLiteMemoryRoute(w, req, route, params)
					return
				}

				body := renderLiteBody(route.Body, req, params)
				w.Header().Set("Content-Type", route.ContentType)
				w.WriteHeader(route.Status)
				_, _ = w.Write([]byte(body))
				return
			}
		}
		if req.Method == http.MethodGet && req.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprint(w, liteWelcomePage())
			return
		}
		http.NotFound(w, req)
	})

	host := config.Get("APP_HOST", "127.0.0.1")
	port := config.Get("APP_PORT", "8080")
	listener, address, moved, err := listenLiteAddress(host, port)
	if err != nil {
		return err
	}
	if moved {
		fmt.Printf("Port %s is already in use. CopyTyGo switched automatically to %s.\n\n", port, strings.TrimPrefix(address, host+":"))
	}

	fmt.Println("CopyTyGo Dev")
	fmt.Println("-------------")
	fmt.Println("Runtime     : lite")
	fmt.Println("Application : http://" + address)
	if !strings.EqualFold(config.Get("APP_ENV", "local"), "production") && config.GetBool("COPYTYGO_STUDIO", true) {
		fmt.Println("Studio      : http://" + address + "/__copytygo")
	}
	fmt.Printf("Routes      : %d\n", len(routes))
	fmt.Println()
	fmt.Println("Lite Runtime keeps Welcome and Studio available even when no project routes can be interpreted.")
	fmt.Println("Supported inline Text/JSON routes still reload automatically.")
	fmt.Println("Complex controllers, middleware, database calls and arbitrary Go packages still use native mode.")
	fmt.Println()

	return http.Serve(listener, requestLogMiddleware(mux))
}

func handleLiteAuthRoute(w http.ResponseWriter, req *http.Request, route liteRoute) {
	data := liteRequestData(req)

	switch route.AuthAction {
	case "register":
		session, err := copyauth.Register(
			data["name"],
			data["email"],
			data["password"],
			route.AuthRole,
		)
		if err != nil {
			switch {
			case errors.Is(err, copyauth.ErrEmailRegistered):
				writeLiteJSON(w, http.StatusConflict, map[string]any{
					"error": map[string]any{
						"status": http.StatusConflict,
						"message": "Email already registered",
					},
				})
			case errors.Is(err, copyauth.ErrInvalidRegistration):
				writeLiteJSON(w, http.StatusBadRequest, map[string]any{
					"error": map[string]any{
						"status": http.StatusBadRequest,
						"message": "Invalid registration data",
					},
				})
			default:
				writeLiteDatabaseError(w, err)
			}
			return
		}
		writeLiteJSON(w, http.StatusCreated, session)

	case "login":
		session, err := copyauth.Login(data["email"], data["password"])
		if err != nil {
			if errors.Is(err, copyauth.ErrInvalidCredentials) {
				writeLiteJSON(w, http.StatusUnauthorized, map[string]any{
					"error": map[string]any{
						"status": http.StatusUnauthorized,
						"message": "Invalid credentials",
					},
				})
				return
			}
			writeLiteDatabaseError(w, err)
			return
		}
		writeLiteJSON(w, http.StatusOK, session)

	case "me":
		const prefix = "Bearer "
		header := req.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			writeLiteJSON(w, http.StatusUnauthorized, map[string]any{
				"error": map[string]any{
					"status": http.StatusUnauthorized,
					"message": "Authentication required",
				},
			})
			return
		}

		claims, err := security.VerifyToken(
			config.Get("APP_KEY"),
			strings.TrimPrefix(header, prefix),
		)
		if err != nil {
			writeLiteJSON(w, http.StatusUnauthorized, map[string]any{
				"error": map[string]any{
					"status": http.StatusUnauthorized,
					"message": "Invalid authentication token",
				},
			})
			return
		}

		writeLiteJSON(w, http.StatusOK, map[string]any{
			"id": claims.Subject,
			"role": claims.Role,
			"name": claims.Data["name"],
			"email": claims.Data["email"],
			"exp": claims.ExpiresAt,
		})
	}
}

func handleLiteDatabaseRoute(w http.ResponseWriter, req *http.Request, route liteRoute, params map[string]string) {
	drivers.Register()

	db, err := database.Connect()
	if err != nil {
		writeLiteJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]any{
				"status": http.StatusInternalServerError,
				"message": err.Error(),
			},
		})
		return
	}

	resource, err := database.NewResource(db, config.Get("DB_DRIVER", "mysql"), route.Resource)
	if err != nil {
		writeLiteJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]any{
				"status": http.StatusInternalServerError,
				"message": err.Error(),
			},
		})
		return
	}
	if route.SoftDeletes {
		resource.WithSoftDeletes()
	}

	id := renderLiteBody(route.ResourceID, req, params)

	switch route.ResourceAction {
	case "db:index":
		page, _ := strconv.Atoi(req.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(req.URL.Query().Get("per_page"))
		filters := make(map[string]string)
		for _, field := range route.Filterable {
			filters[field] = req.URL.Query().Get("filter[" + field + "]")
		}

		result, err := resource.List(database.ResourceListOptions{
			Page:          page,
			PerPage:       perPage,
			Search:        req.URL.Query().Get("q"),
			SearchColumns: route.Searchable,
			Sort:          req.URL.Query().Get("sort"),
			Order:         req.URL.Query().Get("order"),
			Filters:       filters,
			Filterable:    route.Filterable,
			Sortable:      route.Sortable,
		})
		if err != nil {
			writeLiteDatabaseError(w, err)
			return
		}
		writeLiteJSON(w, http.StatusOK, result)

	case "db:show":
		item, found, err := resource.Show(id)
		if err != nil {
			writeLiteDatabaseError(w, err)
			return
		}
		if !found {
			writeLiteNotFound(w)
			return
		}
		writeLiteJSON(w, http.StatusOK, item)

	case "db:store":
		data, ok := decodeLiteResourceData(route.ResourceData, req, params)
		if !ok {
			writeLiteJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]any{
					"status": http.StatusBadRequest,
					"message": "Invalid resource data",
				},
			})
			return
		}
		item, err := resource.Store(data)
		if err != nil {
			writeLiteDatabaseError(w, err)
			return
		}
		writeLiteJSON(w, http.StatusCreated, item)

	case "db:update":
		data, ok := decodeLiteResourceData(route.ResourceData, req, params)
		if !ok {
			writeLiteJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]any{
					"status": http.StatusBadRequest,
					"message": "Invalid resource data",
				},
			})
			return
		}
		item, found, err := resource.Update(id, data)
		if err != nil {
			writeLiteDatabaseError(w, err)
			return
		}
		if !found {
			writeLiteNotFound(w)
			return
		}
		writeLiteJSON(w, http.StatusOK, item)

	case "db:destroy":
		deleted, err := resource.Destroy(id)
		if err != nil {
			writeLiteDatabaseError(w, err)
			return
		}
		if !deleted {
			writeLiteNotFound(w)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeLiteJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]any{
				"status": http.StatusInternalServerError,
				"message": "Unsupported Lite database action",
			},
		})
	}
}

func writeLiteDatabaseError(w http.ResponseWriter, err error) {
	message := "Database operation failed"
	if config.GetBool("APP_DEBUG", false) {
		message = err.Error()
	}
	writeLiteJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]any{
			"status": http.StatusInternalServerError,
			"message": message,
		},
	})
}

func liteMemoryBucketFor(resource string) *liteMemoryBucket {
	liteMemoryStore.Lock()
	defer liteMemoryStore.Unlock()

	bucket, ok := liteMemoryStore.Buckets[resource]
	if !ok {
		bucket = &liteMemoryBucket{
			NextID: 1,
			Items:  make(map[string]map[string]any),
		}
		liteMemoryStore.Buckets[resource] = bucket
	}
	return bucket
}

func handleLiteMemoryRoute(w http.ResponseWriter, req *http.Request, route liteRoute, params map[string]string) {
	bucket := liteMemoryBucketFor(route.Resource)
	id := renderLiteBody(route.ResourceID, req, params)

	switch route.ResourceAction {
	case "index":
		liteMemoryStore.RLock()
		ids := make([]string, 0, len(bucket.Items))
		for itemID := range bucket.Items {
			ids = append(ids, itemID)
		}
		sort.Slice(ids, func(i, j int) bool {
			a, _ := strconv.Atoi(ids[i])
			b, _ := strconv.Atoi(ids[j])
			return a < b
		})
		rows := make([]map[string]any, 0, len(ids))
		for _, itemID := range ids {
			row := map[string]any{"id": itemID}
			for key, value := range bucket.Items[itemID] {
				row[key] = value
			}
			rows = append(rows, row)
		}
		liteMemoryStore.RUnlock()

		writeLiteJSON(w, http.StatusOK, map[string]any{"data": rows})
		return

	case "show":
		liteMemoryStore.RLock()
		item, ok := bucket.Items[id]
		if ok {
			item = cloneLiteRecord(item)
		}
		liteMemoryStore.RUnlock()

		if !ok {
			writeLiteNotFound(w)
			return
		}

		row := map[string]any{"id": id}
		for key, value := range item {
			row[key] = value
		}
		writeLiteJSON(w, http.StatusOK, row)
		return

	case "store":
		data, ok := decodeLiteResourceData(route.ResourceData, req, params)
		if !ok {
			writeLiteJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]any{
					"status":  http.StatusBadRequest,
					"message": "Invalid resource data",
				},
			})
			return
		}

		liteMemoryStore.Lock()
		id := strconv.FormatInt(bucket.NextID, 10)
		bucket.NextID++
		bucket.Items[id] = cloneLiteRecord(data)
		liteMemoryStore.Unlock()

		row := map[string]any{"id": id}
		for key, value := range data {
			row[key] = value
		}
		writeLiteJSON(w, http.StatusCreated, row)
		return

	case "update":
		data, ok := decodeLiteResourceData(route.ResourceData, req, params)
		if !ok {
			writeLiteJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]any{
					"status":  http.StatusBadRequest,
					"message": "Invalid resource data",
				},
			})
			return
		}

		liteMemoryStore.Lock()
		item, exists := bucket.Items[id]
		if exists {
			for key, value := range data {
				item[key] = value
			}
			bucket.Items[id] = item
			item = cloneLiteRecord(item)
		}
		liteMemoryStore.Unlock()

		if !exists {
			writeLiteNotFound(w)
			return
		}

		row := map[string]any{"id": id}
		for key, value := range item {
			row[key] = value
		}
		writeLiteJSON(w, http.StatusOK, row)
		return

	case "destroy":
		liteMemoryStore.Lock()
		_, exists := bucket.Items[id]
		if exists {
			delete(bucket.Items, id)
		}
		liteMemoryStore.Unlock()

		if !exists {
			writeLiteNotFound(w)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeLiteJSON(w, http.StatusInternalServerError, map[string]any{
		"error": map[string]any{
			"status":  http.StatusInternalServerError,
			"message": "Unsupported Lite resource action",
		},
	})
}

func decodeLiteResourceData(template string, req *http.Request, params map[string]string) (map[string]any, bool) {
	rendered := renderLiteBody(template, req, params)
	var data map[string]any
	if err := json.Unmarshal([]byte(rendered), &data); err != nil {
		return nil, false
	}
	return data, true
}

func cloneLiteRecord(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func writeLiteNotFound(w http.ResponseWriter) {
	writeLiteJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"status":  http.StatusNotFound,
			"message": "Resource not found",
		},
	})
}

func writeLiteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func listenLiteAddress(host, preferredPort string) (net.Listener, string, bool, error) {
	start, err := strconv.Atoi(preferredPort)
	if err != nil || start < 1 || start > 65535 {
		return nil, "", false, fmt.Errorf("copytygo lite runtime: invalid APP_PORT %q", preferredPort)
	}

	for port := start; port <= start+20 && port <= 65535; port++ {
		address := net.JoinHostPort(host, strconv.Itoa(port))
		listener, listenErr := net.Listen("tcp", address)
		if listenErr == nil {
			return listener, address, port != start, nil
		}
	}
	return nil, "", false, fmt.Errorf("copytygo lite runtime: no free port found from %d to %d", start, minInt(start+20, 65535))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func discoverLiteRoutes(dir string) ([]liteRoute, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("copytygo lite runtime: %s directory not found", dir)
		}
		return nil, err
	}

	var routes []liteRoute
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		fileRoutes, err := parseLiteRouteFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		routes = append(routes, fileRoutes...)
	}
	return routes, nil
}

func parseLiteRouteFile(path string) ([]liteRoute, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	controllerVars := discoverControllerVars(file)

	var routes []liteRoute
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		if selector.Sel.Name == "Routes" && len(call.Args) == 2 {
			if appIdent, ok := call.Args[0].(*ast.Ident); ok && appIdent.Name == "app" {
				defaultRole, roleOK := stringLiteral(call.Args[1])
				if roleOK {
					routes = append(routes,
						liteRoute{
							Method: "POST",
							Path: "/api/auth/register",
							Status: http.StatusCreated,
							ContentType: "application/json; charset=utf-8",
							Validation: map[string]string{
								"name": "required|min:3",
								"email": "required|email",
								"password": "required|min:8",
							},
							AuthAction: "register",
							AuthRole: defaultRole,
						},
						liteRoute{
							Method: "POST",
							Path: "/api/auth/login",
							Status: http.StatusOK,
							ContentType: "application/json; charset=utf-8",
							Validation: map[string]string{
								"email": "required|email",
								"password": "required|min:8",
							},
							AuthAction: "login",
						},
						liteRoute{
							Method: "GET",
							Path: "/api/auth/me",
							Status: http.StatusOK,
							ContentType: "application/json; charset=utf-8",
							AuthAction: "me",
						},
					)
					return true
				}
			}
		}

		method := strings.ToUpper(selector.Sel.Name)
		routePath, ok := stringLiteral(call.Args[0])
		if !ok {
			return true
		}

		if method == "RESOURCE" {
			instance, ok := call.Args[1].(*ast.Ident)
			if !ok {
				return true
			}
			controllerType, ok := controllerVars[instance.Name]
			if !ok {
				return true
			}
			base := strings.TrimRight(routePath, "/")
			if base == "" {
				base = "/"
			}
			member := base
			if member == "/" {
				member = ""
			}
			member += "/:id"

			definitions := []struct {
				Method string
				Path   string
				Handler string
			}{
				{"GET", base, "Index"},
				{"GET", member, "Show"},
				{"POST", base, "Store"},
				{"PUT", member, "Update"},
				{"DELETE", member, "Destroy"},
			}
			for _, definition := range definitions {
				route, supported, controllerErr := parseLiteControllerHandler(
					filepath.Join("app", "controllers"),
					controllerType,
					definition.Handler,
					definition.Method,
					definition.Path,
				)
				if controllerErr == nil && supported {
					routes = append(routes, route)
				}
			}
			return true
		}

		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE":
		default:
			return true
		}

		if handler, ok := call.Args[1].(*ast.FuncLit); ok {
			route, supported := parseLiteHandler(method, routePath, handler)
			if supported {
				routes = append(routes, route)
			}
			return true
		}

		if handler, ok := call.Args[1].(*ast.SelectorExpr); ok {
			instance, ok := handler.X.(*ast.Ident)
			if !ok {
				return true
			}
			controllerType, ok := controllerVars[instance.Name]
			if !ok {
				return true
			}
			route, supported, controllerErr := parseLiteControllerHandler(
				filepath.Join("app", "controllers"),
				controllerType,
				handler.Sel.Name,
				method,
				routePath,
			)
			if controllerErr != nil {
				return true
			}
			if supported {
				routes = append(routes, route)
			}
		}
		return true
	})
	return routes, nil
}

func discoverControllerVars(file *ast.File) map[string]string {
	out := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range assign.Rhs {
			if i >= len(assign.Lhs) {
				break
			}
			name, ok := assign.Lhs[i].(*ast.Ident)
			if !ok {
				continue
			}
			if controllerType, ok := controllerTypeFromExpr(rhs); ok {
				out[name.Name] = controllerType
			}
		}
		return true
	})
	return out
}

func controllerTypeFromExpr(expr ast.Expr) (string, bool) {
	if unary, ok := expr.(*ast.UnaryExpr); ok {
		expr = unary.X
	}
	composite, ok := expr.(*ast.CompositeLit)
	if !ok {
		return "", false
	}
	switch typ := composite.Type.(type) {
	case *ast.SelectorExpr:
		return typ.Sel.Name, true
	case *ast.Ident:
		return typ.Name, true
	default:
		return "", false
	}
}

func parseLiteControllerHandler(dir, controllerType, method, httpMethod, routePath string) (liteRoute, bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return liteRoute{}, false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return liteRoute{}, false, err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != method || fn.Body == nil {
				continue
			}
			if !receiverMatches(fn, controllerType) {
				continue
			}
			handler := &ast.FuncLit{Body: fn.Body}
			route, supported := parseLiteHandler(httpMethod, routePath, handler)
			return route, supported, nil
		}
	}
	return liteRoute{}, false, nil
}

func receiverMatches(fn *ast.FuncDecl, controllerType string) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}
	expr := fn.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == controllerType
}

func parseLiteHandler(method, path string, handler *ast.FuncLit) (liteRoute, bool) {
	route := liteRoute{
		Method:     method,
		Path:       path,
		Status:     http.StatusOK,
		Validation: parseLiteValidationRules(handler),
	}

	for _, stmt := range handler.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		call, ok := ret.Results[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		if status, ok := liteStatusFromSelector(sel); ok {
			route.Status = status
		}

		switch sel.Sel.Name {
		case "Text":
			if len(call.Args) != 1 {
				return liteRoute{}, false
			}
			value, ok := liteStringExpr(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			route.ContentType = "text/plain; charset=utf-8"
			route.Body = value
			return route, true

		case "JSON":
			if len(call.Args) != 1 {
				return liteRoute{}, false
			}
			body, ok := mapLiteralJSON(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			route.ContentType = "application/json; charset=utf-8"
			route.Body = body + "\n"
			return route, true

		case "DBIndex":
			if len(call.Args) < 1 || len(call.Args) > 2 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			if len(call.Args) == 2 {
				searchable, filterable, sortable, softDeletes, ok := parseLiteDBIndexOptions(call.Args[1])
				if !ok {
					return liteRoute{}, false
				}
				route.Searchable = searchable
				route.Filterable = filterable
				route.Sortable = sortable
				route.SoftDeletes = softDeletes
			}
			route.ResourceAction = "db:index"
			route.Resource = resource
			route.ContentType = "application/json; charset=utf-8"
			return route, true

		case "DBShow":
			if len(call.Args) < 2 || len(call.Args) > 3 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			id, ok := liteStringExpr(call.Args[1])
			if !ok {
				return liteRoute{}, false
			}
			route.ResourceAction = "db:show"
			route.Resource = resource
			route.ResourceID = id
			if len(call.Args) == 3 {
				softDeletes, ok := parseLiteDBResourceOptions(call.Args[2])
				if !ok {
					return liteRoute{}, false
				}
				route.SoftDeletes = softDeletes
			}
			route.ContentType = "application/json; charset=utf-8"
			return route, true

		case "DBStore":
			if len(call.Args) < 2 || len(call.Args) > 3 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			data, ok := mapLiteralJSON(call.Args[1])
			if !ok {
				return liteRoute{}, false
			}
			route.ResourceAction = "db:store"
			route.Resource = resource
			route.ResourceData = data
			if len(call.Args) == 3 {
				softDeletes, ok := parseLiteDBResourceOptions(call.Args[2])
				if !ok {
					return liteRoute{}, false
				}
				route.SoftDeletes = softDeletes
			}
			route.Status = http.StatusCreated
			route.ContentType = "application/json; charset=utf-8"
			return route, true

		case "DBUpdate":
			if len(call.Args) < 3 || len(call.Args) > 4 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			id, ok := liteStringExpr(call.Args[1])
			if !ok {
				return liteRoute{}, false
			}
			data, ok := mapLiteralJSON(call.Args[2])
			if !ok {
				return liteRoute{}, false
			}
			route.ResourceAction = "db:update"
			route.Resource = resource
			route.ResourceID = id
			route.ResourceData = data
			if len(call.Args) == 4 {
				softDeletes, ok := parseLiteDBResourceOptions(call.Args[3])
				if !ok {
					return liteRoute{}, false
				}
				route.SoftDeletes = softDeletes
			}
			route.ContentType = "application/json; charset=utf-8"
			return route, true

		case "DBDestroy":
			if len(call.Args) < 2 || len(call.Args) > 3 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			id, ok := liteStringExpr(call.Args[1])
			if !ok {
				return liteRoute{}, false
			}
			route.ResourceAction = "db:destroy"
			route.Resource = resource
			route.ResourceID = id
			if len(call.Args) == 3 {
				softDeletes, ok := parseLiteDBResourceOptions(call.Args[2])
				if !ok {
					return liteRoute{}, false
				}
				route.SoftDeletes = softDeletes
			}
			route.Status = http.StatusNoContent
			return route, true

		case "MemoryIndex":
			if len(call.Args) != 1 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			route.ResourceAction = "index"
			route.Resource = resource
			route.ContentType = "application/json; charset=utf-8"
			return route, true

		case "MemoryShow":
			if len(call.Args) != 2 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			id, ok := liteStringExpr(call.Args[1])
			if !ok {
				return liteRoute{}, false
			}
			route.ResourceAction = "show"
			route.Resource = resource
			route.ResourceID = id
			route.ContentType = "application/json; charset=utf-8"
			return route, true

		case "MemoryStore":
			if len(call.Args) != 2 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			data, ok := mapLiteralJSON(call.Args[1])
			if !ok {
				return liteRoute{}, false
			}
			route.ResourceAction = "store"
			route.Resource = resource
			route.ResourceData = data
			route.Status = http.StatusCreated
			route.ContentType = "application/json; charset=utf-8"
			return route, true

		case "MemoryUpdate":
			if len(call.Args) != 3 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			id, ok := liteStringExpr(call.Args[1])
			if !ok {
				return liteRoute{}, false
			}
			data, ok := mapLiteralJSON(call.Args[2])
			if !ok {
				return liteRoute{}, false
			}
			route.ResourceAction = "update"
			route.Resource = resource
			route.ResourceID = id
			route.ResourceData = data
			route.ContentType = "application/json; charset=utf-8"
			return route, true

		case "MemoryDestroy":
			if len(call.Args) != 2 {
				return liteRoute{}, false
			}
			resource, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			id, ok := liteStringExpr(call.Args[1])
			if !ok {
				return liteRoute{}, false
			}
			route.ResourceAction = "destroy"
			route.Resource = resource
			route.ResourceID = id
			route.Status = http.StatusNoContent
			return route, true
		}
	}
	return liteRoute{}, false
}

func parseLiteDBIndexOptions(expr ast.Expr) ([]string, []string, []string, bool, bool) {
	composite, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, nil, nil, false, false
	}

	var searchable []string
	var filterable []string
	var sortable []string
	softDeletes := false

	for _, element := range composite.Elts {
		kv, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		if key.Name == "SoftDeletes" {
			ident, ok := kv.Value.(*ast.Ident)
			if !ok || (ident.Name != "true" && ident.Name != "false") {
				return nil, nil, nil, false, false
			}
			softDeletes = ident.Name == "true"
			continue
		}

		values, ok := liteStringSlice(kv.Value)
		if !ok {
			return nil, nil, nil, false, false
		}

		switch key.Name {
		case "Searchable":
			searchable = values
		case "Filterable":
			filterable = values
		case "Sortable":
			sortable = values
		}
	}

	return searchable, filterable, sortable, softDeletes, true
}

func parseLiteDBResourceOptions(expr ast.Expr) (bool, bool) {
	composite, ok := expr.(*ast.CompositeLit)
	if !ok {
		return false, false
	}
	for _, element := range composite.Elts {
		kv, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "SoftDeletes" {
			continue
		}
		ident, ok := kv.Value.(*ast.Ident)
		if !ok || (ident.Name != "true" && ident.Name != "false") {
			return false, false
		}
		return ident.Name == "true", true
	}
	return false, true
}

func liteStringSlice(expr ast.Expr) ([]string, bool) {
	composite, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, false
	}

	values := make([]string, 0, len(composite.Elts))
	for _, element := range composite.Elts {
		value, ok := stringLiteral(element)
		if !ok {
			return nil, false
		}
		values = append(values, value)
	}

	return values, true
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	return value, err == nil
}

func mapLiteralJSON(expr ast.Expr) (string, bool) {
	composite, ok := expr.(*ast.CompositeLit)
	if !ok {
		return "", false
	}
	var b strings.Builder
	b.WriteByte('{')
	first := true
	for _, element := range composite.Elts {
		kv, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return "", false
		}
		key, ok := stringLiteral(kv.Key)
		if !ok {
			ident, identOK := kv.Key.(*ast.Ident)
			if !identOK {
				return "", false
			}
			key = ident.Name
		}
		value, ok := jsonScalar(kv.Value)
		if !ok {
			if composite, compositeOK := kv.Value.(*ast.CompositeLit); compositeOK && len(composite.Elts) == 0 {
				value = "[]"
				ok = true
			}
		}
		if !ok {
			return "", false
		}
		if !first {
			b.WriteByte(',')
		}
		first = false
		b.WriteString(strconv.Quote(key))
		b.WriteByte(':')
		b.WriteString(value)
	}
	b.WriteByte('}')
	return b.String(), true
}

func jsonScalar(expr ast.Expr) (string, bool) {
	switch v := expr.(type) {
	case *ast.BasicLit:
		switch v.Kind {
		case token.STRING:
			s, err := strconv.Unquote(v.Value)
			if err != nil {
				return "", false
			}
			return strconv.Quote(s), true
		case token.INT, token.FLOAT:
			return v.Value, true
		}
	case *ast.Ident:
		if v.Name == "true" || v.Name == "false" || v.Name == "nil" {
			if v.Name == "nil" {
				return "null", true
			}
			return v.Name, true
		}
	}
	if value, ok := liteStringExpr(expr); ok {
		return strconv.Quote(value), true
	}
	return "", false
}

func liteStringExpr(expr ast.Expr) (string, bool) {
	if value, ok := stringLiteral(expr); ok {
		return value, true
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if sel.Sel.Name == "Body" && len(call.Args) == 0 {
		return "{{body}}", true
	}
	if len(call.Args) != 1 {
		return "", false
	}
	name, ok := stringLiteral(call.Args[0])
	if !ok {
		return "", false
	}
	switch sel.Sel.Name {
	case "Param":
		return "{{param:" + name + "}}", true
	case "Query":
		return "{{query:" + name + "}}", true
	case "Input":
		return "{{input:" + name + "}}", true
	}
	return "", false
}

func parseLiteValidationRules(handler *ast.FuncLit) map[string]string {
	rules := map[string]string{}
	ast.Inspect(handler.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Validate" {
			return true
		}
		composite, ok := call.Args[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, element := range composite.Elts {
			kv, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, keyOK := stringLiteral(kv.Key)
			value, valueOK := stringLiteral(kv.Value)
			if keyOK && valueOK {
				rules[key] = value
			}
		}
		return true
	})
	return rules
}

func liteRequestData(req *http.Request) map[string]string {
	data := map[string]string{}
	if req == nil || req.Body == nil {
		return data
	}

	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return data
	}
	req.Body = io.NopCloser(strings.NewReader(string(raw)))

	contentType := strings.ToLower(req.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		var decoded map[string]any
		if json.Unmarshal(raw, &decoded) == nil {
			for key, value := range decoded {
				switch v := value.(type) {
				case string:
					data[key] = v
				default:
					encoded, err := json.Marshal(v)
					if err == nil {
						data[key] = string(encoded)
					}
				}
			}
		}
		return data
	}

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if values, err := urlParseQuery(string(raw)); err == nil {
			for key, items := range values {
				if len(items) > 0 {
					data[key] = items[0]
				}
			}
		}
	}
	return data
}

func urlParseQuery(raw string) (map[string][]string, error) {
	values := make(map[string][]string)
	for _, part := range strings.Split(raw, "&") {
		if part == "" {
			continue
		}
		pieces := strings.SplitN(part, "=", 2)
		key := pieces[0]
		value := ""
		if len(pieces) == 2 {
			value = pieces[1]
		}
		values[key] = append(values[key], value)
	}
	return values, nil
}

func liteValidateRequest(req *http.Request, rules map[string]string) map[string][]string {
	if len(rules) == 0 {
		return nil
	}
	data := liteRequestData(req)
	v := validation.New(data)
	for field, ruleList := range rules {
		for _, rule := range strings.Split(ruleList, "|") {
			rule = strings.TrimSpace(rule)
			switch {
			case rule == "required":
				v.Required(field)
			case rule == "email":
				v.Email(field)
			case rule == "integer":
				v.Integer(field)
			case rule == "numeric":
				v.Numeric(field)
			case rule == "boolean":
				v.Boolean(field)
			case rule == "uuid":
				v.UUID(field)
			case strings.HasPrefix(rule, "min:"):
				if n, err := strconv.Atoi(strings.TrimPrefix(rule, "min:")); err == nil {
					v.Min(field, n)
				}
			case strings.HasPrefix(rule, "max:"):
				if n, err := strconv.Atoi(strings.TrimPrefix(rule, "max:")); err == nil {
					v.Max(field, n)
				}
			case strings.HasPrefix(rule, "oneof:"):
				v.OneOf(field, strings.Split(strings.TrimPrefix(rule, "oneof:"), ",")...)
			}
		}
	}
	if v.Valid() {
		return nil
	}
	return map[string][]string(v.Errors())
}

func liteStatusFromSelector(sel *ast.SelectorExpr) (int, bool) {
	call, ok := sel.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return 0, false
	}
	statusSel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || statusSel.Sel.Name != "Status" {
		return 0, false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	code, err := strconv.Atoi(lit.Value)
	if err != nil || code < 100 || code > 599 {
		return 0, false
	}
	return code, true
}

func matchLitePath(routePath, requestPath string) (map[string]string, bool) {
	routeParts := strings.Split(strings.Trim(routePath, "/"), "/")
	requestParts := strings.Split(strings.Trim(requestPath, "/"), "/")
	if routePath == "/" {
		routeParts = []string{}
	}
	if requestPath == "/" {
		requestParts = []string{}
	}
	if len(routeParts) != len(requestParts) {
		return nil, false
	}
	params := map[string]string{}
	for i, part := range routeParts {
		if strings.HasPrefix(part, ":") {
			name := strings.TrimPrefix(part, ":")
			if name == "" || requestParts[i] == "" {
				return nil, false
			}
			params[name] = requestParts[i]
			continue
		}
		if part != requestParts[i] {
			return nil, false
		}
	}
	return params, true
}

func renderLiteBody(body string, req *http.Request, params map[string]string) string {
	for key, value := range params {
		body = strings.ReplaceAll(body, "{{param:"+key+"}}", value)
	}
	for key, values := range req.URL.Query() {
		if len(values) > 0 {
			body = strings.ReplaceAll(body, "{{query:"+key+"}}", values[0])
		}
	}
	for key, value := range liteRequestData(req) {
		body = strings.ReplaceAll(body, "{{input:"+key+"}}", value)
	}
	if strings.Contains(body, "{{body}}") && req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err == nil {
			body = strings.ReplaceAll(body, "{{body}}", string(raw))
		}
	}
	return body
}

// readFirstLine is intentionally small and dependency-free; it is useful for
// future dev-runner probes without pulling a shell parser into the CLI.
func readFirstLine(data string) string {
	scanner := bufio.NewScanner(strings.NewReader(data))
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}
