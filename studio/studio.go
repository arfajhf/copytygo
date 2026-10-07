package studio

import (
	"fmt"
	"net/url"
	"sort"
	"html/template"
	"strings"

	"github.com/arfajhf/copytygo/v4/cache"
	"github.com/arfajhf/copytygo/v4/events"
	"github.com/arfajhf/copytygo/v4/foundation"
	"github.com/arfajhf/copytygo/v4/health"
	"github.com/arfajhf/copytygo/v4/storage"
	"github.com/arfajhf/copytygo/v4/cli"
	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/core"
	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/database/migration"
	"github.com/arfajhf/copytygo/v4/logging"
	"github.com/arfajhf/copytygo/v4/queue"
	"github.com/arfajhf/copytygo/v4/scheduler"
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
	errorInspector := core.NewErrorInspector(100)
	app.Use(errorInspector.Middleware())

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

	app.Get(path+"/resources", func(ctx *core.Context) error {
		resources, err := cli.DiscoverGeneratedResources(".")
		payload := struct {
			viewData
			Resources []cli.GeneratedResource
			Error     string
		}{
			viewData:  data(),
			Resources: resources,
		}
		if err != nil {
			payload.Error = err.Error()
		}
		var out strings.Builder
		if err := resourcesTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio resources: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.resources")

	app.Get(path+"/resources/:name", func(ctx *core.Context) error {
		name := ctx.Param("name")
		resources, err := cli.DiscoverGeneratedResources(".")
		if err != nil {
			return err
		}

		allowed := false
		for _, resource := range resources {
			if resource.Resource == name {
				allowed = true
				break
			}
		}
		if !allowed {
			return ctx.NotFound("Studio resource not found")
		}

		drivers.Register()
		db, connectErr := database.Connect()
		var page database.ResourcePage
		previewErr := connectErr
		if previewErr == nil {
			resource, resourceErr := database.NewResource(db, config.Get("DB_DRIVER", "mysql"), name)
			if resourceErr != nil {
				previewErr = resourceErr
			} else {
				resource.WithSoftDeletes()
				page, previewErr = resource.List(database.ResourceListOptions{Page: 1, PerPage: 50})
			}
		}

		columns := []string{}
		if len(page.Data) > 0 {
			for column := range page.Data[0] {
				columns = append(columns, column)
			}
			sort.Strings(columns)
		}

		payload := struct {
			viewData
			Name    string
			Page    database.ResourcePage
			Columns []string
			Error   string
		}{
			viewData: data(),
			Name:     name,
			Page:     page,
			Columns:  columns,
		}
		if previewErr != nil {
			payload.Error = previewErr.Error()
		}

		var out strings.Builder
		if err := resourcePreviewTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio resource preview: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.resource.preview")

	app.Get(path+"/api/resources", func(ctx *core.Context) error {
		resources, err := cli.DiscoverGeneratedResources(".")
		if err != nil {
			return err
		}
		return ctx.JSON(core.Map{"data": resources})
	}).Name("copytygo.studio.api.resources")

	app.Get(path+"/services", func(ctx *core.Context) error {
		cacheSize := 0
		cacheDriver := "unavailable"
		if service, err := app.Resolve(foundation.CacheService); err == nil {
			cacheDriver = fmt.Sprintf("%T", service)
			if memory, ok := service.(*cache.Memory); ok {
				cacheSize = memory.Len()
			}
		}

		registrations := []events.Registration{}
		if service, err := app.Resolve(foundation.EventBusService); err == nil {
			if bus, ok := service.(*events.Bus); ok {
				registrations = bus.Registrations()
			}
		}

		storageDriver := "unavailable"
		storageRoot := config.Get("STORAGE_PATH", "storage/app")
		if service, err := app.Resolve(foundation.StorageService); err == nil {
			storageDriver = fmt.Sprintf("%T", service)
			if local, ok := service.(*storage.Local); ok {
				storageRoot = local.Root()
			}
		}

		mailDriver := "unavailable"
		if service, err := app.Resolve(foundation.MailService); err == nil {
			mailDriver = fmt.Sprintf("%T", service)
		}

		payload := struct {
			viewData
			CacheDriver   string
			CacheSize     int
			Events        []events.Registration
			StorageDriver string
			StorageRoot   string
			MailDriver    string
			MailHost      string
			MailPort      int
			MailFrom      string
		}{
			viewData:      data(),
			CacheDriver:   cacheDriver,
			CacheSize:     cacheSize,
			Events:        registrations,
			StorageDriver: storageDriver,
			StorageRoot:   storageRoot,
			MailDriver:    mailDriver,
			MailHost:      config.Get("MAIL_HOST", "127.0.0.1"),
			MailPort:      config.GetInt("MAIL_PORT", 1025),
			MailFrom:      config.Get("MAIL_FROM", "noreply@copytygo.local"),
		}

		var out strings.Builder
		if err := servicesTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio services: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.services")

	app.Post(path+"/services/cache/clear", func(ctx *core.Context) error {
		service, err := app.Resolve(foundation.CacheService)
		if err != nil {
			return err
		}
		store, ok := service.(cache.Store)
		if !ok {
			return fmt.Errorf("copytygo studio: cache service does not implement cache.Store")
		}
		store.Clear()
		return ctx.Redirect(path + "/services")
	}).Name("copytygo.studio.services.cache.clear")

	app.Get(path+"/api/services", func(ctx *core.Context) error {
		cacheSize := 0
		if service, err := app.Resolve(foundation.CacheService); err == nil {
			if memory, ok := service.(*cache.Memory); ok {
				cacheSize = memory.Len()
			}
		}
		registrations := []events.Registration{}
		if service, err := app.Resolve(foundation.EventBusService); err == nil {
			if bus, ok := service.(*events.Bus); ok {
				registrations = bus.Registrations()
			}
		}
		return ctx.JSON(core.Map{
			"cache_entries": cacheSize,
			"events":        registrations,
			"storage_path":  config.Get("STORAGE_PATH", "storage/app"),
			"mail_host":     config.Get("MAIL_HOST", "127.0.0.1"),
			"mail_port":     config.GetInt("MAIL_PORT", 1025),
			"mail_from":     config.Get("MAIL_FROM", "noreply@copytygo.local"),
		})
	}).Name("copytygo.studio.api.services")

	app.Get(path+"/queue", func(ctx *core.Context) error {
		manager := queue.Current()
		memoryStats := queue.Stats{}
		failures := []queue.Failure{}
		durableStats := queue.DurableStats{}
		var pendingDB int64
		var failedDB int64
		var queueError string

		if manager.Memory != nil {
			memoryStats = manager.Memory.Stats()
			failures = manager.Memory.Failures()
		}
		if manager.DatabaseWorker != nil {
			durableStats = manager.DatabaseWorker.Stats()
		}
		if manager.Database != nil {
			if count, err := manager.Database.Pending(ctx.Request.Context()); err == nil {
				pendingDB = count
			} else {
				queueError = err.Error()
			}
			if count, err := manager.Database.Failed(ctx.Request.Context()); err == nil {
				failedDB = count
			} else if queueError == "" {
				queueError = err.Error()
			}
		}

		payload := struct {
			viewData
			Driver       string
			MemoryStats  queue.Stats
			DurableStats queue.DurableStats
			PendingDB    int64
			FailedDB     int64
			Failures     []queue.Failure
			Error        string
		}{
			viewData:     data(),
			Driver:       manager.Driver,
			MemoryStats:  memoryStats,
			DurableStats: durableStats,
			PendingDB:    pendingDB,
			FailedDB:     failedDB,
			Failures:     failures,
			Error:        queueError,
		}
		var out strings.Builder
		if err := queueTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio queue: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.queue")

	app.Get(path+"/api/queue", func(ctx *core.Context) error {
		manager := queue.Current()
		result := core.Map{
			"driver": manager.Driver,
		}
		if manager.Driver == "database" && manager.Database != nil {
			pending, _ := manager.Database.Pending(ctx.Request.Context())
			failed, _ := manager.Database.Failed(ctx.Request.Context())
			result["pending"] = pending
			result["failed"] = failed
			if manager.DatabaseWorker != nil {
				result["worker"] = manager.DatabaseWorker.Stats()
			}
		} else if manager.Memory != nil {
			result["stats"] = manager.Memory.Stats()
			result["failures"] = manager.Memory.Failures()
		}
		return ctx.JSON(result)
	}).Name("copytygo.studio.api.queue")

	app.Get(path+"/scheduler", func(ctx *core.Context) error {
		payload := struct {
			viewData
			Started bool
			Entries []scheduler.Entry
		}{
			viewData: data(),
			Started:  scheduler.Default.Started(),
			Entries:  scheduler.Default.Entries(),
		}
		var out strings.Builder
		if err := schedulerTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio scheduler: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.scheduler")

	app.Get(path+"/api/scheduler", func(ctx *core.Context) error {
		return ctx.JSON(core.Map{
			"started": scheduler.Default.Started(),
			"data":    scheduler.Default.Entries(),
		})
	}).Name("copytygo.studio.api.scheduler")

	app.Get(path+"/errors", func(ctx *core.Context) error {
		payload := struct {
			viewData
			Errors []core.ErrorRecord
		}{
			viewData: data(),
			Errors:    errorInspector.Records(),
		}
		var out strings.Builder
		if err := errorsTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio errors: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.errors")

	app.Get(path+"/api/errors", func(ctx *core.Context) error {
		return ctx.JSON(core.Map{"data": errorInspector.Records()})
	}).Name("copytygo.studio.api.errors")

	app.Post(path+"/errors/clear", func(ctx *core.Context) error {
		errorInspector.Clear()
		return ctx.Redirect(path + "/errors")
	}).Name("copytygo.studio.errors.clear")

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

	app.Get(path+"/health", func(ctx *core.Context) error {
		status, results, err := foundation.RunHealth(ctx.Request.Context(), app.Services)
		payload := struct {
			viewData
			Status  health.Status
			Results []health.Result
			Error   string
		}{
			viewData: data(),
			Status:   status,
			Results:  results,
		}
		if err != nil {
			payload.Status = health.Unhealthy
			payload.Error = err.Error()
		}
		var out strings.Builder
		if err := healthTemplate.Execute(&out, payload); err != nil {
			return fmt.Errorf("copytygo studio health: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio.health")

	app.Get(path+"/api/health", func(ctx *core.Context) error {
		status, results, err := foundation.RunHealth(ctx.Request.Context(), app.Services)
		if err != nil {
			return err
		}
		return ctx.JSON(core.Map{"status": status, "checks": results})
	}).Name("copytygo.studio.api.health")

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

	app.Post(path+"/generator/scaffold", func(ctx *core.Context) error {
		kind := strings.ToLower(strings.TrimSpace(ctx.Input("type")))
		name := strings.TrimSpace(ctx.Input("name"))

		var err error
		switch kind {
		case "model":
			err = cli.MakeModel(name, "app/models")
		case "middleware":
			err = cli.MakeMiddleware(name, "app/middleware")
		case "service":
			err = cli.MakeService(name, "app/services")
		case "job":
			err = cli.MakeJob(name, "app/jobs")
		case "listener":
			err = cli.MakeListener(name, "app/listeners")
		case "seeder":
			err = cli.MakeSeeder(name, "database/seeders")
		case "factory":
			err = cli.MakeFactory(name, "database/factories")
		case "mail":
			err = cli.MakeMail(name, "app/mails")
		default:
			err = fmt.Errorf("copytygo studio: unsupported generator type %q", kind)
		}

		if err != nil {
			return ctx.Redirect(path + "/generator?error=" + urlQueryEscape(err.Error()))
		}
		return ctx.Redirect(path + "/generator?created=" + urlQueryEscape(kind+" "+name))
	}).Name("copytygo.studio.generator.scaffold")

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
<nav><a class="active" href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>{{.AppName}}</h1><div style="color:#6f84a2;margin-top:6px">Local development workspace</div></div><span class="pill">{{.Environment}}</span></div>
<section class="cards"><div class="card"><div class="label">Framework</div><div class="value">{{.Version}}</div></div><div class="card"><div class="label">Routes</div><div class="value">{{.RouteCount}}</div></div><div class="card"><div class="label">Runtime</div><div class="value">Ready</div></div></section>
<div class="notice"><strong>Studio is connected to the running application.</strong><br>Route Explorer is live now. Other v4 ecosystem modules plug into this same workspace as they are completed. <a href="{{.DocsURL}}" target="_blank" rel="noreferrer">Open documentation</a>.</div>
</main></div></body></html>`))

var routesTemplate = template.Must(template.New("copytygo-studio-routes").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Routes · CopyTyGo Studio</title><style>` + studioStyle + `</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a class="active" href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
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
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a class="active" href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Visual Generator</h1><div style="color:#6f84a2;margin-top:6px">Generate framework code without leaving Studio</div></div><span class="pill">{{.Environment}}</span></div>
{{if .Created}}<div class="flash">Resource <strong>{{.Created}}</strong> created successfully. Model, controller, migration and routes were generated.</div>{{end}}
{{if .Error}}<div class="flash error">{{.Error}}</div>{{end}}
<form class="form" method="post" action="{{.BasePath}}/generator/resource">
<div class="field"><label>Resource name</label><input name="name" placeholder="Product" required></div>
<div class="field"><label>Fields</label><textarea name="fields" placeholder="name:string&#10;price:decimal&#10;stock:integer&#10;active:boolean&#10;description:text?"></textarea>
<div class="hint">One field per line or separated by spaces. Supported types follow the same rules as <code>ctg make:resource</code>.</div></div>
<button type="submit">Create Resource</button>
</form>
<form class="form" method="post" action="{{.BasePath}}/generator/scaffold">
<div class="field"><label>Scaffold type</label><select name="type" required style="width:100%;background:#08111f;border:1px solid #263852;color:#e7edf7;border-radius:9px;padding:12px;font:inherit">
<option value="model">Model</option><option value="middleware">Middleware</option><option value="service">Service</option><option value="job">Queue Job</option><option value="listener">Event Listener</option><option value="seeder">Seeder</option><option value="factory">Factory</option><option value="mail">Mail</option>
</select></div>
<div class="field"><label>Name</label><input name="name" placeholder="SendWelcomeEmail" required></div>
<button type="submit">Create Scaffold</button>
</form></main></div></body></html>`))


var requestsTemplate = template.Must(template.New("copytygo-studio-requests").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Requests · CopyTyGo Studio</title><style>` + studioStyle + `
.toolbar{display:flex;justify-content:space-between;align-items:center;margin-top:24px;gap:12px}.muted{color:#6f84a2;font-size:13px}
button{border:1px solid #344863;background:#101c2e;color:#dce8fb;border-radius:8px;padding:9px 12px;cursor:pointer}
.status-ok{color:#74d99f}.status-warn{color:#ffcf70}.status-error{color:#ff8e98}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a class="active" href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
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
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a class="active" href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Database</h1><div style="color:#6f84a2;margin-top:6px">Connection overview without exposing credentials</div></div>{{if .Connected}}<span class="pill ok">Connected</span>{{else}}<span class="pill bad">Disconnected</span>{{end}}</div>
<section class="dbgrid"><div class="dbitem"><div class="label">Driver</div><div class="value">{{.Driver}}</div></div><div class="dbitem"><div class="label">Database</div><div class="value">{{.Database}}</div></div><div class="dbitem"><div class="label">Host</div><div class="value">{{.Host}}</div></div><div class="dbitem"><div class="label">Port</div><div class="value">{{.Port}}</div></div></section>
<div class="message">{{.Message}}</div></main></div></body></html>`))


var doctorTemplate = template.Must(template.New("copytygo-studio-doctor").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Doctor · CopyTyGo Studio</title><style>` + studioStyle + `
.summary{display:flex;gap:12px;margin-top:24px}.summary .card{min-width:140px}.check{display:grid;grid-template-columns:26px 180px 1fr;gap:10px;align-items:start;padding:13px 15px;border-bottom:1px solid #1e2d43}.checks{margin-top:20px;border:1px solid #1e2d43;border-radius:14px;background:#0b1525;overflow:hidden}.good{color:#74d99f}.bad{color:#ff8e98}.reason{color:#8094b1;font-size:13px;word-break:break-word}
@media(max-width:700px){.check{grid-template-columns:26px 1fr}.reason{grid-column:2}}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a class="active" href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Doctor</h1><div style="color:#6f84a2;margin-top:6px">Framework, project, security and database diagnostics</div></div><span class="pill">{{.Environment}}</span></div>
<section class="summary"><div class="card"><div class="label">Passed</div><div class="value good">{{.Passed}}</div></div><div class="card"><div class="label">Failed</div><div class="value {{if .Failed}}bad{{else}}good{{end}}">{{.Failed}}</div></div></section>
<div class="checks">{{range .Results}}<div class="check"><div class="{{if .OK}}good{{else}}bad{{end}}">{{if .OK}}✓{{else}}×{{end}}</div><strong>{{.Name}}</strong><div class="reason">{{if .Error}}{{.Error}}{{else}}Healthy{{end}}</div></div>{{end}}</div>
</main></div></body></html>`))


var migrationsTemplate = template.Must(template.New("copytygo-studio-migrations").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Migrations · CopyTyGo Studio</title><style>` + studioStyle + `
.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:24px}.actions form{margin:0}.actions button{border:1px solid #344863;background:#101c2e;color:#dce8fb;border-radius:8px;padding:10px 13px;cursor:pointer}.actions .primary{background:#f4f7fb;color:#0c1525;border-color:#f4f7fb}.flash{margin-top:18px;padding:13px 15px;border-radius:10px;background:#10233a;border:1px solid #29405e}.flash.error{background:#2a1518;border-color:#6b2a31;color:#ffc5cb}.ran{color:#74d99f}.pending{color:#ffcf70}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a class="active" href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
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
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a class="active" href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Logs</h1><div style="color:#6f84a2;margin-top:6px">Structured framework and request logs</div></div><span class="pill">{{.Environment}}</span></div>
<div class="toolbar"><div style="color:#6f84a2">{{len .Entries}} retained entries</div><form method="post" action="{{.BasePath}}/logs/clear"><button type="submit">Clear</button></form></div>
<table><thead><tr><th>Level</th><th>Message</th><th>Fields</th><th>Time</th></tr></thead><tbody>
{{range .Entries}}<tr><td><span class="level {{.Level}}">{{.Level}}</span></td><td>{{.Message}}</td><td class="fields">{{printf "%v" .Fields}}</td><td class="fields">{{.At}}</td></tr>{{else}}<tr><td colspan="4" class="fields">No logs captured yet.</td></tr>{{end}}
</tbody></table></main></div></body></html>`))


var queueTemplate = template.Must(template.New("copytygo-studio-queue").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Queue · CopyTyGo Studio</title><style>` + studioStyle + `
.state{color:#74d99f}.stopped{color:#ffcf70}.error{color:#ff8e98}.small{color:#758aa8;font-size:12px}.stats4{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px;margin-top:24px}@media(max-width:800px){.stats4{grid-template-columns:repeat(2,1fr)}}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a class="active" href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Queue</h1><div style="color:#6f84a2;margin-top:6px">Background job worker runtime · driver: <strong>{{.Driver}}</strong></div></div>
{{if eq .Driver "database"}}<span class="pill state">Durable Queue</span>{{else}}{{if .MemoryStats.Started}}<span class="pill state">Running · {{.MemoryStats.Concurrency}} worker(s)</span>{{else}}<span class="pill stopped">Stopped</span>{{end}}{{end}}</div>
{{if eq .Driver "database"}}
<section class="stats4"><div class="card"><div class="label">Pending</div><div class="value">{{.PendingDB}}</div></div><div class="card"><div class="label">Running</div><div class="value">{{.DurableStats.Running}}</div></div><div class="card"><div class="label">Completed</div><div class="value">{{.DurableStats.Completed}}</div></div><div class="card"><div class="label">Failed</div><div class="value">{{.FailedDB}}</div></div></section>
{{else}}
<section class="stats4"><div class="card"><div class="label">Pending</div><div class="value">{{.MemoryStats.Pending}}</div></div><div class="card"><div class="label">Running</div><div class="value">{{.MemoryStats.Running}}</div></div><div class="card"><div class="label">Completed</div><div class="value">{{.MemoryStats.Completed}}</div></div><div class="card"><div class="label">Failed</div><div class="value">{{.MemoryStats.Failed}}</div></div></section>
{{end}}
{{if .Error}}<div class="notice error">{{.Error}}</div>{{end}}
<table><thead><tr><th>Failed Job</th><th>Attempts</th><th>Error</th><th>Time</th></tr></thead><tbody>{{range .Failures}}<tr><td>{{.Name}}</td><td>{{.Attempts}}</td><td class="error">{{.Error}}</td><td class="small">{{.At}}</td></tr>{{else}}<tr><td colspan="4" class="small">No failed jobs.</td></tr>{{end}}</tbody></table>
</main></div></body></html>`))

var schedulerTemplate = template.Must(template.New("copytygo-studio-scheduler").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Scheduler · CopyTyGo Studio</title><style>` + studioStyle + `
.state{color:#74d99f}.stopped{color:#ffcf70}.error{color:#ff8e98}.small{color:#758aa8;font-size:12px}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a class="active" href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Scheduler</h1><div style="color:#6f84a2;margin-top:6px">Registered recurring background tasks</div></div>{{if .Started}}<span class="pill state">Running</span>{{else}}<span class="pill stopped">Stopped</span>{{end}}</div>
<table><thead><tr><th>Task</th><th>Schedule</th><th>Runs</th><th>Last Run</th><th>Next Run</th><th>Last Error</th></tr></thead><tbody>{{range .Entries}}<tr><td><strong>{{.Name}}</strong></td><td>{{if .Cron}}<code>{{.Cron}}</code>{{else}}{{.Interval}}{{end}}</td><td>{{.Runs}}</td><td class="small">{{.LastRun}}</td><td class="small">{{.NextRun}}</td><td class="error">{{.LastError}}</td></tr>{{else}}<tr><td colspan="6" class="small">No scheduled tasks registered.</td></tr>{{end}}</tbody></table>
</main></div></body></html>`))


var resourcesTemplate = template.Must(template.New("copytygo-studio-resources").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Resources · CopyTyGo Studio</title><style>` + studioStyle + `
.resource-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px;margin-top:24px}.resource-card{border:1px solid #1e2d43;background:#0b1525;border-radius:14px;padding:18px}.resource-card h3{margin:0 0 8px}.small{color:#758aa8;font-size:12px}.resource-card a{display:inline-block;margin-top:14px;text-decoration:none}@media(max-width:900px){.resource-grid{grid-template-columns:1fr}}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a class="active" href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Resources</h1><div style="color:#6f84a2;margin-top:6px">Generated application resources</div></div><span class="pill">{{len .Resources}} resource(s)</span></div>
{{if .Error}}<div class="notice">{{.Error}}</div>{{end}}<section class="resource-grid">{{range .Resources}}<article class="resource-card"><h3>{{.Resource}}</h3><div class="small">Controller: {{.ControllerType}}</div><div class="small">File: {{.File}}</div><a href="{{$.BasePath}}/resources/{{.Resource}}">Open data preview →</a></article>{{else}}<div class="notice">No generated resources yet. Use Studio Generator or <code>ctg make:resource</code>.</div>{{end}}</section>
</main></div></body></html>`))

var resourcePreviewTemplate = template.Must(template.New("copytygo-studio-resource-preview").Funcs(template.FuncMap{
	"value": func(row map[string]any, key string) any { return row[key] },
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Name}} · CopyTyGo Studio</title><style>` + studioStyle + `
.back{display:inline-block;margin-top:18px;text-decoration:none}.errorbox{margin-top:20px;padding:14px;border:1px solid #6b2a31;background:#2a1518;color:#ffc5cb;border-radius:10px}.scroll{overflow:auto}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a class="active" href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>{{.Name}}</h1><div style="color:#6f84a2;margin-top:6px">{{.Page.Total}} active row(s) · first 50 shown</div></div><span class="pill">Resource Preview</span></div><a class="back" href="{{.BasePath}}/resources">← All resources</a>
{{if .Error}}<div class="errorbox">{{.Error}}</div>{{else}}<div class="scroll"><table><thead><tr>{{range .Columns}}<th>{{.}}</th>{{end}}</tr></thead><tbody>{{range .Page.Data}}{{$row := .}}<tr>{{range $.Columns}}<td><code>{{value $row .}}</code></td>{{end}}</tr>{{else}}<tr><td colspan="99" style="color:#6f84a2">No data.</td></tr>{{end}}</tbody></table></div>{{end}}
</main></div></body></html>`))


var errorsTemplate = template.Must(template.New("copytygo-studio-errors").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Errors · CopyTyGo Studio</title><style>` + studioStyle + `
.toolbar{display:flex;justify-content:space-between;align-items:center;margin-top:24px}.toolbar button{border:1px solid #344863;background:#101c2e;color:#dce8fb;border-radius:8px;padding:9px 12px;cursor:pointer}.panic{color:#ff8e98}.muted{color:#758aa8;font-size:12px}.msg{max-width:520px;word-break:break-word}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a class="active" href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Error Inspector</h1><div style="color:#6f84a2;margin-top:6px">Recent handler errors and panics from the local application</div></div><span class="pill">{{len .Errors}} retained</span></div>
<div class="toolbar"><div class="muted">Newest first</div><form method="post" action="{{.BasePath}}/errors/clear"><button type="submit">Clear</button></form></div>
<table><thead><tr><th>Type</th><th>Method</th><th>Path</th><th>Message</th><th>Request ID</th><th>Time</th></tr></thead><tbody>
{{range .Errors}}<tr><td class="{{if .Panic}}panic{{end}}">{{if .Panic}}Panic{{else}}Error{{end}}</td><td><strong>{{.Method}}</strong></td><td><code>{{.Path}}</code></td><td class="msg">{{.Message}}</td><td class="muted"><code>{{.RequestID}}</code></td><td class="muted">{{.At}}</td></tr>{{else}}<tr><td colspan="6" class="muted">No errors captured.</td></tr>{{end}}
</tbody></table></main></div></body></html>`))


var servicesTemplate = template.Must(template.New("copytygo-studio-services").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Services · CopyTyGo Studio</title><style>` + studioStyle + `
.service-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px;margin-top:24px}.service{border:1px solid #1e2d43;background:#0b1525;border-radius:14px;padding:20px}.service h3{margin:0 0 5px}.meta{color:#758aa8;font-size:12px;line-height:1.7;word-break:break-word}.metric{font-size:28px;font-weight:800;margin:10px 0}.actions{margin-top:14px}.actions button{border:1px solid #344863;background:#101c2e;color:#dce8fb;border-radius:8px;padding:9px 12px;cursor:pointer}.listener{display:flex;justify-content:space-between;gap:12px;padding:8px 0;border-bottom:1px solid #1e2d43;font-size:13px}@media(max-width:800px){.service-grid{grid-template-columns:1fr}}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a class="active" href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Application Services</h1><div style="color:#6f84a2;margin-top:6px">Cache, events, storage and mail registered in the service container</div></div><span class="pill">{{.Environment}}</span></div>
<section class="service-grid">
<article class="service"><h3>Cache</h3><div class="meta">{{.CacheDriver}}</div><div class="metric">{{.CacheSize}}</div><div class="meta">active in-memory entries</div><form class="actions" method="post" action="{{.BasePath}}/services/cache/clear"><button type="submit">Clear Cache</button></form></article>
<article class="service"><h3>Storage</h3><div class="meta">{{.StorageDriver}}</div><div class="metric">Local</div><div class="meta">{{.StorageRoot}}</div></article>
<article class="service"><h3>Mail</h3><div class="meta">{{.MailDriver}}</div><div class="metric">SMTP</div><div class="meta">Host: {{.MailHost}}:{{.MailPort}}<br>From: {{.MailFrom}}<br>Password is never exposed in Studio.</div></article>
<article class="service"><h3>Events</h3><div class="metric">{{len .Events}}</div><div class="meta">registered event names</div>{{range .Events}}<div class="listener"><code>{{.Event}}</code><span>{{.Listeners}} listener(s)</span></div>{{else}}<div class="meta">No event listeners registered yet.</div>{{end}}</article>
</section></main></div></body></html>`))


var healthTemplate = template.Must(template.New("copytygo-studio-health").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Health · CopyTyGo Studio</title><style>` + studioStyle + `
.good{color:#74d99f}.bad{color:#ff8e98}.muted{color:#758aa8;font-size:12px}.health-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px;margin-top:24px}.health-card{border:1px solid #1e2d43;background:#0b1525;border-radius:14px;padding:18px}.health-card h3{margin:0 0 10px}.message{margin-top:8px;color:#8094b1;font-size:12px;word-break:break-word}@media(max-width:800px){.health-grid{grid-template-columns:1fr}}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a href="{{.BasePath}}">Dashboard</a><a href="{{.BasePath}}/routes">Routes</a><a href="{{.BasePath}}/resources">Resources</a><a href="{{.BasePath}}/database">Database</a><a href="{{.BasePath}}/migrations">Migrations</a><a href="{{.BasePath}}/services">Services</a><a href="{{.BasePath}}/queue">Queue</a><a href="{{.BasePath}}/scheduler">Scheduler</a><a href="{{.BasePath}}/requests">Requests</a><a href="{{.BasePath}}/errors">Errors</a><a href="{{.BasePath}}/logs">Logs</a><a href="{{.BasePath}}/generator">Generator</a><a class="active" href="{{.BasePath}}/health">Health</a><a href="{{.BasePath}}/doctor">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>Runtime Health</h1><div style="color:#6f84a2;margin-top:6px">Live dependency checks for the running application</div></div><span class="pill {{if eq .Status "healthy"}}good{{else}}bad{{end}}">{{.Status}}</span></div>
{{if .Error}}<div class="notice bad">{{.Error}}</div>{{end}}
<section class="health-grid">{{range .Results}}<article class="health-card"><h3>{{.Name}}</h3><div class="{{if eq .Status "healthy"}}good{{else}}bad{{end}}">{{.Status}}</div><div class="muted">{{.DurationMS}} ms</div>{{if .Message}}<div class="message">{{.Message}}</div>{{end}}</article>{{else}}<div class="notice">No health checks registered.</div>{{end}}</section>
</main></div></body></html>`))
