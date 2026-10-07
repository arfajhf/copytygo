package auth

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type memoryUsers struct {
	users          map[string]User
	saves, deletes int
	failure        error
}

func (store *memoryUsers) Find(_ context.Context, id string) (User, error) {
	if store.failure != nil {
		return User{}, store.failure
	}
	user, ok := store.users[id]
	if !ok {
		return User{}, errUserMissing
	}
	return user, nil
}
func (store *memoryUsers) Stats(context.Context) (userStats, error) {
	result := userStats{Total: int64(len(store.users))}
	for _, user := range store.users {
		if user.Role == "admin" {
			result.Admins++
		}
	}
	result.Members = result.Total - result.Admins
	return result, store.failure
}
func (store *memoryUsers) List(_ context.Context, search string, page int) ([]User, int64, error) {
	users := []User{}
	for _, user := range store.users {
		if strings.Contains(user.Name, search) || strings.Contains(user.Email, search) {
			users = append(users, user)
		}
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	count := int64(len(users))
	start := (page - 1) * 10
	if start > len(users) {
		start = len(users)
	}
	end := start + 10
	if end > len(users) {
		end = len(users)
	}
	return users[start:end], count, store.failure
}
func (store *memoryUsers) Save(_ context.Context, actor, id string, input userInput) error {
	if store.users[actor].Role != "admin" {
		return errAdminOnly
	}
	if id == actor && input.Role != "admin" {
		return errOwnAdmin
	}
	for _, user := range store.users {
		if user.Email == input.Email && user.ID != id {
			return ErrEmailRegistered
		}
	}
	if id == "" {
		id = strconv.Itoa(len(store.users) + 20)
	} else if _, ok := store.users[id]; !ok {
		return errUserMissing
	}
	store.users[id] = User{ID: id, Name: input.Name, Email: input.Email, Role: input.Role}
	store.saves++
	return store.failure
}
func (store *memoryUsers) Delete(_ context.Context, actor, id string) error {
	if store.users[actor].Role != "admin" {
		return errAdminOnly
	}
	if actor == id {
		return errOwnAdmin
	}
	if _, ok := store.users[id]; !ok {
		return errUserMissing
	}
	delete(store.users, id)
	store.deletes++
	return store.failure
}

func TestDashboardRoleAndAdminCRUD(t *testing.T) {
	app, pages := browserTestApp(t)
	store := pages.users.(*memoryUsers)
	store.users["8"] = User{ID: "8", Name: "Second User", Email: "second@example.com", Role: "user"}
	pages.login = func(string, string) (Session, error) {
		token, err := Issue("7", "user", map[string]string{"name": "Stale Name"}, time.Hour)
		return Session{Token: token}, err
	}
	token, guest := webForm(t, app, "/login")
	login := webRequest(app, "POST", "/login", url.Values{"_token": {token}, "email": {"alex@example.com"}, "password": {"test-password"}}, guest, "")
	cookie := login.Result().Cookies()[0]
	request := func(method, path string, input url.Values, want int) string {
		t.Helper()
		rec := webRequest(app, method, path, input, cookie, "")
		if rec.Code != want {
			t.Fatalf("%s %s got %d want %d: %s", method, path, rec.Code, want, rec.Body.String())
		}
		return rec.Body.String()
	}
	dashboard := request("GET", "/dashboard", nil, 200)
	if !strings.Contains(dashboard, "Alex User") || strings.Contains(dashboard, `href="/users"`) || strings.Contains(dashboard, "Stale Name") {
		t.Fatal("member dashboard leaked admin menu or used stale identity")
	}
	request("GET", "/users", nil, 403)
	request("GET", "/users/8/edit", nil, 403)
	request("POST", "/users/8/delete", url.Values{"_token": {csrfField.FindStringSubmatch(dashboard)[1]}}, 403)
	if store.deletes != 0 {
		t.Fatal("member deleted a user")
	}
	admin := store.users["7"]
	admin.Role = "admin"
	store.users["7"] = admin
	dashboard = request("GET", "/dashboard", nil, 200)
	if !strings.Contains(dashboard, `href="/users"`) || !strings.Contains(dashboard, "Total users") {
		t.Fatal("fresh promotion missing admin dashboard")
	}
	csrf := csrfField.FindStringSubmatch(dashboard)[1]
	request("GET", "/users/create", nil, 200)
	input := url.Values{"_token": {csrf}, "name": {"Managed User"}, "email": {"managed@example.com"}, "password": {"test-password"}, "password_confirmation": {"test-password"}, "role": {"superadmin"}}
	invalid := request("POST", "/users", input, 422)
	if !strings.Contains(invalid, "must be user or admin") || strings.Contains(invalid, "test-password") {
		t.Fatal("validation missing or password reflected")
	}
	input.Set("role", "user")
	input.Set("_token", "wrong")
	request("POST", "/users", input, 403)
	if store.saves != 0 {
		t.Fatal("invalid form reached persistence")
	}
	input.Set("_token", csrf)
	request("POST", "/users", input, 303)
	request("POST", "/users", input, 409)
	edit := request("GET", "/users/8/edit", nil, 200)
	if !strings.Contains(edit, "Leave blank to keep") {
		t.Fatal("optional password editing unclear")
	}
	input.Set("email", "second@example.com")
	input.Set("name", "Updated User")
	input.Set("password", "")
	input.Set("password_confirmation", "")
	input.Set("role", "admin")
	request("POST", "/users/8", input, 303)
	if store.users["8"].Role != "admin" {
		t.Fatal("role update failed")
	}
	input.Set("email", "alex@example.com")
	input.Set("role", "user")
	request("POST", "/users/7", input, 422)
	request("POST", "/users/7/delete", url.Values{"_token": {csrf}}, 422)
	request("POST", "/users/8/delete", url.Values{"_token": {"wrong"}}, 403)
	request("POST", "/users/8/delete", url.Values{"_token": {csrf}}, 303)
	request("GET", "/users/8/edit", nil, 404)
	request("GET", "/users/not-an-id/edit", nil, 404)
	request("GET", "/users?search=no-such-user&page=-1", nil, 200)
	admin.Role = "user"
	store.users["7"] = admin
	request("GET", "/users", nil, 403)
	delete(store.users, "7")
	request("GET", "/dashboard", nil, 303)
}

func TestDashboardSingleRoleAndDatabaseFailure(t *testing.T) {
	app, pages := browserTestApp(t)
	pages.options.DefaultRole = ""
	pages.login = func(string, string) (Session, error) {
		token, err := Issue("7", "admin", nil, time.Hour)
		return Session{Token: token}, err
	}
	token, cookie := webForm(t, app, "/login")
	login := webRequest(app, "POST", "/login", url.Values{"_token": {token}, "email": {"alex@example.com"}, "password": {"test-password"}}, cookie, "")
	cookie = login.Result().Cookies()[0]
	dashboard := webRequest(app, "GET", "/dashboard", nil, cookie, "")
	if dashboard.Code != 200 || strings.Contains(dashboard.Body.String(), `href="/users"`) {
		t.Fatal("single-role auth exposed user management")
	}
	store := pages.users.(*memoryUsers)
	store.failure = errors.New("private database detail")
	failed := webRequest(app, "GET", "/dashboard", nil, cookie, "")
	if failed.Code != 503 || strings.Contains(failed.Body.String(), "private database detail") {
		t.Fatal("database failure leaked or looked like logout")
	}
}
