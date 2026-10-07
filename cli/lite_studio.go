package cli

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/arfajhf/copytygo/v4/branding"
	"github.com/arfajhf/copytygo/v4/config"
	copycontainer "github.com/arfajhf/copytygo/v4/container"
	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/database/query"
	"github.com/arfajhf/copytygo/v4/foundation"
	"github.com/arfajhf/copytygo/v4/version"
)

func handleLiteStudio(w http.ResponseWriter, req *http.Request) bool {
	if !strings.HasPrefix(req.URL.Path, "/__copytygo") {
		return false
	}
	if strings.EqualFold(config.Get("APP_ENV", "local"), "production") || !config.GetBool("COPYTYGO_STUDIO", true) {
		http.NotFound(w, req)
		return true
	}

	base := "/__copytygo"
	switch {
	case req.Method == http.MethodGet && req.URL.Path == base:
		routes, _ := discoverLiteRoutes("routes")
		resources, _ := DiscoverGeneratedResources(".")
		content := liteStudioCards(
			[2]string{"Runtime", "Lite"},
			[2]string{"Routes", fmt.Sprint(len(routes))},
			[2]string{"Resources", fmt.Sprint(len(resources))},
		) + `<div class="notice"><strong>Lite Runtime Studio is active.</strong><br>Project inspection, generators, database tools and migrations work without compiling a temporary application executable. Queue and Scheduler execution require Native Runtime because they execute arbitrary project Go code.</div>`
		writeLiteStudioPage(w, "Dashboard", base, content)
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/routes":
		routes, err := discoverLiteRoutes("routes")
		var body strings.Builder
		if err != nil {
			body.WriteString(liteStudioError(err))
		} else {
			body.WriteString("<table><thead><tr><th>Method</th><th>Path</th><th>Type</th></tr></thead><tbody>")
			for _, route := range routes {
				body.WriteString("<tr><td><strong>" + html.EscapeString(route.Method) + "</strong></td><td><code>" + html.EscapeString(route.Path) + "</code></td><td>" + html.EscapeString(liteRouteKind(route)) + "</td></tr>")
			}
			body.WriteString("</tbody></table>")
		}
		writeLiteStudioPage(w, "Route Explorer", base, body.String())
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/models":
		models, err := DiscoverModels("app/models")
		var body strings.Builder
		if err != nil {
			body.WriteString(liteStudioError(err))
		} else {
			body.WriteString(`<div class="grid">`)
			for _, model := range models {
				body.WriteString(`<div class="card"><div class="label">Model</div><div class="value">` + html.EscapeString(model.Name) + `</div><p class="muted">` + html.EscapeString(model.File) + ` · ` + fmt.Sprint(len(model.Fields)) + ` field(s)</p></div>`)
			}
			if len(models) == 0 {
				body.WriteString(`<div class="notice">No models found.</div>`)
			}
			body.WriteString("</div>")
		}
		writeLiteStudioPage(w, "Models", base, body.String())
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/auth":
		raw, err := os.ReadFile("app/auth/user.go")
		if err != nil {
			if os.IsNotExist(err) {
				writeLiteStudioPage(w, "Authentication", base, `<div class="notice">Authentication is not installed. Use <code>ctg install:auth single</code> or <code>ctg install:auth multi</code>.</div>`)
				return true
			}
			writeLiteStudioPage(w, "Authentication", base, liteStudioError(err))
			return true
		}

		multiRole := strings.Contains(string(raw), "Role string")
		drivers.Register()
		db, connectErr := database.Connect()
		if connectErr != nil {
			writeLiteStudioPage(w, "Authentication", base, liteStudioError(connectErr))
			return true
		}

		columns := []string{"id", "name", "email", "created_at", "updated_at"}
		if multiRole {
			columns = []string{"id", "name", "email", "role", "created_at", "updated_at"}
		}
		users, queryErr := query.Table(db, config.Get("DB_DRIVER", "mysql"), "users").
			Select(columns...).
			OrderBy("id", "DESC").
			Limit(50).
			AllMaps(req.Context())
		if queryErr != nil {
			writeLiteStudioPage(w, "Authentication", base, liteStudioError(queryErr))
			return true
		}

		var body strings.Builder
		body.WriteString(`<div class="notice good">Auth installed. Password hashes are never selected by this inspector.</div>`)
		body.WriteString("<table><thead><tr><th>ID</th><th>Name</th><th>Email</th>")
		if multiRole {
			body.WriteString("<th>Role</th>")
		}
		body.WriteString("</tr></thead><tbody>")
		for _, user := range users {
			body.WriteString("<tr><td>" + html.EscapeString(fmt.Sprint(user["id"])) + "</td><td>" + html.EscapeString(fmt.Sprint(user["name"])) + "</td><td>" + html.EscapeString(fmt.Sprint(user["email"])) + "</td>")
			if multiRole {
				body.WriteString("<td>" + html.EscapeString(fmt.Sprint(user["role"])) + "</td>")
			}
			body.WriteString("</tr>")
		}
		body.WriteString("</tbody></table>")
		writeLiteStudioPage(w, "Authentication", base, body.String())
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/services":
		body := liteStudioCards(
			[2]string{"Cache", "Memory"},
			[2]string{"Storage", config.Get("STORAGE_PATH", "storage/app")},
			[2]string{"Mail", config.Get("MAIL_HOST", "127.0.0.1") + ":" + fmt.Sprint(config.GetInt("MAIL_PORT", 1025))},
		)
		body += `<div class="notice">Lite Studio shows configured service defaults. Runtime service-container state is available in Native Studio.</div>`
		writeLiteStudioPage(w, "Application Services", base, body)
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/health":
		services := copycontainer.New()
		if err := foundation.RegisterDefaults(services); err != nil {
			writeLiteStudioPage(w, "Runtime Health", base, liteStudioError(err))
			return true
		}
		status, results, err := foundation.RunHealth(req.Context(), services)
		var body strings.Builder
		body.WriteString(`<div class="notice"><strong>Status: ` + html.EscapeString(string(status)) + `</strong></div>`)
		if err != nil {
			body.WriteString(liteStudioError(err))
		} else {
			body.WriteString("<table><thead><tr><th>Check</th><th>Status</th><th>Latency</th><th>Message</th></tr></thead><tbody>")
			for _, result := range results {
				body.WriteString("<tr><td><strong>" + html.EscapeString(result.Name) + "</strong></td><td>" + html.EscapeString(string(result.Status)) + "</td><td>" + fmt.Sprint(result.DurationMS) + " ms</td><td>" + html.EscapeString(result.Message) + "</td></tr>")
			}
			body.WriteString("</tbody></table>")
		}
		writeLiteStudioPage(w, "Runtime Health", base, body.String())
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/resources":
		resources, err := DiscoverGeneratedResources(".")
		var body strings.Builder
		if err != nil {
			body.WriteString(liteStudioError(err))
		} else {
			body.WriteString(`<div class="grid">`)
			for _, item := range resources {
				body.WriteString(`<div class="card"><div class="label">Resource</div><div class="value">` + html.EscapeString(item.Resource) + `</div><p>` + html.EscapeString(item.ControllerType) + `</p></div>`)
			}
			if len(resources) == 0 {
				body.WriteString(`<div class="notice">No generated resources yet.</div>`)
			}
			body.WriteString("</div>")
		}
		writeLiteStudioPage(w, "Resources", base, body.String())
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/database":
		drivers.Register()
		cfg := database.LoadConfig()
		_, err := database.Connect()
		status := `<span class="good">Connected</span>`
		if err != nil {
			status = `<span class="bad">Disconnected</span><div class="notice">` + html.EscapeString(err.Error()) + `</div>`
		}
		body := liteStudioCards(
			[2]string{"Driver", cfg.Driver},
			[2]string{"Database", cfg.Database},
			[2]string{"Host", cfg.Host + ":" + cfg.Port},
		) + `<div class="notice">Status: ` + status + `</div>`
		writeLiteStudioPage(w, "Database", base, body)
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/migrations":
		statuses, err := ProjectMigrationStatus()
		var body strings.Builder
		body.WriteString(`<div class="actions"><form method="post" action="` + base + `/migrations/run"><button>Run Migrations</button></form><form method="post" action="` + base + `/migrations/rollback"><button>Rollback Last Batch</button></form></div>`)
		if message := req.URL.Query().Get("message"); message != "" {
			body.WriteString(`<div class="notice good">` + html.EscapeString(message) + `</div>`)
		}
		if queryErr := req.URL.Query().Get("error"); queryErr != "" {
			body.WriteString(`<div class="notice bad">` + html.EscapeString(queryErr) + `</div>`)
		}
		if err != nil {
			body.WriteString(liteStudioError(err))
		} else {
			body.WriteString("<table><thead><tr><th>Status</th><th>Batch</th><th>Migration</th></tr></thead><tbody>")
			for _, item := range statuses {
				status := "Pending"
				batch := "—"
				if item.Ran {
					status = "Ran"
					batch = fmt.Sprint(item.Batch)
				}
				body.WriteString("<tr><td>" + status + "</td><td>" + batch + "</td><td><code>" + html.EscapeString(item.Migration) + "</code></td></tr>")
			}
			body.WriteString("</tbody></table>")
		}
		writeLiteStudioPage(w, "Migrations", base, body.String())
		return true

	case req.Method == http.MethodPost && req.URL.Path == base+"/migrations/run":
		if err := ProjectMigrate(); err != nil {
			http.Redirect(w, req, base+"/migrations?error="+url.QueryEscape(err.Error()), http.StatusFound)
		} else {
			http.Redirect(w, req, base+"/migrations?message="+url.QueryEscape("Migrations completed"), http.StatusFound)
		}
		return true

	case req.Method == http.MethodPost && req.URL.Path == base+"/migrations/rollback":
		if err := ProjectRollback(); err != nil {
			http.Redirect(w, req, base+"/migrations?error="+url.QueryEscape(err.Error()), http.StatusFound)
		} else {
			http.Redirect(w, req, base+"/migrations?message="+url.QueryEscape("Last batch rolled back"), http.StatusFound)
		}
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/generator":
		body := `<form class="form" method="post" action="` + base + `/generator/resource">
<label>Resource name</label><input name="name" placeholder="Product" required>
<label>Fields</label><textarea name="fields" placeholder="name:string&#10;price:decimal&#10;stock:integer&#10;active:boolean&#10;description:text?"></textarea>
<p class="muted">One field per line or separated by spaces.</p><button>Create Resource</button></form>
<form class="form" method="post" action="` + base + `/generator/scaffold">
<label>Type</label><select name="type"><option value="model">Model</option><option value="middleware">Middleware</option><option value="service">Service</option><option value="job">Job</option><option value="listener">Listener</option><option value="seeder">Seeder</option><option value="factory">Factory</option><option value="mail">Mail</option></select>
<label>Name</label><input name="name" placeholder="SendWelcomeEmail" required><button>Create Scaffold</button></form>`
		if created := req.URL.Query().Get("created"); created != "" {
			body = `<div class="notice good">Generated <strong>` + html.EscapeString(created) + `</strong> created.</div>` + body
		}
		if queryErr := req.URL.Query().Get("error"); queryErr != "" {
			body = `<div class="notice bad">` + html.EscapeString(queryErr) + `</div>` + body
		}
		writeLiteStudioPage(w, "Visual Generator", base, body)
		return true

	case req.Method == http.MethodPost && req.URL.Path == base+"/generator/resource":
		if err := req.ParseForm(); err != nil {
			http.Redirect(w, req, base+"/generator?error="+url.QueryEscape("Invalid form"), http.StatusFound)
			return true
		}
		name := strings.TrimSpace(req.FormValue("name"))
		fields := strings.Fields(req.FormValue("fields"))
		if err := MakeResourceWithFields(name, fields); err != nil {
			http.Redirect(w, req, base+"/generator?error="+url.QueryEscape(err.Error()), http.StatusFound)
		} else {
			http.Redirect(w, req, base+"/generator?created="+url.QueryEscape(name), http.StatusFound)
		}
		return true

	case req.Method == http.MethodPost && req.URL.Path == base+"/generator/scaffold":
		if err := req.ParseForm(); err != nil {
			http.Redirect(w, req, base+"/generator?error="+url.QueryEscape("Invalid form"), http.StatusFound)
			return true
		}
		kind := strings.ToLower(strings.TrimSpace(req.FormValue("type")))
		name := strings.TrimSpace(req.FormValue("name"))
		if err := MakeScaffold(kind, name); err != nil {
			http.Redirect(w, req, base+"/generator?error="+url.QueryEscape(err.Error()), http.StatusFound)
		} else {
			http.Redirect(w, req, base+"/generator?created="+url.QueryEscape(kind+" "+name), http.StatusFound)
		}
		return true

	case req.Method == http.MethodGet && req.URL.Path == base+"/doctor":
		results := RunDoctorChecks([]string{"--db"})
		var body strings.Builder
		body.WriteString(`<div class="checks">`)
		for _, item := range results {
			icon := `<span class="good">✓</span>`
			message := "Healthy"
			if !item.OK {
				icon = `<span class="bad">×</span>`
				message = item.Error
			}
			body.WriteString(`<div class="check">` + icon + ` <strong>` + html.EscapeString(item.Name) + `</strong><span class="muted">` + html.EscapeString(message) + `</span></div>`)
		}
		body.WriteString("</div>")
		writeLiteStudioPage(w, "Doctor", base, body.String())
		return true

	case req.Method == http.MethodGet && (req.URL.Path == base+"/queue" || req.URL.Path == base+"/scheduler"):
		title := "Queue"
		if strings.HasSuffix(req.URL.Path, "/scheduler") {
			title = "Scheduler"
		}
		writeLiteStudioPage(w, title, base, `<div class="notice"><strong>Native Runtime required for execution.</strong><br>Lite Studio can inspect and generate project code, but arbitrary background Go jobs are intentionally not executed under Lite Runtime.</div>`)
		return true
	}

	http.NotFound(w, req)
	return true
}

