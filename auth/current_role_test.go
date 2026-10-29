package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arfajhf/copytygo/v5/core"
	"github.com/arfajhf/copytygo/v5/security"
)

type fakeCurrentRoleFinder func(context.Context, string) (User, error)

func (f fakeCurrentRoleFinder) Find(ctx context.Context, id string) (User, error) {
	return f(ctx, id)
}

func TestRequireStoredRoleEnforcesCurrentDatabaseRole(t *testing.T) {
	tests := []struct {
		name string
		user User
		err error
		roles []string
		want int
	}{
		{name: "demoted admin token", user: User{Role: "user"}, roles: []string{"admin"}, want: 403},
		{name: "current admin", user: User{Role: "admin"}, roles: []string{"admin"}, want: 200},
		{name: "deleted account", err: errUserMissing, roles: []string{"admin"}, want: 401},
		{name: "database unavailable", err: errors.New("db unavailable"), want: 500},
		{name: "authenticated current member", user: User{Role: "user"}, want: 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := core.New()
			finder := fakeCurrentRoleFinder(func(ctx context.Context, id string) (User, error) {
				if id != "42" {
					t.Fatalf("unexpected lookup %q", id)
				}
				return tc.user, tc.err
			})
			seedTokenClaims := func(next core.Handler) core.Handler {
				return func(ctx *core.Context) error {
					ctx.Set(ClaimsContextKey, security.Claims{Subject: "42", Role: "admin"})
					return next(ctx)
				}
			}
			app.Get("/products", func(ctx *core.Context) error {
				return ctx.JSON(core.Map{"ok": true})
			}).Middleware(seedTokenClaims, requireStoredRole(finder, tc.roles...))
			w := httptest.NewRecorder()
			app.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/products", nil))
			if w.Code != tc.want {
				t.Fatalf("got %d, want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}
