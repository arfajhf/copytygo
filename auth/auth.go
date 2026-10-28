package auth

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/arfajhf/copytygo/v5/config"
	"github.com/arfajhf/copytygo/v5/core"
	"github.com/arfajhf/copytygo/v5/security"
)

const ClaimsContextKey = "copytygo.auth.claims"

func Issue(subject, role string, data map[string]string, ttl time.Duration) (string, error) {
	claims := security.Claims{
		Subject: subject,
		Role:    role,
		Data:    data,
	}

	if ttl > 0 {
		claims.ExpiresAt = time.Now().Add(ttl).Unix()
	}

	return security.SignToken(config.Get("APP_KEY"), claims)
}

func Middleware() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx *core.Context) error {
			token := ctx.BearerToken()
			if token == "" {
				return core.NewHTTPError(http.StatusUnauthorized, "Authentication required")
			}

			claims, err := security.VerifyToken(config.Get("APP_KEY"), token)
			if err != nil {
				return core.NewHTTPError(http.StatusUnauthorized, "Invalid authentication token")
			}

			ctx.Set(ClaimsContextKey, claims)
			return next(ctx)
		}
	}
}

func Claims(ctx *core.Context) (security.Claims, bool) {
	return core.ContextValue[security.Claims](ctx, ClaimsContextKey)
}

func UserID(ctx *core.Context) string {
	claims, ok := Claims(ctx)
	if !ok {
		return ""
	}
	return claims.Subject
}

func Role(ctx *core.Context) string {
	claims, ok := Claims(ctx)
	if !ok {
		return ""
	}
	return claims.Role
}

func RequireRole(roles ...string) core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx *core.Context) error {
			claims, ok := Claims(ctx)
			if !ok {
				return core.NewHTTPError(http.StatusUnauthorized, "Authentication required")
			}

			if len(roles) > 0 && !slices.Contains(roles, claims.Role) {
				return core.NewHTTPError(http.StatusForbidden, "Insufficient role")
			}

			return next(ctx)
		}
	}
}

// RequireCurrentRole checks the current database account on every request.
// Middleware() must run first to authenticate the signed bearer token. Role
// claims from a token alone are not enough if an admin was demoted/deleted.
func RequireCurrentRole(multi bool, roles ...string) core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx *core.Context) error {
			claims, ok := Claims(ctx)
			if !ok {
				return core.NewHTTPError(http.StatusUnauthorized, "Authentication required")
			}
			requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
			defer cancel()
			user, err := (databaseUsers{multi: multi}).Find(requestCtx, claims.Subject)
			if errors.Is(err, errUserMissing) {
				return core.NewHTTPError(http.StatusUnauthorized, "Account no longer exists")
			}
			if err != nil {
				return err
			}
			claims.Role = user.Role
			claims.Data = map[string]string{"name": user.Name, "email": user.Email}
			ctx.Set(ClaimsContextKey, claims)
			if len(roles) > 0 && !slices.Contains(roles, user.Role) {
				return core.NewHTTPError(http.StatusForbidden, "Insufficient role")
			}
			return next(ctx)
		}
	}
}
