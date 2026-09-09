package postgres

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

const (
	reservationTransitionConstraint = "reserva_estados_transition_chk"
	reservationIdempotencyIndex     = "reservas_usuario_alta_idempotency_key_idx"
	reservationActiveIndex          = "reservas_lote_id_activa_idx"
	reservationRetryCount           = 3
)

type ReservationRepository struct {
	pool  *pgxpool.Pool
	clock reservationClock
}

type reservationClock interface {
	Now() time.Time
}

type systemReservationClock struct{}

func (systemReservationClock) Now() time.Time { return time.Now() }

func NewReservationRepository(pool *pgxpool.Pool, clocks ...reservationClock) *ReservationRepository {
	clock := reservationClock(systemReservationClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &ReservationRepository{pool: pool, clock: clock}
}

func (repository *ReservationRepository) Create(ctx context.Context, command gateway.CreateReservationCommand) (domain.Reservation, error) {
	reservation, err := retryReservationCreate(ctx, func() (domain.Reservation, error) {
		return repository.create(ctx, command)
	})
	if err == nil || !errors.Is(err, errRetryReservationWrite) {
		return reservation, err
	}
	return repository.reconcileIdempotentCreate(ctx, command, err)
}

var errRetryReservationWrite = errors.New("retry reservation write")

func retryReservationCreate(ctx context.Context, create func() (domain.Reservation, error)) (domain.Reservation, error) {
	for attempt := 0; attempt < reservationRetryCount; attempt++ {
		reservation, err := create()
		if err == nil || errors.Is(err, errRetryReservationWrite) || !isTransientDBError(err) || attempt == reservationRetryCount-1 {
			return reservation, err
		}
		if err := waitForReservationRetry(ctx, attempt); err != nil {
			return domain.Reservation{}, err
		}
	}
	return domain.Reservation{}, nil
}

func (repository *ReservationRepository) create(ctx context.Context, command gateway.CreateReservationCommand) (domain.Reservation, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Reservation{}, err
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
		return domain.Reservation{}, domain.ErrLoteNotFound
	}
	if err != nil {
		return domain.Reservation{}, err
	}

	var lotState domain.LotState
	var lotNumber string
	var lotPrice *float64
	err = tx.QueryRow(ctx, `
		SELECT estado_actual, COALESCE(numero, ''), precio::float8
		FROM lotes
		WHERE id = $1::uuid AND loteo_id = $2::uuid AND fecha_baja IS NULL
		FOR UPDATE
	`, command.LoteID, command.LoteoID).Scan(&lotState, &lotNumber, &lotPrice)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Reservation{}, domain.ErrLoteNotFound
	}
	if err != nil {
		return domain.Reservation{}, err
	}

	actorRole, actorAgency, err := lockActor(ctx, tx, command.ActorID)
	if err != nil {
		return domain.Reservation{}, err
	}
	if !domain.IsReservationRole(actorRole) {
		return domain.Reservation{}, domain.ErrNoAutorizado
	}
	if actorRole == domain.RolInmobiliaria {
		assigned, assignmentErr := agencyAssigned(ctx, tx, actorAgency, loteoID)
		if assignmentErr != nil {
			return domain.Reservation{}, assignmentErr
		}
		if !assigned {
			return domain.Reservation{}, domain.ErrLoteNotFound
		}
	}

	var existingID, existingHash string
	err = tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(idempotency_payload_hash, '')
		FROM reservas
		WHERE usuario_alta = $1::uuid AND idempotency_key = $2
		FOR UPDATE
	`, command.ActorID, command.IdempotencyKey).Scan(&existingID, &existingHash)
	if err == nil {
		if existingHash != command.IdempotencyPayloadHash {
			return domain.Reservation{}, domain.ErrReservationIdempotencyConflict
		}
		result, readErr := loadReservation(ctx, tx, existingID, false)
		if readErr != nil {
			return domain.Reservation{}, readErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return domain.Reservation{}, fmt.Errorf("%w: %w", errRetryReservationWrite, commitErr)
		}
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Reservation{}, err
	}

	var sellerRole domain.Rol
	var sellerAgency *string
	var sellerActive bool
	if command.SellerIsActor {
		sellerRole, sellerAgency, sellerActive = actorRole, actorAgency, true
	} else {
		sellerRole, sellerAgency, sellerActive, err = lockSeller(ctx, tx, command.VendedorID)
		if err != nil {
			return domain.Reservation{}, err
		}
	}
	if !sellerActive || !domain.IsReservationRole(sellerRole) {
		return domain.Reservation{}, domain.ErrReservationSellerNotEligible
	}
	if command.SellerIsActor && command.VendedorID != command.ActorID {
		return domain.Reservation{}, domain.ErrReservationSellerNotEligible
	}
	if command.SellerIsActor && actorRole != domain.RolInmobiliaria {
		return domain.Reservation{}, domain.ErrReservationSellerNotEligible
	}
	if sellerRole == domain.RolInmobiliaria {
		assigned, assignmentErr := agencyAssigned(ctx, tx, sellerAgency, loteoID)
		if assignmentErr != nil {
			return domain.Reservation{}, assignmentErr
		}
		if !assigned {
			return domain.Reservation{}, domain.ErrReservationSellerNotEligible
		}
	}

	if err := lockActiveClient(ctx, tx, command.ClienteID); err != nil {
		return domain.Reservation{}, err
	}

	var activeID string
	var activeDue time.Time
	err = tx.QueryRow(ctx, `
		SELECT id::text, fecha_vencimiento
		FROM reservas
		WHERE lote_id = $1::uuid AND estado_actual = 'activa'
		FOR UPDATE
	`, command.LoteID).Scan(&activeID, &activeDue)
	activeErr := err
	createdAt := repository.clock.Now().UTC()
	if activeErr == nil {
		if !createdAt.Before(activeDue) {
			if lotState != domain.LotStateReserved {
				return domain.Reservation{}, domain.ErrReservationLotUnavailable
			}
			if err := expireReservationTx(ctx, tx, loteoID, command.LoteID, activeID, createdAt); err != nil {
				return domain.Reservation{}, err
			}
			lotState = domain.LotStateAvailable
		} else {
			return domain.Reservation{}, domain.ErrReservationActiveConflict
		}
	} else if !errors.Is(activeErr, pgx.ErrNoRows) {
		return domain.Reservation{}, activeErr
	}
	if lotNumber == "" || lotPrice == nil || *lotPrice <= 0 {
		return domain.Reservation{}, domain.ErrReservationLotIncomplete
	}
	if lotState != domain.LotStateAvailable {
		return domain.Reservation{}, domain.ErrReservationLotUnavailable
	}

	var reservationID string
	err = tx.QueryRow(ctx, `
		INSERT INTO reservas (
			lote_id, cliente_id, vendedor_id, usuario_alta,
			fecha_vencimiento, fecha_creacion, fecha_modificacion,
			idempotency_key, idempotency_payload_hash
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid,
			$5, $6, $6, $7, $8)
		RETURNING id::text
	`, command.LoteID, command.ClienteID, command.VendedorID, command.ActorID,
		createdAt.Add(domain.ReservationDuration), createdAt,
		command.IdempotencyKey, command.IdempotencyPayloadHash).Scan(&reservationID)
	if err != nil {
		if isConstraint(err, reservationIdempotencyIndex) {
			return domain.Reservation{}, errRetryReservationWrite
		}
		if isConstraint(err, reservationActiveIndex) {
			return domain.Reservation{}, domain.ErrReservationActiveConflict
		}
		return domain.Reservation{}, err
	}

	if _, err := transitionLotStateWithLockedLot(ctx, tx, domain.LotStateTransition{
		DevelopmentID: loteoID,
		LotID:         command.LoteID,
		ExpectedState: domain.LotStateAvailable,
		NextState:     domain.LotStateReserved,
		Origin:        domain.LotStateOriginReservation,
		ActorID:       command.ActorID,
		ReservationID: stringReference(reservationID),
	}, lotState); err != nil {
		return domain.Reservation{}, err
	}

	result, err := loadReservation(ctx, tx, reservationID, false)
	if err != nil {
		return domain.Reservation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Reservation{}, fmt.Errorf("%w: %w", errRetryReservationWrite, err)
	}
	return result, nil
}

func (repository *ReservationRepository) reconcileIdempotentCreate(ctx context.Context, command gateway.CreateReservationCommand, original error) (domain.Reservation, error) {
	var id, hash string
	err := repository.pool.QueryRow(ctx, `
		SELECT id::text, COALESCE(idempotency_payload_hash, '')
		FROM reservas
		WHERE usuario_alta = $1::uuid AND idempotency_key = $2
	`, command.ActorID, command.IdempotencyKey).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Reservation{}, original
	}
	if err != nil {
		return domain.Reservation{}, err
	}
	if hash != command.IdempotencyPayloadHash {
		return domain.Reservation{}, domain.ErrReservationIdempotencyConflict
	}
	return repository.Get(ctx, id, gateway.ReservationScope{})
}

func (repository *ReservationRepository) List(ctx context.Context, filter domain.ReservationListFilter, scope gateway.ReservationScope) (domain.ReservationPage, error) {
	var err error
	filter, err = filter.Normalize()
	if err != nil {
		return domain.ReservationPage{}, err
	}
	states := make([]string, len(filter.States))
	for i, state := range filter.States {
		states[i] = string(state)
	}
	page := domain.ReservationPage{Page: filter.Page, Limit: filter.Limit, Items: []domain.Reservation{}}
	offset := (filter.Page - 1) * filter.Limit
	predicate := fmt.Sprintf(reservationScopePredicate, 5, 6)
	rows, err := repository.pool.Query(ctx, `
		SELECT `+reservationColumns+`, count(*) OVER()
		FROM reservas r
		JOIN lotes lo ON lo.id = r.lote_id AND lo.fecha_baja IS NULL
		JOIN loteos l ON l.id = lo.loteo_id AND l.fecha_baja IS NULL
		JOIN clientes c ON c.id = r.cliente_id
		JOIN usuarios seller ON seller.id = r.vendedor_id
		JOIN usuarios alta ON alta.id = r.usuario_alta
		WHERE (cardinality($1::text[]) = 0 OR r.estado_actual = ANY($1::text[]))
		  AND ($2 = '' OR l.id::text = $2)
		  AND ($3 = '' OR l.nombre ILIKE $7 ESCAPE '\'
		       OR COALESCE(lo.numero, '') ILIKE $7 ESCAPE '\'
		       OR c.nombre ILIKE $7 ESCAPE '\' OR c.apellido ILIKE $7 ESCAPE '\'
		       OR c.dni ILIKE $7 ESCAPE '\'
		       OR seller.nombre ILIKE $7 ESCAPE '\' OR seller.apellido ILIKE $7 ESCAPE '\')
		  AND ($4 = '' OR lo.id::text = $4)
		  AND `+predicate+`
		ORDER BY r.fecha_creacion DESC, r.id DESC
		LIMIT $8 OFFSET $9
	`, states, filter.LoteoID, filter.Search, filter.LoteID, scope.AssigneeAuthProviderID,
		scope.ByAgencyAssignment, containsPattern(filter.Search), filter.Limit, offset)
	if err != nil {
		return domain.ReservationPage{}, err
	}
	defer rows.Close()

	hasTotal := false
	for rows.Next() {
		reservation, total, err := scanReservationWithTotal(rows)
		if err != nil {
			return domain.ReservationPage{}, err
		}
		page.Items = append(page.Items, reservation)
		page.Total = int(total)
		hasTotal = true
	}
	if err := rows.Err(); err != nil {
		return domain.ReservationPage{}, err
	}
	if !hasTotal {
		countQuery := `
			SELECT count(*)
			FROM reservas r
			JOIN lotes lo ON lo.id = r.lote_id AND lo.fecha_baja IS NULL
			JOIN loteos l ON l.id = lo.loteo_id AND l.fecha_baja IS NULL
			JOIN clientes c ON c.id = r.cliente_id
			JOIN usuarios seller ON seller.id = r.vendedor_id
			JOIN usuarios alta ON alta.id = r.usuario_alta
			WHERE (cardinality($1::text[]) = 0 OR r.estado_actual = ANY($1::text[]))
			  AND ($2 = '' OR l.id::text = $2)
			  AND ($3 = '' OR l.nombre ILIKE $7 ESCAPE '\'
			       OR COALESCE(lo.numero, '') ILIKE $7 ESCAPE '\'
			       OR c.nombre ILIKE $7 ESCAPE '\' OR c.apellido ILIKE $7 ESCAPE '\'
			       OR c.dni ILIKE $7 ESCAPE '\'
			       OR seller.nombre ILIKE $7 ESCAPE '\' OR seller.apellido ILIKE $7 ESCAPE '\')
			  AND ($4 = '' OR lo.id::text = $4)
			  AND ` + predicate
		if err := repository.pool.QueryRow(ctx, countQuery, states, filter.LoteoID, filter.Search, filter.LoteID,
			scope.AssigneeAuthProviderID, scope.ByAgencyAssignment, containsPattern(filter.Search)).Scan(&page.Total); err != nil {
			return domain.ReservationPage{}, err
		}
	}
	page.TotalPages = (page.Total + page.Limit - 1) / page.Limit
	return page, nil
}

func (repository *ReservationRepository) Get(ctx context.Context, id string, scope gateway.ReservationScope) (domain.Reservation, error) {
	return loadReservationFromPool(ctx, repository.pool, id, scope, true)
}

func (repository *ReservationRepository) Cancel(ctx context.Context, command gateway.CancelReservationCommand, scope gateway.ReservationScope) (domain.Reservation, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Reservation{}, err
	}
	defer rollbackTransaction(tx)

	var lotID, loteoID string
	err = tx.QueryRow(ctx, `
		SELECT r.lote_id::text, lo.loteo_id::text
		FROM reservas r
		JOIN lotes lo ON lo.id = r.lote_id AND lo.fecha_baja IS NULL
		JOIN loteos l ON l.id = lo.loteo_id AND l.fecha_baja IS NULL
		WHERE r.id = $1::uuid AND `+fmt.Sprintf(reservationScopePredicate, 2, 3)+`
	`, command.ReservationID, scope.AssigneeAuthProviderID, scope.ByAgencyAssignment).Scan(&lotID, &loteoID)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	if err != nil {
		return domain.Reservation{}, err
	}

	var lotState domain.LotState
	err = tx.QueryRow(ctx, `SELECT estado_actual FROM lotes WHERE id = $1::uuid FOR UPDATE`, lotID).Scan(&lotState)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	if err != nil {
		return domain.Reservation{}, err
	}

	var state domain.ReservationState
	var due time.Time
	var sellerID, actorRole string
	var sellerAgency, actorAgency *string
	var actorInactive *time.Time
	err = tx.QueryRow(ctx, `
		SELECT r.estado_actual, r.fecha_vencimiento, r.vendedor_id::text,
		       seller.inmobiliaria_id::text, actor.rol,
		       actor.inmobiliaria_id::text, actor.fecha_baja
		FROM reservas r
		JOIN usuarios seller ON seller.id = r.vendedor_id
		JOIN usuarios actor ON actor.id = $2::uuid
		WHERE r.id = $1::uuid
		FOR UPDATE
	`, command.ReservationID, command.ActorID).Scan(&state, &due, &sellerID, &sellerAgency, &actorRole, &actorAgency, &actorInactive)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	if err != nil {
		return domain.Reservation{}, err
	}
	if actorInactive != nil {
		return domain.Reservation{}, domain.ErrCuentaInactiva
	}
	inScope, err := reservationInScope(ctx, tx, command.ReservationID, scope)
	if err != nil {
		return domain.Reservation{}, err
	}
	if !inScope {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	assigned := false
	if domain.Rol(actorRole) == domain.RolInmobiliaria {
		assigned, err = agencyAssigned(ctx, tx, actorAgency, loteoID)
		if err != nil {
			return domain.Reservation{}, err
		}
	}
	if !domain.CanCancelReservation(domain.Rol(actorRole), command.ActorID, sellerID, sameString(actorAgency, sellerAgency), assigned) {
		return domain.Reservation{}, domain.ErrNoAutorizado
	}

	if state == domain.ReservationStateCancelled {
		result, err := loadReservation(ctx, tx, command.ReservationID, true)
		if err != nil {
			return domain.Reservation{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.Reservation{}, err
		}
		return result, nil
	}
	if state == domain.ReservationStateExpired {
		return domain.Reservation{}, domain.ErrReservationExpired
	}
	if state == domain.ReservationStateConverted {
		return domain.Reservation{}, domain.ErrReservationConverted
	}
	cancelledAt := repository.clock.Now().UTC()
	if !cancelledAt.Before(due) {
		if lotState == domain.LotStateReserved {
			if err := expireReservationTx(ctx, tx, loteoID, lotID, command.ReservationID, cancelledAt); err != nil {
				return domain.Reservation{}, err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return domain.Reservation{}, err
		}
		return domain.Reservation{}, domain.ErrReservationExpired
	}
	if lotState != domain.LotStateReserved {
		return domain.Reservation{}, domain.ErrReservationLotUnavailable
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO reserva_estados (reserva_id, estado, razon, usuario_modificacion)
		VALUES ($1::uuid, 'cancelada', $2, $3::uuid)
	`, command.ReservationID, command.Reason, command.ActorID); err != nil {
		return domain.Reservation{}, mapReservationWriteError(err)
	}
	if _, err := transitionLotState(ctx, tx, domain.LotStateTransition{
		DevelopmentID: loteoID,
		LotID:         lotID,
		ExpectedState: domain.LotStateReserved,
		NextState:     domain.LotStateAvailable,
		Origin:        domain.LotStateOriginReservation,
		Reason:        command.Reason,
		ActorID:       command.ActorID,
		ReservationID: stringReference(command.ReservationID),
	}); err != nil {
		return domain.Reservation{}, err
	}
	result, err := loadReservation(ctx, tx, command.ReservationID, true)
	if err != nil {
		return domain.Reservation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Reservation{}, err
	}
	return result, nil
}

