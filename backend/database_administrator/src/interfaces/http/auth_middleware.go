// Package httpiface — auth_middleware.go introduces the Go-side
// JWE-cookie verifier middleware for cachicamas-github-login PR-3.
//
// Reference: openspec/changes/cachicamas-github-login/specs/backend-auth-middleware/spec.md
//   R-BAM-001..R-BAM-031 — the IdentityFromCookie middleware reads
//   the authjs.session-token JWE cookie, decrypts it with the
//   shared AUTH_SECRET, resolves the matching identity.user row,
//   and populates c.Set("identity", *Identity) for downstream
//   handlers. On any failure it returns a 401 with the locked
//   "code=unauthorized" envelope.
//
// Design: openspec/changes/cachicamas-github-login/design.md §3.
//   The byte-level envelope contract is load-bearing:
//     alg=dir, enc=A256CBC-HS512, HKDF-SHA256 over AUTH_SECRET
//     with salt=cookieName, info="Auth.js Generated Encryption
//     Key (cookieName)", length=64. Auth.js v0.34.3+ produces
//     this exact envelope (verified against @auth/core@0.41.2
//     source in this PR).
//
// Why this lives in interfaces/http:
//   The middleware is the boundary between Auth.js (browser-side)
//   and the Go service. It belongs in the transport layer (next to
//   CORS, health, organization handler) rather than domain or
//   application — it's a request-time policy, not a domain rule.
//   The middleware depends on domain.IdentityRepository (the
//   hexagonal port introduced in PR-1) so the pgx adapter stays
//   out of the httpiface package.
package httpiface

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/hkdf"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// IdentityContextKey is the key under which the middleware stores the
// resolved *Identity on the Echo context. Handlers fetch it via
// `c.Get(IdentityContextKey)` (returns `*domain.Identity` or nil).
const IdentityContextKey = "identity"

// IdentityMiddlewareConfig configures the IdentityFromCookie middleware.
// The struct is exported so future slices (e.g., a project-scoped
// middleware) can embed it.
type IdentityMiddlewareConfig struct {
	// AuthSecret is the SAME secret the Auth.js encoder uses on the
	// Qwik side. MUST be at least 32 raw bytes; the constructor
	// panics otherwise (fail-fast at startup, per S-BAM-080/081).
	AuthSecret string

	// CookieName is the cookie to read: "authjs.session-token" in
	// dev (HTTP), "__Secure-authjs.session-token" in production
	// (HTTPS). The salt in HKDF is THIS value, so dev and prod
	// sessions use DIFFERENT derived keys even with the same secret.
	CookieName string

	// IdentityRepo is the hexagonal port used to resolve the email
	// claim from the JWE into a full *domain.Identity.
	IdentityRepo domain.IdentityRepository

	// Logger is the slog logger used for the PII-safe audit line.
	// Required.
	Logger *slog.Logger

	// TracerProvider is the OTel SDK TracerProvider used to emit
	// the auth.verify span. When nil, the global provider is used.
	TracerProvider trace.TracerProvider
}

// authJSCookiePayload mirrors the canonical Auth.js JWT claims the
// middleware reads. Field names are LOCKED — they match @auth/core's
// Encode<JWT>() shape (sub/email/name/picture/exp/iat/jti).
type authJSCookiePayload struct {
	Sub     string `json:"sub,omitempty"`
	Email   string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
	Picture string `json:"picture,omitempty"`
	Iss     string `json:"iss,omitempty"`
	Aud     any    `json:"aud,omitempty"`
	Exp     int64  `json:"exp,omitempty"`
	Iat     int64  `json:"iat,omitempty"`
	Jti     string `json:"jti,omitempty"`
}

