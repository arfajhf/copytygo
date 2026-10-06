package core

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/arfajhf/copytygo/config"
	"github.com/arfajhf/copytygo/version"
)

type Application struct {
	router *Router
}

func New() *Application {
	return &Application{
		router: NewRouter(),
	}
}

func (app *Application) Use(middlewares ...Middleware) *Application {
	app.router.Use(middlewares...)
	return app
}

func (app *Application) Routes() []*Route { return app.router.Routes() }

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

	fmt.Println()
	fmt.Println("CopyTyGo v" + version.Framework)
	fmt.Println("----------------------------")
	fmt.Println("Application :", name)
	fmt.Println("Environment :", environment)
	fmt.Println("Debug       :", debug)
	fmt.Println()
	fmt.Println(
		"Server running at http://" + address,
	)
	fmt.Println()

	return http.ListenAndServe(
		address,
		app.router,
	)
}

func (app *Application) URL(name string, params ...map[string]string) (string, bool) {
	values := map[string]string{}
	if len(params) > 0 {
		values = params[0]
	}
	return app.router.URL(name, values)
}