func (repository *ReservationRepository) ListEligibleSellers(ctx context.Context, loteoID string, scope gateway.ReservationScope) ([]domain.SellerOption, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT u.id::text, u.nombre, u.apellido, u.email, u.rol
		FROM usuarios u
		LEFT JOIN inmobiliarias a ON a.id = u.inmobiliaria_id
		JOIN loteos l ON l.id = $1::uuid AND l.fecha_baja IS NULL
		WHERE u.fecha_baja IS NULL
		  AND u.rol IN ('administrador', 'administrativo', 'inmobiliaria')
		  AND ($2::uuid IS NULL OR ($3 AND u.auth_provider_id = $2::uuid))
		  AND (
			 u.rol IN ('administrador', 'administrativo')
			 OR (a.fecha_baja IS NULL AND EXISTS (
				SELECT 1 FROM inmobiliaria_loteos il
				WHERE il.inmobiliaria_id = u.inmobiliaria_id
				  AND il.loteo_id = l.id AND il.fecha_baja IS NULL
			 ))
		  )
		ORDER BY u.apellido, u.nombre, u.id
	`, loteoID, scope.AssigneeAuthProviderID, scope.ByAgencyAssignment)
	if err != nil {
		if isInvalidUUID(err) {
			return []domain.SellerOption{}, domain.ErrLoteoNotFound
		}
		return nil, err
	}
	defer rows.Close()

	result := make([]domain.SellerOption, 0)
	for rows.Next() {
		var seller domain.SellerOption
		if err := rows.Scan(&seller.ID, &seller.Nombre, &seller.Apellido, &seller.Email, &seller.Rol); err != nil {
			return nil, err
		}
		result = append(result, seller)
	}
	if err := rows.Err(); err != nil {
		if isInvalidUUID(err) {
			return []domain.SellerOption{}, domain.ErrLoteoNotFound
		}
		return nil, err
	}
	return result, nil
}

func (repository *ReservationRepository) ExpireDue(ctx context.Context, now time.Time, limit int) (domain.ExpirationReport, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id::text
		FROM reservas
		WHERE estado_actual = 'activa' AND fecha_vencimiento <= $1
		ORDER BY fecha_vencimiento, id
		LIMIT $2
	`, now, limit)
	if err != nil {
		return domain.ExpirationReport{}, err
	}
	var candidates []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return domain.ExpirationReport{}, err
		}
		candidates = append(candidates, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return domain.ExpirationReport{}, err
	}
	rows.Close()

	report := domain.ExpirationReport{Candidates: len(candidates), Failures: []domain.ExpirationFailure{}}
	for _, id := range candidates {
		processed, err := repository.expireWithRetry(ctx, id, now)
		if err != nil {
			report.Failures = append(report.Failures, domain.ExpirationFailure{ReservationID: id, Cause: err})
			continue
		}
		if processed {
			report.Processed++
		} else {
			report.Skipped++
		}
	}
	return report, nil
}

