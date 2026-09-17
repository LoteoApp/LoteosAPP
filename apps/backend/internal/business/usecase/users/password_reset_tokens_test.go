package users

import (
	"testing"
	"time"
)

func TestPasswordResetTokensIssueAndConsume(t *testing.T) {
	t.Parallel()

	store := NewPasswordResetTokens()
	now := time.Now()

	token, err := store.Issue("user-1", now, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if token == "" {
		t.Fatal("Issue() returned an empty token")
	}

	usuarioID, ok := store.Consume(token, now.Add(time.Minute))
	if !ok {
		t.Fatal("Consume() ok = false, want true for a fresh token")
	}
	if usuarioID != "user-1" {
		t.Errorf("Consume() usuarioID = %q, want %q", usuarioID, "user-1")
	}
}

func TestPasswordResetTokensConsumeIsSingleUse(t *testing.T) {
	t.Parallel()

	store := NewPasswordResetTokens()
	now := time.Now()

	token, err := store.Issue("user-1", now, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, ok := store.Consume(token, now); !ok {
		t.Fatal("Consume() first call ok = false, want true")
	}
	if _, ok := store.Consume(token, now); ok {
		t.Error("Consume() second call ok = true, want false — a token must not be redeemable twice")
	}
}

func TestPasswordResetTokensConsumeRejectsExpired(t *testing.T) {
	t.Parallel()

	store := NewPasswordResetTokens()
	now := time.Now()

	token, err := store.Issue("user-1", now, time.Hour)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, ok := store.Consume(token, now.Add(time.Hour+time.Second)); ok {
		t.Error("Consume() ok = true, want false for an expired token")
	}
}

func TestPasswordResetTokensConsumeRejectsUnknownToken(t *testing.T) {
	t.Parallel()

	store := NewPasswordResetTokens()

	if _, ok := store.Consume("does-not-exist", time.Now()); ok {
		t.Error("Consume() ok = true, want false for a token that was never issued")
	}
}

// TestPasswordResetTokensIssueSweepsExpiredEntries guards against unbounded
// growth: unlike the cooldown map, every Issue call adds a brand new key, so
// nothing but a sweep keeps expired, never-consumed tokens from piling up
// for the lifetime of the process.
func TestPasswordResetTokensIssueSweepsExpiredEntries(t *testing.T) {
	t.Parallel()

	store := NewPasswordResetTokens()
	now := time.Now()

	for i := 0; i < 5; i++ {
		if _, err := store.Issue("user-1", now, time.Minute); err != nil {
			t.Fatalf("Issue() error = %v", err)
		}
	}
	if len(store.tokens) != 5 {
		t.Fatalf("len(tokens) = %d, want 5 before expiry", len(store.tokens))
	}

	if _, err := store.Issue("user-1", now.Add(time.Hour), time.Minute); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if len(store.tokens) != 1 {
		t.Errorf("len(tokens) = %d, want 1 — the sweep should have dropped the 5 expired tokens", len(store.tokens))
	}
}
