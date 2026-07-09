// sse_test_helper.go — minimal Echo + auth stub for the SSE
// handler tests. The production RegisterAuthenticatedWorkspaceRoutes
// wires a real IdentityFromCookie middleware; for the SSE tests
// we use a stub that injects a fake identity so the auth check
// in the SSE handler passes without a real JWE session.
package httpiface

import (
	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// newSSEEcho constructs a fresh Echo router (root) with a stub
// auth middleware applied at the root. The handler test
// fixtures call RegisterSyncStreamRoute on the root to attach
// the SSE route.
func newSSEEcho() *echo.Echo {
	e := echo.New()
	// Stub auth: always sets a fake identity so the SSE
	// handler's IdentityFromContext check passes. The
	// production route uses IdentityFromCookie + LoadGitHubToken.
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set(IdentityContextKey, &domain.Identity{
				ID:                1,
				Email:             "test@example.com",
				Provider:          "github",
				ProviderAccountID: "12345",
			})
			return next(c)
		}
	})
	return e
}
