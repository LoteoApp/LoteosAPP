package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type ReservationState string

const (
	ReservationStateActive    ReservationState = "activa"
	ReservationStateExpired   ReservationState = "vencida"
	ReservationStateCancelled ReservationState = "cancelada"
	ReservationStateConverted ReservationState = "convertida"
)

const (
	ReservationDuration      = 360 * time.Hour
	MaxReservationReasonSize = 500
	MaxIdempotencyKeySize    = 128
	DefaultReservationPage   = 1
	DefaultReservationLimit  = 25
	MaxReservationLimit      = 100
)

var (
	ErrReservationNotFound            = &Error{Kind: KindNotFound, Code: "reservation_not_found", Message: "La reserva solicitada no existe"}
	ErrReservationInvalidClient       = &Error{Kind: KindInvalid, Code: "invalid_reservation_client", Message: "El cliente seleccionado no es válido"}
	ErrReservationClientInactive      = &Error{Kind: KindConflict, Code: "reservation_client_inactive", Message: "El cliente seleccionado está dado de baja"}
	ErrReservationSellerRequired      = &Error{Kind: KindInvalid, Code: "reservation_seller_required", Message: "Tenés que seleccionar un vendedor"}
	ErrReservationSellerNotEligible   = &Error{Kind: KindForbidden, Code: "reservation_seller_not_eligible", Message: "El vendedor no está habilitado para este loteo"}
	ErrReservationLotUnavailable      = &Error{Kind: KindConflict, Code: "reservation_lot_unavailable", Message: "El lote no está disponible para reservar"}
	ErrReservationLotIncomplete       = &Error{Kind: KindConflict, Code: "reservation_lot_incomplete", Message: "El lote necesita número y precio para reservar"}
	ErrReservationActiveConflict      = &Error{Kind: KindConflict, Code: "reservation_already_active", Message: "El lote ya tiene una reserva activa"}
	ErrReservationReasonRequired      = &Error{Kind: KindInvalid, Code: "reservation_reason_required", Message: "La justificación es obligatoria"}
	ErrReservationReasonTooLong       = &Error{Kind: KindInvalid, Code: "reservation_reason_too_long", Message: "La justificación no puede superar los 500 caracteres"}
	ErrReservationIdempotencyRequired = &Error{Kind: KindInvalid, Code: "idempotency_key_required", Message: "La solicitud necesita una clave de idempotencia"}
	ErrReservationIdempotencyConflict = &Error{Kind: KindConflict, Code: "idempotency_key_conflict", Message: "La clave de idempotencia ya fue utilizada con otros datos"}
	ErrReservationExpired             = &Error{Kind: KindConflict, Code: "reservation_expired", Message: "La reserva ya venció y el lote fue liberado"}
	ErrReservationConverted           = &Error{Kind: KindConflict, Code: "reservation_converted", Message: "La reserva ya fue convertida en una venta"}
	ErrReservationAlreadyCancelled    = &Error{Kind: KindConflict, Code: "reservation_already_cancelled", Message: "La reserva ya fue cancelada"}
	ErrReservationInvalidState        = &Error{Kind: KindInvalid, Code: "invalid_reservation_state", Message: "El estado de la reserva no es válido"}
	ErrReservationInvalidPage         = &Error{Kind: KindInvalid, Code: "invalid_reservation_page", Message: "La paginación solicitada no es válida"}
)

func (state ReservationState) IsValid() bool {
	switch state {
	case ReservationStateActive, ReservationStateExpired, ReservationStateCancelled, ReservationStateConverted:
		return true
	default:
		return false
	}
}

func (state ReservationState) IsTerminal() bool {
	return state == ReservationStateExpired || state == ReservationStateCancelled || state == ReservationStateConverted
}

func (state ReservationState) CanTransitionTo(next ReservationState) bool {
	return state == ReservationStateActive && next.IsTerminal()
}

func IsReservationRole(role Rol) bool {
	return role == RolAdministrador || role == RolAdministrativo || role == RolInmobiliaria
}

func IsAdministrativeRole(role Rol) bool {
	return role == RolAdministrador || role == RolAdministrativo
}

