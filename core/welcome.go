package core

import (
	"html/template"
	"strings"

	"github.com/arfajhf/copytygo/v5/branding"
	"github.com/arfajhf/copytygo/v5/config"
	"github.com/arfajhf/copytygo/v5/version"
)

type WelcomeOptions struct {
	AppName     string
	DocsURL     string
	GitHubURL   string
	StudioURL   string
	LoginURL    string
	RegisterURL string
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
			LogoURL     template.URL
			AppName     string
			Version     string
			Environment string
			DocsURL     string
			GitHubURL   string
			StudioURL   string
			ShowStudio  bool
			LoginURL    string
			RegisterURL string
		}{
			LogoURL:     branding.LogoURL(),
			AppName:     appName,
			Version:     version.Framework,
			Environment: config.Get("APP_ENV", "local"),
			DocsURL:     docsURL,
			GitHubURL:   githubURL,
			StudioURL:   studioURL,
			ShowStudio:  !strings.EqualFold(config.Get("APP_ENV", "local"), "production") && config.GetBool("COPYTYGO_STUDIO", true),
		}
		loginURL, registerURL := opts.LoginURL, opts.RegisterURL
		if loginURL == "" {
			loginURL = "/login"
		}
		if registerURL == "" {
			registerURL = "/register"
		}
		if ctx.router != nil {
			for _, route := range ctx.router.Routes() {
				if route.Method != "GET" {
					continue
				}
				if route.Path == loginURL {
					data.LoginURL = loginURL
				}
				if route.Path == registerURL {
					data.RegisterURL = registerURL
				}
			}
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
<title>{{.AppName}} · CopyTyGo</title><link rel="icon" href="{{.LogoURL}}">
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#e7edf7;background:#08101f}
*{box-sizing:border-box}body{margin:0;min-height:100vh;background:radial-gradient(circle at top,#15213a 0,#08101f 42%,#050a13 100%);display:flex;flex-direction:column}
.navbar{width:100%;background:#0b1424eF;border-bottom:1px solid #26354d;position:relative;z-index:5}
.nav-inner{max-width:1160px;min-height:78px;margin:0 auto;padding:14px 28px;display:flex;align-items:center;justify-content:space-between;gap:24px}
.brand{display:inline-flex;align-items:center;gap:13px;min-width:0;text-decoration:none;color:#f2f7ff}
.brand:hover{text-decoration:none}
.brand-icon{width:48px;height:48px;border:1px solid #3a526e;border-radius:14px;display:grid;place-items:center;background:linear-gradient(145deg,#ffffff,#dfedf9);flex-shrink:0;box-shadow:0 6px 24px #0003}
.brand img{width:36px;height:36px;object-fit:contain}
.brand-copy{min-width:0;display:flex;flex-direction:column;gap:3px}
.brand-name{font-weight:800;font-size:17px;letter-spacing:-.02em;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:310px}
.brand-caption{color:#8ca2c1;font-size:11px;letter-spacing:.035em;font-weight:500}
.navbar nav{display:flex;align-items:center;justify-content:flex-end;gap:10px;flex-wrap:wrap}
.navbar a.nav-link{display:inline-flex;align-items:center;color:#b6c5dc;text-decoration:none;font-weight:650;font-size:13px;padding:11px 12px;border-radius:9px;white-space:nowrap}
.navbar a.nav-link:hover{background:#1a2a42;color:#fff}
.navbar a.nav-action{display:inline-flex;align-items:center;justify-content:center;min-height:39px;padding:9px 15px;border-radius:10px;text-decoration:none;font-weight:700;font-size:13px;color:#e5effe;background:#17263c;border:1px solid #36506e;white-space:nowrap}
.navbar a.nav-action:hover{background:#243a57;border-color:#507199}
.navbar a.nav-action-primary{background:#ecf3ff;border-color:#ecf3ff;color:#0d1a2f}
.navbar a.nav-action-primary:hover{background:#fff;border-color:#fff}
.navbar a:focus-visible{outline:2px solid #8fb5ff;outline-offset:3px}
.dot{width:8px;height:8px;border-radius:50%;background:#59d98e;box-shadow:0 0 18px #59d98e}
.card{margin-top:16px;border:1px solid #24324a;border-radius:24px;background:#0c1525d9;box-shadow:0 28px 90px #0008;overflow:hidden}
.hero{padding:58px 54px 40px}.mark{font-size:14px;letter-spacing:.14em;text-transform:uppercase;color:#80a7ff;font-weight:700}h1{font-size:clamp(42px,7vw,76px);line-height:.98;margin:16px 0;color:#f8fbff;letter-spacing:-.055em}
.lead{max-width:660px;color:#9fb0ca;font-size:18px;line-height:1.7;margin:0}.actions{display:flex;gap:12px;flex-wrap:wrap;margin-top:30px}
a.btn{padding:12px 16px;border-radius:11px;text-decoration:none;font-weight:650;font-size:14px;border:1px solid #314461;color:#dce8fb;background:#111e32}
a.primary{background:#f4f7fb;color:#0c1525;border-color:#f4f7fb}.grid{border-top:1px solid #24324a;display:grid;grid-template-columns:repeat(3,1fr)}
.stat{padding:22px 28px}.stat+.stat{border-left:1px solid #24324a}.label{font-size:12px;text-transform:uppercase;letter-spacing:.12em;color:#657894}.value{margin-top:8px;font-weight:700;color:#dce8fb}
.foot{margin-top:14px;text-align:center;color:#526680;font-size:12px}
@media(max-width:700px){
.nav-inner{min-height:68px;padding:12px 20px;gap:12px}
.brand-icon{width:42px;height:42px;border-radius:12px}
.brand img{width:33px;height:33px}
.brand-name{font-size:15px;max-width:180px}
.brand-caption{font-size:10px}
.navbar nav{gap:5px}
.navbar a.nav-link{padding:8px 7px;font-size:12px}
.navbar a.nav-action{padding:8px 11px;min-height:36px;font-size:12px}
.navbar a.nav-github{display:none}
.welcome{padding:32px 20px}
.hero{padding:38px 26px 30px}
.grid{grid-template-columns:1fr}
.stat+.stat{border-left:0;border-top:1px solid #24324a}
}
@media(max-width:520px){
.nav-inner{flex-wrap:wrap;align-items:center;row-gap:10px}
.brand-copy{max-width:calc(100vw - 120px)}
.brand-caption{display:none}
.navbar nav{width:100%;justify-content:flex-start;gap:6px}
.navbar a.nav-link{padding:8px 11px}
}
</style>
</head>
<body>
<header class="navbar">
<div class="nav-inner">
<a class="brand" href="/" aria-label="{{.AppName}} home"><span class="brand-icon"><img src="{{.LogoURL}}" alt="CopyTyGo logo"></span><span class="brand-copy"><span class="brand-name">{{.AppName}}</span><span class="brand-caption">Powered by CopyTyGo</span></span></a>
<nav aria-label="Main navigation">
<a class="nav-link" href="{{.DocsURL}}" target="_blank" rel="noopener noreferrer">Docs</a>
<a class="nav-link nav-github" href="{{.GitHubURL}}" target="_blank" rel="noopener noreferrer">GitHub</a>
{{if .ShowStudio}}<a class="nav-action" href="{{.StudioURL}}">Open Studio</a>{{end}}
{{if .LoginURL}}<a class="nav-link" href="{{.LoginURL}}">Log in</a>{{end}}
{{if .RegisterURL}}<a class="nav-action nav-action-primary" href="{{.RegisterURL}}">Register</a>{{end}}
</nav>
</div>
</header>
<main class="welcome"><div class="shell">
<div class="badge"><span class="dot"></span>Your application is running</div>
<section class="card">
<div class="hero">
<div class="mark">Your next application starts here</div>
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
</div></main></body></html>`))
