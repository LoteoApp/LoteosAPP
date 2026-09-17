package users

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// PasswordResetTokenTTL is how long a password reset link stays valid.
const PasswordResetTokenTTL = time.Hour

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// PasswordResetTokens issues and consumes single-use password reset tokens,
// in memory only (no migration): the backend runs as a single instance, so
// there's nothing to coordinate across replicas, and losing pending tokens
// on a restart is harmless — the user just requests another link. It's
// shared between RequestPasswordReset (issues) and ResetPassword (consumes),
// so it's constructed once and passed to both, not owned by either.
type PasswordResetTokens struct {
	mu     sync.Mutex
	tokens map[string]passwordResetEntry
}

type passwordResetEntry struct {
	usuarioID string
	expiresAt time.Time
}

func NewPasswordResetTokens() *PasswordResetTokens {
	return &PasswordResetTokens{tokens: make(map[string]passwordResetEntry)}
}

// Issue mints a new single-use token for usuarioID, valid until now+ttl.
func (store *PasswordResetTokens) Issue(usuarioID string, now time.Time, ttl time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	store.mu.Lock()
	defer store.mu.Unlock()
	store.tokens[token] = passwordResetEntry{usuarioID: usuarioID, expiresAt: now.Add(ttl)}

	return token, nil
}

// Consume reports the usuarioID a still-valid token was issued for, and
// invalidates it: a token can only ever be redeemed once, whether it
// succeeds or the caller decides not to use the result.
func (store *PasswordResetTokens) Consume(token string, now time.Time) (usuarioID string, ok bool) {
	store.mu.Lock()
	defer store.mu.Unlock()

	entry, found := store.tokens[token]
	delete(store.tokens, token)
	if !found || now.After(entry.expiresAt) {
		return "", false
	}

	return entry.usuarioID, true
}
