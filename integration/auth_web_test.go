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
	if created.Header.Get("Location") != "/account" {
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
}
