package keycloak

import (
	"context"
	"fmt"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Principal represents the caller identified by a validated Keycloak token.
type Principal struct {
	Subject string
	Roles   []string
}

type Verifier struct {
	keyfunc  keyfunc.Keyfunc
	issuer   string
	audience string
	cancel   context.CancelFunc
}

// failFastOnFirstFetch overrides keyfunc's default of tolerating a failed
// initial JWKS fetch, so NewVerifier reports an error instead of starting
// with an empty key set.
var failFastOnFirstFetch = false

// NewVerifier fetches the JWKS from jwksBaseURL and keeps it refreshed in
// the background until Close is called. It fails if the initial fetch
// fails.
//
// jwksBaseURL and issuer are often the same value, but can differ behind
// split-horizon DNS (e.g. in Docker Compose): jwksBaseURL must be reachable
// from this process, while issuer must match the canonical `iss` Keycloak
// stamps on tokens (governed by KC_HOSTNAME), which callers may reach
// through a different, publicly reachable URL.
func NewVerifier(ctx context.Context, jwksBaseURL, issuer, audience string) (*Verifier, error) {
	refreshCtx, cancel := context.WithCancel(context.Background())

	jwksURL := jwksBaseURL + "/protocol/openid-connect/certs"

	k, err := keyfunc.NewDefaultOverrideCtx(refreshCtx, []string{jwksURL}, keyfunc.Override{
		NoErrorReturnFirstHTTPReq: &failFastOnFirstFetch,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("fetch keycloak jwks: %w", err)
	}

	return &Verifier{
		keyfunc:  k,
		issuer:   issuer,
		audience: audience,
		cancel:   cancel,
	}, nil
}

// Close stops the background JWKS refresh.
func (v *Verifier) Close() {
	v.cancel()
}

type realmAccess struct {
	Roles []string `json:"roles"`
}

type resourceAccess struct {
	Roles []string `json:"roles"`
}

type claims struct {
	RealmAccess    realmAccess               `json:"realm_access"`
	ResourceAccess map[string]resourceAccess `json:"resource_access"`
	jwt.RegisteredClaims
}

// Verify validates the token's signature, issuer, audience and expiration,
// and extracts the caller's realm and client roles.
func (v *Verifier) Verify(ctx context.Context, tokenString string) (Principal, error) {
	tokenClaims := &claims{}

	_, err := jwt.ParseWithClaims(
		tokenString,
		tokenClaims,
		v.keyfunc.KeyfuncCtx(ctx),
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Principal{}, fmt.Errorf("verify token: %w", err)
	}

	return Principal{
		Subject: tokenClaims.Subject,
		Roles:   mergeRoles(tokenClaims.RealmAccess.Roles, tokenClaims.ResourceAccess[v.audience].Roles),
	}, nil
}

func mergeRoles(roleSets ...[]string) []string {
	seen := make(map[string]struct{})
	var roles []string

	for _, set := range roleSets {
		for _, role := range set {
			if _, ok := seen[role]; ok {
				continue
			}
			seen[role] = struct{}{}
			roles = append(roles, role)
		}
	}

	return roles
}
