package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

const saleActiveIndex = "ventas_lote_id_activa_idx"

type SaleRepository struct {
	pool *pgxpool.Pool
}

func NewSaleRepository(pool *pgxpool.Pool) *SaleRepository {
	return &SaleRepository{pool: pool}
}

// Create registers the venta and moves the lote to vendido in one
// transaction. The lote row is locked first, so two sales of the same lote
// serialize on it and the second one finds it already vendido.
func (repository *SaleRepository) Create(ctx context.Context, command gateway.CreateSaleCommand) (domain.Sale, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Sale{}, err
	}
	defer rollbackTransaction(tx)

	var loteoID string
	err = tx.QueryRow(ctx, `
		SELECT id::text
		FROM loteos
		WHERE id = $1::uuid AND fecha_baja IS NULL
		FOR SHARE
	`, command.LoteoID).Scan(&loteoID)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Sale{}, domain.ErrLoteNotFound
	}
	if err != nil {
		return domain.Sale{}, err
	}

	var lotState domain.LotState
	var lotNumber, currency string
	var lotPrice *float64
	err = tx.QueryRow(ctx, `
		SELECT estado_actual, COALESCE(numero, ''), precio::float8, COALESCE(moneda, '')
		FROM lotes
		WHERE id = $1::uuid AND loteo_id = $2::uuid AND fecha_baja IS NULL
		FOR UPDATE
	`, command.LoteID, command.LoteoID).Scan(&lotState, &lotNumber, &lotPrice, &currency)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Sale{}, domain.ErrLoteNotFound
	}
	if err != nil {
		return domain.Sale{}, err
	}

	actorRole, actorAgency, err := lockActor(ctx, tx, command.ActorID)
	if err != nil {
		return domain.Sale{}, err
	}
	if !domain.IsSaleRole(actorRole) {
		return domain.Sale{}, domain.ErrNoAutorizado
	}

	sellerRole, sellerAgency, sellerActive, err := lockSeller(ctx, tx, command.VendedorID)
	if err != nil {
		if errors.Is(err, domain.ErrReservationSellerNotEligible) {
			return domain.Sale{}, domain.ErrSaleSellerNotEligible
		}
		return domain.Sale{}, err
	}
	if !sellerActive || !domain.IsSaleRole(sellerRole) {
		return domain.Sale{}, domain.ErrSaleSellerNotEligible
	}
	// An agency user sells for their own agency only; the seller's agency
	// itself must still be active, though it needn't be assigned to the loteo.
	if actorRole == domain.RolInmobiliaria && !sameString(actorAgency, sellerAgency) {
		return domain.Sale{}, domain.ErrSaleSellerNotEligible
	}
	if sellerRole == domain.RolInmobiliaria {
		active, activeErr := agencyActive(ctx, tx, sellerAgency)
		if activeErr != nil {
			return domain.Sale{}, activeErr
		}
		if !active {
			return domain.Sale{}, domain.ErrSaleSellerNotEligible
		}
	}

	if err := lockActiveClient(ctx, tx, command.ClienteID); err != nil {
		if errors.Is(err, domain.ErrReservationInvalidClient) {
			return domain.Sale{}, domain.ErrSaleInvalidClient
		}
		return domain.Sale{}, err
	}

	if lotNumber == "" || lotPrice == nil || *lotPrice <= 0 || currency == "" {
		return domain.Sale{}, domain.ErrSaleLotIncomplete
	}
	if lotState != domain.LotStateAvailable {
		return domain.Sale{}, domain.ErrSaleLotUnavailable
	}

	var saleID string
	err = tx.QueryRow(ctx, `
		INSERT INTO ventas (
			lote_id, cliente_id, modalidad_pago, monto, moneda, vendedor_id, usuario_alta,
			fecha_creacion, fecha_modificacion
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::uuid, $7::uuid, $8, $8)
		RETURNING id::text
	`, command.LoteID, command.ClienteID, command.PaymentMethod, *lotPrice, currency,
		command.VendedorID, command.ActorID, command.CreatedAt).Scan(&saleID)
	if err != nil {
		if isConstraint(err, saleActiveIndex) {
			return domain.Sale{}, domain.ErrSaleActiveConflict
		}
		return domain.Sale{}, err
	}

	if _, err := transitionLotStateWithLockedLot(ctx, tx, domain.LotStateTransition{
		DevelopmentID: loteoID,
		LotID:         command.LoteID,
		ExpectedState: domain.LotStateAvailable,
		NextState:     domain.LotStateSold,
		Origin:        domain.LotStateOriginSale,
		ActorID:       command.ActorID,
		SaleID:        stringReference(saleID),
	}, lotState); err != nil {
		return domain.Sale{}, err
	}

	sale, err := loadSale(ctx, tx, saleID, gateway.SaleScope{}, true)
	if err != nil {
		return domain.Sale{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Sale{}, err
	}
	return sale, nil
}

