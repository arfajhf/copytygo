package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/arfajhf/copytygo/v4/auth"
	"github.com/arfajhf/copytygo/v4/core"
	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/database/query"
	"github.com/arfajhf/copytygo/v4/security"
)

func TestBrowserAuthDatabaseIntegration(t *testing.T) {
	if os.Getenv("COPYTYGO_INTEGRATION_DB") != "1" {
		t.Skip("set COPYTYGO_INTEGRATION_DB=1 to run database integration tests")
	}
	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_KEY", "browser-auth-integration-key-0123456789")
	drivers.Register()
	db, err := database.Connect()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	driver := database.LoadConfig().Driver
	identity := "BIGINT AUTO_INCREMENT"
	if driver == "postgres" {
		identity = "BIGSERIAL"
	}
	ddl := fmt.Sprintf("CREATE TABLE IF NOT EXISTS users (id %s PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, password VARCHAR(512) NOT NULL, role VARCHAR(50) NOT NULL DEFAULT 'user')", identity)
	if _, err := db.Exec(ddl); err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("browser-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() { _, _ = query.Table(db, driver, "users").WhereEq("email", email).Delete(context.Background()) })
	pages := auth.NewWebAuth(auth.WebOptions{DefaultRole: "user"})
	app := core.New()
	app.Get("/", core.Welcome())
	app.Get("/login", pages.LoginPage)
	app.Post("/login", pages.Login)
	app.Get("/register", pages.RegisterPage)
	app.Post("/register", pages.Register)
	app.Get("/dashboard", pages.Dashboard).Middleware(pages.Middleware())
	app.Get("/users", pages.Users).Middleware(pages.Middleware(), pages.RequireAdmin())
	app.Get("/users/create", pages.UserCreatePage).Middleware(pages.Middleware(), pages.RequireAdmin())
	app.Post("/users", pages.UserCreate).Middleware(pages.Middleware(), pages.RequireAdmin())
	app.Get("/users/:id/edit", pages.UserEditPage).Middleware(pages.Middleware(), pages.RequireAdmin())
	app.Post("/users/:id", pages.UserUpdate).Middleware(pages.Middleware(), pages.RequireAdmin())
	app.Post("/users/:id/delete", pages.UserDelete).Middleware(pages.Middleware(), pages.RequireAdmin())
	app.Get("/account", pages.Account).Middleware(pages.Middleware())
	app.Post("/logout", pages.Logout).Middleware(pages.Middleware())
	auth.Routes(app, "user")
	server := httptest.NewServer(app)
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request := func(method, path string, data url.Values, expected int) (string, *http.Response) {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(data.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		if method == "POST" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Origin", server.URL)
		}
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		raw, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != expected {
			t.Fatalf("%s %s expected %d got %d: %s", method, path, expected, response.StatusCode, raw)
		}
		return string(raw), response
	}
	csrf := func(body string) string {
		t.Helper()
		match := regexp.MustCompile(`name="_token" value="([^"]+)"`).FindStringSubmatch(body)
		if len(match) != 2 {
			t.Fatal("CSRF input missing")
		}
		return match[1]
	}
	welcome, _ := request("GET", "/", nil, 200)
	if !strings.Contains(welcome, `href="/login"`) || !strings.Contains(welcome, `href="/register"`) {
		t.Fatal("welcome auth buttons missing")
	}
	register, _ := request("GET", "/register", nil, 200)
	password := "a-valid-test-password"
	_, created := request("POST", "/register", url.Values{"_token": {csrf(register)}, "name": {"Browser User"}, "email": {email}, "password": {password}, "password_confirmation": {password}, "role": {"admin"}}, 303)
	if created.Header.Get("Location") != "/dashboard" {
		t.Fatal("registration did not sign in")
	}
	row, found, err := query.Table(db, driver, "users").WhereEq("email", email).FirstMap(context.Background())
	if err != nil || !found {
		t.Fatalf("registered user missing: %v", err)
	}
	if fmt.Sprint(row["role"]) != "user" {
		t.Fatal("public registration escalated role")
	}
	if !security.VerifyPassword(password, fmt.Sprint(row["password"])) {
		t.Fatal("stored password hash invalid")
	}
	account, _ := request("GET", "/account", nil, 200)
	if !strings.Contains(account, "Browser User") || !strings.Contains(account, email) {
		t.Fatal("account profile missing")
	}
	dashboard, _ := request("GET", "/dashboard", nil, 200)
	if strings.Contains(dashboard, `href="/users"`) {
		t.Fatal("member saw administrator menu")
	}
	request("GET", "/users", nil, 403)
	request("GET", "/users/create", nil, 403)
	request("POST", "/users", url.Values{"_token": {csrf(account)}, "role": {"admin"}}, 403)
	if err := auth.PromoteAdmin(context.Background(), email); err != nil {
		t.Fatal(err)
	}
	// A promotion must become visible even with the original user-role cookie.
	dashboard, _ = request("GET", "/dashboard", nil, 200)
	if !strings.Contains(dashboard, `href="/users"`) || !strings.Contains(dashboard, "Total users") {
		t.Fatal("admin dashboard missing")
	}
	users, _ := request("GET", "/users", nil, 200)
	userEmail := fmt.Sprintf("managed-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = query.Table(db, driver, "users").WhereEq("email", userEmail).Delete(context.Background())
	})
	input := url.Values{"_token": {csrf(users)}, "name": {"Managed User"}, "email": {userEmail}, "password": {"managed-password"}, "password_confirmation": {"managed-password"}, "role": {"user"}}
	missingToken := url.Values{"name": {"Managed User"}, "email": {userEmail}, "password": {"managed-password"}, "password_confirmation": {"managed-password"}, "role": {"user"}}
	request("POST", "/users", missingToken, 403)
	request("POST", "/users", input, 303)
	managed, found, err := query.Table(db, driver, "users").WhereEq("email", userEmail).FirstMap(context.Background())
	if err != nil || !found {
		t.Fatalf("admin creation failed: %v", err)
	}
	id := fmt.Sprint(managed["id"])
	if !security.VerifyPassword("managed-password", fmt.Sprint(managed["password"])) {
		t.Fatal("managed password not hashed")
	}
	request("POST", "/users", input, 409)
	edit, _ := request("GET", "/users/"+id+"/edit", nil, 200)
	if strings.Contains(edit, fmt.Sprint(managed["password"])) || strings.Contains(edit, "managed-password") {
		t.Fatal("edit page exposed password")
	}
	input.Set("_token", csrf(edit))
	input.Set("name", "Updated User")
	input.Set("role", "admin")
	input.Set("password", "")
	input.Set("password_confirmation", "")
	request("POST", "/users/"+id, input, 303)
	updated, _, err := query.Table(db, driver, "users").WhereEq("id", id).FirstMap(context.Background())
	if err != nil || fmt.Sprint(updated["role"]) != "admin" || fmt.Sprint(updated["password"]) != fmt.Sprint(managed["password"]) {
		t.Fatal("update failed or blank password replaced hash")
	}
	input.Set("password", "replacement-password")
	input.Set("password_confirmation", "replacement-password")
	request("POST", "/users/"+id, input, 303)
	replacement, _, _ := query.Table(db, driver, "users").WhereEq("id", id).FirstMap(context.Background())
	if !security.VerifyPassword("replacement-password", fmt.Sprint(replacement["password"])) {
		t.Fatal("password update was not hashed")
	}
	ownID := fmt.Sprint(row["id"])
	request("POST", "/users/"+ownID+"/delete", url.Values{"_token": {csrf(users)}}, 422)
	ownInput := url.Values{"_token": {csrf(users)}, "name": {"Browser User"}, "email": {email}, "role": {"user"}}
	request("POST", "/users/"+ownID, ownInput, 422)
	results, _ := request("GET", "/users?search="+url.QueryEscape(userEmail), nil, 200)
	if !strings.Contains(results, "Updated User") {
		t.Fatal("search did not find updated user")
	}
	request("POST", "/users/"+id+"/delete", url.Values{"_token": {"wrong"}}, 403)
	request("POST", "/users/"+id+"/delete", url.Values{"_token": {csrf(users)}}, 303)
	_, found, err = query.Table(db, driver, "users").WhereEq("id", id).FirstMap(context.Background())
	if err != nil || found {
		t.Fatal("deleted user remained")
	}
	request("GET", "/users/"+id+"/edit", nil, 404)
	// Revoked roles must invalidate access immediately, even with an admin cookie.
	_, err = query.Table(db, driver, "users").WhereEq("email", email).Update(context.Background(), map[string]any{"role": "user"})
	if err != nil {
		t.Fatal(err)
	}
	request("GET", "/users", nil, 403)
	request("GET", "/api/auth/me", nil, 401)
	request("POST", "/logout", url.Values{"_token": {csrf(account)}}, 303)
	request("GET", "/account", nil, 303)
	login, _ := request("GET", "/login", nil, 200)
	request("POST", "/login", url.Values{"_token": {csrf(login)}, "email": {email}, "password": {"wrong-password"}}, 401)
	request("POST", "/login", url.Values{"_token": {csrf(login)}, "email": {email}, "password": {password}}, 303)
	account, _ = request("GET", "/account", nil, 200)
	request("POST", "/logout", url.Values{"_token": {csrf(account)}}, 303)
	register, _ = request("GET", "/register", nil, 200)
	request("POST", "/register", url.Values{"_token": {csrf(register)}, "name": {"Browser User"}, "email": {email}, "password": {password}, "password_confirmation": {password}}, 409)
	apiBody, _ := request("POST", "/api/auth/login", url.Values{"email": {email}, "password": {password}}, 200)
	var apiSession auth.Session
	if err := json.Unmarshal([]byte(apiBody), &apiSession); err != nil || apiSession.Token == "" {
		t.Fatalf("API login failed: %v", err)
	}
	req, _ := http.NewRequest("GET", server.URL+"/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+apiSession.Token)
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("API bearer authentication regressed")
	}
	login, _ = request("GET", "/login", nil, 200)
	request("POST", "/login", url.Values{"_token": {csrf(login)}, "email": {email}, "password": {password}}, 303)
	_, err = query.Table(db, driver, "users").WhereEq("email", email).Delete(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, gone := request("GET", "/dashboard", nil, 303)
	if gone.Header.Get("Location") != "/login" {
		t.Fatal("deleted user remained signed in")
	}
}
