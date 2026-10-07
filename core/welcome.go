package core

import (
	"html/template"
	"strings"

	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/version"
)

type WelcomeOptions struct {
	AppName   string
	DocsURL   string
	GitHubURL string
	StudioURL string
}

const defaultGitHubURL = "https://github.com/arfajhf/copytygo"

func Welcome(options ...WelcomeOptions) Handler {
	opts := WelcomeOptions{}
	if len(options) > 0 {
		opts = options[0]
	}

	return func(ctx *Context) error {
		appName := strings.TrimSpace(opts.AppName)
		if appName == "" {
			appName = config.Get("APP_NAME", "CopyTyGo")
		}

		docsURL := strings.TrimSpace(opts.DocsURL)
		if docsURL == "" {
			docsURL = config.Get("COPYTYGO_DOCS_URL", version.DocsURL)
		}

		githubURL := strings.TrimSpace(opts.GitHubURL)
		if githubURL == "" {
			githubURL = defaultGitHubURL
		}

		studioURL := strings.TrimSpace(opts.StudioURL)
		if studioURL == "" {
			studioURL = "/__copytygo"
		}

		data := struct {
			AppName     string
			Version     string
			Environment string
			DocsURL     string
			GitHubURL   string
			StudioURL   string
			ShowStudio  bool
		}{
			AppName:     appName,
			Version:     version.Framework,
			Environment: config.Get("APP_ENV", "local"),
			DocsURL:     docsURL,
			GitHubURL:   githubURL,
			StudioURL:   studioURL,
			ShowStudio:  !strings.EqualFold(config.Get("APP_ENV", "local"), "production"),
		}

		var out strings.Builder
		if err := welcomeTemplate.Execute(&out, data); err != nil {
			return err
		}

		return ctx.HTML(out.String())
	}
}

var welcomeTemplate = template.Must(template.New("copytygo-welcome").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.AppName}} · CopyTyGo</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#e7edf7;background:#08101f}
*{box-sizing:border-box}body{margin:0;min-height:100vh;background:radial-gradient(circle at top,#15213a 0,#08101f 42%,#050a13 100%);display:grid;place-items:center;padding:28px}
.shell{width:min(1040px,100%)}.badge{display:inline-flex;gap:8px;align-items:center;padding:7px 11px;border:1px solid #293955;border-radius:999px;color:#a9bbd7;font-size:13px;background:#0d1729cc}
.dot{width:8px;height:8px;border-radius:50%;background:#59d98e;box-shadow:0 0 18px #59d98e}
.card{margin-top:16px;border:1px solid #24324a;border-radius:24px;background:#0c1525d9;box-shadow:0 28px 90px #0008;overflow:hidden}
.hero{padding:58px 54px 40px}.mark{font-size:14px;letter-spacing:.14em;text-transform:uppercase;color:#80a7ff;font-weight:700}h1{font-size:clamp(42px,7vw,76px);line-height:.98;margin:16px 0;color:#f8fbff;letter-spacing:-.055em}
.lead{max-width:660px;color:#9fb0ca;font-size:18px;line-height:1.7;margin:0}.actions{display:flex;gap:12px;flex-wrap:wrap;margin-top:30px}
a.btn{padding:12px 16px;border-radius:11px;text-decoration:none;font-weight:650;font-size:14px;border:1px solid #314461;color:#dce8fb;background:#111e32}
a.primary{background:#f4f7fb;color:#0c1525;border-color:#f4f7fb}.grid{border-top:1px solid #24324a;display:grid;grid-template-columns:repeat(3,1fr)}
.stat{padding:22px 28px}.stat+.stat{border-left:1px solid #24324a}.label{font-size:12px;text-transform:uppercase;letter-spacing:.12em;color:#657894}.value{margin-top:8px;font-weight:700;color:#dce8fb}
.foot{margin-top:14px;text-align:center;color:#526680;font-size:12px}@media(max-width:700px){.hero{padding:38px 26px 30px}.grid{grid-template-columns:1fr}.stat+.stat{border-left:0;border-top:1px solid #24324a}}
</style>
</head>
<body><main class="shell">
<div class="badge"><span class="dot"></span>Your application is running</div>
<section class="card">
<div class="hero">
<div class="mark">CopyTyGo</div>
<h1>{{.AppName}}</h1>
<p class="lead">Build applications with Go using a productive framework experience. Your project is ready — start building from the CLI or open the local development Studio.</p>
<div class="actions">
<a class="btn primary" href="{{.DocsURL}}" target="_blank" rel="noreferrer">Documentation</a>
{{if .ShowStudio}}<a class="btn" href="{{.StudioURL}}">Open Studio</a>{{end}}
<a class="btn" href="{{.GitHubURL}}" target="_blank" rel="noreferrer">GitHub</a>
</div></div>
<div class="grid">
<div class="stat"><div class="label">Framework</div><div class="value">CopyTyGo {{.Version}}</div></div>
<div class="stat"><div class="label">Environment</div><div class="value">{{.Environment}}</div></div>
<div class="stat"><div class="label">Runtime</div><div class="value">Go</div></div>
</div></section>
<div class="foot">CopyTyGo · Developer-first Go framework</div>
</main></body></html>`))