func (repository *SaleRepository) List(ctx context.Context, filter domain.SaleListFilter, scope gateway.SaleScope) (domain.SalePage, error) {
	scope = normalizeSaleScope(scope)
	var err error
	filter, err = filter.Normalize()
	if err != nil {
		return domain.SalePage{}, err
	}
	states := make([]string, len(filter.States))
	for i, state := range filter.States {
		states[i] = string(state)
	}
	page := domain.SalePage{Page: filter.Page, Limit: filter.Limit, Items: []domain.Sale{}}
	offset := (filter.Page - 1) * filter.Limit
	where := `
		WHERE (cardinality($1::text[]) = 0 OR v.estado_actual = ANY($1::text[]))
		  AND ($2 = '' OR l.id::text = $2)
		  AND ($3 = '' OR l.nombre ILIKE $7 ESCAPE '\'
		       OR COALESCE(lo.numero, '') ILIKE $7 ESCAPE '\'
		       OR c.nombre ILIKE $7 ESCAPE '\' OR c.apellido ILIKE $7 ESCAPE '\'
		       OR c.dni ILIKE $7 ESCAPE '\'
		       OR seller.nombre ILIKE $7 ESCAPE '\' OR seller.apellido ILIKE $7 ESCAPE '\')
		  AND ($4 = '' OR lo.id::text = $4)
		  AND ` + fmt.Sprintf(saleScopePredicate, 5, 6)
	rows, err := repository.pool.Query(ctx, `
		SELECT `+saleColumns+`, count(*) OVER()
		`+saleFromClause+where+`
		ORDER BY v.fecha_creacion DESC, v.id DESC
		LIMIT $8 OFFSET $9
	`, states, filter.LoteoID, filter.Search, filter.LoteID, scope.AssigneeAuthProviderID,
		scope.ByAgency, containsPattern(filter.Search), filter.Limit, offset)
	if err != nil {
		return domain.SalePage{}, err
	}
	defer rows.Close()

	hasTotal := false
	for rows.Next() {
		sale, total, err := scanSaleWithTotal(rows)
		if err != nil {
			return domain.SalePage{}, err
		}
		page.Items = append(page.Items, sale)
		page.Total = int(total)
		hasTotal = true
	}
	if err := rows.Err(); err != nil {
		return domain.SalePage{}, err
	}
	if !hasTotal {
		if err := repository.pool.QueryRow(ctx, `SELECT count(*) `+saleFromClause+where,
			states, filter.LoteoID, filter.Search, filter.LoteID, scope.AssigneeAuthProviderID,
			scope.ByAgency, containsPattern(filter.Search)).Scan(&page.Total); err != nil {
			return domain.SalePage{}, err
		}
	}
	page.TotalPages = (page.Total + page.Limit - 1) / page.Limit
	return page, nil
}

func (repository *SaleRepository) Get(ctx context.Context, id string, scope gateway.SaleScope) (domain.Sale, error) {
	return loadSale(ctx, repository.pool, id, scope, true)
}

const saleFromClause = `
	FROM ventas v
	JOIN lotes lo ON lo.id = v.lote_id AND lo.fecha_baja IS NULL
	JOIN loteos l ON l.id = lo.loteo_id AND l.fecha_baja IS NULL
	LEFT JOIN manzanas mz ON mz.id = lo.manzana_id
	JOIN clientes c ON c.id = v.cliente_id
	JOIN usuarios seller ON seller.id = v.vendedor_id
	JOIN usuarios alta ON alta.id = v.usuario_alta
	LEFT JOIN inmobiliarias agency ON agency.id = seller.inmobiliaria_id`

const saleColumns = `
	v.id::text, l.id::text, l.nombre, lo.id::text, COALESCE(lo.numero, ''), COALESCE(mz.numero, ''),
	lo.superficie::float8,
	c.id::text, c.nombre, c.apellido, c.dni, c.celular, c.email,
	seller.id::text, seller.nombre, seller.apellido, seller.email, seller.rol,
	alta.id::text, alta.nombre, alta.apellido, alta.email, alta.rol,
	agency.id::text, COALESCE(agency.razon_social, ''),
	v.modalidad_pago, v.monto::float8, v.moneda,
	v.estado_actual, v.fecha_creacion, v.fecha_modificacion`

// saleScopePredicate keeps an agency actor within the sales of their own
// agency: the seller's agency, since ventas doesn't store one.
const saleScopePredicate = `($%[1]d::uuid IS NULL OR ($%[2]d AND EXISTS (
	SELECT 1
	FROM usuarios actor
	WHERE actor.auth_provider_id = $%[1]d::uuid
	  AND actor.fecha_baja IS NULL
	  AND actor.inmobiliaria_id IS NOT NULL
	  AND actor.inmobiliaria_id = seller.inmobiliaria_id
)))`

