package domain

import (
	"math"
	"strings"
	"time"
)

type PaymentType string

const (
	PaymentTypeRegular    PaymentType = "pago"
	PaymentTypeSettlement PaymentType = "cancelacion_total"
)

type PaymentMedium string

const (
	PaymentMediumCash     PaymentMedium = "efectivo"
	PaymentMediumTransfer PaymentMedium = "transferencia"
	PaymentMediumCheque   PaymentMedium = "cheque"
	PaymentMediumOther    PaymentMedium = "otro"
)

const (
	MaxPaymentObservationLength = 500
	// PaymentDateTolerance absorbs clock skew between the browser and the
	// server when the caller sends "now" as the payment date.
	PaymentDateTolerance       = 5 * time.Minute
	DefaultDueInstallmentPage  = 1
	DefaultDueInstallmentLimit = 25
	MaxDueInstallmentLimit     = 100
	// DueSoonWindow is how far ahead the vencimientos summary looks for
	// cuotas about to fall due.
	DueSoonWindow = 30 * 24 * time.Hour

	SaleSettledByPaymentReason    = "Plan de pago completado"
	SaleSettledBySettlementReason = "Cancelación anticipada del saldo"
)

var (
	ErrSaleNotFinanced                  = &Error{Kind: KindInvalid, Code: "sale_not_financed", Message: "Una venta al contado no tiene deuda que cobrar"}
	ErrSaleNotActive                    = &Error{Kind: KindConflict, Code: "sale_not_active", Message: "La venta no está activa, no admite cobros"}
	ErrPaymentNothingSelected           = &Error{Kind: KindInvalid, Code: "payment_nothing_selected", Message: "Seleccioná al menos una cuota o la entrega para cobrar"}
	ErrPaymentInstallmentUnknown        = &Error{Kind: KindInvalid, Code: "payment_installment_unknown", Message: "Alguna de las cuotas seleccionadas no pertenece a la venta"}
	ErrPaymentInstallmentPaid           = &Error{Kind: KindConflict, Code: "payment_installment_paid", Message: "Alguna de las cuotas seleccionadas ya está pagada"}
	ErrPaymentInstallmentOrder          = &Error{Kind: KindInvalid, Code: "payment_installment_order", Message: "Las cuotas se cobran en orden: no se puede saltear una pendiente"}
	ErrPaymentDownPaymentPaid           = &Error{Kind: KindConflict, Code: "payment_down_payment_paid", Message: "La entrega ya está cobrada"}
	ErrPaymentNoDownPayment             = &Error{Kind: KindInvalid, Code: "payment_no_down_payment", Message: "La venta no tiene entrega para cobrar"}
	ErrPaymentInvalidMedium             = &Error{Kind: KindInvalid, Code: "invalid_payment_medium", Message: "El medio de pago no es válido"}
	ErrPaymentInvalidDate               = &Error{Kind: KindInvalid, Code: "invalid_payment_date", Message: "La fecha de pago no es válida"}
	ErrPaymentDateInFuture              = &Error{Kind: KindInvalid, Code: "payment_date_in_future", Message: "La fecha de pago no puede ser futura"}
	ErrPaymentObservationTooLong        = &Error{Kind: KindInvalid, Code: "payment_observation_too_long", Message: "La observación no puede superar los 500 caracteres"}
	ErrSettlementNothingOwed            = &Error{Kind: KindConflict, Code: "settlement_nothing_owed", Message: "La venta no tiene saldo pendiente"}
	ErrSettlementAmountMismatch         = &Error{Kind: KindConflict, Code: "settlement_amount_mismatch", Message: "El saldo de la venta cambió; revisá el estado de deuda y volvé a intentar"}
	ErrSettlementExpectedAmountRequired = &Error{Kind: KindInvalid, Code: "settlement_expected_amount_required", Message: "Falta el saldo a cancelar; revisá el estado de deuda y volvé a intentar"}
	ErrDueInstallmentInvalidPage        = &Error{Kind: KindInvalid, Code: "invalid_due_installment_page", Message: "La paginación solicitada no es válida"}
	ErrDueInstallmentInvalidState       = &Error{Kind: KindInvalid, Code: "invalid_due_installment_state", Message: "El estado de cuota no es válido"}
	ErrDueInstallmentInvalidPeriod      = &Error{Kind: KindInvalid, Code: "invalid_due_installment_period", Message: "El rango de vencimientos no es válido"}
)

func (medium PaymentMedium) IsValid() bool {
	switch medium {
	case PaymentMediumCash, PaymentMediumTransfer, PaymentMediumCheque, PaymentMediumOther:
		return true
	default:
		return false
	}
}

