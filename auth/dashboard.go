package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/arfajhf/copytygo/v4/core"
)

// RequireAdmin follows Middleware. Handlers also check access themselves.
func (web *WebAuth) RequireAdmin() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx *core.Context) error {
			if !web.isAdmin(ctx) {
				return web.adminDenied(ctx)
			}
			return next(ctx)
		}
	}
}
func (web *WebAuth) adminDenied(ctx *core.Context) error {
	return web.render(ctx.Status(http.StatusForbidden), "error", webPage{Title: "Access denied", Message: "Administrator access is required."})
}
func (web *WebAuth) isAdmin(ctx *core.Context) bool {
	claims, ok := Claims(ctx)
	return ok && web.options.DefaultRole != "" && claims.Role == "admin"
}
func (web *WebAuth) Dashboard(ctx *core.Context) error {
	if _, ok := Claims(ctx); !ok {
		return webRedirect(ctx, "/login")
	}
	page := webPage{Title: "Dashboard"}
	if web.isAdmin(ctx) {
		requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
		defer cancel()
		stats, err := web.users.Stats(requestContext)
		if err != nil {
			return web.failure(ctx, err)
		}
		page.Stats = stats
	}
	return web.render(ctx, "dashboard", page)
}
func (web *WebAuth) Users(ctx *core.Context) error {
	if !web.isAdmin(ctx) {
		return web.adminDenied(ctx)
	}
	pageNumber, _ := strconv.Atoi(ctx.Query("page"))
	if pageNumber < 1 || pageNumber > 100000 {
		pageNumber = 1
	}
	search := strings.TrimSpace(ctx.Query("search"))
	if len(search) > 200 {
		search = search[:200]
	}
	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	users, total, err := web.users.List(requestContext, search, pageNumber)
	if err != nil {
		return web.failure(ctx, err)
	}
	page := webPage{Title: "Users", Users: users, Total: total, Search: search}
	pageURL := func(n int) string {
		return "/users?" + url.Values{"page": {strconv.Itoa(n)}, "search": {search}}.Encode()
	}
	if pageNumber > 1 {
		page.PreviousURL = pageURL(pageNumber - 1)
	}
	if int64(pageNumber*10) < total {
		page.NextURL = pageURL(pageNumber + 1)
	}
	switch ctx.Query("saved") {
	case "created":
		page.Success = "User created successfully."
	case "updated":
		page.Success = "User updated successfully."
	case "deleted":
		page.Success = "User deleted successfully."
	}
	return web.render(ctx, "users", page)
}
func (web *WebAuth) UserCreatePage(ctx *core.Context) error {
	if !web.isAdmin(ctx) {
		return web.adminDenied(ctx)
	}
	return web.render(ctx, "user-form", webPage{Title: "Add user", Editing: User{Role: "user"}, FormAction: "/users"})
}
func (web *WebAuth) UserEditPage(ctx *core.Context) error {
	if !web.isAdmin(ctx) {
		return web.adminDenied(ctx)
	}
	id, err := webUserID(ctx)
	if err != nil {
		return web.failure(ctx, err)
	}
	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	user, err := web.users.Find(requestContext, id)
	if err != nil {
		return web.failure(ctx, err)
	}
	return web.render(ctx, "user-form", webPage{Title: "Edit user", Editing: user, FormAction: "/users/" + id})
}
func webUserID(ctx *core.Context) (string, error) {
	id := ctx.Param("id")
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil || n < 1 {
		return "", errUserMissing
	}
	return strconv.FormatInt(n, 10), nil
}
func (web *WebAuth) UserCreate(ctx *core.Context) error { return web.saveUser(ctx, false) }
func (web *WebAuth) UserUpdate(ctx *core.Context) error { return web.saveUser(ctx, true) }
func (web *WebAuth) saveUser(ctx *core.Context, editing bool) error {
	if !web.isAdmin(ctx) {
		return web.adminDenied(ctx)
	}
	input := userInput{Name: strings.TrimSpace(ctx.Input("name")), Email: strings.TrimSpace(ctx.Input("email")), Role: ctx.Input("role"), Password: ctx.Input("password")}
	page := webPage{Title: "Add user", Editing: User{Name: input.Name, Email: input.Email, Role: input.Role}, FormAction: "/users"}
	if editing {
		id, err := webUserID(ctx)
		if err != nil {
			return web.failure(ctx, err)
		}
		page.Editing.ID = id
		page.Title = "Edit user"
		page.FormAction += "/" + id
	}
	if !web.validCSRF(ctx) {
		page.Message = "This form expired. Please try again."
		return web.render(ctx.Status(http.StatusForbidden), "user-form", page)
	}
	rules := map[string]string{"name": "required|min:3|max:255", "email": "required|email|max:255"}
	if !editing || input.Password != "" {
		rules["password"] = "required|min:8"
	}
	page.Errors = webValidationErrors(ctx.Validate(rules))
	if input.Role != "user" && input.Role != "admin" {
		page.Errors["role"] = []string{"must be user or admin"}
	}
	if input.Password != "" && input.Password != ctx.Input("password_confirmation") {
		page.Errors["password_confirmation"] = []string{"must match your password"}
	}
	if len(page.Errors) > 0 {
		return web.render(ctx.Status(http.StatusUnprocessableEntity), "user-form", page)
	}
	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	err := web.users.Save(requestContext, UserID(ctx), page.Editing.ID, input)
	if errors.Is(err, ErrEmailRegistered) {
		page.Errors["email"] = []string{"is already registered"}
		return web.render(ctx.Status(http.StatusConflict), "user-form", page)
	}
	if errors.Is(err, errOwnAdmin) {
		page.Message = err.Error()
		return web.render(ctx.Status(http.StatusUnprocessableEntity), "user-form", page)
	}
	if err != nil {
		return web.failure(ctx, err)
	}
	saved := "created"
	if editing {
		saved = "updated"
	}
	return webRedirect(ctx, "/users?saved="+saved)
}
func (web *WebAuth) UserDelete(ctx *core.Context) error {
	if !web.isAdmin(ctx) {
		return web.adminDenied(ctx)
	}
	if !web.validCSRF(ctx) {
		return web.render(ctx.Status(http.StatusForbidden), "error", webPage{Title: "Form expired", Message: "This form expired. Please try again."})
	}
	id, err := webUserID(ctx)
	if err != nil {
		return web.failure(ctx, err)
	}
	requestContext, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	if err := web.users.Delete(requestContext, UserID(ctx), id); err != nil {
		return web.failure(ctx, err)
	}
	return webRedirect(ctx, "/users?saved=deleted")
}
func (web *WebAuth) failure(ctx *core.Context, err error) error {
	status, message := http.StatusServiceUnavailable, "We couldn't load your account right now. Please try again."
	switch {
	case errors.Is(err, errUserMissing):
		status, message = http.StatusNotFound, err.Error()
	case errors.Is(err, errAdminOnly):
		status, message = http.StatusForbidden, err.Error()
	case errors.Is(err, errOwnAdmin):
		status, message = http.StatusUnprocessableEntity, err.Error()
	default:
		log.Printf("copytygo browser account: %v", err)
	}
	return web.render(ctx.Status(status), "error", webPage{Title: "Request could not be completed", Message: message})
}
