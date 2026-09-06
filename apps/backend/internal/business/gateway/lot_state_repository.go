package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

type LotStateRepository interface {
	LotExists(ctx context.Context, developmentID, lotID string, scope LoteoScope) (bool, error)
	Transition(ctx context.Context, command domain.LotStateTransition) (domain.LotStateEvent, error)
}
