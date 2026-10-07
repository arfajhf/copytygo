package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/arfajhf/copytygo/v4/auth"
	"github.com/arfajhf/copytygo/v4/core"
	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/database/query"
	"github.com/arfajhf/copytygo/v4/security"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTypeScriptBrowserAPIDatabaseIntegration(t *testing.T) {
	if os.Getenv("COPYTYGO_INTEGRATION_DB") != "1" {
		t.Skip("set COPYTYGO_INTEGRATION_DB=1")
	}
	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_KEY", "typescript-browser-integration-key-0123456789")
	drivers.Register()
	db, err := database.Connect()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	driver := database.LoadConfig().Driver
	identity := "BIGINT AUTO_INCREMENT"
	if driver == "postgres" {
		identity = "BIGSERIAL"
	}
	if _, err := db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS users (id %s PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, password VARCHAR(512) NOT NULL, role VARCHAR(50) NOT NULL DEFAULT 'user')", identity)); err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("ts-browser-%d@example.com", time.Now().UnixNano())
	managedEmail := fmt.Sprintf("ts-managed-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		for _, email := range []string{email, managedEmail} {
			query.Table(db, driver, "users").WhereEq("email", email).Delete(context.Background())
		}
	})
	sessions := auth.NewBrowserAuth("user")
	app := core.New()
	app.Get("/api/session", sessions.Session)
	app.Post("/api/session/register", sessions.Register).Middleware(sessions.CSRF())
	app.Post("/api/session/login", sessions.Login).Middleware(sessions.CSRF())
	app.Post("/api/session/logout", sessions.Logout).Middleware(sessions.Middleware(), sessions.CSRF())
	app.Get("/api/dashboard", sessions.Dashboard).Middleware(sessions.Middleware())
	app.Get("/api/users", sessions.Users).Middleware(sessions.Middleware(), sessions.RequireAdmin())
	app.Get("/api/users/:id", sessions.User).Middleware(sessions.Middleware(), sessions.RequireAdmin())
	app.Post("/api/users", sessions.CreateUser).Middleware(sessions.Middleware(), sessions.RequireAdmin(), sessions.CSRF())
	app.Put("/api/users/:id", sessions.UpdateUser).Middleware(sessions.Middleware(), sessions.RequireAdmin(), sessions.CSRF())
	app.Delete("/api/users/:id", sessions.DeleteUser).Middleware(sessions.Middleware(), sessions.RequireAdmin(), sessions.CSRF())
	auth.Routes(app, "user")
	server := httptest.NewServer(app)
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}
	csrf := ""
	request := func(method, path string, data map[string]string, want int) string {
		t.Helper()
		raw, _ := json.Marshal(data)
		req, _ := http.NewRequest(method, server.URL+path, strings.NewReader(string(raw)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", server.URL)
		req.Header.Set("X-CSRF-Token", csrf)
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		if response.StatusCode != want {
			t.Fatalf("%s %s got %d want %d: %s", method, path, response.StatusCode, want, body)
		}
		var result struct {
			CSRF string `json:"csrf"`
		}
		json.Unmarshal(body, &result)
		if result.CSRF != "" {
			csrf = result.CSRF
		}
		return string(body)
	}
	request("GET", "/api/session", nil, 200)
	guestCSRF := csrf
	input := map[string]string{"name": "TypeScript User", "email": email, "password": "test-password", "password_confirmation": "test-password", "role": "admin"}
	signedIn := request("POST", "/api/session/register", input, 201)
	if csrf == guestCSRF || strings.Contains(signedIn, `"token"`) || !strings.Contains(signedIn, `"role":"user"`) {
		t.Fatal("unsafe registration result")
	}
	row, found, err := query.Table(db, driver, "users").WhereEq("email", email).FirstMap(context.Background())
	if err != nil || !found || fmt.Sprint(row["role"]) != "user" || !security.VerifyPassword("test-password", fmt.Sprint(row["password"])) {
		t.Fatal("invalid persisted public account")
	}
	ownID := fmt.Sprint(row["id"])
	if strings.Contains(request("GET", "/api/dashboard", nil, 200), `"stats"`) {
		t.Fatal("member got admin stats")
	}
	request("GET", "/api/users", nil, 403)
	request("POST", "/api/users", input, 403)
	if err := auth.PromoteAdmin(context.Background(), email); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(request("GET", "/api/dashboard", nil, 200), `"stats"`) {
		t.Fatal("promotion did not refresh cookie role")
	}
	input["name"] = "Managed User"
	input["email"] = managedEmail
	input["role"] = "user"
	validCSRF := csrf
	csrf = "wrong"
	request("POST", "/api/users", input, 403)
	csrf = validCSRF
	request("POST", "/api/users", input, 201)
	request("POST", "/api/users", input, 409)
	managed, found, err := query.Table(db, driver, "users").WhereEq("email", managedEmail).FirstMap(context.Background())
	if err != nil || !found || !security.VerifyPassword("test-password", fmt.Sprint(managed["password"])) {
		t.Fatal("managed account not hashed")
	}
	id := fmt.Sprint(managed["id"])
	if strings.Contains(request("GET", "/api/users/"+id, nil, 200), "password") {
		t.Fatal("password exposed to frontend")
	}
	input["name"] = "Updated User"
	input["role"] = "admin"
	input["password"] = ""
	input["password_confirmation"] = ""
	request("PUT", "/api/users/"+id, input, 200)
	updated, _, _ := query.Table(db, driver, "users").WhereEq("id", id).FirstMap(context.Background())
	if fmt.Sprint(updated["name"]) != "Updated User" || fmt.Sprint(updated["role"]) != "admin" || fmt.Sprint(updated["password"]) != fmt.Sprint(managed["password"]) {
		t.Fatal("update or blank-password preservation failed")
	}
	input["password"] = "replacement-password"
	input["password_confirmation"] = "replacement-password"
	request("PUT", "/api/users/"+id, input, 200)
	updated, _, _ = query.Table(db, driver, "users").WhereEq("id", id).FirstMap(context.Background())
	if !security.VerifyPassword("replacement-password", fmt.Sprint(updated["password"])) {
		t.Fatal("password update not hashed")
	}
	input["email"] = email
	input["role"] = "user"
	request("PUT", "/api/users/"+ownID, input, 422)
	request("DELETE", "/api/users/"+ownID, nil, 422)
	if !strings.Contains(request("GET", "/api/users?search="+url.QueryEscape(managedEmail), nil, 200), "Updated User") {
		t.Fatal("search failed")
	}
	request("DELETE", "/api/users/"+id, nil, 200)
	request("GET", "/api/users/"+id, nil, 404)
	query.Table(db, driver, "users").WhereEq("id", ownID).Update(context.Background(), map[string]any{"role": "user"})
	request("GET", "/api/users", nil, 403)
	request("DELETE", "/api/users/"+id, nil, 403)
	request("GET", "/api/auth/me", nil, 401)
	request("POST", "/api/session/logout", nil, 200)
	request("GET", "/api/dashboard", nil, 401)
	request("GET", "/api/session", nil, 200)
	request("POST", "/api/session/login", map[string]string{"email": email, "password": "wrong-password"}, 401)
	request("POST", "/api/session/login", map[string]string{"email": email, "password": "test-password"}, 200)
	request("GET", "/api/dashboard", nil, 200)
	query.Table(db, driver, "users").WhereEq("id", ownID).Delete(context.Background())
	request("GET", "/api/dashboard", nil, 401)
	if !strings.Contains(request("GET", "/api/session", nil, 200), `"user":null`) {
		t.Fatal("deleted user stayed signed in")
	}
}
