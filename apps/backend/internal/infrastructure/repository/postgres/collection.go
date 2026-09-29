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
	// afterStatementInstallments runs between reading the cuotas and the
	// cobros of an estado de deuda. Tests use it to commit a cobro mid-read;
	// it is nil in production.
	afterStatementInstallments func()
}

func NewCollectionRepository(pool *pgxpool.Pool) *CollectionRepository {
	return &CollectionRepository{pool: pool}
}

// readSnapshot runs read in a read-only REPEATABLE READ transaction, so every
// query it makes sees the same committed data even while cobros commit.
func (repository *CollectionRepository) readSnapshot(ctx context.Context, read func(tx pgx.Tx) error) error {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer rollbackTransaction(tx)
	if err := read(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
	entrega := &domain.DownPayment{Amount: *row.downPayment, State: domain.InstallmentStatePending}
	if row.deliveredAt != nil {
		entrega.State = domain.InstallmentStatePaid
		entrega.PaidAt = row.deliveredAt
		entrega.PaymentID = stringValue(row.deliveryPaymentID)
	}
	return entrega
}

// GetDebtStatement reads the venta, plan, cuotas and cobros in one snapshot:
// a statement may be printed, so it must never show a cuota pendiente next to
// the cobro that already paid it.
func (repository *CollectionRepository) GetDebtStatement(ctx context.Context, saleID string, scope gateway.SaleScope, now time.Time) (domain.DebtStatement, error) {
	var statement domain.DebtStatement
	err := repository.readSnapshot(ctx, func(tx pgx.Tx) error {
		sale, err := loadSale(ctx, tx, saleID, scope, false)
		if err != nil {
			return err
		}
		if sale.PlanPago == nil {
			return domain.ErrSaleNotFinanced
		}
		plan, err := loadPlanRow(ctx, tx, sale.PlanPago.ID)
		if err != nil {
			return err
		}
		installments, err := loadInstallmentsAsOf(ctx, tx, plan.id, now, false)
		if err != nil {
			return err
		}
		if repository.afterStatementInstallments != nil {
			repository.afterStatementInstallments()
		}
		payments, err := loadPayments(ctx, tx, sale.ID, plan)
		if err != nil {
			return err
		}
		entrega := plan.entrega()
		charges := make([]domain.PaymentCharge, 0)
		for _, payment := range payments {
			charges = append(charges, payment.Charges...)
		}
		statement = domain.DebtStatement{
			Sale:             sale,
			DownPayment:      entrega,
			Installments:     installments,
			Summary:          domain.ComputeDebtSummary(entrega, installments),
			Payments:         payments,
			CollectedCharges: domain.ChargeTotals(charges),
			IssuedAt:         now,
		}
		return nil
	})
	if err != nil {
		return domain.DebtStatement{}, err
	}
	return statement, nil
}

// dueInstallmentState is the cuota's state given the start of the current
// business day in %[1]s: the same rule domain.EffectiveInstallmentState
// applies, so the list can filter on it.
const dueInstallmentState = `CASE
	WHEN c.estado = 'pagada' THEN 'pagada'
	WHEN c.fecha_vencimiento < %[1]s THEN 'vencida'
	ELSE 'pendiente'
END`

// dueInstallmentFilterFrom joins only what the filters and the scope read.
// Every join is inner, so it also decides which cuotas are listed at all.
const dueInstallmentFilterFrom = `
	FROM cuotas c
	JOIN planes_pago plan ON plan.id = c.plan_pago_id AND plan.fecha_baja IS NULL
	JOIN ventas v ON v.id = plan.venta_id AND v.estado_actual <> 'cancelada'
	JOIN lotes lo ON lo.id = v.lote_id AND lo.fecha_baja IS NULL
	JOIN loteos l ON l.id = lo.loteo_id AND l.fecha_baja IS NULL
	JOIN clientes cl ON cl.id = v.cliente_id
	JOIN usuarios seller ON seller.id = v.vendedor_id`

const dueInstallmentDetailJoins = `
	LEFT JOIN manzanas mz ON mz.id = lo.manzana_id
	LEFT JOIN inmobiliarias agency ON agency.id = seller.inmobiliaria_id`

const dueInstallmentOrder = `ORDER BY c.fecha_vencimiento, c.numero, c.id`

// ListDueInstallments pages the ids first and loads the detail only for that
// page, so a page doesn't pay for the detail of every cuota in scope. The
// total is a plain count, skipped when the page itself tells it. Everything
// is read in one snapshot so the page, the total and the summary agree.
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
	today := domain.StartOfBusinessDay(now)
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
	args := []any{today, states, filter.DevelopmentID, filter.Search, containsPattern(filter.Search),
		filter.From, filter.To, scope.AssigneeAuthProviderID, scope.ByAgency}

	err = repository.readSnapshot(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT c.id::text `+dueInstallmentFilterFrom+where+`
			`+dueInstallmentOrder+`
			LIMIT $10 OFFSET $11`, append(args, filter.Limit, offset)...)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		if (len(ids) > 0 && len(ids) < filter.Limit) || (len(ids) == 0 && offset == 0) {
			page.Total = offset + len(ids)
		} else if err := tx.QueryRow(ctx, `SELECT count(*) `+dueInstallmentFilterFrom+where, args...).Scan(&page.Total); err != nil {
			return err
		}
		if page.Items, err = loadDueInstallments(ctx, tx, ids, today); err != nil {
			return err
		}

		// The header counts over the whole scope (and loteo, when one is
		// picked), not over the filtered page, so it reads the same whatever
		// the user is looking at.
		return tx.QueryRow(ctx, `
			SELECT count(*) FILTER (WHERE c.fecha_vencimiento < $1::timestamptz),
			       count(*) FILTER (WHERE c.fecha_vencimiento >= $1::timestamptz AND c.fecha_vencimiento < $2::timestamptz)
			`+dueInstallmentFilterFrom+`
			WHERE c.estado <> 'pagada'
			  AND ($3 = '' OR l.id::text = $3)
			  AND `+fmt.Sprintf(saleScopePredicate, 4, 5),
			today, today.Add(domain.DueSoonWindow), filter.DevelopmentID, scope.AssigneeAuthProviderID, scope.ByAgency,
		).Scan(&page.Summary.OverdueInstallments, &page.Summary.UpcomingInstallments)
	})
	if err != nil {
		return domain.DueInstallmentPage{}, err
	}
	page.TotalPages = (page.Total + page.Limit - 1) / page.Limit
	return page, nil
}

// loadDueInstallments reads the detail of one page of cuotas, in list order.
func loadDueInstallments(ctx context.Context, tx pgx.Tx, ids []string, today time.Time) ([]domain.DueInstallment, error) {
	items := make([]domain.DueInstallment, 0, len(ids))
	if len(ids) == 0 {
		return items, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT c.id::text, v.id::text, c.numero, plan.cantidad_cuotas, c.monto::float8, plan.moneda,
		       `+fmt.Sprintf(dueInstallmentState, "$1::timestamptz")+`, c.fecha_vencimiento, c.fecha_pago,
		       l.id::text, l.nombre, lo.id::text, COALESCE(lo.numero, ''), COALESCE(mz.numero, ''),
		       cl.id::text, cl.nombre, cl.apellido, cl.dni, cl.celular, cl.email,
		       seller.id::text, seller.nombre, seller.apellido, seller.email, seller.rol,
		       agency.id::text, COALESCE(agency.razon_social, '')
		`+dueInstallmentFilterFrom+dueInstallmentDetailJoins+`
		WHERE c.id = ANY($2::uuid[])
		`+dueInstallmentOrder, today, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			item       domain.DueInstallment
			agencyID   *string
			agencyName string
		)
		if err := rows.Scan(&item.ID, &item.SaleID, &item.Number, &item.InstallmentCount, &item.Amount, &item.Currency,
			&item.State, &item.DueDate, &item.PaidAt,
			&item.DevelopmentID, &item.DevelopmentName, &item.LotID, &item.LotNumber, &item.BlockNumber,
			&item.Client.ID, &item.Client.Nombre, &item.Client.Apellido, &item.Client.DNI, &item.Client.Celular, &item.Client.Email,
			&item.Seller.ID, &item.Seller.Nombre, &item.Seller.Apellido, &item.Seller.Email, &item.Seller.Rol,
			&agencyID, &agencyName); err != nil {
			return nil, err
		}
		if agencyID != nil {
			item.Agency = &domain.ReservationAgency{ID: *agencyID, BusinessName: agencyName}
		}
		items = append(items, item)
	}
	return items, rows.Err()
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
	entregaSettled := entrega == nil || entrega.State == domain.InstallmentStatePaid || selection.IncludeDownPayment
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
		`, paymentID, charge.Amount, charge.Currency, string(charge.Type), charge.Detail,
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
		if err := rows.Scan(&installment.ID, &installment.Numero, &installment.Monto, &installment.Estado, &installment.FechaVencimiento, &installment.FechaPago, &installment.PaymentID); err != nil {
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
	if err := scanner.Scan(&payment.ID, &payment.SaleID, &payment.Type, &payment.Amount, &payment.Currency, &payment.Medium, &payment.Observation,
		&payment.PaidAt, &payment.CreatedAt,
		&payment.CreatedBy.ID, &payment.CreatedBy.Nombre, &payment.CreatedBy.Apellido, &payment.CreatedBy.Email, &payment.CreatedBy.Rol,
		&payment.IncludesDownPayment, &downPayment); err != nil {
		return domain.Payment{}, err
	}
	if payment.IncludesDownPayment {
		payment.DownPaymentAmount = downPayment
	}
	payment.Installments = []domain.Installment{}
	payment.Charges = []domain.PaymentCharge{}
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
		if err := rows.Scan(&installment.ID, &installment.Numero, &installment.Monto, &installment.Estado, &installment.FechaVencimiento, &installment.FechaPago, &installment.PaymentID); err != nil {
			return domain.Payment{}, err
		}
		payment.Installments = append(payment.Installments, installment)
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
		payment.Charges = append(payment.Charges, charge)
	}
	if err := chargeRows.Err(); err != nil {
		return domain.Payment{}, err
	}
	payment.Totals = domain.PaymentTotals(payment.Amount, payment.Currency, payment.Charges)
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
	if err := scanner.Scan(&charge.ID, &charge.Type, &charge.Amount, &charge.Currency, &charge.Detail, &paymentID); err != nil {
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
		if err := paidRows.Scan(&installment.ID, &installment.Numero, &installment.Monto, &installment.Estado, &installment.FechaVencimiento, &installment.FechaPago, &installment.PaymentID); err != nil {
			return nil, err
		}
		if position, ok := index[installment.PaymentID]; ok {
			payments[position].Installments = append(payments[position].Installments, installment)
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
			payments[position].Charges = append(payments[position].Charges, charge)
		}
	}
	if err := chargeRows.Err(); err != nil {
		return nil, err
	}
	for i := range payments {
		payments[i].Totals = domain.PaymentTotals(payments[i].Amount, payments[i].Currency, payments[i].Charges)
	}
	return payments, nil
}