func (state InstallmentState) IsValid() bool {
	switch state {
	case InstallmentStatePending, InstallmentStatePaid, InstallmentStateOverdue:
		return true
	default:
		return false
	}
}

// IsCollectionRole reports who may collect on a venta: the same roles that
// register one. An agency user only reaches the ventas of their own agency,
// which the repository enforces through the sale scope.
func IsCollectionRole(role Rol) bool {
	return IsSaleRole(role)
}

// PaymentTerms is what every cobro carries besides what it pays.
type PaymentTerms struct {
	Medium      PaymentMedium
	PaidAt      time.Time
	Observation string
}

// ValidatePaymentTerms checks the medio de pago, that the payment isn't dated
// in the future (beyond a small skew tolerance) and the observación length.
func ValidatePaymentTerms(terms PaymentTerms, now time.Time) error {
	if !terms.Medium.IsValid() {
		return ErrPaymentInvalidMedium
	}
	if terms.PaidAt.IsZero() {
		return ErrPaymentInvalidDate
	}
	if terms.PaidAt.After(now.Add(PaymentDateTolerance)) {
		return ErrPaymentDateInFuture
	}
	if len([]rune(strings.TrimSpace(terms.Observation))) > MaxPaymentObservationLength {
		return ErrPaymentObservationTooLong
	}
	return nil
}

// DownPayment is the entrega of an entrega_financiada sale as the statement
// publishes it. It has no due date: it is collected whenever the client pays
// it, usually when the sale is registered.
type DownPayment struct {
	Amount    float64          `json:"monto"`
	State     InstallmentState `json:"estado"`
	PaidAt    *time.Time       `json:"fechaPago,omitempty"`
	PaymentID string           `json:"cobroId,omitempty"`
}

// Payment is a cobro as the API publishes it: what was collected, over which
// cuotas (and the entrega, when it was included), with which extra charges,
// by whom and when.
//
// Amount and Currency are only what was applied to the plan, always in the
// sale's currency. A charge may be in another currency, so what the client
// handed over is Totals, one amount per currency; nothing is converted.
type Payment struct {
	ID                  string           `json:"id"`
	SaleID              string           `json:"ventaId"`
	Type                PaymentType      `json:"tipo"`
	Amount              float64          `json:"monto"`
	Currency            string           `json:"moneda"`
	Medium              PaymentMedium    `json:"medioPago"`
	Observation         string           `json:"observacion,omitempty"`
	IncludesDownPayment bool             `json:"incluyeEntrega"`
	DownPaymentAmount   float64          `json:"montoEntrega"`
	Installments        []Installment    `json:"cuotas"`
	Charges             []PaymentCharge  `json:"cargos"`
	Totals              []CurrencyTotal  `json:"totales"`
	CreatedBy           ReservationActor `json:"usuarioAlta"`
	PaidAt              time.Time        `json:"fechaPago"`
	CreatedAt           time.Time        `json:"fechaCreacion"`
}

// DebtSummary totals the entrega and the cuotas of a plan: what the client
// owes in all, what was collected, what is still pending and how much of
// that is already overdue.
type DebtSummary struct {
	TotalAmount         float64    `json:"montoTotal"`
	PaidAmount          float64    `json:"montoPagado"`
	PendingAmount       float64    `json:"montoPendiente"`
	OverdueAmount       float64    `json:"montoVencido"`
	PaidInstallments    int        `json:"cuotasPagadas"`
	PendingInstallments int        `json:"cuotasPendientes"`
	OverdueInstallments int        `json:"cuotasVencidas"`
	NextDueDate         *time.Time `json:"proximoVencimiento,omitempty"`
}

// DebtStatement is the estado de deuda of a financed sale: the sale, its
// entrega and cuotas with their current state, the totals and every cobro
// registered so far.
//
// Summary covers the plan only. CollectedCharges is what the cobros added on
// top of it, per currency, since those charges aren't debt of the plan and
// may be in a currency the plan never uses.
type DebtStatement struct {
	Sale             Sale            `json:"venta"`
	DownPayment      *DownPayment    `json:"entrega,omitempty"`
	Installments     []Installment   `json:"cuotas"`
	Summary          DebtSummary     `json:"resumen"`
	Payments         []Payment       `json:"cobros"`
	CollectedCharges []CurrencyTotal `json:"cargosCobrados"`
	IssuedAt         time.Time       `json:"emitidoEl"`
}

