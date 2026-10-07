package studio

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/core"
	"github.com/arfajhf/copytygo/v4/version"
)

const DefaultPath = "/__copytygo"

type Options struct {
	Path string
}

type routeView struct {
	Method string
	Path   string
	Name   string
}

type viewData struct {
	AppName     string
	Version     string
	Environment string
	RouteCount  int
	DocsURL     string
	BasePath    string
	Routes      []routeView
}

func Register(app *core.Application, options ...Options) bool {
	if app == nil || strings.EqualFold(config.Get("APP_ENV", "local"), "production") {
		return false
	}
	if !config.GetBool("COPYTYGO_STUDIO", true) {
		return false
	}

	path := DefaultPath
	if len(options) > 0 && strings.TrimSpace(options[0].Path) != "" {
		path = strings.TrimRight(strings.TrimSpace(options[0].Path), "/")
		if path == "" {
			path = DefaultPath
		}
	}

	data := func() viewData {
		routes := app.Routes()
		items := make([]routeView, 0, len(routes))
		for _, route := range routes {
			if route == nil {
				continue
			}
			items = append(items, routeView{
				Method: route.Method,
				Path:   route.Path,
				Name:   route.NameValue(),
			})
		}
		return viewData{
			AppName:     config.Get("APP_NAME", "CopyTyGo"),
			Version:     version.Framework,
			Environment: config.Get("APP_ENV", "local"),
			RouteCount:  len(items),
			DocsURL:     config.Get("COPYTYGO_DOCS_URL", version.DocsURL),
			BasePath:    path,
			Routes:      items,
		}
	}

	app.Get(path, func(ctx *core.Context) error {
		var out strings.Builder
		if err := dashboardTemplate.Execute(&out, data()); err != nil {
			return fmt.Errorf("copytygo studio: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio")

	app.Get(path+"/routes", func(ctx *core.Context) error {
		var out strings.Builder
		if err := routesTemplate.Execute(&out, data()); err != nil {
			return fmt.Errorf("copytygo studio routes: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.routes")

	app.Get(path+"/api/routes", func(ctx *core.Context) error {
		return ctx.JSON(core.Map{
			"data":  data().Routes,
			"count": len(data().Routes),
		})
	}).Name("copytygo.studio.api.routes")

	return true
}

const studioStyle = `
:root{font-family:Inter,ui-sans-serif,system-ui;background:#070d18;color:#e7edf7}*{box-sizing:border-box}body{margin:0}
.layout{min-height:100vh;display:grid;grid-template-columns:230px 1fr}.side{border-right:1px solid #1d2a3d;padding:24px 18px;background:#09111f}
.brand{font-weight:800;font-size:20px;padding:5px 8px 22px}.brand small{display:block;font-size:11px;color:#647896;font-weight:600;margin-top:4px}
nav a{display:block;color:#8ea2bf;text-decoration:none;padding:10px 11px;border-radius:9px;margin:2px 0;font-size:14px}nav a.active{background:#12213a;color:#eaf2ff}
.main{padding:34px}.top{display:flex;justify-content:space-between;align-items:center;gap:20px}.top h1{margin:0;font-size:28px}.pill{border:1px solid #29405e;border-radius:999px;padding:7px 10px;font-size:12px;color:#91a8c7}
.cards{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px;margin-top:28px}.card{border:1px solid #1e2d43;background:#0b1525;border-radius:15px;padding:20px}.label{text-transform:uppercase;font-size:11px;letter-spacing:.12em;color:#667b99}.value{font-size:24px;font-weight:750;margin-top:9px}
.notice{margin-top:18px;border:1px dashed #2b405e;border-radius:14px;padding:20px;color:#8297b5;line-height:1.6}.notice strong{color:#dbe7f9}a{color:#9bbcff}
table{width:100%;border-collapse:collapse;margin-top:24px;border:1px solid #1e2d43;background:#0b1525}th,td{padding:13px 15px;text-align:left;border-bottom:1px solid #1e2d43;font-size:14px}th{color:#758aa8;font-size:11px;text-transform:uppercase;letter-spacing:.1em}code{color:#cbd9ee}
@media(max-width:800px){.layout{grid-template-columns:1fr}.side{display:none}.main{padding:24px}.cards{grid-template-columns:1fr}}
`

var dashboardTemplate = template.Must(template.New("copytygo-studio").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>CopyTyGo Studio</title><style>` + studioStyle + `</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a class="active" href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a href="#">Database</a><a href="#">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="#">Logs</a><a href="#">Generator</a><a href="#">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>{{.AppName}}</h1><div style="color:#6f84a2;margin-top:6px">Local development workspace</div></div><span class="pill">{{.Environment}}</span></div>
<section class="cards"><div class="card"><div class="label">Framework</div><div class="value">{{.Version}}</div></div><div class="card"><div class="label">Routes</div><div class="value">{{.RouteCount}}</div></div><div class="card"><div class="label">Runtime</div><div class="value">Ready</div></div></section>
<div class="notice"><strong>Studio is connected to the running application.</strong><br>Route Explorer is live now. Other v4 ecosystem modules plug into this same workspace as they are completed. <a href="{{.DocsURL}}" target="_blank" rel="noreferrer">Open documentation</a>.</div>
</main></div></body></html>`))

var routesTemplate = template.Must(template.New("copytygo-studio-routes").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Routes · CopyTyGo Studio</title><style>` + studioStyle + `</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a class="active" href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a href="#">Database</a><a href="#">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="#">Logs</a><a href="#">Generator</a><a href="#">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Route Explorer</h1><div style="color:#6f84a2;margin-top:6px">{{.RouteCount}} registered routes</div></div><span class="pill">{{.Environment}}</span></div>
<table><thead><tr><th>Method</th><th>Path</th><th>Name</th></tr></thead><tbody>
{{range .Routes}}<tr><td><strong>{{.Method}}</strong></td><td><code>{{.Path}}</code></td><td>{{if .Name}}{{.Name}}{{else}}—{{end}}</td></tr>{{end}}
</tbody></table></main></div></body></html>`))
