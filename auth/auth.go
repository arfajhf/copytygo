package auth

import (
	"net/http"
	"slices"
	"time"

	"github.com/arfajhf/copytygo/v2/config"
	"github.com/arfajhf/copytygo/v2/core"
	"github.com/arfajhf/copytygo/v2/security"
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