func (repository *ReservationRepository) expireWithRetry(ctx context.Context, id string, now time.Time) (bool, error) {
	for attempt := 0; attempt < reservationRetryCount; attempt++ {
		processed, err := repository.expireOne(ctx, id, now)
		if err == nil || !isTransientDBError(err) || attempt == reservationRetryCount-1 {
			return processed, err
		}
		if err := waitForReservationRetry(ctx, attempt); err != nil {
			return false, err
		}
	}
	return false, nil
}

func (repository *ReservationRepository) expireOne(ctx context.Context, id string, now time.Time) (bool, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer rollbackTransaction(tx)

	var lotID string
	err = tx.QueryRow(ctx, `SELECT lote_id::text FROM reservas WHERE id = $1::uuid`, id).Scan(&lotID)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var lotState domain.LotState
	var loteoID string
	var lotDeleted *time.Time
	lotExists := true
	if err := tx.QueryRow(ctx, `SELECT estado_actual, fecha_baja, loteo_id::text FROM lotes WHERE id = $1::uuid FOR UPDATE`, lotID).Scan(&lotState, &lotDeleted, &loteoID); errors.Is(err, pgx.ErrNoRows) {
		lotExists = false
	} else if err != nil {
		return false, err
	}

	var state domain.ReservationState
	var due time.Time
	var reservationLotID string
	if err := tx.QueryRow(ctx, `SELECT lote_id::text, estado_actual, fecha_vencimiento FROM reservas WHERE id = $1::uuid FOR UPDATE`, id).Scan(&reservationLotID, &state, &due); errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	effectiveNow := repository.clock.Now().UTC()
	if state != domain.ReservationStateActive || reservationLotID != lotID || effectiveNow.Before(due) {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}
	if !lotExists || lotDeleted != nil {
		if err := expireReservationWithoutLotTransitionTx(ctx, tx, id, "La reserva venció automáticamente; el lote ya no está activo"); err != nil {
			return false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return true, nil
	}

	var currentActiveID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM reservas WHERE lote_id = $1::uuid AND estado_actual = 'activa' FOR UPDATE`, lotID).Scan(&currentActiveID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if currentActiveID != id {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}
	if lotState != domain.LotStateReserved {
		if err := expireReservationWithoutLotTransitionTx(ctx, tx, id, "La reserva venció automáticamente; el lote ya no estaba reservado"); err != nil {
			return false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := expireReservationTx(ctx, tx, loteoID, lotID, id, effectiveNow); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func expireReservationTx(ctx context.Context, tx pgx.Tx, loteoID, lotID, reservationID string, now time.Time) error {
	var state domain.ReservationState
	var due time.Time
	if err := tx.QueryRow(ctx, `SELECT estado_actual, fecha_vencimiento FROM reservas WHERE id = $1::uuid AND lote_id = $2::uuid FOR UPDATE`, reservationID, lotID).Scan(&state, &due); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrReservationNotFound
		}
		return err
	}
	if state != domain.ReservationStateActive {
		return nil
	}
	if now.Before(due) {
		return domain.ErrReservationActiveConflict
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO reserva_estados (reserva_id, estado, razon)
		VALUES ($1::uuid, 'vencida', $2)
	`, reservationID, "La reserva venció automáticamente"); err != nil {
		return mapReservationWriteError(err)
	}
	_, err := transitionLotState(ctx, tx, domain.LotStateTransition{
		DevelopmentID: loteoID,
		LotID:         lotID,
		ExpectedState: domain.LotStateReserved,
		NextState:     domain.LotStateAvailable,
		Origin:        domain.LotStateOriginSystem,
		Reason:        "La reserva venció automáticamente",
		ReservationID: stringReference(reservationID),
	})
	return err
}

