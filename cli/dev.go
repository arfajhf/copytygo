package cli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/arfajhf/copytygo/config"
)

type liteRoute struct {
	Method      string
	Path        string
	ContentType string
	Status      int
	Body        string
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
	if len(routes) == 0 {
		return errors.New("copytygo lite runtime: no supported routes found; use native runtime for this project")
	}

	mux := http.NewServeMux()
	for _, route := range routes {
		r := route
		pattern := r.Method + " " + r.Path
		mux.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", r.ContentType)
			w.WriteHeader(r.Status)
			_, _ = w.Write([]byte(r.Body))
		})
	}

	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<!doctype html><html><head><meta charset="utf-8"><title>CopyTyGo</title></head><body style="font-family:system-ui;max-width:760px;margin:60px auto;padding:0 20px"><h1>CopyTyGo Lite Runtime</h1><p>Your development server is running without a generated application executable.</p><p>Try <a href="/api/health">/api/health</a>.</p></body></html>`)
	})

	host := config.Get("APP_HOST", "127.0.0.1")
	port := config.Get("APP_PORT", "8080")
	address := host + ":" + port

	fmt.Println("CopyTyGo Dev")
	fmt.Println("-------------")
	fmt.Println("Runtime : lite")
	fmt.Println("Backend : http://" + address)
	fmt.Printf("Routes  : %d\n", len(routes))
	fmt.Println()
	fmt.Println("Lite Runtime supports inline Text/JSON routes with automatic route hot reload.")
	fmt.Println("Complex controllers, middleware, database calls and arbitrary Go packages still use native mode.")
	fmt.Println()

	return http.ListenAndServe(address, requestLogMiddleware(mux))
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
		method := strings.ToUpper(selector.Sel.Name)
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE":
		default:
			return true
		}
		routePath, ok := stringLiteral(call.Args[0])
		if !ok {
			return true
		}
		handler, ok := call.Args[1].(*ast.FuncLit)
		if !ok {
			return true
		}
		route, ok := parseLiteHandler(method, routePath, handler)
		if ok {
			routes = append(routes, route)
		}
		return true
	})
	return routes, nil
}

func parseLiteHandler(method, path string, handler *ast.FuncLit) (liteRoute, bool) {
	route := liteRoute{Method: method, Path: path, Status: http.StatusOK}
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
		if !ok || len(call.Args) != 1 {
			continue
		}
		switch sel.Sel.Name {
		case "Text":
			value, ok := stringLiteral(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			route.ContentType = "text/plain; charset=utf-8"
			route.Body = value
			return route, true
		case "JSON":
			body, ok := mapLiteralJSON(call.Args[0])
			if !ok {
				return liteRoute{}, false
			}
			route.ContentType = "application/json; charset=utf-8"
			route.Body = body + "\n"
			return route, true
		}
	}
	return liteRoute{}, false
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
	return "", false
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
