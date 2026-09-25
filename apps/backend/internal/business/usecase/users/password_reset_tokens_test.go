package users

import (
	"fmt"
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
// growth: nothing but a sweep keeps expired, never-consumed tokens from
// piling up for the lifetime of the process.
func TestPasswordResetTokensIssueSweepsExpiredEntries(t *testing.T) {
	t.Parallel()

	store := NewPasswordResetTokens()
	now := time.Now()

	for i := 0; i < 5; i++ {
		usuarioID := fmt.Sprintf("user-%d", i)
		if _, err := store.Issue(usuarioID, now, time.Minute); err != nil {
			t.Fatalf("Issue() error = %v", err)
		}
	}
	if len(store.tokens) != 5 {
		t.Fatalf("len(tokens) = %d, want 5 before expiry", len(store.tokens))
	}

	if _, err := store.Issue("user-5", now.Add(time.Hour), time.Minute); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if len(store.tokens) != 1 {
		t.Errorf("len(tokens) = %d, want 1 — the sweep should have dropped the 5 expired tokens", len(store.tokens))
	}
}

// TestPasswordResetTokensIssueInvalidatesPreviousToken guards against an
// intercepted old link staying valid: requesting a second reset for the same
// user must retire the first token, not just add another valid one.
func TestPasswordResetTokensIssueInvalidatesPreviousToken(t *testing.T) {
	t.Parallel()

	store := NewPasswordResetTokens()
	now := time.Now()

	first, err := store.Issue("user-1", now, time.Hour)
	if err != nil {
		t.Fatalf("Issue() first error = %v", err)
	}
	second, err := store.Issue("user-1", now, time.Hour)
	if err != nil {
		t.Fatalf("Issue() second error = %v", err)
	}

	if _, ok := store.Consume(first, now); ok {
		t.Error("Consume() on the first token ok = true, want false — it should have been invalidated")
	}
	if usuarioID, ok := store.Consume(second, now); !ok || usuarioID != "user-1" {
		t.Errorf("Consume() on the second token = (%q, %v), want (%q, true)", usuarioID, ok, "user-1")
	}
}

// TestPasswordResetTokensIssueKeepsTokensForDifferentUsers guards against an
// overly broad invalidation: issuing a token for one user must not touch a
// still-pending token belonging to another.
func TestPasswordResetTokensIssueKeepsTokensForDifferentUsers(t *testing.T) {
	t.Parallel()

	store := NewPasswordResetTokens()
	now := time.Now()

	tokenForUserOne, err := store.Issue("user-1", now, time.Hour)
	if err != nil {
		t.Fatalf("Issue() for user-1 error = %v", err)
	}
	if _, err := store.Issue("user-2", now, time.Hour); err != nil {
		t.Fatalf("Issue() for user-2 error = %v", err)
	}

	if usuarioID, ok := store.Consume(tokenForUserOne, now); !ok || usuarioID != "user-1" {
		t.Errorf("Consume() for user-1's token = (%q, %v), want (%q, true)", usuarioID, ok, "user-1")
	}
}
