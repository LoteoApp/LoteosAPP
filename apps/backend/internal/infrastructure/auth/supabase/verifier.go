package supabase

import (
	"context"
	"fmt"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Principal represents the caller identified by a validated Supabase Auth
// token.
type Principal struct {
	Subject string
	Email   string
	Roles   []string
}

type Verifier struct {
	keyfunc keyfunc.Keyfunc
	issuer  string
	cancel  context.CancelFunc
}

// failFastOnFirstFetch overrides keyfunc's default of tolerating a failed
// initial JWKS fetch, so NewVerifier reports an error instead of starting
// with an empty key set.
var failFastOnFirstFetch = false

// audience is the fixed value Supabase Auth stamps on every access token's
// aud claim, regardless of project.
const audience = "authenticated"

// NewVerifier fetches the JWKS Supabase Auth publishes for supabaseURL (the
// project's API URL, e.g. https://<project-ref>.supabase.co) and keeps it
// refreshed in the background until Close is called. It fails if the
// initial fetch fails.
func NewVerifier(ctx context.Context, supabaseURL string) (*Verifier, error) {
	refreshCtx, cancel := context.WithCancel(context.Background())

	jwksURL := supabaseURL + "/auth/v1/.well-known/jwks.json"

	k, err := keyfunc.NewDefaultOverrideCtx(refreshCtx, []string{jwksURL}, keyfunc.Override{
		NoErrorReturnFirstHTTPReq: &failFastOnFirstFetch,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("fetch supabase jwks: %w", err)
	}

	return &Verifier{
		keyfunc: k,
		issuer:  supabaseURL + "/auth/v1",
		cancel:  cancel,
	}, nil
}

// Close stops the background JWKS refresh.
func (v *Verifier) Close() {
	v.cancel()
}

type appMetadata struct {
	Role string `json:"role"`
}

type claims struct {
	Email       string      `json:"email"`
	AppMetadata appMetadata `json:"app_metadata"`
	jwt.RegisteredClaims
}

// Verify validates the token's signature, issuer, audience and expiration,
// and extracts the caller's subject, email and domain role. The role comes
// from app_metadata.role, set by AdminClient.CreateUser when the account is
// provisioned.
func (v *Verifier) Verify(ctx context.Context, tokenString string) (Principal, error) {
	tokenClaims := &claims{}

	_, err := jwt.ParseWithClaims(
		tokenString,
		tokenClaims,
		v.keyfunc.KeyfuncCtx(ctx),
		jwt.WithValidMethods([]string{"RS256", "ES256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Principal{}, fmt.Errorf("verify token: %w", err)
	}

	principal := Principal{
		Subject: tokenClaims.Subject,
		Email:   tokenClaims.Email,
	}
	if tokenClaims.AppMetadata.Role != "" {
		principal.Roles = []string{tokenClaims.AppMetadata.Role}
	}

	return principal, nil
}
