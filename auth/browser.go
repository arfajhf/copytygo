package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/core"
)

// BrowserAuth exposes JSON endpoints for the TypeScript frontend. Authentication
// lives in an encrypted HttpOnly cookie; tokens are never returned to JavaScript.
type BrowserAuth struct{ web *WebAuth }

func NewBrowserAuth(defaultRole string) *BrowserAuth {
	return &BrowserAuth{web: NewWebAuth(WebOptions{DefaultRole: defaultRole})}
}

func (browser *BrowserAuth) Middleware() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx *core.Context) error {
			ctx.Header("Cache-Control", "no-store")
			claims, ok := browser.web.claims(ctx)
			if !ok {
				return core.NewHTTPError(401, "Please log in to continue.")
			}
			requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
			defer cancel()
			user, err := browser.web.users.Find(requestContext, claims.Subject)
			if errors.Is(err, errUserMissing) {
				browser.web.manager(ctx).Forget(ctx)
				return core.NewHTTPError(401, "Please log in to continue.")
			}
			if err != nil {
				return browser.failure(err)
			}
			claims.Role = user.Role
			claims.Data = map[string]string{"name": user.Name, "email": user.Email}
			ctx.Set(ClaimsContextKey, claims)
			return next(ctx)
		}
	}
}

func (browser *BrowserAuth) RequireAdmin() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx *core.Context) error {
			if !browser.web.isAdmin(ctx) {
				return core.NewHTTPError(403, errAdminOnly.Error())
			}
			return next(ctx)
		}
	}
}

func (browser *BrowserAuth) CSRF() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx *core.Context) error {
			if origin := ctx.Request.Header.Get("Origin"); origin != "" {
				parsed, err := url.Parse(origin)
				if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || !strings.EqualFold(parsed.Host, ctx.Request.Host) {
					return core.NewHTTPError(403, "This request expired. Reload the page and try again.")
				}
			}
			expected := browser.web.manager(ctx).Get(ctx, "csrf")
			supplied := ctx.Request.Header.Get("X-CSRF-Token")
			if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(supplied)) != 1 {
				return core.NewHTTPError(403, "This request expired. Reload the page and try again.")
			}
			ctx.Header("Cache-Control", "no-store")
			return next(ctx)
		}
	}
}

// Session also serves guests so login/register can obtain a CSRF token safely.
func (browser *BrowserAuth) Session(ctx *core.Context) error {
	var user *User
	if claims, ok := browser.web.claims(ctx); ok {
		requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
		defer cancel()
		found, err := browser.web.users.Find(requestContext, claims.Subject)
		if err != nil && !errors.Is(err, errUserMissing) {
			return browser.failure(err)
		}
		if err == nil {
			user = &found
		} else {
			// Discard the deleted identity, then issue a fresh guest session.
			if err := browser.web.manager(ctx).Write(ctx, map[string]string{}); err != nil {
				return err
			}
		}
	}
	csrf := browser.web.manager(ctx).Get(ctx, "csrf")
	if csrf == "" {
		var err error
		csrf, err = newWebCSRF()
		if err != nil {
			return err
		}
		if err := browser.web.manager(ctx).Write(ctx, map[string]string{"csrf": csrf}); err != nil {
			return err
		}
	}
	ctx.Header("Cache-Control", "no-store")
	return ctx.JSON(core.Map{"appName": config.Get("APP_NAME", "CopyTyGo"), "multi": browser.web.options.DefaultRole != "", "csrf": csrf, "user": user})
}

func (browser *BrowserAuth) Login(ctx *core.Context) error {
	if err := ctx.Validate(map[string]string{"email": "required|email|max:255", "password": "required|min:8|max:72"}); err != nil {
		return err
	}
	result, err := browser.web.login(strings.TrimSpace(ctx.Input("email")), ctx.Input("password"))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return core.NewHTTPError(401, "The email or password is incorrect.")
		}
		return browser.failure(err)
	}
	return browser.signIn(ctx, result, 200)
}

func (browser *BrowserAuth) Register(ctx *core.Context) error {
	fields := webValidationErrors(ctx.Validate(map[string]string{"name": "required|min:3|max:255", "email": "required|email|max:255", "password": "required|min:8|max:72"}))
	if ctx.Input("password") != ctx.Input("password_confirmation") {
		fields["password_confirmation"] = []string{"must match your password"}
	}
	if len(fields) != 0 {
		return &core.ValidationError{Fields: fields}
	}
	// Public clients cannot choose their role.
	result, err := browser.web.register(strings.TrimSpace(ctx.Input("name")), strings.TrimSpace(ctx.Input("email")), ctx.Input("password"), browser.web.options.DefaultRole)
	if err != nil {
		return browser.failure(err)
	}
	return browser.signIn(ctx, result, 201)
}

func (browser *BrowserAuth) signIn(ctx *core.Context, result Session, status int) error {
	csrf, err := newWebCSRF()
	if err != nil {
		return err
	}
	if err := browser.web.manager(ctx).Write(ctx, map[string]string{"token": result.Token, "csrf": csrf}); err != nil {
		return err
	}
	return ctx.Status(status).JSON(core.Map{"user": result.User, "csrf": csrf})
}

