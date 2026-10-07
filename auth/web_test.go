package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/arfajhf/copytygo/v4/core"
	"github.com/arfajhf/copytygo/v4/security"
)

var csrfField = regexp.MustCompile(`name="_token" value="([^"]+)"`)

func browserTestApp(t *testing.T) (*core.Application, *WebAuth) {
	t.Helper()
	t.Setenv("APP_KEY", "copytygo-browser-test-secret-0123456789")
	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_NAME", "Example App")
	pages := NewWebAuth(WebOptions{DefaultRole: "user"})
	app := core.New()
	app.Get("/", core.Welcome())
	app.Get("/login", pages.LoginPage)
	app.Post("/login", pages.Login)
	app.Get("/register", pages.RegisterPage)
	app.Post("/register", pages.Register)
	app.Get("/account", pages.Account).Middleware(pages.Middleware())
	app.Post("/logout", pages.Logout).Middleware(pages.Middleware())
	Routes(app, "user")
	return app, pages
}

func webRequest(app *core.Application, method, path string, data url.Values, cookie *http.Cookie, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(data.Encode()))
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func webForm(t *testing.T, app *core.Application, path string) (string, *http.Cookie) {
	t.Helper()
	rec := webRequest(app, "GET", path, nil, nil, "")
	if rec.Code != 200 {
		t.Fatalf("form failed: %d %s", rec.Code, rec.Body.String())
	}
	matches := csrfField.FindStringSubmatch(rec.Body.String())
	cookies := rec.Result().Cookies()
	if len(matches) != 2 || len(cookies) != 1 {
		t.Fatal("form did not create a CSRF session")
	}
	if !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("unsafe browser session or cache policy")
	}
	return matches[1], cookies[0]
}

func TestBrowserRegistrationAccountAndLogout(t *testing.T) {
	app, pages := browserTestApp(t)
	token, cookie := webForm(t, app, "/register")
	password := "correct-password"
	pages.register = func(name, email, suppliedPassword, role string) (Session, error) {
		if name != "Alex User" || email != "alex@example.com" || suppliedPassword != password || role != "user" {
			t.Fatalf("unexpected registration: %s %s %s", name, email, role)
		}
		bearer, err := Issue("7", role, map[string]string{"name": name, "email": email}, time.Hour)
		return Session{User: User{ID: "7", Name: name, Email: email, Role: role}, Token: bearer}, err
	}
	rec := webRequest(app, "POST", "/register", url.Values{"_token": {token}, "name": {"Alex User"}, "email": {"alex@example.com"}, "password": {password}, "password_confirmation": {password}, "role": {"admin"}}, cookie, "http://example.com")
	if rec.Code != 303 || rec.Header().Get("Location") != "/account" {
		t.Fatalf("registration failed: %d %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value == cookie.Value || strings.Contains(cookies[0].Value, password) {
		t.Fatal("sign-in did not rotate encrypted session")
	}
	authenticated := cookies[0]
	account := webRequest(app, "GET", "/account", nil, authenticated, "")
	if account.Code != 200 || !strings.Contains(account.Body.String(), "Alex User") || !strings.Contains(account.Body.String(), "alex@example.com") {
		t.Fatal("signed-in account missing")
	}
	newToken := csrfField.FindStringSubmatch(account.Body.String())[1]
	if newToken == token {
		t.Fatal("sign-in did not rotate CSRF")
	}
	api := webRequest(app, "GET", "/api/auth/me", nil, authenticated, "")
	if api.Code != 401 {
		t.Fatal("browser cookie unexpectedly authenticated bearer API")
	}
	if got := webRequest(app, "GET", "/login", nil, authenticated, ""); got.Code != 303 || got.Header().Get("Location") != "/account" {
		t.Fatal("signed-in visitor saw login form")
	}
	stale := webRequest(app, "POST", "/logout", url.Values{"_token": {token}}, authenticated, "http://example.com")
	if stale.Code != 403 {
		t.Fatal("old CSRF token remained valid after sign-in")
	}
	logout := webRequest(app, "POST", "/logout", url.Values{"_token": {newToken}}, authenticated, "http://example.com")
	if logout.Code != 303 || logout.Header().Get("Location") != "/login" || logout.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout did not clear the session")
	}
	guest := webRequest(app, "GET", "/account", nil, nil, "")
	if guest.Code != 303 || guest.Header().Get("Location") != "/login" {
		t.Fatal("guest account access was not redirected")
	}
}