// Saldada reports whether nothing is left to collect: every cuota paid and
// the entrega, when there is one, collected.
func (statement DebtStatement) Saldada() bool {
	if statement.DownPayment != nil && statement.DownPayment.State != InstallmentStatePaid {
		return false
	}
	for _, installment := range statement.Installments {
		if installment.Estado != InstallmentStatePaid {
			return false
		}
	}
	return true
}

// StartOfBusinessDay is midnight, in BusinessLocation, of the day t falls on.
func StartOfBusinessDay(t time.Time) time.Time {
	year, month, day := t.In(BusinessLocation).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, BusinessLocation)
}

// EffectiveInstallmentState is the cuota's state as of now: a pendiente
// whose vencimiento already passed is vencida even though the row still says
// pendiente, since nothing rewrites cuotas as time goes by.
//
// Days are compared in BusinessLocation, not instants: fecha_vencimiento
// carries the clock of the sale, but a vencimiento is a date, so a cuota due
// today stays pendiente until the day is over.
func EffectiveInstallmentState(installment Installment, now time.Time) InstallmentState {
	if installment.Estado == InstallmentStatePaid {
		return InstallmentStatePaid
	}
	if installment.FechaVencimiento.Before(StartOfBusinessDay(now)) {
		return InstallmentStateOverdue
	}
	return InstallmentStatePending
}

// ComputeDebtSummary totals the entrega and the cuotas. It expects every
// cuota to already carry its effective state (see EffectiveInstallmentState).
func ComputeDebtSummary(entrega *DownPayment, installments []Installment) DebtSummary {
	var summary DebtSummary
	if entrega != nil {
		summary.TotalAmount += entrega.Amount
		if entrega.State == InstallmentStatePaid {
			summary.PaidAmount += entrega.Amount
		} else {
			summary.PendingAmount += entrega.Amount
		}
	}
	for _, installment := range installments {
		summary.TotalAmount += installment.Monto
		switch installment.Estado {
		case InstallmentStatePaid:
			summary.PaidAmount += installment.Monto
			summary.PaidInstallments++
		case InstallmentStateOverdue:
			summary.PendingAmount += installment.Monto
			summary.OverdueAmount += installment.Monto
			summary.PendingInstallments++
			summary.OverdueInstallments++
		default:
			summary.PendingAmount += installment.Monto
			summary.PendingInstallments++
			if summary.NextDueDate == nil || installment.FechaVencimiento.Before(*summary.NextDueDate) {
				due := installment.FechaVencimiento
				summary.NextDueDate = &due
			}
		}
	}
	summary.TotalAmount = RoundMoney(summary.TotalAmount)
	summary.PaidAmount = RoundMoney(summary.PaidAmount)
	summary.PendingAmount = RoundMoney(summary.PendingAmount)
	summary.OverdueAmount = RoundMoney(summary.OverdueAmount)
	return summary
}

// PaymentSelection is what a cobro pays: the entrega and/or a run of cuotas.
type PaymentSelection struct {
	IncludeDownPayment bool
	Installments       []Installment
	DownPaymentAmount  float64
}

// Amount is what the client hands over for this selection.
func (selection PaymentSelection) Amount() float64 {
	total := 0.0
	if selection.IncludeDownPayment {
		total += selection.DownPaymentAmount
	}
	for _, installment := range selection.Installments {
		total += installment.Monto
	}
	return RoundMoney(total)
}

// SelectPayment resolves which cuotas a cobro pays. Cuotas are collected in
// order, so the selected ids must be exactly the first pending ones: paying
// cuota 3 while 2 is still owed is rejected. installments must be sorted by
// numero, which is how the repository loads them.
func SelectPayment(entrega *DownPayment, installments []Installment, installmentIDs []string, includeDownPayment bool) (PaymentSelection, error) {
	if includeDownPayment {
		if entrega == nil {
			return PaymentSelection{}, ErrPaymentNoDownPayment
		}
		if entrega.State == InstallmentStatePaid {
			return PaymentSelection{}, ErrPaymentDownPaymentPaid
		}
	}
	selected := make(map[string]bool, len(installmentIDs))
	for _, id := range installmentIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		selected[id] = true
	}
	if !includeDownPayment && len(selected) == 0 {
		return PaymentSelection{}, ErrPaymentNothingSelected
	}
	byID := make(map[string]Installment, len(installments))
	for _, installment := range installments {
		byID[installment.ID] = installment
	}
	for id := range selected {
		installment, ok := byID[id]
		if !ok {
			return PaymentSelection{}, ErrPaymentInstallmentUnknown
		}
		if installment.Estado == InstallmentStatePaid {
			return PaymentSelection{}, ErrPaymentInstallmentPaid
		}
	}
	selection := PaymentSelection{IncludeDownPayment: includeDownPayment}
	if includeDownPayment {
		selection.DownPaymentAmount = entrega.Amount
	}
	remaining := len(selected)
	for _, installment := range installments {
		if installment.Estado == InstallmentStatePaid {
			continue
		}
		if remaining == 0 {
			break
		}
		if !selected[installment.ID] {
			return PaymentSelection{}, ErrPaymentInstallmentOrder
		}
		selection.Installments = append(selection.Installments, installment)
		remaining--
	}
	return selection, nil
}

