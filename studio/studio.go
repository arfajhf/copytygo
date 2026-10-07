package studio

import (
	"fmt"
	"net/url"
	"html/template"
	"strings"

	"github.com/arfajhf/copytygo/v4/cli"
	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/core"
	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/database/migration"
	"github.com/arfajhf/copytygo/v4/logging"
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

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
}

func Register(app *core.Application, options ...Options) bool {
	if app == nil || strings.EqualFold(config.Get("APP_ENV", "local"), "production") {
		return false
	}
	if !config.GetBool("COPYTYGO_STUDIO", true) {
		return false
	}

	inspector := core.NewRequestInspector(200)
	app.Use(inspector.Middleware())

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

	app.Get(path+"/logs", func(ctx *core.Context) error {
		payload := struct {
			viewData
			Entries []logging.Entry
		}{
			viewData: data(),
			Entries:  logging.Default.Entries(),
		}
		var out strings.Builder
		if err := logsTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio logs: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.logs")

	app.Get(path+"/api/logs", func(ctx *core.Context) error {
		return ctx.JSON(core.Map{"data": logging.Default.Entries()})
	}).Name("copytygo.studio.api.logs")

	app.Post(path+"/logs/clear", func(ctx *core.Context) error {
		logging.Default.Clear()
		return ctx.Redirect(path + "/logs")
	}).Name("copytygo.studio.logs.clear")

	app.Get(path+"/migrations", func(ctx *core.Context) error {
		statuses, err := cli.ProjectMigrationStatus()
		payload := struct {
			viewData
			Statuses []migration.StatusRecord
			Message  string
			Error    string
		}{
			viewData: data(),
			Statuses: statuses,
			Message:  ctx.Query("message"),
		}
		if err != nil {
			payload.Error = err.Error()
		}
		if queryErr := ctx.Query("error"); queryErr != "" {
			payload.Error = queryErr
		}

		var out strings.Builder
		if err := migrationsTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio migrations: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.migrations")

	app.Post(path+"/migrations/run", func(ctx *core.Context) error {
		if err := cli.ProjectMigrate(); err != nil {
			return ctx.Redirect(path + "/migrations?error=" + urlQueryEscape(err.Error()))
		}
		return ctx.Redirect(path + "/migrations?message=" + urlQueryEscape("Migrations completed"))
	}).Name("copytygo.studio.migrations.run")

	app.Post(path+"/migrations/rollback", func(ctx *core.Context) error {
		if err := cli.ProjectRollback(); err != nil {
			return ctx.Redirect(path + "/migrations?error=" + urlQueryEscape(err.Error()))
		}
		return ctx.Redirect(path + "/migrations?message=" + urlQueryEscape("Last migration batch rolled back"))
	}).Name("copytygo.studio.migrations.rollback")

	app.Get(path+"/api/migrations", func(ctx *core.Context) error {
		statuses, err := cli.ProjectMigrationStatus()
		if err != nil {
			return err
		}
		return ctx.JSON(core.Map{"data": statuses})
	}).Name("copytygo.studio.api.migrations")

	app.Get(path+"/doctor", func(ctx *core.Context) error {
		results := cli.RunDoctorChecks([]string{"--db"})
		passed := 0
		for _, result := range results {
			if result.OK {
				passed++
			}
		}
		payload := struct {
			viewData
			Results []cli.DoctorResult
			Passed  int
			Failed  int
		}{
			viewData: data(),
			Results:  results,
			Passed:   passed,
			Failed:   len(results) - passed,
		}
		var out strings.Builder
		if err := doctorTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio doctor: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.doctor")

	app.Get(path+"/api/doctor", func(ctx *core.Context) error {
		results := cli.RunDoctorChecks([]string{"--db"})
		return ctx.JSON(core.Map{"data": results})
	}).Name("copytygo.studio.api.doctor")

	app.Get(path+"/database", func(ctx *core.Context) error {
		drivers.Register()
		cfg := database.LoadConfig()
		connected := true
		message := "Connected"
		if _, err := database.Connect(); err != nil {
			connected = false
			message = err.Error()
		}

		payload := struct {
			viewData
			Driver    string
			Host      string
			Port      string
			Database  string
			Connected bool
			Message   string
		}{
			viewData:  data(),
			Driver:    cfg.Driver,
			Host:      cfg.Host,
			Port:      cfg.Port,
			Database:  cfg.Database,
			Connected: connected,
			Message:   message,
		}
		var out strings.Builder
		if err := databaseTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio database: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.database")

	app.Get(path+"/api/database", func(ctx *core.Context) error {
		drivers.Register()
		cfg := database.LoadConfig()
		_, err := database.Connect()
		return ctx.JSON(core.Map{
			"driver":    cfg.Driver,
			"host":      cfg.Host,
			"port":      cfg.Port,
			"database":  cfg.Database,
			"connected": err == nil,
		})
	}).Name("copytygo.studio.api.database")

	app.Get(path+"/requests", func(ctx *core.Context) error {
		payload := struct {
			viewData
			Requests []core.RequestRecord
		}{
			viewData:  data(),
			Requests: inspector.Records(),
		}
		var out strings.Builder
		if err := requestsTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio requests: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.requests")

	app.Get(path+"/api/requests", func(ctx *core.Context) error {
		return ctx.JSON(core.Map{
			"data": inspector.Records(),
		})
	}).Name("copytygo.studio.api.requests")

	app.Post(path+"/requests/clear", func(ctx *core.Context) error {
		inspector.Clear()
		return ctx.Redirect(path + "/requests")
	}).Name("copytygo.studio.requests.clear")

	app.Get(path+"/generator", func(ctx *core.Context) error {
		page := data()
		payload := struct {
			viewData
			Created string
			Error   string
		}{
			viewData: page,
			Created:  ctx.Query("created"),
			Error:    ctx.Query("error"),
		}
		var out strings.Builder
		if err := generatorTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio generator: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.generator")

	app.Post(path+"/generator/resource", func(ctx *core.Context) error {
		name := strings.TrimSpace(ctx.Input("name"))
		fieldsRaw := strings.TrimSpace(ctx.Input("fields"))
		fields := strings.Fields(fieldsRaw)

		if err := cli.MakeResourceWithFields(name, fields); err != nil {
			return ctx.Redirect(path + "/generator?error=" + urlQueryEscape(err.Error()))
		}

		return ctx.Redirect(path + "/generator?created=" + urlQueryEscape(name))
	}).Name("copytygo.studio.generator.resource")

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
<nav><a class="active" href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>{{.AppName}}</h1><div style="color:#6f84a2;margin-top:6px">Local development workspace</div></div><span class="pill">{{.Environment}}</span></div>
<section class="cards"><div class="card"><div class="label">Framework</div><div class="value">{{.Version}}</div></div><div class="card"><div class="label">Routes</div><div class="value">{{.RouteCount}}</div></div><div class="card"><div class="label">Runtime</div><div class="value">Ready</div></div></section>
<div class="notice"><strong>Studio is connected to the running application.</strong><br>Route Explorer is live now. Other v4 ecosystem modules plug into this same workspace as they are completed. <a href="{{.DocsURL}}" target="_blank" rel="noreferrer">Open documentation</a>.</div>
</main></div></body></html>`))

var routesTemplate = template.Must(template.New("copytygo-studio-routes").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Routes · CopyTyGo Studio</title><style>` + studioStyle + `</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a class="active" href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Route Explorer</h1><div style="color:#6f84a2;margin-top:6px">{{.RouteCount}} registered routes</div></div><span class="pill">{{.Environment}}</span></div>
<table><thead><tr><th>Method</th><th>Path</th><th>Name</th></tr></thead><tbody>
{{range .Routes}}<tr><td><strong>{{.Method}}</strong></td><td><code>{{.Path}}</code></td><td>{{if .Name}}{{.Name}}{{else}}—{{end}}</td></tr>{{end}}
</tbody></table></main></div></body></html>`))


var generatorTemplate = template.Must(template.New("copytygo-studio-generator").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Generator · CopyTyGo Studio</title><style>` + studioStyle + `
.form{margin-top:24px;border:1px solid #1e2d43;background:#0b1525;border-radius:15px;padding:22px;max-width:760px}
.field{margin-bottom:18px}.field label{display:block;font-size:12px;text-transform:uppercase;letter-spacing:.1em;color:#758aa8;margin-bottom:8px}
input,textarea{width:100%;background:#08111f;border:1px solid #263852;color:#e7edf7;border-radius:9px;padding:12px;font:inherit}
textarea{min-height:150px;resize:vertical}button{border:0;border-radius:9px;padding:12px 16px;background:#f4f7fb;color:#0c1525;font-weight:700;cursor:pointer}
.flash{margin-top:18px;padding:13px 15px;border-radius:10px;background:#10233a;border:1px solid #29405e}.error{background:#2a1518;border-color:#6b2a31;color:#ffc5cb}
.hint{color:#6f84a2;font-size:13px;line-height:1.6;margin-top:7px}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/logs">Logs</a><a class="active" href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Visual Generator</h1><div style="color:#6f84a2;margin-top:6px">Generate framework code without leaving Studio</div></div><span class="pill">{{.Environment}}</span></div>
{{if .Created}}<div class="flash">Resource <strong>{{.Created}}</strong> created successfully. Model, controller, migration and routes were generated.</div>{{end}}
{{if .Error}}<div class="flash error">{{.Error}}</div>{{end}}
<form class="form" method="post" action="{{.BasePath}}/generator/resource">
<div class="field"><label>Resource name</label><input name="name" placeholder="Product" required></div>
<div class="field"><label>Fields</label><textarea name="fields" placeholder="name:string&#10;price:decimal&#10;stock:integer&#10;active:boolean&#10;description:text?"></textarea>
<div class="hint">One field per line or separated by spaces. Supported types follow the same rules as <code>ctg make:resource</code>.</div></div>
<button type="submit">Create Resource</button>
</form></main></div></body></html>`))


var requestsTemplate = template.Must(template.New("copytygo-studio-requests").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Requests · CopyTyGo Studio</title><style>` + studioStyle + `
.toolbar{display:flex;justify-content:space-between;align-items:center;margin-top:24px;gap:12px}.muted{color:#6f84a2;font-size:13px}
button{border:1px solid #344863;background:#101c2e;color:#dce8fb;border-radius:8px;padding:9px 12px;cursor:pointer}
.status-ok{color:#74d99f}.status-warn{color:#ffcf70}.status-error{color:#ff8e98}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a class="active" href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Request Inspector</h1><div style="color:#6f84a2;margin-top:6px">Recent requests captured from the running local application</div></div><span class="pill">{{.Environment}}</span></div>
<div class="toolbar"><div class="muted">{{len .Requests}} requests retained</div><form method="post" action="{{.BasePath}}/requests/clear"><button type="submit">Clear</button></form></div>
<table><thead><tr><th>Method</th><th>Path</th><th>Status</th><th>Latency</th><th>Request ID</th></tr></thead><tbody>
{{range .Requests}}<tr><td><strong>{{.Method}}</strong></td><td><code>{{.Path}}</code></td><td>{{.Status}}</td><td>{{.DurationMS}} ms</td><td><code>{{.RequestID}}</code></td></tr>{{else}}<tr><td colspan="5" class="muted">No requests captured yet.</td></tr>{{end}}
</tbody></table></main></div></body></html>`))


var databaseTemplate = template.Must(template.New("copytygo-studio-database").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Database · CopyTyGo Studio</title><style>` + studioStyle + `
.dbgrid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px;margin-top:24px}.dbitem{border:1px solid #1e2d43;background:#0b1525;border-radius:13px;padding:18px}.dbitem .value{font-size:18px}.ok{color:#74d99f}.bad{color:#ff8e98}.message{margin-top:18px;padding:14px 16px;border:1px solid #263a55;border-radius:11px;background:#0b1525;color:#9eb0c9;word-break:break-word}
@media(max-width:700px){.dbgrid{grid-template-columns:1fr}}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a class="active" href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Database</h1><div style="color:#6f84a2;margin-top:6px">Connection overview without exposing credentials</div></div>{{if .Connected}}<span class="pill ok">Connected</span>{{else}}<span class="pill bad">Disconnected</span>{{end}}</div>
<section class="dbgrid"><div class="dbitem"><div class="label">Driver</div><div class="value">{{.Driver}}</div></div><div class="dbitem"><div class="label">Database</div><div class="value">{{.Database}}</div></div><div class="dbitem"><div class="label">Host</div><div class="value">{{.Host}}</div></div><div class="dbitem"><div class="label">Port</div><div class="value">{{.Port}}</div></div></section>
<div class="message">{{.Message}}</div></main></div></body></html>`))


var doctorTemplate = template.Must(template.New("copytygo-studio-doctor").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Doctor · CopyTyGo Studio</title><style>` + studioStyle + `
.summary{display:flex;gap:12px;margin-top:24px}.summary .card{min-width:140px}.check{display:grid;grid-template-columns:26px 180px 1fr;gap:10px;align-items:start;padding:13px 15px;border-bottom:1px solid #1e2d43}.checks{margin-top:20px;border:1px solid #1e2d43;border-radius:14px;background:#0b1525;overflow:hidden}.good{color:#74d99f}.bad{color:#ff8e98}.reason{color:#8094b1;font-size:13px;word-break:break-word}
@media(max-width:700px){.check{grid-template-columns:26px 1fr}.reason{grid-column:2}}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a class="active" href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Doctor</h1><div style="color:#6f84a2;margin-top:6px">Framework, project, security and database diagnostics</div></div><span class="pill">{{.Environment}}</span></div>
<section class="summary"><div class="card"><div class="label">Passed</div><div class="value good">{{.Passed}}</div></div><div class="card"><div class="label">Failed</div><div class="value {{if .Failed}}bad{{else}}good{{end}}">{{.Failed}}</div></div></section>
<div class="checks">{{range .Results}}<div class="check"><div class="{{if .OK}}good{{else}}bad{{end}}">{{if .OK}}✓{{else}}×{{end}}</div><strong>{{.Name}}</strong><div class="reason">{{if .Error}}{{.Error}}{{else}}Healthy{{end}}</div></div>{{end}}</div>
</main></div></body></html>`))


var migrationsTemplate = template.Must(template.New("copytygo-studio-migrations").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Migrations · CopyTyGo Studio</title><style>` + studioStyle + `
.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:24px}.actions form{margin:0}.actions button{border:1px solid #344863;background:#101c2e;color:#dce8fb;border-radius:8px;padding:10px 13px;cursor:pointer}.actions .primary{background:#f4f7fb;color:#0c1525;border-color:#f4f7fb}.flash{margin-top:18px;padding:13px 15px;border-radius:10px;background:#10233a;border:1px solid #29405e}.flash.error{background:#2a1518;border-color:#6b2a31;color:#ffc5cb}.ran{color:#74d99f}.pending{color:#ffcf70}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a href="{{.BasePath}}/database">Database</a><a class="active" href="{{.BasePath}}/migrations">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Migrations</h1><div style="color:#6f84a2;margin-top:6px">Inspect and run project database migrations</div></div><span class="pill">{{.Environment}}</span></div>
<div class="actions"><form method="post" action="{{.BasePath}}/migrations/run"><button class="primary" type="submit">Run Migrations</button></form><form method="post" action="{{.BasePath}}/migrations/rollback"><button type="submit">Rollback Last Batch</button></form></div>
{{if .Message}}<div class="flash">{{.Message}}</div>{{end}}{{if .Error}}<div class="flash error">{{.Error}}</div>{{end}}
<table><thead><tr><th>Status</th><th>Batch</th><th>Migration</th></tr></thead><tbody>
{{range .Statuses}}<tr><td class="{{if .Ran}}ran{{else}}pending{{end}}">{{if .Ran}}Ran{{else}}Pending{{end}}</td><td>{{if .Ran}}{{.Batch}}{{else}}—{{end}}</td><td><code>{{.Migration}}</code></td></tr>{{else}}<tr><td colspan="3" style="color:#6f84a2">No migrations registered.</td></tr>{{end}}
</tbody></table></main></div></body></html>`))


var logsTemplate = template.Must(template.New("copytygo-studio-logs").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Logs · CopyTyGo Studio</title><style>` + studioStyle + `
.toolbar{display:flex;justify-content:space-between;align-items:center;margin-top:24px}.toolbar button{border:1px solid #344863;background:#101c2e;color:#dce8fb;border-radius:8px;padding:9px 12px;cursor:pointer}
.level{font-weight:800;text-transform:uppercase;font-size:11px}.debug{color:#9fb0ca}.info{color:#74d99f}.warn{color:#ffcf70}.error{color:#ff8e98}.fields{color:#7f94b2;font-size:12px;word-break:break-word}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="#">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a class="active" href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Logs</h1><div style="color:#6f84a2;margin-top:6px">Structured framework and request logs</div></div><span class="pill">{{.Environment}}</span></div>
<div class="toolbar"><div style="color:#6f84a2">{{len .Entries}} retained entries</div><form method="post" action="{{.BasePath}}/logs/clear"><button type="submit">Clear</button></form></div>
<table><thead><tr><th>Level</th><th>Message</th><th>Fields</th><th>Time</th></tr></thead><tbody>
{{range .Entries}}<tr><td><span class="level {{.Level}}">{{.Level}}</span></td><td>{{.Message}}</td><td class="fields">{{printf "%v" .Fields}}</td><td class="fields">{{.At}}</td></tr>{{else}}<tr><td colspan="4" class="fields">No logs captured yet.</td></tr>{{end}}
</tbody></table></main></div></body></html>`))