func expireReservationWithoutLotTransitionTx(ctx context.Context, tx pgx.Tx, reservationID, reason string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO reserva_estados (reserva_id, estado, razon)
		VALUES ($1::uuid, 'vencida', $2)
	`, reservationID, reason)
	return mapReservationWriteError(err)
}

func waitForReservationRetry(ctx context.Context, attempt int) error {
	backoff := time.Duration(20*(1<<attempt)+rand.Intn(20)) * time.Millisecond
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

const reservationColumns = `
	r.id::text, l.id::text, l.nombre, lo.id::text, COALESCE(lo.numero, ''),
	c.id::text, c.nombre, c.apellido, c.dni, c.celular, c.email,
	seller.id::text, seller.nombre, seller.apellido, seller.email, seller.rol,
	alta.id::text, alta.nombre, alta.apellido, alta.email, alta.rol,
	r.estado_actual, r.fecha_vencimiento, r.fecha_creacion, r.fecha_modificacion`

const reservationScopePredicate = `($%[1]d::uuid IS NULL OR ($%[2]d AND EXISTS (
	SELECT 1
	FROM usuarios actor
	JOIN inmobiliarias agency ON agency.id = actor.inmobiliaria_id
	JOIN usuarios seller_scope ON seller_scope.id = r.vendedor_id
	JOIN inmobiliaria_loteos il ON il.inmobiliaria_id = agency.id AND il.loteo_id = l.id
	WHERE actor.auth_provider_id = $%[1]d::uuid
	  AND actor.fecha_baja IS NULL AND agency.fecha_baja IS NULL
	  AND seller_scope.inmobiliaria_id = agency.id
	  AND il.fecha_baja IS NULL
)))`

func reservationInScope(ctx context.Context, queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string, scope gateway.ReservationScope) (bool, error) {
	predicate := fmt.Sprintf(reservationScopePredicate, 2, 3)
	var exists bool
	err := queryer.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM reservas r
			JOIN lotes lo ON lo.id = r.lote_id AND lo.fecha_baja IS NULL
			JOIN loteos l ON l.id = lo.loteo_id AND l.fecha_baja IS NULL
			WHERE r.id = $1::uuid AND `+predicate+`
		)
	`, id, scope.AssigneeAuthProviderID, scope.ByAgencyAssignment).Scan(&exists)
	if isInvalidUUID(err) {
		return false, domain.ErrReservationNotFound
	}
	return exists, err
}

type reservationQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadReservationFromPool(ctx context.Context, pool reservationQueryer, id string, scope gateway.ReservationScope, withHistory bool) (domain.Reservation, error) {
	predicate := fmt.Sprintf(reservationScopePredicate, 2, 3)
	row := pool.QueryRow(ctx, `SELECT `+reservationColumns+` FROM reservas r
		JOIN lotes lo ON lo.id = r.lote_id AND lo.fecha_baja IS NULL
		JOIN loteos l ON l.id = lo.loteo_id AND l.fecha_baja IS NULL
		JOIN clientes c ON c.id = r.cliente_id
		JOIN usuarios seller ON seller.id = r.vendedor_id
		JOIN usuarios alta ON alta.id = r.usuario_alta
		WHERE r.id = $1::uuid AND `+predicate,
		id, scope.AssigneeAuthProviderID, scope.ByAgencyAssignment)
	reservation, err := scanReservation(row)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	if err != nil {
		return domain.Reservation{}, err
	}
	if withHistory {
		reservation.Historial, err = loadHistory(ctx, pool, reservation.ID)
		if err != nil {
			return domain.Reservation{}, err
		}
	}
	return reservation, nil
}

func loadReservation(ctx context.Context, tx pgx.Tx, id string, withHistory bool) (domain.Reservation, error) {
	return loadReservationFromPool(ctx, tx, id, gateway.ReservationScope{}, withHistory)
}

