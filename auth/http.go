package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/arfajhf/copytygo/v3/core"
)

func RegisterHandler(defaultRole string) core.Handler {
	return func(ctx *core.Context) error {
		if err := ctx.Validate(map[string]string{
			"name":     "required|min:3",
			"email":    "required|email",
			"password": "required|min:8",
		}); err != nil {
			return err
		}

		session, err := Register(
			ctx.Input("name"),
			ctx.Input("email"),
			ctx.Input("password"),
			defaultRole,
		)
		if err != nil {
			switch {
			case errors.Is(err, ErrEmailRegistered):
				return core.NewHTTPError(http.StatusConflict, "Email already registered")
			case errors.Is(err, ErrInvalidRegistration):
				return core.NewHTTPError(http.StatusBadRequest, "Invalid registration data")
			default:
				return err
			}
		}

		return ctx.Status(http.StatusCreated).JSON(session)
	}
}

func LoginHandler() core.Handler {
	return func(ctx *core.Context) error {
		if err := ctx.Validate(map[string]string{
			"email":    "required|email",
			"password": "required|min:8",
		}); err != nil {
			return err
		}

		session, err := Login(
			ctx.Input("email"),
			ctx.Input("password"),
		)
		if err != nil {
			if errors.Is(err, ErrInvalidCredentials) {
				return core.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
			}
			return err
		}

		return ctx.JSON(session)
	}
}

func MeHandler() core.Handler {
	return func(ctx *core.Context) error {
		claims, ok := Claims(ctx)
		if !ok {
			return core.NewHTTPError(http.StatusUnauthorized, "Authentication required")
		}

		return ctx.JSON(core.Map{
			"id":    claims.Subject,
			"role":  claims.Role,
			"name":  claims.Data["name"],
			"email": claims.Data["email"],
			"exp":   claims.ExpiresAt,
		})
	}
}

func Routes(app *core.Application, defaultRole string) {
	app.Post("/api/auth/register", RegisterHandler(defaultRole))
	app.Post("/api/auth/login", LoginHandler())
	app.Get("/api/auth/me", MeHandler()).Middleware(Middleware())
}

func IssueDefault(subject, role string, data map[string]string) (string, error) {
	return Issue(subject, role, data, 24*time.Hour)
}
