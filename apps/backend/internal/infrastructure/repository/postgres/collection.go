package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type CollectionRepository struct {
	pool *pgxpool.Pool
}

func NewCollectionRepository(pool *pgxpool.Pool) *CollectionRepository {
	return &CollectionRepository{pool: pool}
}

// planRow is the collectable side of a plan de pago: the entrega and its
// cobro, when the plan has one.
type planRow struct {
	id                string
	downPayment       *float64
	deliveredAt       *time.Time
	deliveryPaymentID *string
}

func (row planRow) entrega() *domain.DownPayment {
	if row.downPayment == nil || *row.downPayment <= 0 {
		return nil
	}
	entrega := &domain.DownPayment{Monto: *row.downPayment, Estado: domain.InstallmentStatePending}
	if row.deliveredAt != nil {
		entrega.Estado = domain.InstallmentStatePaid
		entrega.FechaPago = row.deliveredAt
		entrega.CobroID = stringValue(row.deliveryPaymentID)
	}
	return entrega
}

func (repository *CollectionRepository) GetDebtStatement(ctx context.Context, saleID string, scope gateway.SaleScope, now time.Time) (domain.DebtStatement, error) {
	sale, err := loadSale(ctx, repository.pool, saleID, scope, false)
	if err != nil {
		return domain.DebtStatement{}, err
	}
	if sale.PlanPago == nil {
		return domain.DebtStatement{}, domain.ErrSaleNotFinanced
	}
	plan, err := loadPlanRow(ctx, repository.pool, sale.PlanPago.ID)
	if err != nil {
		return domain.DebtStatement{}, err
	}
	installments, err := loadInstallmentsAsOf(ctx, repository.pool, plan.id, now, false)
	if err != nil {
		return domain.DebtStatement{}, err
	}
	payments, err := loadPayments(ctx, repository.pool, sale.ID, plan)
	if err != nil {
		return domain.DebtStatement{}, err
	}
	entrega := plan.entrega()
	charges := make([]domain.PaymentCharge, 0)
	for _, payment := range payments {
		charges = append(charges, payment.Cargos...)
	}
	return domain.DebtStatement{
		Venta:          sale,
		Entrega:        entrega,
		Cuotas:         installments,
		Resumen:        domain.ComputeDebtSummary(entrega, installments),
		Cobros:         payments,
		CargosCobrados: domain.ChargeTotals(charges),
		EmitidoEl:      now,
	}, nil
}

// dueInstallmentState is the cuota's state as of $now, the same rule
// domain.EffectiveInstallmentState applies, so the list can filter on it.
const dueInstallmentState = `CASE
	WHEN c.estado = 'pagada' THEN 'pagada'
	WHEN c.fecha_vencimiento < %[1]s THEN 'vencida'
	ELSE 'pendiente'
END`

const dueInstallmentFromClause = `
	FROM cuotas c
	JOIN planes_pago plan ON plan.id = c.plan_pago_id AND plan.fecha_baja IS NULL
	JOIN ventas v ON v.id = plan.venta_id AND v.estado_actual <> 'cancelada'
	JOIN lotes lo ON lo.id = v.lote_id AND lo.fecha_baja IS NULL
	JOIN loteos l ON l.id = lo.loteo_id AND l.fecha_baja IS NULL
	LEFT JOIN manzanas mz ON mz.id = lo.manzana_id
	JOIN clientes cl ON cl.id = v.cliente_id
	JOIN usuarios seller ON seller.id = v.vendedor_id
	LEFT JOIN inmobiliarias agency ON agency.id = seller.inmobiliaria_id`