func liteRouteKind(route liteRoute) string {
	switch {
	case route.AuthAction != "":
		return "Auth"
	case strings.HasPrefix(route.ResourceAction, "db:"):
		return "Database Resource"
	case route.ResourceAction != "":
		return "Memory Resource"
	default:
		return "Route"
	}
}

func liteStudioCards(items ...[2]string) string {
	var body strings.Builder
	body.WriteString(`<div class="grid">`)
	for _, item := range items {
		body.WriteString(`<div class="card"><div class="label">` + html.EscapeString(item[0]) + `</div><div class="value">` + html.EscapeString(item[1]) + `</div></div>`)
	}
	body.WriteString("</div>")
	return body.String()
}

func liteStudioError(err error) string {
	if err == nil {
		return ""
	}
	return `<div class="notice bad">` + html.EscapeString(err.Error()) + `</div>`
}

func writeLiteStudioPage(w http.ResponseWriter, title, base, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	nav := func(label, href string) string {
		return `<a href="` + base + href + `">` + label + `</a>`
	}
	fmt.Fprint(w, `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>`+html.EscapeString(title)+` · CopyTyGo Studio</title><style>
:root{font-family:Inter,system-ui;background:#070d18;color:#e7edf7}*{box-sizing:border-box}body{margin:0}.layout{min-height:100vh;display:grid;grid-template-columns:230px 1fr}.side{padding:24px 18px;border-right:1px solid #1d2a3d;background:#09111f}.brand{font-weight:800;font-size:20px;margin:5px 8px 22px}.brand small{display:block;color:#647896;font-size:11px;margin-top:4px}nav a{display:block;color:#8ea2bf;text-decoration:none;padding:10px 11px;border-radius:9px}.main{padding:34px}.pill{display:inline-block;border:1px solid #29405e;border-radius:999px;padding:7px 10px;color:#91a8c7;font-size:12px}.grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px;margin-top:24px}.card,.notice,.form{border:1px solid #1e2d43;background:#0b1525;border-radius:14px;padding:18px}.label{font-size:11px;text-transform:uppercase;letter-spacing:.12em;color:#667b99}.value{font-size:24px;font-weight:750;margin-top:9px}.notice{margin-top:20px;color:#91a5c3;line-height:1.6}.good{color:#74d99f}.bad{color:#ff8e98}.muted{color:#758aa8;font-size:12px}table{width:100%;border-collapse:collapse;margin-top:24px;background:#0b1525}th,td{padding:12px;border-bottom:1px solid #1e2d43;text-align:left;font-size:13px}th{color:#758aa8}.actions{display:flex;gap:10px;margin-top:20px}.actions form{margin:0}.form{max-width:760px;margin-top:24px}.form label{display:block;margin:14px 0 7px;color:#758aa8;font-size:12px;text-transform:uppercase}.form input,.form textarea,.form select{width:100%;background:#08111f;border:1px solid #263852;color:#e7edf7;border-radius:9px;padding:11px}.form textarea{min-height:150px}button{border:1px solid #344863;background:#f4f7fb;color:#0c1525;border-radius:8px;padding:10px 13px;font-weight:700;cursor:pointer}.checks{margin-top:20px}.check{display:grid;grid-template-columns:25px 180px 1fr;padding:12px;border-bottom:1px solid #1e2d43}@media(max-width:800px){.layout{grid-template-columns:1fr}.side{display:none}.main{padding:22px}.grid{grid-template-columns:1fr}}
</style></head><body><div class="layout"><aside class="side"><div class="brand"><img src="`+string(branding.LogoURL())+`" alt="CopyTyGo logo" style="width:46px;height:46px;object-fit:contain;display:block;margin-bottom:10px;background:#fff;padding:5px;border-radius:10px">CopyTyGo<small>Lite Studio · `+version.Framework+`</small></div><nav>`+
		nav("Dashboard", "")+
		nav("Routes", "/routes")+
		nav("Models", "/models")+
		nav("Resources", "/resources")+
		nav("Database", "/database")+
		nav("Migrations", "/migrations")+
		nav("Auth", "/auth")+
		nav("Services", "/services")+
		nav("Queue", "/queue")+
		nav("Scheduler", "/scheduler")+
		nav("Generator", "/generator")+
		nav("Health", "/health")+
		nav("Doctor", "/doctor")+
		`</nav></aside><main class="main"><h1>`+html.EscapeString(title)+`</h1><span class="pill">Lite Runtime</span>`+content+`</main></div></body></html>`)
}