func loadHistory(ctx context.Context, queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, reservationID string) ([]domain.ReservationHistoryEntry, error) {
	rows, err := queryer.Query(ctx, `
		SELECT re.id::text, re.estado, COALESCE(re.razon, ''), re.fecha_creacion,
		       u.id::text, u.nombre, u.apellido, u.email, u.rol
		FROM reserva_estados re
		LEFT JOIN usuarios u ON u.id = re.usuario_modificacion
		WHERE re.reserva_id = $1::uuid
		ORDER BY re.fecha_creacion, re.id
	`, reservationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := make([]domain.ReservationHistoryEntry, 0)
	for rows.Next() {
		var (
			entry                               domain.ReservationHistoryEntry
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

type reservationScanner interface {
	Scan(dest ...any) error
}

func scanReservation(row reservationScanner) (domain.Reservation, error) {
	var (
		reservation domain.Reservation
		client      domain.Cliente
		seller      domain.ReservationActor
		alta        domain.ReservationActor
	)
	err := row.Scan(
		&reservation.ID, &reservation.LoteoID, &reservation.LoteoNombre,
		&reservation.LoteID, &reservation.LoteNumero,
		&client.ID, &client.Nombre, &client.Apellido, &client.DNI, &client.Celular, &client.Email,
		&seller.ID, &seller.Nombre, &seller.Apellido, &seller.Email, &seller.Rol,
		&alta.ID, &alta.Nombre, &alta.Apellido, &alta.Email, &alta.Rol,
		&reservation.Estado, &reservation.FechaVencimiento, &reservation.FechaCreacion, &reservation.FechaModificacion,
	)
	if err != nil {
		return domain.Reservation{}, err
	}
	reservation.Cliente = client
	reservation.Vendedor = seller
	reservation.UsuarioAlta = alta
	return reservation, nil
}

func scanReservationWithTotal(row reservationScanner) (domain.Reservation, int64, error) {
	var total int64
	var reservation domain.Reservation
	var client domain.Cliente
	var seller, alta domain.ReservationActor
	err := row.Scan(
		&reservation.ID, &reservation.LoteoID, &reservation.LoteoNombre,
		&reservation.LoteID, &reservation.LoteNumero,
		&client.ID, &client.Nombre, &client.Apellido, &client.DNI, &client.Celular, &client.Email,
		&seller.ID, &seller.Nombre, &seller.Apellido, &seller.Email, &seller.Rol,
		&alta.ID, &alta.Nombre, &alta.Apellido, &alta.Email, &alta.Rol,
		&reservation.Estado, &reservation.FechaVencimiento, &reservation.FechaCreacion, &reservation.FechaModificacion,
		&total,
	)
	if err != nil {
		return domain.Reservation{}, 0, err
	}
	reservation.Cliente, reservation.Vendedor, reservation.UsuarioAlta = client, seller, alta
	return reservation, total, nil
}

func lockActor(ctx context.Context, tx pgx.Tx, id string) (domain.Rol, *string, error) {
	var role string
	var agency *string
	var inactive *time.Time
	err := tx.QueryRow(ctx, `SELECT rol, inmobiliaria_id::text, fecha_baja FROM usuarios WHERE id = $1::uuid FOR UPDATE`, id).Scan(&role, &agency, &inactive)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return "", nil, domain.ErrActorNoAprovisionado
	}
	if err != nil {
		return "", nil, err
	}
	if inactive != nil {
		return "", nil, domain.ErrCuentaInactiva
	}
	return domain.Rol(role), agency, nil
}

func lockSeller(ctx context.Context, tx pgx.Tx, id string) (domain.Rol, *string, bool, error) {
	var role string
	var agency *string
	var inactive *time.Time
	err := tx.QueryRow(ctx, `SELECT rol, inmobiliaria_id::text, fecha_baja FROM usuarios WHERE id = $1::uuid FOR UPDATE`, id).Scan(&role, &agency, &inactive)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return "", nil, false, domain.ErrReservationSellerNotEligible
	}
	if err != nil {
		return "", nil, false, err
	}
	return domain.Rol(role), agency, inactive == nil, nil
}

func lockActiveClient(ctx context.Context, tx pgx.Tx, id string) error {
	var active bool
	err := tx.QueryRow(ctx, `SELECT true FROM clientes WHERE id = $1::uuid AND fecha_baja IS NULL FOR UPDATE`, id).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.ErrReservationInvalidClient
	}
	if err != nil {
		return err
	}
	return nil
}

func agencyAssigned(ctx context.Context, tx pgx.Tx, agencyID *string, loteoID string) (bool, error) {
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
	if err != nil {
		return false, err
	}

	var assigned bool
	err = tx.QueryRow(ctx, `
		SELECT true
		FROM inmobiliaria_loteos il
		WHERE il.inmobiliaria_id = $1::uuid
		  AND il.loteo_id = $2::uuid
		  AND il.fecha_baja IS NULL
		LIMIT 1
		FOR SHARE
	`, *agencyID, loteoID).Scan(&assigned)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return assigned, err
}

func sameString(left, right *string) bool {
	return left != nil && right != nil && *left == *right
}

func stringReference(value string) *string { return &value }

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func rollbackTransaction(tx pgx.Tx) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = tx.Rollback(cleanupCtx)
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode
}

func isConstraint(err error, name string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == name
}

func mapReservationWriteError(err error) error {
	if isConstraint(err, reservationTransitionConstraint) {
		return domain.ErrReservationInvalidState
	}
	if isConstraint(err, reservationActiveIndex) {
		return domain.ErrReservationActiveConflict
	}
	return err
}

func isTransientDBError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001", "40P01", "55P03":
			return true
		}
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