func (repository *CollectionRepository) ListDueInstallments(ctx context.Context, filter domain.DueInstallmentFilter, scope gateway.SaleScope, now time.Time) (domain.DueInstallmentPage, error) {
	scope = normalizeSaleScope(scope)
	var err error
	filter, err = filter.Normalize()
	if err != nil {
		return domain.DueInstallmentPage{}, err
	}
	states := make([]string, len(filter.States))
	for i, state := range filter.States {
		states[i] = string(state)
	}
	page := domain.DueInstallmentPage{Page: filter.Page, Limit: filter.Limit, Items: []domain.DueInstallment{}}
	offset := (filter.Page - 1) * filter.Limit
	stateExpr := fmt.Sprintf(dueInstallmentState, "$1::timestamptz")
	where := `
		WHERE (` + stateExpr + `) = ANY($2::text[])
		  AND ($3 = '' OR l.id::text = $3)
		  AND ($4 = '' OR l.nombre ILIKE $5 ESCAPE '\'
		       OR COALESCE(lo.numero, '') ILIKE $5 ESCAPE '\'
		       OR cl.nombre ILIKE $5 ESCAPE '\' OR cl.apellido ILIKE $5 ESCAPE '\'
		       OR cl.dni ILIKE $5 ESCAPE '\'
		       OR seller.nombre ILIKE $5 ESCAPE '\' OR seller.apellido ILIKE $5 ESCAPE '\')
		  AND ($6::timestamptz IS NULL OR c.fecha_vencimiento >= $6::timestamptz)
		  AND ($7::timestamptz IS NULL OR c.fecha_vencimiento <= $7::timestamptz)
		  AND ` + fmt.Sprintf(saleScopePredicate, 8, 9)
	args := []any{now, states, filter.DevelopmentID, filter.Search, containsPattern(filter.Search),
		filter.From, filter.To, scope.AssigneeAuthProviderID, scope.ByAgency}
	rows, err := repository.pool.Query(ctx, `
		SELECT c.id::text, v.id::text, c.numero, plan.cantidad_cuotas, c.monto::float8, plan.moneda,
		       `+stateExpr+`, c.fecha_vencimiento, c.fecha_pago,
		       l.id::text, l.nombre, lo.id::text, COALESCE(lo.numero, ''), COALESCE(mz.numero, ''),
		       cl.id::text, cl.nombre, cl.apellido, cl.dni, cl.celular, cl.email,
		       seller.id::text, seller.nombre, seller.apellido, seller.email, seller.rol,
		       agency.id::text, COALESCE(agency.razon_social, ''),
		       count(*) OVER()
		`+dueInstallmentFromClause+where+`
		ORDER BY c.fecha_vencimiento, c.numero, c.id
		LIMIT $10 OFFSET $11
	`, append(args, filter.Limit, offset)...)
	if err != nil {
		return domain.DueInstallmentPage{}, err
	}
	defer rows.Close()
	hasTotal := false
	for rows.Next() {
		var (
			item       domain.DueInstallment
			agencyID   *string
			agencyName string
			total      int64
		)
		if err := rows.Scan(&item.ID, &item.VentaID, &item.Numero, &item.CantidadCuotas, &item.Monto, &item.Moneda,
			&item.Estado, &item.FechaVencimiento, &item.FechaPago,
			&item.LoteoID, &item.LoteoNombre, &item.LoteID, &item.LoteNumero, &item.ManzanaNumero,
			&item.Cliente.ID, &item.Cliente.Nombre, &item.Cliente.Apellido, &item.Cliente.DNI, &item.Cliente.Celular, &item.Cliente.Email,
			&item.Vendedor.ID, &item.Vendedor.Nombre, &item.Vendedor.Apellido, &item.Vendedor.Email, &item.Vendedor.Rol,
			&agencyID, &agencyName, &total); err != nil {
			return domain.DueInstallmentPage{}, err
		}
		if agencyID != nil {
			item.Inmobiliaria = &domain.ReservationAgency{ID: *agencyID, BusinessName: agencyName}
		}
		page.Items = append(page.Items, item)
		page.Total = int(total)
		hasTotal = true
	}
	if err := rows.Err(); err != nil {
		return domain.DueInstallmentPage{}, err
	}
	if !hasTotal {
		if err := repository.pool.QueryRow(ctx, `SELECT count(*) `+dueInstallmentFromClause+where, args...).Scan(&page.Total); err != nil {
			return domain.DueInstallmentPage{}, err
		}
	}
	page.TotalPages = (page.Total + page.Limit - 1) / page.Limit

	// The header counts over the whole scope (and loteo, when one is
	// picked), not over the filtered page, so it reads the same whatever
	// the user is looking at.
	err = repository.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE c.fecha_vencimiento < $1::timestamptz),
		       count(*) FILTER (WHERE c.fecha_vencimiento >= $1::timestamptz AND c.fecha_vencimiento < $2::timestamptz)
		`+dueInstallmentFromClause+`
		WHERE c.estado <> 'pagada'
		  AND ($3 = '' OR l.id::text = $3)
		  AND `+fmt.Sprintf(saleScopePredicate, 4, 5),
		now, now.Add(domain.DueSoonWindow), filter.DevelopmentID, scope.AssigneeAuthProviderID, scope.ByAgency,
	).Scan(&page.Resumen.CuotasVencidas, &page.Resumen.CuotasProximas)
	if err != nil {
		return domain.DueInstallmentPage{}, err
	}
	return page, nil
}

// RegisterPayment collects in one transaction: the venta, its plan and its
// cuotas are locked first, so two cobros over the same venta serialize and
// the second one sees what the first already paid.
func (repository *CollectionRepository) RegisterPayment(ctx context.Context, command gateway.RegisterPaymentCommand, scope gateway.SaleScope) (domain.Payment, error) {
	scope = normalizeSaleScope(scope)
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Payment{}, err
	}
	defer rollbackTransaction(tx)

	var saleID, saleState, currency, lotID, developmentID string
	err = tx.QueryRow(ctx, `
		SELECT v.id::text, v.estado_actual, v.moneda, lo.id::text, l.id::text
		`+saleFromClause+`
		WHERE v.id = $1::uuid AND `+fmt.Sprintf(saleScopePredicate, 2, 3)+`
		FOR UPDATE OF v
	`, command.SaleID, scope.AssigneeAuthProviderID, scope.ByAgency).Scan(&saleID, &saleState, &currency, &lotID, &developmentID)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Payment{}, domain.ErrSaleNotFound
	}
	if err != nil {
		return domain.Payment{}, err
	}
	if domain.SaleState(saleState) != domain.SaleStateActive {
		return domain.Payment{}, domain.ErrSaleNotActive
	}
	plan, err := loadPlanRowBySale(ctx, tx, saleID)
	if err != nil {
		return domain.Payment{}, err
	}
	installments, err := loadInstallmentsAsOf(ctx, tx, plan.id, command.Now, true)
	if err != nil {
		return domain.Payment{}, err
	}
	entrega := plan.entrega()

	var selection domain.PaymentSelection
	if command.Type == domain.PaymentTypeSettlement {
		selection, err = domain.SelectSettlement(entrega, installments, command.ExpectedAmount)
	} else {
		selection, err = domain.SelectPayment(entrega, installments, command.InstallmentIDs, command.IncludeDownPayment)
	}
	if err != nil {
		return domain.Payment{}, err
	}

	var paymentID string
	err = tx.QueryRow(ctx, `
		INSERT INTO cobros (venta_id, tipo, monto, moneda, medio_pago, observacion, usuario_alta, fecha_pago, fecha_creacion)
		VALUES ($1::uuid, $2, $3, $4, $5, NULLIF($6, ''), $7::uuid, $8, $9)
		RETURNING id::text
	`, saleID, string(command.Type), selection.Amount(), currency, string(command.Medium), command.Observation,
		command.ActorID, command.PaidAt, command.Now).Scan(&paymentID)
	if err != nil {
		return domain.Payment{}, err
	}
	if err := insertPaymentCharges(ctx, tx, paymentID, currency, command); err != nil {
		return domain.Payment{}, err
	}
	if len(selection.Installments) > 0 {
		ids := make([]string, len(selection.Installments))
		for i, installment := range selection.Installments {
			ids[i] = installment.ID
		}
		tag, err := tx.Exec(ctx, `
			UPDATE cuotas
			SET estado = 'pagada', fecha_pago = $2, cobro_id = $3::uuid,
			    usuario_modificacion = $4::uuid, fecha_modificacion = $5
			WHERE id = ANY($1::uuid[]) AND plan_pago_id = $6::uuid AND estado <> 'pagada'
		`, ids, command.PaidAt, paymentID, command.ActorID, command.Now, plan.id)
		if err != nil {
			return domain.Payment{}, err
		}
		if int(tag.RowsAffected()) != len(ids) {
			return domain.Payment{}, domain.ErrPaymentInstallmentPaid
		}
	}
	if selection.IncludeDownPayment {
		tag, err := tx.Exec(ctx, `
			UPDATE planes_pago
			SET fecha_entrega = $2, cobro_entrega_id = $3::uuid,
			    usuario_modificacion = $4::uuid, fecha_modificacion = $5
			WHERE id = $1::uuid AND cobro_entrega_id IS NULL AND fecha_entrega IS NULL
		`, plan.id, command.PaidAt, paymentID, command.ActorID, command.Now)
		if err != nil {
			return domain.Payment{}, err
		}
		if tag.RowsAffected() != 1 {
			return domain.Payment{}, domain.ErrPaymentDownPaymentPaid
		}
	}

	var remaining int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM cuotas WHERE plan_pago_id = $1::uuid AND estado <> 'pagada'
	`, plan.id).Scan(&remaining); err != nil {
		return domain.Payment{}, err
	}
	entregaSettled := entrega == nil || entrega.Estado == domain.InstallmentStatePaid || selection.IncludeDownPayment
	if remaining == 0 && entregaSettled {
		reason := domain.SaleSettledByPaymentReason
		if command.Type == domain.PaymentTypeSettlement {
			reason = domain.SaleSettledBySettlementReason
		}
		if err := completeSale(ctx, tx, saleID, developmentID, lotID, command.ActorID, reason); err != nil {
			return domain.Payment{}, err
		}
	}

	payment, err := loadPayment(ctx, tx, paymentID)
	if err != nil {
		return domain.Payment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Payment{}, err
	}
	return payment, nil
}

