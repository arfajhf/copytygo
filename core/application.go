package core

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/arfajhf/copytygo/v4/config"
	copycontainer "github.com/arfajhf/copytygo/v4/container"
	"github.com/arfajhf/copytygo/v4/version"
	"github.com/arfajhf/copytygo/v4/logging"
)

type BackgroundTask func(context.Context) error

type backgroundRegistration struct {
	name string
	run  BackgroundTask
}

type Application struct {
	router     *Router
	background []backgroundRegistration
	Services   *copycontainer.Container
}

func New() *Application {
	return &Application{
		router:     NewRouter(),
		background: make([]backgroundRegistration, 0),
		Services:   copycontainer.New(),
	}
}

func (app *Application) Use(middlewares ...Middleware) *Application {
	app.router.Use(middlewares...)
	return app
}

func (app *Application) Background(name string, task BackgroundTask) *Application {
	if task == nil {
		return app
	}
	if strings.TrimSpace(name) == "" {
		name = "background"
	}
	app.background = append(app.background, backgroundRegistration{name: name, run: task})
	return app
}

func (app *Application) Routes() []*Route { return app.router.Routes() }

func (app *Application) Resolve(name string) (any, error) {
	if app.Services == nil {
		return nil, fmt.Errorf("copytygo: application service container is unavailable")
	}
	return app.Services.Resolve(name)
}

func (app *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.router.ServeHTTP(w, r)
}

func (app *Application) Get(
	path string,
	handler Handler,
) *RouteBuilder {

	return app.router.Get(
		path,
		handler,
	)
}

func (app *Application) Post(
	path string,
	handler Handler,
) *RouteBuilder {

	return app.router.Post(
		path,
		handler,
	)
}

func (app *Application) Put(
	path string,
	handler Handler,
) *RouteBuilder {

	return app.router.Put(
		path,
		handler,
	)
}

func (app *Application) Patch(
	path string,
	handler Handler,
) *RouteBuilder {

	return app.router.Patch(
		path,
		handler,
	)
}

func (app *Application) Delete(
	path string,
	handler Handler,
) *RouteBuilder {

	return app.router.Delete(
		path,
		handler,
	)
}

type ResourceController interface {
	Index(*Context) error
	Show(*Context) error
	Store(*Context) error
	Update(*Context) error
	Destroy(*Context) error
}

func (app *Application) Resource(path string, controller ResourceController) *Application {
	base := strings.TrimRight(path, "/")
	if base == "" {
		base = "/"
	}
	member := base
	if member == "/" {
		member = ""
	}
	member += "/:id"

	app.Get(base, controller.Index)
	app.Get(member, controller.Show)
	app.Post(base, controller.Store)
	app.Put(member, controller.Update)
	app.Delete(member, controller.Destroy)
	return app
}

func (app *Application) Group(
	prefix string,
) *RouteGroup {

	return app.router.Group(prefix)
}

func (app *Application) APIVersion(version string) *RouteGroup {
	version = strings.Trim(strings.TrimSpace(version), "/")
	if version == "" {
		version = "v1"
	}
	return app.Group("/api/" + version).Name("api." + version + ".")
}

func (app *Application) Run() error {
	host := config.Get(
		"APP_HOST",
		"127.0.0.1",
	)

	port := config.Get(
		"APP_PORT",
		"8080",
	)

	name := config.Get(
		"APP_NAME",
		"CopyTyGo",
	)

	environment := config.Get(
		"APP_ENV",
		"local",
	)

	debug := config.GetBool(
		"APP_DEBUG",
		false,
	)

	address := host + ":" + port

	backgroundCtx, cancelBackground := context.WithCancel(context.Background())
	defer cancelBackground()
	for _, registration := range app.background {
		registration := registration
		go func() {
			if err := registration.run(backgroundCtx); err != nil && backgroundCtx.Err() == nil {
				logging.Default.Error("background task stopped", map[string]any{
					"name":  registration.name,
					"error": err.Error(),
				})
			}
		}()
	}

	fmt.Println()
	fmt.Println("CopyTyGo v" + version.Framework)
	fmt.Println("----------------------------")
	fmt.Println("Application :", name)
	fmt.Println("Environment :", environment)
	fmt.Println("Debug       :", debug)
	fmt.Println()
	fmt.Println("Application  : http://" + address)
	if !strings.EqualFold(environment, "production") && config.GetBool("COPYTYGO_STUDIO", true) {
		fmt.Println("Studio       : http://" + address + "/__copytygo")
	}
	fmt.Println("Documentation:", config.Get("COPYTYGO_DOCS_URL", version.DocsURL))
	fmt.Println()

	server := &http.Server{
		Addr:              address,
		Handler:           app.router,
		ReadHeaderTimeout: time.Duration(config.GetInt("SERVER_READ_HEADER_TIMEOUT", 5)) * time.Second,
		ReadTimeout:       time.Duration(config.GetInt("SERVER_READ_TIMEOUT", 15)) * time.Second,
		WriteTimeout:      time.Duration(config.GetInt("SERVER_WRITE_TIMEOUT", 30)) * time.Second,
		IdleTimeout:       time.Duration(config.GetInt("SERVER_IDLE_TIMEOUT", 60)) * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err

	case <-stop:
		fmt.Println()
		fmt.Println("Shutting down gracefully...")
		cancelBackground()

		timeout := time.Duration(config.GetInt("SERVER_SHUTDOWN_TIMEOUT", 10)) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return err
		}

		fmt.Println("Server stopped.")
		return nil
	}
}

func (app *Application) URL(name string, params ...map[string]string) (string, bool) {
	values := map[string]string{}
	if len(params) > 0 {
		values = params[0]
	}
	return app.router.URL(name, values)
}
