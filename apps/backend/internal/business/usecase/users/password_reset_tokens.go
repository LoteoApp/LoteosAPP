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
	store.sweepLocked(now)
	store.invalidateLocked(usuarioID)
	store.tokens[token] = passwordResetEntry{usuarioID: usuarioID, expiresAt: now.Add(ttl)}

	return token, nil
}

// invalidateLocked drops every still-pending token for usuarioID, so at most
// one reset link is ever redeemable per user: issuing a new one retires
// whatever was sent before, so an intercepted old link can't be used after
// the user requests (or completes) a newer reset. Caller must hold mu.
func (store *PasswordResetTokens) invalidateLocked(usuarioID string) {
	for token, entry := range store.tokens {
		if entry.usuarioID == usuarioID {
			delete(store.tokens, token)
		}
	}
}

// sweepLocked drops every token that has already expired, so the store stays
// bounded to tokens someone could still redeem instead of growing with every
// reset ever requested since the process started. Caller must hold mu.
func (store *PasswordResetTokens) sweepLocked(now time.Time) {
	for token, entry := range store.tokens {
		if now.After(entry.expiresAt) {
			delete(store.tokens, token)
		}
	}
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
