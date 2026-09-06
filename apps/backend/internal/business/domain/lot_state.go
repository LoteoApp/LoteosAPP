package domain

import (
	"strings"
	"time"
)

type LotState string

const (
	LotStateAvailable LotState = "disponible"
	LotStateReserved  LotState = "reservado"
	LotStateSold      LotState = "vendido"
	LotStateCompleted LotState = "finalizado"
)

type LotStateOrigin string

const (
	LotStateOriginCreation    LotStateOrigin = "alta"
	LotStateOriginReservation LotStateOrigin = "reserva"
	LotStateOriginSale        LotStateOrigin = "venta"
	LotStateOriginCollection  LotStateOrigin = "cobranza"
	LotStateOriginSystem      LotStateOrigin = "sistema"
	LotStateOriginCorrection  LotStateOrigin = "correccion"
)

var (
	ErrInvalidLotState = &Error{
		Kind: KindInvalid, Code: "invalid_lot_state", Message: "El estado del lote no es v\u00e1lido",
	}
	ErrInvalidLotStateOrigin = &Error{
		Kind: KindInvalid, Code: "invalid_lot_state_origin", Message: "El origen del cambio de estado no es v\u00e1lido",
	}
	ErrInvalidLotStateTransition = &Error{
		Kind: KindInvalid, Code: "invalid_lot_state_transition", Message: "El cambio de estado solicitado no est\u00e1 permitido",
	}
	ErrLotStateConflict = &Error{
		Kind: KindConflict, Code: "lot_state_conflict", Message: "El lote cambi\u00f3 de estado; actualiz\u00e1 la informaci\u00f3n e intent\u00e1 nuevamente",
	}
	ErrLotStateReasonRequired = &Error{
		Kind: KindInvalid, Code: "lot_state_reason_required", Message: "La justificaci\u00f3n del cambio de estado es obligatoria",
	}
	ErrLotStateReferenceRequired = &Error{
		Kind: KindInvalid, Code: "lot_state_reference_required", Message: "El cambio de estado requiere la referencia de la operaci\u00f3n que lo origin\u00f3",
	}
	ErrInvalidLotStateReference = &Error{
		Kind: KindInvalid, Code: "invalid_lot_state_reference", Message: "La operaci\u00f3n referenciada no corresponde al lote",
	}
)

func (state LotState) IsValid() bool {
	switch state {
	case LotStateAvailable, LotStateReserved, LotStateSold, LotStateCompleted:
		return true
	default:
		return false
	}
}

func (state LotState) CanTransitionTo(next LotState) bool {
	switch state {
	case LotStateAvailable:
		return next == LotStateReserved || next == LotStateSold
	case LotStateReserved:
		return next == LotStateAvailable || next == LotStateSold
	case LotStateSold:
		return next == LotStateAvailable || next == LotStateCompleted
	default:
		return false
	}
}

func (origin LotStateOrigin) IsValid() bool {
	switch origin {
	case LotStateOriginCreation,
		LotStateOriginReservation,
		LotStateOriginSale,
		LotStateOriginCollection,
		LotStateOriginSystem,
		LotStateOriginCorrection:
		return true
	default:
		return false
	}
}

type LotStateTransition struct {
	DevelopmentID string
	LotID         string
	ExpectedState LotState
	NextState     LotState
	Origin        LotStateOrigin
	Reason        string
	ActorID       string
	ReservationID *string
	SaleID        *string
}

func (transition LotStateTransition) Validate() error {
	if !transition.ExpectedState.IsValid() || !transition.NextState.IsValid() {
		return ErrInvalidLotState
	}
	if !transition.ExpectedState.CanTransitionTo(transition.NextState) {
		return ErrInvalidLotStateTransition
	}
	if !transition.Origin.IsValid() || !transition.originMatchesTransition() {
		return ErrInvalidLotStateOrigin
	}
	if transition.requiresReason() && strings.TrimSpace(transition.Reason) == "" {
		return ErrLotStateReasonRequired
	}
	if transition.Origin == LotStateOriginReservation && emptyReference(transition.ReservationID) {
		return ErrLotStateReferenceRequired
	}
	if (transition.Origin == LotStateOriginSale || transition.Origin == LotStateOriginCollection) && emptyReference(transition.SaleID) {
		return ErrLotStateReferenceRequired
	}

	return nil
}

func (transition LotStateTransition) originMatchesTransition() bool {
	switch {
	case transition.ExpectedState == LotStateAvailable && transition.NextState == LotStateReserved:
		return transition.Origin == LotStateOriginReservation
	case transition.ExpectedState == LotStateAvailable && transition.NextState == LotStateSold:
		return transition.Origin == LotStateOriginSale
	case transition.ExpectedState == LotStateReserved && transition.NextState == LotStateAvailable:
		return transition.Origin == LotStateOriginReservation || transition.Origin == LotStateOriginSystem
	case transition.ExpectedState == LotStateReserved && transition.NextState == LotStateSold:
		return transition.Origin == LotStateOriginSale
	case transition.ExpectedState == LotStateSold && transition.NextState == LotStateAvailable:
		return transition.Origin == LotStateOriginSale || transition.Origin == LotStateOriginSystem
	case transition.ExpectedState == LotStateSold && transition.NextState == LotStateCompleted:
		return transition.Origin == LotStateOriginCollection
	default:
		return false
	}
}

func (transition LotStateTransition) requiresReason() bool {
	return transition.NextState == LotStateAvailable && transition.Origin != LotStateOriginSystem
}

func emptyReference(reference *string) bool {
	return reference == nil || strings.TrimSpace(*reference) == ""
}

type LotStateEvent struct {
	ID            string
	LotID         string
	PreviousState LotState
	State         LotState
	Origin        LotStateOrigin
	Reason        string
	ActorID       string
	ReservationID *string
	SaleID        *string
	OccurredAt    time.Time
}