// IdentityFromCookie returns an Echo middleware that reads the
// configured cookie, decrypts the JWE using AUTH_SECRET, resolves
// the identity.user row, and stores the *domain.Identity on the
// Echo context.
//
// Panics (fail-fast at startup):
//   - if cfg.AuthSecret is empty (S-BAM-080)
//   - if cfg.AuthSecret is shorter than 32 raw bytes (S-BAM-081)
//   - if cfg.CookieName is empty
//   - if cfg.IdentityRepo is nil
//   - if cfg.Logger is nil
//
// On any per-request failure the middleware writes a 401 with the
// locked "code=unauthorized" envelope and does NOT call next(c).
func IdentityFromCookie(cfg IdentityMiddlewareConfig) echo.MiddlewareFunc {
	if cfg.AuthSecret == "" {
		panic("IdentityFromCookie: AUTH_SECRET must be set (cachicamas-github-login S-BAM-080)")
	}
	if len(cfg.AuthSecret) < 32 {
		panic(fmt.Sprintf("IdentityFromCookie: AUTH_SECRET must be at least 32 raw bytes, got %d (S-BAM-081)", len(cfg.AuthSecret)))
	}
	if cfg.CookieName == "" {
		panic("IdentityFromCookie: CookieName must be set")
	}
	if cfg.IdentityRepo == nil {
		panic("IdentityFromCookie: IdentityRepo must be non-nil")
	}
	if cfg.Logger == nil {
		panic("IdentityFromCookie: Logger must be non-nil")
	}

	// Pre-derive the HKDF key once at construction. HKDF is pure,
	// so re-deriving per request is wasteful; this is the same
	// pattern Auth.js uses (the salt+info are stable across the
	// process lifetime).
	derivedKey := deriveEncryptionKey(cfg.AuthSecret, cfg.CookieName)
	jwkKey, err := jwk.FromRaw(derivedKey)
	if err != nil {
		// jwk.FromRaw only fails for nil input; we control derivedKey
		// so this is a programmer error. Fail-fast at startup.
		panic(fmt.Sprintf("IdentityFromCookie: jwk.FromRaw: %v", err))
	}

	tracerProvider := cfg.TracerProvider
	if tracerProvider == nil {
		tracerProvider = otel.GetTracerProvider()
	}
	tracer := tracerProvider.Tracer("cachicamas/auth-middleware")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ctx, span := tracer.Start(c.Request().Context(), "auth.verify")
			defer span.End()

			// 1. Read the cookie.
			cookie, err := c.Request().Cookie(cfg.CookieName)
			if err != nil {
				writeUnauthorized(c, span, cfg.Logger, "missing_cookie", "", "no cookie in request")
				return nil
			}
			jweValue := cookie.Value

			// 2. Decrypt the JWE.
			plaintext, err := jwe.Decrypt(
				[]byte(jweValue),
				jwe.WithKey(jwa.DIRECT, jwkKey),
			)
			if err != nil {
				writeUnauthorized(c, span, cfg.Logger, "decrypt_failed", "", "jwe decrypt: "+err.Error())
				return nil
			}

			// 3. Parse the JWT claims.
			var claims authJSCookiePayload
			if err := json.Unmarshal(plaintext, &claims); err != nil {
				writeUnauthorized(c, span, cfg.Logger, "invalid_payload", "", "json unmarshal: "+err.Error())
				return nil
			}
			if claims.Email == "" {
				writeUnauthorized(c, span, cfg.Logger, "missing_email_claim", "", "no email claim in JWE")
				return nil
			}

			// 4. Resolve the identity.user row.
			identity, err := cfg.IdentityRepo.LookupByEmail(ctx, claims.Email)
			if err != nil {
				// Distinguish "not found" (configurable in the future)
				// from "db down" (always 401 for the middleware path,
				// to keep the response surface uniform).
				emailHash := emailHash(claims.Email)
				span.SetAttributes(attribute.String("auth.email_hash", emailHash))
				span.SetStatus(codes.Error, "identity lookup failed")
				cfg.Logger.WarnContext(ctx, "auth.verify",
					slog.String("result", "lookup_failed"),
					slog.String("email_hash", emailHash),
					slog.String("error", err.Error()),
				)
				writeUnauthorized(c, span, cfg.Logger, "identity_lookup_failed", emailHash, err.Error())
				return nil
			}

			// 5. Populate the Echo context and emit the audit line.
			c.Set(IdentityContextKey, identity)
			emailHash := emailHash(claims.Email)
			span.SetAttributes(
				attribute.String("auth.email_hash", emailHash),
				attribute.String("auth.provider", identity.Provider),
				attribute.Int64("auth.identity_id", identity.ID),
			)
			span.SetStatus(codes.Ok, "")
			cfg.Logger.InfoContext(ctx, "auth.verify",
				slog.String("result", "ok"),
				slog.String("email_hash", emailHash),
				slog.String("provider", identity.Provider),
				slog.Int64("identity_id", identity.ID),
			)
			return next(c)
		}
	}
}

// writeUnauthorized emits the locked 401 envelope and the OTel span
// status. It does NOT call next(c).
func writeUnauthorized(c *echo.Context, span trace.Span, logger *slog.Logger, reason, emailHash, _ string) {
	span.SetStatus(codes.Error, reason)
	if emailHash == "" {
		emailHash = "-"
	}
	logger.WarnContext(c.Request().Context(), "auth.verify",
		slog.String("result", "denied"),
		slog.String("reason", reason),
		slog.String("email_hash", emailHash),
	)
	c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Response().WriteHeader(http.StatusUnauthorized)
	// Use a stable envelope shape; the locked "code=unauthorized"
	// matches the PR-3 spec scenario S-BAM-011/012.
	_, _ = io.WriteString(c.Response(), `{"code":"unauthorized","message":"authentication required"}`)
}

// deriveEncryptionKey applies the HKDF-SHA256 contract from
// @auth/core/src/jwt.ts (verified for 0.34.3+; the same shape
// applies to 0.41.x). salt is the cookie name; info is the
// literal "Auth.js Generated Encryption Key (salt)"; length is 64.
//
// Why this matches design §3 byte-for-byte:
//   Auth.js calls `hkdf("sha256", keyMaterial, salt, info, length)`
//   where keyMaterial is the raw AUTH_SECRET string (NOT base64-decoded),
//   salt is the cookie name, info is the literal above, length is 64
//   for A256CBC-HS512 (or 32 for A256GCM, but Auth.js' default is the
//   former).
func deriveEncryptionKey(authSecret, cookieName string) []byte {
	info := []byte("Auth.js Generated Encryption Key (" + cookieName + ")")
	r := hkdf.New(sha256.New, []byte(authSecret), []byte(cookieName), info)
	out := make([]byte, 64)
	if _, err := io.ReadFull(r, out); err != nil {
		// hkdf.New with a fixed-length reader can only error on
		// programmer bugs (length=0). Treat as fatal.
		panic(fmt.Sprintf("deriveEncryptionKey: hkdf read: %v", err))
	}
	return out
}

// emailHash returns sha256(email) truncated to 12 hex chars. This is
// the canonical PII-safe identifier used in slog + span attributes
// (S-BAM-072). 12 hex chars = 48 bits of entropy, enough for log
// dedupe without leaking the address.
func emailHash(email string) string {
	h := sha256.Sum256([]byte(email))
	return hex.EncodeToString(h[:])[:12]
}

// Compile-time check that IdentityFromCookie returns echo.MiddlewareFunc.
// (Constructor panics on bad config — we cannot call it at package
// init time without panicking the test binary. The function signature
// is verified by the function bodies themselves.)

// silentReader discards input; placeholder for future logging hooks.
var _ = bytes.NewReader(nil)