func loadSale(ctx context.Context, queryer reservationQueryer, id string, scope gateway.SaleScope, withHistory bool) (domain.Sale, error) {
	scope = normalizeSaleScope(scope)
	row := queryer.QueryRow(ctx, `SELECT `+saleColumns+saleFromClause+`
		WHERE v.id = $1::uuid AND `+fmt.Sprintf(saleScopePredicate, 2, 3),
		id, scope.AssigneeAuthProviderID, scope.ByAgency)
	sale, err := scanSale(row)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Sale{}, domain.ErrSaleNotFound
	}
	if err != nil {
		return domain.Sale{}, err
	}
	if withHistory {
		sale.Historial, err = loadSaleHistory(ctx, queryer, sale.ID)
		if err != nil {
			return domain.Sale{}, err
		}
	}
	return sale, nil
}

func normalizeSaleScope(scope gateway.SaleScope) gateway.SaleScope {
	scope.AssigneeAuthProviderID = nonBlankUUIDReference(scope.AssigneeAuthProviderID)
	return scope
}

func loadSaleHistory(ctx context.Context, queryer reservationQueryer, saleID string) ([]domain.SaleHistoryEntry, error) {
	rows, err := queryer.Query(ctx, `
		SELECT ve.id::text, ve.estado, COALESCE(ve.razon, ''), ve.fecha_creacion,
		       u.id::text, u.nombre, u.apellido, u.email, u.rol
		FROM venta_estados ve
		LEFT JOIN usuarios u ON u.id = ve.usuario_modificacion
		WHERE ve.venta_id = $1::uuid
		ORDER BY ve.fecha_creacion, ve.id
	`, saleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]domain.SaleHistoryEntry, 0)
	for rows.Next() {
		var (
			entry                               domain.SaleHistoryEntry
			actorID, name, surname, email, role *string
		)
		if err := rows.Scan(&entry.ID, &entry.Estado, &entry.Razon, &entry.Fecha, &actorID, &name, &surname, &email, &role); err != nil {
			return nil, err
		}
		if actorID != nil {
			entry.Usuario = &domain.ReservationActor{ID: *actorID, Nombre: stringValue(name), Apellido: stringValue(surname), Email: stringValue(email), Rol: domain.Rol(stringValue(role))}
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func saleScanTargets(sale *domain.Sale, client *domain.Cliente, seller, alta *domain.ReservationActor, agencyID **string, agencyName *string) []any {
	return []any{
		&sale.ID, &sale.LoteoID, &sale.LoteoNombre, &sale.LoteID, &sale.LoteNumero, &sale.ManzanaNumero,
		&sale.LoteSuperficie,
		&client.ID, &client.Nombre, &client.Apellido, &client.DNI, &client.Celular, &client.Email,
		&seller.ID, &seller.Nombre, &seller.Apellido, &seller.Email, &seller.Rol,
		&alta.ID, &alta.Nombre, &alta.Apellido, &alta.Email, &alta.Rol,
		agencyID, agencyName,
		&sale.ModalidadPago, &sale.Monto, &sale.Moneda,
		&sale.Estado, &sale.FechaCreacion, &sale.FechaModificacion,
	}
}

func assembleSale(sale domain.Sale, client domain.Cliente, seller, alta domain.ReservationActor, agencyID *string, agencyName string) domain.Sale {
	sale.Cliente, sale.Vendedor, sale.UsuarioAlta = client, seller, alta
	if agencyID != nil {
		sale.Inmobiliaria = &domain.ReservationAgency{ID: *agencyID, BusinessName: agencyName}
	}
	return sale
}

func scanSale(row reservationScanner) (domain.Sale, error) {
	var (
		sale         domain.Sale
		client       domain.Cliente
		seller, alta domain.ReservationActor
		agencyID     *string
		agencyName   string
	)
	if err := row.Scan(saleScanTargets(&sale, &client, &seller, &alta, &agencyID, &agencyName)...); err != nil {
		return domain.Sale{}, err
	}
	return assembleSale(sale, client, seller, alta, agencyID, agencyName), nil
}

func scanSaleWithTotal(row reservationScanner) (domain.Sale, int64, error) {
	var (
		total        int64
		sale         domain.Sale
		client       domain.Cliente
		seller, alta domain.ReservationActor
		agencyID     *string
		agencyName   string
	)
	targets := append(saleScanTargets(&sale, &client, &seller, &alta, &agencyID, &agencyName), &total)
	if err := row.Scan(targets...); err != nil {
		return domain.Sale{}, 0, err
	}
	return assembleSale(sale, client, seller, alta, agencyID, agencyName), total, nil
}

func agencyActive(ctx context.Context, tx pgx.Tx, agencyID *string) (bool, error) {
	if agencyID == nil {
		return false, nil
	}
	var present bool
	err := tx.QueryRow(ctx, `
		SELECT true FROM inmobiliarias
		WHERE id = $1::uuid AND fecha_baja IS NULL
		FOR SHARE
	`, *agencyID).Scan(&present)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return false, nil
	}
	return present, err
}