func (browser *BrowserAuth) Logout(ctx *core.Context) error {
	browser.web.manager(ctx).Forget(ctx)
	return ctx.JSON(core.Map{"ok": true})
}

func (browser *BrowserAuth) Dashboard(ctx *core.Context) error {
	result := core.Map{"user": browser.currentUser(ctx)}
	if browser.web.isAdmin(ctx) {
		requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
		defer cancel()
		stats, err := browser.web.users.Stats(requestContext)
		if err != nil {
			return browser.failure(err)
		}
		result["stats"] = core.Map{"total": stats.Total, "admins": stats.Admins, "members": stats.Members}
	}
	return ctx.JSON(result)
}

func (browser *BrowserAuth) currentUser(ctx *core.Context) User {
	claims, _ := Claims(ctx)
	return User{ID: claims.Subject, Name: claims.Data["name"], Email: claims.Data["email"], Role: claims.Role}
}

func (browser *BrowserAuth) Users(ctx *core.Context) error {
	if !browser.web.isAdmin(ctx) {
		return core.NewHTTPError(403, errAdminOnly.Error())
	}
	page, _ := strconv.Atoi(ctx.Query("page"))
	if page < 1 || page > 100000 {
		page = 1
	}
	search := strings.TrimSpace(ctx.Query("search"))
	if len(search) > 200 {
		search = search[:200]
	}
	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	users, total, err := browser.web.users.List(requestContext, search, page)
	if err != nil {
		return browser.failure(err)
	}
	return ctx.JSON(core.Map{"users": users, "total": total, "page": page, "pageSize": 10})
}

func (browser *BrowserAuth) User(ctx *core.Context) error {
	if !browser.web.isAdmin(ctx) {
		return core.NewHTTPError(403, errAdminOnly.Error())
	}
	id, err := webUserID(ctx)
	if err != nil {
		return browser.failure(err)
	}
	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	user, err := browser.web.users.Find(requestContext, id)
	if err != nil {
		return browser.failure(err)
	}
	return ctx.JSON(user)
}
func (browser *BrowserAuth) CreateUser(ctx *core.Context) error { return browser.saveUser(ctx, false) }
func (browser *BrowserAuth) UpdateUser(ctx *core.Context) error { return browser.saveUser(ctx, true) }
func (browser *BrowserAuth) saveUser(ctx *core.Context, editing bool) error {
	if !browser.web.isAdmin(ctx) {
		return core.NewHTTPError(403, errAdminOnly.Error())
	}
	id := ""
	if editing {
		var err error
		id, err = webUserID(ctx)
		if err != nil {
			return browser.failure(err)
		}
	}
	rules := map[string]string{"name": "required|min:3|max:255", "email": "required|email|max:255", "role": "required|oneof:user,admin"}
	if !editing || ctx.Input("password") != "" {
		rules["password"] = "required|min:8|max:72"
	}
	fields := webValidationErrors(ctx.Validate(rules))
	if ctx.Input("password") != ctx.Input("password_confirmation") {
		fields["password_confirmation"] = []string{"must match your password"}
	}
	// Explicitly enforce the two supported roles independently of validation configuration.
	if role := ctx.Input("role"); role != "user" && role != "admin" {
		fields["role"] = []string{"must be user or admin"}
	}
	if len(fields) != 0 {
		return &core.ValidationError{Fields: fields}
	}
	input := userInput{Name: ctx.Input("name"), Email: ctx.Input("email"), Role: ctx.Input("role"), Password: ctx.Input("password")}
	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	if err := browser.web.users.Save(requestContext, UserID(ctx), id, input); err != nil {
		return browser.failure(err)
	}
	status := 200
	if !editing {
		status = 201
	}
	return ctx.Status(status).JSON(core.Map{"ok": true})
}
func (browser *BrowserAuth) DeleteUser(ctx *core.Context) error {
	if !browser.web.isAdmin(ctx) {
		return core.NewHTTPError(403, errAdminOnly.Error())
	}
	id, err := webUserID(ctx)
	if err != nil {
		return browser.failure(err)
	}
	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	if err := browser.web.users.Delete(requestContext, UserID(ctx), id); err != nil {
		return browser.failure(err)
	}
	return ctx.JSON(core.Map{"ok": true})
}
func (browser *BrowserAuth) failure(err error) error {
	switch {
	case errors.Is(err, ErrEmailRegistered):
		return core.NewHTTPError(409, "Email is already registered.")
	case errors.Is(err, ErrInvalidCredentials):
		return core.NewHTTPError(401, "The email or password is incorrect.")
	case errors.Is(err, ErrInvalidRegistration):
		return core.NewHTTPError(422, "Please check your account details.")
	case errors.Is(err, errUserMissing):
		return core.NewHTTPError(404, err.Error())
	case errors.Is(err, errAdminOnly):
		return core.NewHTTPError(403, err.Error())
	case errors.Is(err, errOwnAdmin):
		return core.NewHTTPError(422, err.Error())
	default:
		log.Printf("copytygo browser API: %v", err)
		return core.NewHTTPError(http.StatusServiceUnavailable, "We couldn't complete this request. Check the database connection and migrations, then try again.")
	}
}
