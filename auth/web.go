package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/core"
	"github.com/arfajhf/copytygo/v4/security"
	"github.com/arfajhf/copytygo/v4/session"
)

//go:embed views/*.html
var webViews embed.FS

// WebOptions configures the browser starter independently of bearer-token APIs.
type WebOptions struct {
	DefaultRole string
	Views       fs.FS // contains views/layout.html, login.html, register.html and account.html
}

// WebAuth serves browser forms with encrypted, HttpOnly cookie sessions.
type WebAuth struct {
	options  WebOptions
	sessions *session.Manager
	login    func(string, string) (Session, error)
	register func(string, string, string, string) (Session, error)
}

type webPage struct {
	AppName string
	Title   string
	CSRF    string
	Name    string
	Email   string
	Role    string
	Message string
	Errors  map[string][]string
}

func NewWebAuth(options WebOptions) *WebAuth {
	if options.Views == nil {
		options.Views = webViews
	}
	sessions := session.New()
	sessions.Name = "copytygo_web_session"
	if sessions.Secure {
		sessions.Name = "__Host-copytygo_web_session"
	}
	return &WebAuth{options: options, sessions: sessions, login: Login, register: Register}
}

// StarterViews returns editable HTML files for install:auth.
func StarterViews() (map[string]string, error) {
	files := make(map[string]string)
	for _, name := range []string{"layout.html", "login.html", "register.html", "account.html"} {
		raw, err := webViews.ReadFile("views/" + name)
		if err != nil {
			return nil, err
		}
		files[name] = string(raw)
	}
	return files, nil
}

func (web *WebAuth) LoginPage(ctx *core.Context) error {
	if _, ok := web.claims(ctx); ok {
		return webRedirect(ctx, "/account")
	}
	return web.render(ctx, "login", webPage{Title: "Log in"})
}

func (web *WebAuth) RegisterPage(ctx *core.Context) error {
	if _, ok := web.claims(ctx); ok {
		return webRedirect(ctx, "/account")
	}
	return web.render(ctx, "register", webPage{Title: "Create account"})
}

func (web *WebAuth) Login(ctx *core.Context) error {
	page := webPage{Title: "Log in", Email: strings.TrimSpace(ctx.Input("email"))}
	if !web.validCSRF(ctx) {
		page.Message = "This form expired. Please try again."
		return web.render(ctx.Status(http.StatusForbidden), "login", page)
	}
	if err := ctx.Validate(map[string]string{"email": "required|email", "password": "required|min:8"}); err != nil {
		page.Errors = webValidationErrors(err)
		return web.render(ctx.Status(http.StatusUnprocessableEntity), "login", page)
	}
	result, err := web.login(page.Email, ctx.Input("password"))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			page.Message = "The email or password is incorrect."
			return web.render(ctx.Status(http.StatusUnauthorized), "login", page)
		}
		log.Printf("copytygo web login: %v", err)
		page.Message = "We couldn't log you in right now. Please try again."
		return web.render(ctx.Status(http.StatusServiceUnavailable), "login", page)
	}
	return web.signIn(ctx, result)
}

func (web *WebAuth) Register(ctx *core.Context) error {
	page := webPage{Title: "Create account", Name: strings.TrimSpace(ctx.Input("name")), Email: strings.TrimSpace(ctx.Input("email"))}
	if !web.validCSRF(ctx) {
		page.Message = "This form expired. Please try again."
		return web.render(ctx.Status(http.StatusForbidden), "register", page)
	}
	validationErr := ctx.Validate(map[string]string{"name": "required|min:3", "email": "required|email", "password": "required|min:8"})
	page.Errors = webValidationErrors(validationErr)
	if ctx.Input("password") != ctx.Input("password_confirmation") {
		page.Errors["password_confirmation"] = []string{"must match your password"}
	}
	if len(page.Errors) > 0 {
		return web.render(ctx.Status(http.StatusUnprocessableEntity), "register", page)
	}
	// Public registration always uses the configured role, never a submitted role.
	result, err := web.register(page.Name, page.Email, ctx.Input("password"), web.options.DefaultRole)
	if err != nil {
		if errors.Is(err, ErrEmailRegistered) {
			page.Errors["email"] = []string{"is already registered. Try logging in instead"}
			return web.render(ctx.Status(http.StatusConflict), "register", page)
		}
		log.Printf("copytygo web registration: %v", err)
		page.Message = "We couldn't create your account right now. Please try again."
		return web.render(ctx.Status(http.StatusServiceUnavailable), "register", page)
	}
	return web.signIn(ctx, result)
}