// completeSale closes a venta whose plan was collected in full: completada
// in venta_estados and the lote from vendido to finalizado with origin
// cobranza. The lote row is locked here since the cobro didn't need it
// until now.
func completeSale(ctx context.Context, tx pgx.Tx, saleID, developmentID, lotID, actorID, reason string) error {
	var lotState domain.LotState
	err := tx.QueryRow(ctx, `
		SELECT estado_actual FROM lotes WHERE id = $1::uuid AND fecha_baja IS NULL FOR UPDATE
	`, lotID).Scan(&lotState)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrLoteNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO venta_estados (venta_id, estado, razon, usuario_modificacion, fecha_creacion)
		VALUES ($1::uuid, $2, $3, $4::uuid, clock_timestamp())
	`, saleID, string(domain.SaleStateCompleted), reason, actorID); err != nil {
		return err
	}
	_, err = transitionLotStateWithLockedLot(ctx, tx, domain.LotStateTransition{
		DevelopmentID: developmentID,
		LotID:         lotID,
		ExpectedState: domain.LotStateSold,
		NextState:     domain.LotStateCompleted,
		Origin:        domain.LotStateOriginCollection,
		ActorID:       actorID,
		SaleID:        stringReference(saleID),
	}, lotState)
	return err
}

// insertPaymentCharges writes the cargos adicionales of a cobro. A charge
// without a currency takes the sale's; one in another currency is stored as
// typed, so a cuota in USD can travel with services in ARS.
func insertPaymentCharges(ctx context.Context, tx pgx.Tx, paymentID, currency string, command gateway.RegisterPaymentCommand) error {
	charges, err := domain.NormalizePaymentCharges(command.Charges, currency)
	if err != nil {
		return err
	}
	for _, charge := range charges {
		if _, err := tx.Exec(ctx, `
			INSERT INTO cargos_adicionales (
				cobro_id, monto, moneda, tipo, observacion,
				usuario_modificacion, fecha_creacion, fecha_modificacion
			)
			VALUES ($1::uuid, $2, $3, $4, NULLIF($5, ''), $6::uuid, $7, $7)
		`, paymentID, charge.Monto, charge.Moneda, string(charge.Tipo), charge.Detalle,
			command.ActorID, command.Now); err != nil {
			return err
		}
	}
	return nil
}

func loadPlanRow(ctx context.Context, queryer reservationQueryer, planID string) (planRow, error) {
	var row planRow
	err := queryer.QueryRow(ctx, `
		SELECT id::text, monto_entrega::float8, fecha_entrega, cobro_entrega_id::text
		FROM planes_pago
		WHERE id = $1::uuid AND fecha_baja IS NULL
	`, planID).Scan(&row.id, &row.downPayment, &row.deliveredAt, &row.deliveryPaymentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return planRow{}, domain.ErrSaleNotFinanced
	}
	return row, err
}

func loadPlanRowBySale(ctx context.Context, tx pgx.Tx, saleID string) (planRow, error) {
	var row planRow
	err := tx.QueryRow(ctx, `
		SELECT id::text, monto_entrega::float8, fecha_entrega, cobro_entrega_id::text
		FROM planes_pago
		WHERE venta_id = $1::uuid AND fecha_baja IS NULL
		FOR UPDATE
	`, saleID).Scan(&row.id, &row.downPayment, &row.deliveredAt, &row.deliveryPaymentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return planRow{}, domain.ErrSaleNotFinanced
	}
	return row, err
}

// loadInstallmentsAsOf reads the cuotas of a plan with the state they have
// as of now, so a pendiente past its vencimiento comes back vencida.
func loadInstallmentsAsOf(ctx context.Context, queryer reservationQueryer, planID string, now time.Time, lock bool) ([]domain.Installment, error) {
	query := `
		SELECT id::text, numero, monto::float8, estado, fecha_vencimiento, fecha_pago, COALESCE(cobro_id::text, '')
		FROM cuotas
		WHERE plan_pago_id = $1::uuid
		ORDER BY numero`
	if lock {
		query += ` FOR UPDATE`
	}
	rows, err := queryer.Query(ctx, query, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	installments := make([]domain.Installment, 0)
	for rows.Next() {
		var installment domain.Installment
		if err := rows.Scan(&installment.ID, &installment.Numero, &installment.Monto, &installment.Estado, &installment.FechaVencimiento, &installment.FechaPago, &installment.CobroID); err != nil {
			return nil, err
		}
		installment.Estado = domain.EffectiveInstallmentState(installment, now)
		installments = append(installments, installment)
	}
	return installments, rows.Err()
}

const paymentColumns = `
	p.id::text, p.venta_id::text, p.tipo, p.monto::float8, p.moneda, p.medio_pago, COALESCE(p.observacion, ''),
	p.fecha_pago, p.fecha_creacion,
	u.id::text, u.nombre, u.apellido, u.email, u.rol,
	COALESCE(plan.cobro_entrega_id = p.id, false), COALESCE(plan.monto_entrega, 0)::float8`

const paymentFromClause = `
	FROM cobros p
	JOIN usuarios u ON u.id = p.usuario_alta
	LEFT JOIN planes_pago plan ON plan.venta_id = p.venta_id AND plan.fecha_baja IS NULL`

func scanPayment(scanner reservationScanner) (domain.Payment, error) {
	var (
		payment     domain.Payment
		downPayment float64
	)
	if err := scanner.Scan(&payment.ID, &payment.VentaID, &payment.Tipo, &payment.Monto, &payment.Moneda, &payment.MedioPago, &payment.Observacion,
		&payment.FechaPago, &payment.FechaCreacion,
		&payment.UsuarioAlta.ID, &payment.UsuarioAlta.Nombre, &payment.UsuarioAlta.Apellido, &payment.UsuarioAlta.Email, &payment.UsuarioAlta.Rol,
		&payment.IncluyeEntrega, &downPayment); err != nil {
		return domain.Payment{}, err
	}
	if payment.IncluyeEntrega {
		payment.MontoEntrega = downPayment
	}
	payment.Cuotas = []domain.Installment{}
	payment.Cargos = []domain.PaymentCharge{}
	return payment, nil
}

func loadPayment(ctx context.Context, queryer reservationQueryer, paymentID string) (domain.Payment, error) {
	payment, err := scanPayment(queryer.QueryRow(ctx, `SELECT `+paymentColumns+paymentFromClause+` WHERE p.id = $1::uuid`, paymentID))
	if err != nil {
		return domain.Payment{}, err
	}
	rows, err := queryer.Query(ctx, `
		SELECT id::text, numero, monto::float8, estado, fecha_vencimiento, fecha_pago, cobro_id::text
		FROM cuotas
		WHERE cobro_id = $1::uuid
		ORDER BY numero
	`, paymentID)
	if err != nil {
		return domain.Payment{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var installment domain.Installment
		if err := rows.Scan(&installment.ID, &installment.Numero, &installment.Monto, &installment.Estado, &installment.FechaVencimiento, &installment.FechaPago, &installment.CobroID); err != nil {
			return domain.Payment{}, err
		}
		payment.Cuotas = append(payment.Cuotas, installment)
	}
	if err := rows.Err(); err != nil {
		return domain.Payment{}, err
	}
	chargeRows, err := queryer.Query(ctx, `
		SELECT `+chargeColumns+chargeFromClause+`
		WHERE c.cobro_id = $1::uuid AND c.fecha_baja IS NULL
		ORDER BY `+chargeOrder, paymentID)
	if err != nil {
		return domain.Payment{}, err
	}
	defer chargeRows.Close()
	for chargeRows.Next() {
		charge, _, err := scanCharge(chargeRows)
		if err != nil {
			return domain.Payment{}, err
		}
		payment.Cargos = append(payment.Cargos, charge)
	}
	if err := chargeRows.Err(); err != nil {
		return domain.Payment{}, err
	}
	payment.Totales = domain.PaymentTotals(payment.Monto, payment.Moneda, payment.Cargos)
	return payment, nil
}

const chargeColumns = `c.id::text, c.tipo, c.monto::float8, c.moneda, COALESCE(c.observacion, ''), c.cobro_id::text`

const chargeFromClause = `
	FROM cargos_adicionales c
	JOIN cobros p ON p.id = c.cobro_id`

// chargeOrder reads the charges the way the totals do: first the ones in the
// sale's currency, then the other currencies alphabetically. Charges entered
// in one cobro share fecha_creacion, so ordering by it decides nothing.
const chargeOrder = `(c.moneda <> p.moneda), c.moneda, c.tipo, c.id`

func scanCharge(scanner reservationScanner) (domain.PaymentCharge, string, error) {
	var (
		charge    domain.PaymentCharge
		paymentID string
	)
	if err := scanner.Scan(&charge.ID, &charge.Tipo, &charge.Monto, &charge.Moneda, &charge.Detalle, &paymentID); err != nil {
		return domain.PaymentCharge{}, "", err
	}
	return charge, paymentID, nil
}

// loadPayments reads every cobro of a venta, newest first, each with the
// cuotas it paid and the cargos adicionales it collected.
func loadPayments(ctx context.Context, queryer reservationQueryer, saleID string, plan planRow) ([]domain.Payment, error) {
	rows, err := queryer.Query(ctx, `SELECT `+paymentColumns+paymentFromClause+`
		WHERE p.venta_id = $1::uuid
		ORDER BY p.fecha_pago DESC, p.fecha_creacion DESC, p.id DESC`, saleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	payments := make([]domain.Payment, 0)
	index := make(map[string]int)
	for rows.Next() {
		payment, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		index[payment.ID] = len(payments)
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(payments) == 0 {
		return payments, nil
	}
	paidRows, err := queryer.Query(ctx, `
		SELECT id::text, numero, monto::float8, estado, fecha_vencimiento, fecha_pago, cobro_id::text
		FROM cuotas
		WHERE plan_pago_id = $1::uuid AND cobro_id IS NOT NULL
		ORDER BY numero
	`, plan.id)
	if err != nil {
		return nil, err
	}
	defer paidRows.Close()
	for paidRows.Next() {
		var installment domain.Installment
		if err := paidRows.Scan(&installment.ID, &installment.Numero, &installment.Monto, &installment.Estado, &installment.FechaVencimiento, &installment.FechaPago, &installment.CobroID); err != nil {
			return nil, err
		}
		if position, ok := index[installment.CobroID]; ok {
			payments[position].Cuotas = append(payments[position].Cuotas, installment)
		}
	}
	if err := paidRows.Err(); err != nil {
		return nil, err
	}
	chargeRows, err := queryer.Query(ctx, `
		SELECT `+chargeColumns+chargeFromClause+`
		WHERE p.venta_id = $1::uuid AND c.fecha_baja IS NULL
		ORDER BY `+chargeOrder, saleID)
	if err != nil {
		return nil, err
	}
	defer chargeRows.Close()
	for chargeRows.Next() {
		charge, paymentID, err := scanCharge(chargeRows)
		if err != nil {
			return nil, err
		}
		if position, ok := index[paymentID]; ok {
			payments[position].Cargos = append(payments[position].Cargos, charge)
		}
	}
	if err := chargeRows.Err(); err != nil {
		return nil, err
	}
	for i := range payments {
		payments[i].Totales = domain.PaymentTotals(payments[i].Monto, payments[i].Moneda, payments[i].Cargos)
	}
	return payments, nil
}
