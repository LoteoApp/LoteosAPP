package postgres

import (
	"testing"

	"loteosapp/backend/internal/business/gateway"
)

func TestNormalizeReservationScopeSkipsBlankAuthProviderIDs(t *testing.T) {
	blank := "  "
	actor := "  52ce9940-b846-4ff9-9a7c-94238149ea53  "
	scope := normalizeReservationScope(gateway.ReservationScope{
		AssigneeAuthProviderID: &blank,
		ActorAuthProviderID:    &actor,
	})

	if scope.AssigneeAuthProviderID != nil {
		t.Fatalf("blank assignee auth provider ID = %q, want nil", *scope.AssigneeAuthProviderID)
	}
	if scope.ActorAuthProviderID == nil || *scope.ActorAuthProviderID != "52ce9940-b846-4ff9-9a7c-94238149ea53" {
		t.Fatalf("actor auth provider ID = %v, want a trimmed UUID", scope.ActorAuthProviderID)
	}

	if actorID := uuidReference(""); actorID != nil {
		t.Fatalf("empty actor auth provider ID = %q, want nil", *actorID)
	}
}