func liteWelcomePage() string {
	docs := config.Get("COPYTYGO_DOCS_URL", version.DocsURL)
	appName := config.Get("APP_NAME", "CopyTyGo")
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + html.EscapeString(appName) + ` · CopyTyGo</title><style>body{font-family:system-ui;background:#08101f;color:#e7edf7;min-height:100vh;display:grid;place-items:center;margin:0;padding:24px}.card{max-width:760px;border:1px solid #24324a;background:#0c1525;border-radius:22px;padding:42px}h1{font-size:48px;margin:8px 0}.muted{color:#9fb0ca;line-height:1.7}.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:24px}a{padding:11px 14px;border:1px solid #314461;border-radius:10px;color:#dce8fb;text-decoration:none}.primary{background:#f4f7fb;color:#0c1525}</style></head><body><main class="card"><img src="` + string(branding.LogoURL()) + `" alt="CopyTyGo logo" style="width:64px;height:64px;object-fit:contain;background:#fff;padding:5px;border-radius:10px"><div style="color:#80a7ff;font-weight:700">COPYTYGO · LITE RUNTIME</div><h1>` + html.EscapeString(appName) + `</h1><p class="muted">Your application is running with CopyTyGo Lite Runtime because a generated executable is not required. Studio remains available for project tooling and inspection.</p><div class="actions"><a class="primary" href="` + html.EscapeString(docs) + `" target="_blank">Documentation</a><a href="/__copytygo">Open Studio</a><a href="https://github.com/arfajhf/copytygo" target="_blank">GitHub</a></div></main></body></html>`
}
