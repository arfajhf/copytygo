package authorization

import (
	"net/http"
	"sync"

	"github.com/arfajhf/copytygo/v4/core"
)

type Policy func(*core.Context, any) bool

type Gate struct {
	mu       sync.RWMutex
	policies map[string]Policy
}

func New() *Gate {
	return &Gate{policies: make(map[string]Policy)}
}

func (g *Gate) Define(ability string, policy Policy) {
	g.mu.Lock()
	g.policies[ability] = policy
	g.mu.Unlock()
}

func (g *Gate) Allows(ctx *core.Context, ability string, subject any) bool {
	g.mu.RLock()
	policy, ok := g.policies[ability]
	g.mu.RUnlock()
	return ok && policy != nil && policy(ctx, subject)
}

func (g *Gate) Denies(ctx *core.Context, ability string, subject any) bool {
	return !g.Allows(ctx, ability, subject)
}

func (g *Gate) Authorize(ability string, subject func(*core.Context) any) core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx *core.Context) error {
			var value any
			if subject != nil {
				value = subject(ctx)
			}
			if !g.Allows(ctx, ability, value) {
				return core.NewHTTPError(http.StatusForbidden, "This action is not authorized")
			}
			return next(ctx)
		}
	}
}