// SelectSettlement resolves a cancelación anticipada: everything still owed,
// entrega included. expectedAmount must match what is owed so a stale screen
// can't collect a different total than the one it showed.
func SelectSettlement(entrega *DownPayment, installments []Installment, expectedAmount float64) (PaymentSelection, error) {
	selection := PaymentSelection{}
	if entrega != nil && entrega.State != InstallmentStatePaid {
		selection.IncludeDownPayment = true
		selection.DownPaymentAmount = entrega.Amount
	}
	for _, installment := range installments {
		if installment.Estado != InstallmentStatePaid {
			selection.Installments = append(selection.Installments, installment)
		}
	}
	if !selection.IncludeDownPayment && len(selection.Installments) == 0 {
		return PaymentSelection{}, ErrSettlementNothingOwed
	}
	if math.Abs(expectedAmount-selection.Amount()) >= 0.005 {
		return PaymentSelection{}, ErrSettlementAmountMismatch
	}
	return selection, nil
}

// DueInstallment is a cuota as the vencimientos list publishes it: the cuota
// plus enough of its venta to know whose it is and where to collect it.
type DueInstallment struct {
	ID               string             `json:"id"`
	SaleID           string             `json:"ventaId"`
	Number           int                `json:"numero"`
	InstallmentCount int                `json:"cantidadCuotas"`
	Amount           float64            `json:"monto"`
	Currency         string             `json:"moneda"`
	State            InstallmentState   `json:"estado"`
	DueDate          time.Time          `json:"fechaVencimiento"`
	PaidAt           *time.Time         `json:"fechaPago,omitempty"`
	DevelopmentID    string             `json:"loteoId"`
	DevelopmentName  string             `json:"loteoNombre"`
	LotID            string             `json:"loteId"`
	LotNumber        string             `json:"loteNumero"`
	BlockNumber      string             `json:"manzanaNumero"`
	Client           Cliente            `json:"cliente"`
	Seller           ReservationActor   `json:"vendedor"`
	Agency           *ReservationAgency `json:"inmobiliaria,omitempty"`
}

// DueInstallmentFilter narrows the vencimientos list. From and To bound the
// fecha_vencimiento (inclusive); an empty States lists every cuota still
// owed (pendiente and vencida).
type DueInstallmentFilter struct {
	States        []InstallmentState
	DevelopmentID string
	Search        string
	From          *time.Time
	To            *time.Time
	Page          int
	Limit         int
}

func (filter DueInstallmentFilter) Normalize() (DueInstallmentFilter, error) {
	filter.DevelopmentID = strings.TrimSpace(filter.DevelopmentID)
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Page == 0 {
		filter.Page = DefaultDueInstallmentPage
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultDueInstallmentLimit
	}
	if filter.Page < 1 || filter.Limit < 1 || filter.Limit > MaxDueInstallmentLimit {
		return DueInstallmentFilter{}, ErrDueInstallmentInvalidPage
	}
	for _, state := range filter.States {
		if !state.IsValid() {
			return DueInstallmentFilter{}, ErrDueInstallmentInvalidState
		}
	}
	if len(filter.States) == 0 {
		filter.States = []InstallmentState{InstallmentStatePending, InstallmentStateOverdue}
	}
	if filter.From != nil && filter.To != nil && filter.To.Before(*filter.From) {
		return DueInstallmentFilter{}, ErrDueInstallmentInvalidPeriod
	}
	return filter, nil
}

// DueSummary is the header of the vencimientos list over the whole scope
// (not just the page): how many cuotas are overdue and how many fall due
// within DueSoonWindow. Amounts aren't totalled because ventas are priced
// in whatever currency the lote had, so a sum across them means nothing.
type DueSummary struct {
	OverdueInstallments  int `json:"cuotasVencidas"`
	UpcomingInstallments int `json:"cuotasProximas"`
}

type DueInstallmentPage struct {
	Items      []DueInstallment `json:"cuotas"`
	Summary    DueSummary       `json:"resumen"`
	Page       int              `json:"pagina"`
	Limit      int              `json:"porPagina"`
	Total      int              `json:"total"`
	TotalPages int              `json:"paginas"`
}
