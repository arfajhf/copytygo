package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arfajhf/copytygo/v4/core"
)

func jsonBrowserApp(t *testing.T) (*core.Application, *BrowserAuth, *memoryUsers) {
	t.Helper()
	t.Setenv("APP_KEY", "copytygo-json-browser-test-secret-0123456789")
	t.Setenv("APP_ENV", "local")
	browser := NewBrowserAuth("user")
	store := &memoryUsers{users: map[string]User{"7": {ID: "7", Name: "Alex User", Email: "alex@example.com", Role: "user"}, "8": {ID: "8", Name: "Second User", Email: "second@example.com", Role: "user"}}}
	browser.web.users = store
	browser.web.login = func(email, password string) (Session, error) {
		if email != "alex@example.com" || password != "test-password" {
			return Session{}, ErrInvalidCredentials
		}
		token, err := Issue("7", "admin", nil, time.Hour)
		return Session{User: store.users["7"], Token: token}, err
	}
	app := core.New()
	app.Get("/api/session", browser.Session)
	app.Post("/api/session/login", browser.Login).Middleware(browser.CSRF())
	app.Post("/api/session/register", browser.Register).Middleware(browser.CSRF())
	app.Post("/api/session/logout", browser.Logout).Middleware(browser.Middleware(), browser.CSRF())
	app.Get("/api/dashboard", browser.Dashboard).Middleware(browser.Middleware())
	app.Get("/api/users", browser.Users).Middleware(browser.Middleware(), browser.RequireAdmin())
	app.Get("/api/users/:id", browser.User).Middleware(browser.Middleware(), browser.RequireAdmin())
	app.Post("/api/users", browser.CreateUser).Middleware(browser.Middleware(), browser.RequireAdmin(), browser.CSRF())
	app.Put("/api/users/:id", browser.UpdateUser).Middleware(browser.Middleware(), browser.RequireAdmin(), browser.CSRF())
	app.Delete("/api/users/:id", browser.DeleteUser).Middleware(browser.Middleware(), browser.RequireAdmin(), browser.CSRF())
	Routes(app, "user")
	return app, browser, store
}
func jsonBrowserRequest(app *core.Application, method, path, body, csrf string, cookie *http.Cookie, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrf)
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
func browserCSRFFrom(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var result struct {
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.CSRF == "" {
		t.Fatal("missing CSRF token")
	}
	return result.CSRF
}

func TestJSONBrowserSessionAndRoleCRUD(t *testing.T) {
	app, _, store := jsonBrowserApp(t)
	guest := jsonBrowserRequest(app, "GET", "/api/session", "", "", nil, "")
	csrf := browserCSRFFrom(t, guest)
	cookie := guest.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || guest.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("unsafe guest session")
	}
	input := `{"email":"alex@example.com","password":"test-password"}`
	for _, origin := range []string{"http://evil.example.com", "null"} {
		if got := jsonBrowserRequest(app, "POST", "/api/session/login", input, csrf, cookie, origin); got.Code != 403 {
			t.Fatal("cross-origin login accepted")
		}
	}
	if got := jsonBrowserRequest(app, "POST", "/api/session/login", input, "", cookie, ""); got.Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	login := jsonBrowserRequest(app, "POST", "/api/session/login", input, csrf, cookie, "http://example.com")
	if login.Code != 200 {
		t.Fatalf("login %d %s", login.Code, login.Body.String())
	}
	newCSRF := browserCSRFFrom(t, login)
	authenticated := login.Result().Cookies()[0]
	if newCSRF == csrf || authenticated.Value == cookie.Value || strings.Contains(login.Body.String(), `"token"`) || strings.Contains(login.Body.String(), "test-password") {
		t.Fatal("unsafe sign-in result")
	}
	request := func(method, path, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		got := jsonBrowserRequest(app, method, path, body, newCSRF, authenticated, "")
		if got.Code != want {
			t.Fatalf("%s %s got %d want %d: %s", method, path, got.Code, want, got.Body.String())
		}
		return got
	}
	request("GET", "/api/auth/me", "", 401)
	dashboard := request("GET", "/api/dashboard", "", 200)
	if strings.Contains(dashboard.Body.String(), `"stats"`) || !strings.Contains(dashboard.Body.String(), `"role":"user"`) {
		t.Fatal("cookie role bypassed fresh database identity")
	}
	request("GET", "/api/users", "", 403)
	request("DELETE", "/api/users/8", "", 403)
	admin := store.users["7"]
	admin.Role = "admin"
	store.users["7"] = admin
	if !strings.Contains(request("GET", "/api/dashboard", "", 200).Body.String(), `"stats"`) {
		t.Fatal("promotion not reflected")
	}
	request("GET", "/api/users?search=Second", "", 200)
	request("GET", "/api/users/8", "", 200)
	payload := `{"name":"Managed User","email":"managed@example.com","role":"admin","password":"test-password","password_confirmation":"test-password"}`
	if got := jsonBrowserRequest(app, "POST", "/api/users", payload, csrf, authenticated, ""); got.Code != 403 {
		t.Fatal("stale CSRF accepted")
	}
	request("POST", "/api/users", strings.Replace(payload, `"admin"`, `"superadmin"`, 1), 422)
	request("POST", "/api/users", payload, 201)
	request("PUT", "/api/users/8", `{"name":"Updated User","email":"updated@example.com","role":"user","password":"","password_confirmation":""}`, 200)
	request("PUT", "/api/users/7", `{"name":"Alex User","email":"alex@example.com","role":"user"}`, 422)
	request("DELETE", "/api/users/7", "", 422)
	request("DELETE", "/api/users/8", "", 200)
	admin.Role = "user"
	store.users["7"] = admin
	request("POST", "/api/users", payload, 403)
	if store.saves != 2 || store.deletes != 1 {
		t.Fatal("unexpected persisted mutations")
	}
	logout := request("POST", "/api/session/logout", "", 200)
	if logout.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout did not clear cookie")
	}
	delete(store.users, "7")
	request("GET", "/api/dashboard", "", 401)
	deleted := request("GET", "/api/session", "", 200)
	if !strings.Contains(deleted.Body.String(), `"user":null`) || len(deleted.Result().Cookies()) == 0 {
		t.Fatal("deleted identity was not replaced by guest session")
	}
}
func TestJSONPublicRegistrationCannotChooseAdmin(t *testing.T) {
	app, browser, _ := jsonBrowserApp(t)
	guest := jsonBrowserRequest(app, "GET", "/api/session", "", "", nil, "")
	csrf := browserCSRFFrom(t, guest)
	cookie := guest.Result().Cookies()[0]
	called := false
	browser.web.register = func(name, email, password, role string) (Session, error) {
		called = true
		if role != "user" {
			t.Fatal("public role escalation")
		}
		token, err := Issue("7", role, nil, time.Hour)
		return Session{User: User{ID: "7", Name: name, Email: email, Role: role}, Token: token}, err
	}
	payload := `{"name":"Alex User","email":"alex@example.com","password":"test-password","password_confirmation":"wrong","role":"admin"}`
	got := jsonBrowserRequest(app, "POST", "/api/session/register", payload, csrf, cookie, "")
	if got.Code != 422 || called || strings.Contains(got.Body.String(), "test-password") {
		t.Fatal("invalid registration reached persistence or reflected password")
	}
	payload = strings.Replace(payload, `"wrong"`, `"test-password"`, 1)
	got = jsonBrowserRequest(app, "POST", "/api/session/register", payload, csrf, cookie, "")
	if got.Code != 201 || !called || !strings.Contains(got.Body.String(), `"role":"user"`) {
		t.Fatalf("registration failed %s", got.Body.String())
	}
}