func CanCancelReservation(actorRole Rol, actorID, sellerID string, sameAgency, agencyAssigned bool) bool {
	if IsAdministrativeRole(actorRole) || actorID == sellerID {
		return true
	}
	return actorRole == RolInmobiliaria && sameAgency && agencyAssigned
}

func CanCancelReservationForAgency(actorRole Rol, actorAgency, reservationAgency *string) bool {
	if IsAdministrativeRole(actorRole) {
		return true
	}
	return actorRole == RolInmobiliaria && samePointer(actorAgency, reservationAgency)
}

func samePointer(left, right *string) bool {
	return left != nil && right != nil && *left == *right
}

func ValidateReservationReason(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrReservationReasonRequired
	}
	if utf8.RuneCountInString(reason) > MaxReservationReasonSize {
		return ErrReservationReasonTooLong
	}
	return nil
}

func NormalizeIdempotencyKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", ErrReservationIdempotencyRequired
	}
	if utf8.RuneCountInString(key) > MaxIdempotencyKeySize {
		return "", ErrReservationIdempotencyRequired
	}
	return key, nil
}

type ReservationActor struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	Email    string `json:"email,omitempty"`
	Rol      Rol    `json:"rol"`
}

type ReservationAgency struct {
	ID           string `json:"id"`
	BusinessName string `json:"razonSocial"`
}

type ReservationHistoryEntry struct {
	ID      string            `json:"id"`
	Estado  ReservationState  `json:"estado"`
	Razon   string            `json:"razon,omitempty"`
	Usuario *ReservationActor `json:"usuario,omitempty"`
	Fecha   time.Time         `json:"fecha"`
}

type Reservation struct {
	ID                string                    `json:"id"`
	LoteoID           string                    `json:"loteoId"`
	LoteoNombre       string                    `json:"loteoNombre"`
	LoteID            string                    `json:"loteId"`
	LoteNumero        string                    `json:"loteNumero"`
	Cliente           Cliente                   `json:"cliente"`
	Vendedor          ReservationActor          `json:"vendedor"`
	UsuarioAlta       ReservationActor          `json:"usuarioAlta"`
	Inmobiliaria      *ReservationAgency        `json:"inmobiliaria,omitempty"`
	Estado            ReservationState          `json:"estado"`
	PuedeCancelar     bool                      `json:"puedeCancelar"`
	FechaVencimiento  time.Time                 `json:"fechaVencimiento"`
	FechaCreacion     time.Time                 `json:"fechaCreacion"`
	FechaModificacion time.Time                 `json:"fechaModificacion"`
	Historial         []ReservationHistoryEntry `json:"historial,omitempty"`
}

type ReservationStateFilter []ReservationState

type ReservationListFilter struct {
	States  ReservationStateFilter
	LoteoID string
	LoteID  string
	Search  string
	Page    int
	Limit   int
}

func (filter ReservationListFilter) Normalize() (ReservationListFilter, error) {
	filter.LoteoID = strings.TrimSpace(filter.LoteoID)
	filter.LoteID = strings.TrimSpace(filter.LoteID)
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Page == 0 {
		filter.Page = DefaultReservationPage
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultReservationLimit
	}
	if filter.Page < 1 || filter.Limit < 1 || filter.Limit > MaxReservationLimit {
		return ReservationListFilter{}, ErrReservationInvalidPage
	}
	for _, state := range filter.States {
		if !state.IsValid() {
			return ReservationListFilter{}, ErrReservationInvalidState
		}
	}
	return filter, nil
}

type ReservationPage struct {
	Items      []Reservation `json:"reservas"`
	Page       int           `json:"pagina"`
	Limit      int           `json:"porPagina"`
	Total      int           `json:"total"`
	TotalPages int           `json:"paginas"`
}

type SellerOption struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	Email    string `json:"email,omitempty"`
	Rol      Rol    `json:"rol"`
}

type ExpirationFailure struct {
	ReservationID string
	Cause         error
}

type ExpirationReport struct {
	Candidates int
	Processed  int
	Skipped    int
	Failures   []ExpirationFailure
}
