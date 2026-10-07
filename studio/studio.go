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

func Register(app *core.Application, options ...Options) bool {
	if app == nil || strings.EqualFold(config.Get("APP_ENV", "local"), "production") {
		return false
	}
	if !config.GetBool("COPYTYGO_STUDIO", true) {
		return false
	}

	path := DefaultPath
	if len(options) > 0 && strings.TrimSpace(options[0].Path) != "" {
		path = strings.TrimSpace(options[0].Path)
	}

	app.Get(path, func(ctx *core.Context) error {
		routes := app.Routes()
		data := struct {
			AppName     string
			Version     string
			Environment string
			RouteCount  int
			DocsURL     string
		}{
			AppName:     config.Get("APP_NAME", "CopyTyGo"),
			Version:     version.Framework,
			Environment: config.Get("APP_ENV", "local"),
			RouteCount:  len(routes),
			DocsURL:     config.Get("COPYTYGO_DOCS_URL", version.DocsURL),
		}
		var out strings.Builder
		if err := studioTemplate.Execute(&out, data); err != nil {
			return fmt.Errorf("copytygo studio: %w", err)
		}
		return ctx.HTML(out.String())
	}).Name("copytygo.studio")

	return true
}

var studioTemplate = template.Must(template.New("copytygo-studio").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>CopyTyGo Studio</title><style>
:root{font-family:Inter,ui-sans-serif,system-ui;background:#070d18;color:#e7edf7}*{box-sizing:border-box}body{margin:0}
.layout{min-height:100vh;display:grid;grid-template-columns:230px 1fr}.side{border-right:1px solid #1d2a3d;padding:24px 18px;background:#09111f}
.brand{font-weight:800;font-size:20px;padding:5px 8px 22px}.brand small{display:block;font-size:11px;color:#647896;font-weight:600;margin-top:4px}
nav a{display:block;color:#8ea2bf;text-decoration:none;padding:10px 11px;border-radius:9px;margin:2px 0;font-size:14px}nav a.active{background:#12213a;color:#eaf2ff}
.main{padding:34px}.top{display:flex;justify-content:space-between;align-items:center;gap:20px}.top h1{margin:0;font-size:28px}.pill{border:1px solid #29405e;border-radius:999px;padding:7px 10px;font-size:12px;color:#91a8c7}
.cards{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px;margin-top:28px}.card{border:1px solid #1e2d43;background:#0b1525;border-radius:15px;padding:20px}.label{text-transform:uppercase;font-size:11px;letter-spacing:.12em;color:#667b99}.value{font-size:24px;font-weight:750;margin-top:9px}
.notice{margin-top:18px;border:1px dashed #2b405e;border-radius:14px;padding:20px;color:#8297b5;line-height:1.6}.notice strong{color:#dbe7f9}a{color:#9bbcff}
@media(max-width:800px){.layout{grid-template-columns:1fr}.side{display:none}.main{padding:24px}.cards{grid-template-columns:1fr}}
</style></head><body><div class="layout"><aside class="side"><div class="brand">CopyTyGo<small>Studio · {{.Version}}</small></div>
<nav><a class="active" href="#">Dashboard</a><a href="#">Routes</a><a href="#">Resources</a><a href="#">Database</a><a href="#">Migrations</a><a href="#">Queue</a><a href="#">Scheduler</a><a href="#">Logs</a><a href="#">Generator</a><a href="#">Doctor</a></nav></aside>
<main class="main"><div class="top"><div><h1>{{.AppName}}</h1><div style="color:#6f84a2;margin-top:6px">Local development workspace</div></div><span class="pill">{{.Environment}}</span></div>
<section class="cards"><div class="card"><div class="label">Framework</div><div class="value">{{.Version}}</div></div><div class="card"><div class="label">Routes</div><div class="value">{{.RouteCount}}</div></div><div class="card"><div class="label">Runtime</div><div class="value">Ready</div></div></section>
<div class="notice"><strong>CopyTyGo Studio foundation is active.</strong><br>Route explorer, migrations, resources, queues, scheduler, logs, generators and diagnostics will plug into this same workspace during v4 development. <a href="{{.DocsURL}}" target="_blank" rel="noreferrer">Open documentation</a>.</div>
</main></div></body></html>`))