func TestBrowserValidationAndCredentialErrors(t *testing.T) {
	app, pages := browserTestApp(t)
	pages.login = func(string, string) (Session, error) { return Session{}, ErrInvalidCredentials }
	pages.register = func(string, string, string, string) (Session, error) { return Session{}, ErrEmailRegistered }
	token, cookie := webForm(t, app, "/register")
	invalid := url.Values{"_token": {token}, "name": {"<script>alert(1)</script>"}, "email": {"invalid"}, "password": {"secret-one"}, "password_confirmation": {"secret-two"}}
	rec := webRequest(app, "POST", "/register", invalid, cookie, "")
	body := rec.Body.String()
	if rec.Code != 422 || !strings.Contains(body, "must match your password") || !strings.Contains(body, "must be a valid email") {
		t.Fatalf("missing HTML validation: %d %s", rec.Code, body)
	}
	if strings.Contains(body, "<script>alert(1)</script>") || strings.Contains(body, "secret-one") || strings.Contains(body, "secret-two") {
		t.Fatal("form reflected executable markup or a password")
	}
	duplicate := webRequest(app, "POST", "/register", url.Values{"_token": {token}, "name": {"Alex User"}, "email": {"alex@example.com"}, "password": {"secret-one"}, "password_confirmation": {"secret-one"}}, cookie, "")
	if duplicate.Code != 409 || !strings.Contains(duplicate.Body.String(), "already registered") {
		t.Fatal("duplicate registration missing clear message")
	}
	login := webRequest(app, "POST", "/login", url.Values{"_token": {token}, "email": {"alex@example.com"}, "password": {"secret-one"}}, cookie, "")
	if login.Code != 401 || !strings.Contains(login.Body.String(), "email or password is incorrect") {
		t.Fatal("invalid login missing clear message")
	}
}

func TestBrowserRejectsMissingCrossSiteAndForgedCSRF(t *testing.T) {
	app, pages := browserTestApp(t)
	pages.login = func(string, string) (Session, error) {
		t.Fatal("CSRF failure reached authentication")
		return Session{}, nil
	}
	token, cookie := webForm(t, app, "/login")
	for _, test := range []struct {
		token, origin string
		cookie        *http.Cookie
	}{
		{"", "", cookie},
		{token, "http://attacker.example", cookie},
		{token, "null", cookie},
		{"wrong-token", "", cookie},
		{token, "", &http.Cookie{Name: cookie.Name, Value: cookie.Value + "tampered"}},
	} {
		rec := webRequest(app, "POST", "/login", url.Values{"_token": {test.token}, "email": {"alex@example.com"}, "password": {"secret-one"}}, test.cookie, test.origin)
		if rec.Code != 403 || !strings.Contains(rec.Body.String(), "form expired") {
			t.Fatal("unsafe form submission was not rejected")
		}
	}
}

func TestBrowserSessionProductionAndExpiration(t *testing.T) {
	app, pages := browserTestApp(t)
	expired, err := security.SignToken("copytygo-browser-test-secret-0123456789", security.Claims{Subject: "7", ExpiresAt: time.Now().Add(-time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	pages.login = func(string, string) (Session, error) { return Session{Token: expired}, nil }
	token, cookie := webForm(t, app, "/login")
	result := webRequest(app, "POST", "/login", url.Values{"_token": {token}, "email": {"alex@example.com"}, "password": {"secret-one"}}, cookie, "")
	account := webRequest(app, "GET", "/account", nil, result.Result().Cookies()[0], "")
	if account.Code != 303 || account.Header().Get("Location") != "/login" {
		t.Fatal("expired session remained authenticated")
	}
	t.Setenv("APP_ENV", "production")
	production := NewWebAuth(WebOptions{})
	prodApp := core.New()
	prodApp.Get("/login", production.LoginPage)
	_, secureCookie := webForm(t, prodApp, "/login")
	if !secureCookie.Secure || !strings.HasPrefix(secureCookie.Name, "__Host-") {
		t.Fatal("production cookie lacks secure host scope")
	}
}

func TestBrowserDatabaseFailureShowsSafeHTML(t *testing.T) {
	app, pages := browserTestApp(t)
	pages.login = func(string, string) (Session, error) { return Session{}, errors.New("private database details") }
	token, cookie := webForm(t, app, "/login")
	result := webRequest(app, "POST", "/login", url.Values{"_token": {token}, "email": {"alex@example.com"}, "password": {"secret-one"}}, cookie, "")
	if result.Code != 503 || strings.Contains(result.Body.String(), "private database details") || !strings.Contains(result.Body.String(), "Please try again") {
		t.Fatal("backend error leaked into browser page")
	}
}