// Middleware protects browser routes and redirects guests to the login form.
// Bearer-token APIs continue to use auth.Middleware().
func (web *WebAuth) Middleware() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx *core.Context) error {
			claims, ok := web.claims(ctx)
			if !ok {
				web.manager(ctx).Forget(ctx)
				return webRedirect(ctx, "/login")
			}
			ctx.Set(ClaimsContextKey, claims)
			return next(ctx)
		}
	}
}

func (web *WebAuth) Account(ctx *core.Context) error {
	claims, ok := Claims(ctx)
	if !ok {
		return webRedirect(ctx, "/login")
	}
	return web.render(ctx, "account", webPage{Title: "Your account", Name: claims.Data["name"], Email: claims.Data["email"], Role: claims.Role})
}

func (web *WebAuth) Logout(ctx *core.Context) error {
	if !web.validCSRF(ctx) {
		claims, _ := Claims(ctx)
		return web.render(ctx.Status(http.StatusForbidden), "account", webPage{Title: "Your account", Name: claims.Data["name"], Email: claims.Data["email"], Role: claims.Role, Message: "This form expired. Please try again."})
	}
	web.manager(ctx).Forget(ctx)
	return webRedirect(ctx, "/login")
}

func (web *WebAuth) manager(ctx *core.Context) *session.Manager {
	manager := *web.sessions
	manager.Secure = manager.Secure || ctx.Request.TLS != nil
	return &manager
}

func (web *WebAuth) claims(ctx *core.Context) (security.Claims, bool) {
	token := web.manager(ctx).Get(ctx, "token")
	if token == "" {
		return security.Claims{}, false
	}
	claims, err := security.VerifyToken(config.Get("APP_KEY"), token)
	return claims, err == nil
}

func (web *WebAuth) validCSRF(ctx *core.Context) bool {
	if origin := ctx.Request.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || !strings.EqualFold(parsed.Host, ctx.Request.Host) {
			return false
		}
	}
	expected := web.manager(ctx).Get(ctx, "csrf")
	provided := ctx.Request.PostFormValue("_token")
	return expected != "" && subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func (web *WebAuth) signIn(ctx *core.Context, result Session) error {
	csrf, err := newWebCSRF()
	if err != nil {
		return err
	}
	if err := web.manager(ctx).Write(ctx, map[string]string{"token": result.Token, "csrf": csrf}); err != nil {
		return err
	}
	return webRedirect(ctx, "/account")
}

func (web *WebAuth) render(ctx *core.Context, name string, page webPage) error {
	ctx.Header("Cache-Control", "no-store")
	page.AppName = config.Get("APP_NAME", "CopyTyGo")
	page.CSRF = web.manager(ctx).Get(ctx, "csrf")
	if page.CSRF == "" {
		token, err := newWebCSRF()
		if err != nil {
			return err
		}
		if err := web.manager(ctx).Put(ctx, "csrf", token); err != nil {
			return err
		}
		page.CSRF = token
	}
	views, err := template.ParseFS(web.options.Views, "views/layout.html", "views/"+name+".html")
	if err != nil {
		return fmt.Errorf("copytygo auth views: %w", err)
	}
	var out strings.Builder
	if err := views.ExecuteTemplate(&out, "layout", page); err != nil {
		return err
	}
	return ctx.HTML(out.String())
}

func webValidationErrors(err error) map[string][]string {
	fields := make(map[string][]string)
	var validation *core.ValidationError
	if errors.As(err, &validation) {
		for field, messages := range validation.Fields {
			fields[field] = messages
		}
	}
	return fields
}

func newWebCSRF() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func webRedirect(ctx *core.Context, path string) error {
	ctx.Header("Cache-Control", "no-store")
	http.Redirect(ctx.Response, ctx.Request, path, http.StatusSeeOther)
	return nil
}
