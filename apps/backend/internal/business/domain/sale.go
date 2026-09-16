package domain

import (
	"strings"
	"time"
)

type SaleState string

const (
	SaleStateActive    SaleState = "activa"
	SaleStateCompleted SaleState = "completada"
	SaleStateCancelled SaleState = "cancelada"
)

type PaymentMethod string

const (
	PaymentMethodCash      PaymentMethod = "contado"
	PaymentMethodFinanced  PaymentMethod = "financiado"
	PaymentMethodDownAndFi PaymentMethod = "entrega_financiada"
)

const (
	DefaultSalePage  = 1
	DefaultSaleLimit = 25
	MaxSaleLimit     = 100
)

var (
	ErrSaleNotFound                 = &Error{Kind: KindNotFound, Code: "sale_not_found", Message: "La venta solicitada no existe"}
	ErrSaleInvalidClient            = &Error{Kind: KindInvalid, Code: "invalid_sale_client", Message: "El cliente seleccionado no es válido"}
	ErrSaleSellerRequired           = &Error{Kind: KindInvalid, Code: "sale_seller_required", Message: "Tenés que seleccionar un vendedor"}
	ErrSaleSellerNotEligible        = &Error{Kind: KindForbidden, Code: "sale_seller_not_eligible", Message: "El vendedor no está habilitado para vender"}
	ErrSaleAgencyNotAssigned        = &Error{Kind: KindForbidden, Code: "sale_agency_not_assigned", Message: "Tu inmobiliaria no está asignada a este loteo"}
	ErrSaleLotUnavailable           = &Error{Kind: KindConflict, Code: "sale_lot_unavailable", Message: "El lote no está disponible para vender"}
	ErrSaleLotIncomplete            = &Error{Kind: KindConflict, Code: "sale_lot_incomplete", Message: "El lote necesita número y precio para vender"}
	ErrSaleActiveConflict           = &Error{Kind: KindConflict, Code: "sale_already_active", Message: "El lote ya tiene una venta registrada"}
	ErrSaleInvalidPaymentMethod     = &Error{Kind: KindInvalid, Code: "invalid_payment_method", Message: "La modalidad de pago no es válida"}
	ErrSalePaymentMethodUnavailable = &Error{Kind: KindInvalid, Code: "payment_method_unavailable", Message: "Por ahora solo se puede registrar una venta al contado"}
	ErrSaleInvalidState             = &Error{Kind: KindInvalid, Code: "invalid_sale_state", Message: "El estado de la venta no es válido"}
	ErrSaleInvalidPage              = &Error{Kind: KindInvalid, Code: "invalid_sale_page", Message: "La paginación solicitada no es válida"}
	ErrSaleIdempotencyRequired      = &Error{Kind: KindInvalid, Code: "idempotency_key_required", Message: "La solicitud necesita una clave de idempotencia"}
	ErrSaleIdempotencyConflict      = &Error{Kind: KindConflict, Code: "idempotency_key_conflict", Message: "La clave de idempotencia ya fue utilizada con otros datos"}
)

func (state SaleState) IsValid() bool {
	switch state {
	case SaleStateActive, SaleStateCompleted, SaleStateCancelled:
		return true
	default:
		return false
	}
}

func (method PaymentMethod) IsValid() bool {
	switch method {
	case PaymentMethodCash, PaymentMethodFinanced, PaymentMethodDownAndFi:
		return true
	default:
		return false
	}
}

// IsAvailable reports whether the app can register a sale with this method
// yet. Only contado is implemented; the other two exist in the contract so
// the selector already lists them.
func (method PaymentMethod) IsAvailable() bool {
	return method == PaymentMethodCash
}

// IsSaleRole reports who may register a venta: internal users for any loteo,
// and agency users only for a loteo their agency is assigned to, which the
// repository checks since the domain doesn't know the assignments.
func IsSaleRole(role Rol) bool {
	return IsReservationRole(role)
}

type SaleHistoryEntry struct {
	ID      string            `json:"id"`
	Estado  SaleState         `json:"estado"`
	Razon   string            `json:"razon,omitempty"`
	Usuario *ReservationActor `json:"usuario,omitempty"`
	Fecha   time.Time         `json:"fecha"`
}

// Sale is a venta as the API publishes it. The seller's agency is read from
// usuarios.inmobiliaria_id, so Inmobiliaria is nil for a venta directa by an
// internal user.
type Sale struct {
	ID                string             `json:"id"`
	LoteoID           string             `json:"loteoId"`
	LoteoNombre       string             `json:"loteoNombre"`
	LoteID            string             `json:"loteId"`
	LoteNumero        string             `json:"loteNumero"`
	ManzanaNumero     string             `json:"manzanaNumero"`
	LoteSuperficie    *float64           `json:"loteSuperficie"`
	Cliente           Cliente            `json:"cliente"`
	Vendedor          ReservationActor   `json:"vendedor"`
	UsuarioAlta       ReservationActor   `json:"usuarioAlta"`
	Inmobiliaria      *ReservationAgency `json:"inmobiliaria,omitempty"`
	ModalidadPago     PaymentMethod      `json:"modalidadPago"`
	Monto             float64            `json:"monto"`
	Moneda            string             `json:"moneda"`
	Estado            SaleState          `json:"estado"`
	FechaCreacion     time.Time          `json:"fechaCreacion"`
	FechaModificacion time.Time          `json:"fechaModificacion"`
	Historial         []SaleHistoryEntry `json:"historial,omitempty"`
}

type SaleListFilter struct {
	States        []SaleState
	DevelopmentID string
	LotID         string
	Search        string
	Page          int
	Limit         int
}

func (filter SaleListFilter) Normalize() (SaleListFilter, error) {
	filter.DevelopmentID = strings.TrimSpace(filter.DevelopmentID)
	filter.LotID = strings.TrimSpace(filter.LotID)
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Page == 0 {
		filter.Page = DefaultSalePage
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultSaleLimit
	}
	if filter.Page < 1 || filter.Limit < 1 || filter.Limit > MaxSaleLimit {
		return SaleListFilter{}, ErrSaleInvalidPage
	}
	for _, state := range filter.States {
		if !state.IsValid() {
			return SaleListFilter{}, ErrSaleInvalidState
		}
	}
	return filter, nil
}

type SalePage struct {
	Items      []Sale `json:"ventas"`
	Page       int    `json:"pagina"`
	Limit      int    `json:"porPagina"`
	Total      int    `json:"total"`
	TotalPages int    `json:"paginas"`
}
