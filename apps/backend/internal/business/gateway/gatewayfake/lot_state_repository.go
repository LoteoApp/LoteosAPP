package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type LotStateRepository struct {
	LotExistsCalls        int
	LotExistsResult       bool
	LotExistsErr          error
	ReceivedDevelopmentID string
	ReceivedLotID         string
	ReceivedLotScope      gateway.LoteoScope

	TransitionCalls  int
	TransitionResult domain.LotStateEvent
	TransitionErr    error
	ReceivedCommand  domain.LotStateTransition
}

func (fake *LotStateRepository) LotExists(
	_ context.Context,
	developmentID, lotID string,
	scope gateway.LoteoScope,
) (bool, error) {
	fake.LotExistsCalls++
	fake.ReceivedDevelopmentID = developmentID
	fake.ReceivedLotID = lotID
	fake.ReceivedLotScope = scope
	if fake.LotExistsErr != nil {
		return false, fake.LotExistsErr
	}

	return fake.LotExistsResult, nil
}

func (fake *LotStateRepository) Transition(
	_ context.Context,
	command domain.LotStateTransition,
) (domain.LotStateEvent, error) {
	fake.TransitionCalls++
	fake.ReceivedCommand = command
	if fake.TransitionErr != nil {
		return domain.LotStateEvent{}, fake.TransitionErr
	}

	return fake.TransitionResult, nil
}
