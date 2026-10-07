package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

const reservationConvertedReason = "Reserva convertida en venta"

// ConvertReservation registers the venta of an active reserva in one
// transaction: the venta (with its plan), the reserva's convertida event and
// the lote's reservado -> vendido event, plus the contado settlement. The
// loteo, lote and reserva are locked in the same order cancellation and the
// expiry worker use, so only one of them wins over a reserva and a converted
// reserva is never released.
func (repository *SaleRepository) ConvertReservation(ctx context.Context, command gateway.ConvertReservationCommand) (domain.Sale, error) {
	var sale domain.Sale
	var err error
	for attempt := 0; attempt < saleRetryCount; attempt++ {
		sale, err = repository.convertReservation(ctx, command)
		if err == nil || errors.Is(err, errRetrySaleWrite) || !isTransientDBError(err) || attempt == saleRetryCount-1 {
			break
		}
		if waitErr := waitForReservationRetry(ctx, attempt); waitErr != nil {
			return domain.Sale{}, waitErr
		}
	}
	if err == nil || !errors.Is(err, errRetrySaleWrite) {
		return sale, err
	}
	return repository.reconcileConversion(ctx, command, err)
}

type lockedReservation struct {
	clientID, sellerID string
	state              domain.ReservationState
	due                time.Time
}

func (repository *SaleRepository) convertReservation(ctx context.Context, command gateway.ConvertReservationCommand) (domain.Sale, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Sale{}, err
	}
	defer rollbackTransaction(tx)

	// Locates the lote only, within the actor's scope as of now; vigencia and
	// permissions are decided again after the locks below.
	var unlockedRole string
	err = tx.QueryRow(ctx, `SELECT rol FROM usuarios WHERE id = $1::uuid`, command.ActorID).Scan(&unlockedRole)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Sale{}, domain.ErrActorNoAprovisionado
	}
	if err != nil {
		return domain.Sale{}, err
	}
	locateScope := conversionScope(domain.Rol(unlockedRole), command.ActorAuthProviderID)
	var lotID, developmentID string
	err = tx.QueryRow(ctx, `
		SELECT r.lote_id::text, lo.loteo_id::text
		FROM reservas r
		JOIN lotes lo ON lo.id = r.lote_id AND lo.fecha_baja IS NULL
		JOIN loteos l ON l.id = lo.loteo_id AND l.fecha_baja IS NULL
		WHERE r.id = $1::uuid AND `+fmt.Sprintf(reservationScopePredicate, 2, 3),
		command.ReservationID, locateScope.AssigneeAuthProviderID, locateScope.ByAgencyAssignment).Scan(&lotID, &developmentID)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return domain.Sale{}, domain.ErrReservationNotFound
	}
	if err != nil {
		return domain.Sale{}, err
	}

	if _, err := lockActiveDevelopment(ctx, tx, developmentID, domain.ErrReservationNotFound); err != nil {
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
	`, lotID, developmentID).Scan(&lotState, &lotNumber, &lotPrice, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Sale{}, domain.ErrReservationNotFound
	}
	if err != nil {
		return domain.Sale{}, err
	}
	reservation, err := lockReservation(ctx, tx, command.ReservationID, lotID)
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
	inScope, err := reservationInScope(ctx, tx, command.ReservationID, conversionScope(actorRole, command.ActorAuthProviderID))
	if err != nil {
		return domain.Sale{}, err
	}
	if !inScope {
		return domain.Sale{}, domain.ErrReservationNotFound
	}
	if !domain.CanConvertReservation(actorRole, command.ActorID, reservation.sellerID) {
		return domain.Sale{}, domain.ErrReservationConvertForbidden
	}

	// A replay returns the venta already registered before any check on the
	// reserva's current state, which the first attempt already changed.
	existing, replayed, err := replayConversion(ctx, tx, command)
	if err != nil {
		return domain.Sale{}, err
	}
	if replayed {
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return domain.Sale{}, fmt.Errorf("%w: %w", errRetrySaleWrite, commitErr)
		}
		return existing, nil
	}

	switch reservation.state {
	case domain.ReservationStateCancelled:
		return domain.Sale{}, domain.ErrReservationAlreadyCancelled
	case domain.ReservationStateExpired:
		return domain.Sale{}, domain.ErrReservationConversionExpired
	case domain.ReservationStateConverted:
		return domain.Sale{}, domain.ErrReservationConverted
	}
	if lotState != domain.LotStateReserved {
		return domain.Sale{}, domain.ErrSaleLotUnavailable
	}
	reservedByIt, err := lotReservedBy(ctx, tx, lotID, command.ReservationID)
	if err != nil {
		return domain.Sale{}, err
	}
	if !reservedByIt {
		return domain.Sale{}, domain.ErrSaleLotUnavailable
	}
	if actorRole == domain.RolInmobiliaria {
		assigned, assignmentErr := agencyAssigned(ctx, tx, actorAgency, developmentID)
		if assignmentErr != nil {
			return domain.Sale{}, assignmentErr
		}
		if !assigned {
			return domain.Sale{}, domain.ErrSaleAgencyNotAssigned
		}
	}
	if reservation.sellerID != command.ActorID {
		if _, err := lockEligibleSeller(ctx, tx, reservation.sellerID); err != nil {
			return domain.Sale{}, err
		}
	}
	if err := lockActiveClient(ctx, tx, reservation.clientID); err != nil {
		if errors.Is(err, domain.ErrReservationInvalidClient) {
			return domain.Sale{}, domain.ErrSaleInvalidClient
		}
		return domain.Sale{}, err
	}
	if lotNumber == "" || lotPrice == nil || *lotPrice <= 0 || currency == "" {
		return domain.Sale{}, domain.ErrSaleLotIncomplete
	}

	// Read only after every lock that may wait: a slow conversion must not
	// stretch a reserva past its due instant. An expired reserva is left for
	// the expiry worker to release; the conversion never touches the lote.
	convertedAt := repository.clock.Now().UTC()
	if !convertedAt.Before(reservation.due) {
		return domain.Sale{}, domain.ErrReservationConversionExpired
	}

	saleID, err := writeSale(ctx, tx, saleWrite{
		developmentID:          developmentID,
		lotID:                  lotID,
		lotState:               lotState,
		clientID:               reservation.clientID,
		sellerID:               reservation.sellerID,
		actorID:                command.ActorID,
		reservationID:          stringReference(command.ReservationID),
		method:                 command.PaymentMethod,
		plan:                   command.PaymentPlan,
		price:                  *lotPrice,
		currency:               currency,
		createdAt:              convertedAt,
		idempotencyKey:         command.IdempotencyKey,
		idempotencyPayloadHash: command.IdempotencyPayloadHash,
	})
	if err != nil {
		return domain.Sale{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reserva_estados (reserva_id, estado, razon, usuario_modificacion)
		VALUES ($1::uuid, 'convertida', $2, $3::uuid)
	`, command.ReservationID, reservationConvertedReason, command.ActorID); err != nil {
		return domain.Sale{}, mapReservationWriteError(err)
	}
	if repository.afterConversionWrite != nil {
		if err := repository.afterConversionWrite(ctx, tx); err != nil {
			return domain.Sale{}, err
		}
	}

	sale, err := loadSale(ctx, tx, saleID, gateway.SaleScope{}, true)
	if err != nil {
		return domain.Sale{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Sale{}, fmt.Errorf("%w: %w", errRetrySaleWrite, err)
	}
	return sale, nil
}

// conversionScope is the reserva read scope of the actor's persisted role,
// never of the token's claims: an agency user only reaches the reservas of
// their own agency, so one outside it is reported as missing.
func conversionScope(role domain.Rol, authProviderID string) gateway.ReservationScope {
	if role != domain.RolInmobiliaria {
		return gateway.ReservationScope{}
	}
	return normalizeReservationScope(gateway.ReservationScope{AssigneeAuthProviderID: &authProviderID, ByAgencyAssignment: true})
}

// lotReservedBy reports whether the lote's latest state event is the
// reservado written by this reserva, so a lote held by another operation is
// never sold through it.
func lotReservedBy(ctx context.Context, tx pgx.Tx, lotID, reservationID string) (bool, error) {
	var reservedBy *string
	err := tx.QueryRow(ctx, `
		SELECT CASE WHEN estado = 'reservado' THEN reserva_id::text END
		FROM lote_estados
		WHERE lote_id = $1::uuid
		ORDER BY fecha_creacion DESC, id DESC
		LIMIT 1
	`, lotID).Scan(&reservedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return reservedBy != nil && *reservedBy == reservationID, nil
}

func lockReservation(ctx context.Context, tx pgx.Tx, reservationID, lotID string) (lockedReservation, error) {
	reservation := lockedReservation{}
	err := tx.QueryRow(ctx, `
		SELECT cliente_id::text, vendedor_id::text, estado_actual, fecha_vencimiento
		FROM reservas
		WHERE id = $1::uuid AND lote_id = $2::uuid
		FOR UPDATE
	`, reservationID, lotID).Scan(&reservation.clientID, &reservation.sellerID, &reservation.state, &reservation.due)
	if errors.Is(err, pgx.ErrNoRows) {
		return lockedReservation{}, domain.ErrReservationNotFound
	}
	return reservation, err
}

// replayConversion looks the actor's key up. The same key with other terms,
// another reserva or an ordinary sale is a conflict.
func replayConversion(ctx context.Context, tx pgx.Tx, command gateway.ConvertReservationCommand) (domain.Sale, bool, error) {
	var existingID, existingHash string
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(idempotency_payload_hash, '')
		FROM ventas
		WHERE usuario_alta = $1::uuid AND idempotency_key = $2
		FOR UPDATE
	`, command.ActorID, command.IdempotencyKey).Scan(&existingID, &existingHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Sale{}, false, nil
	}
	if err != nil {
		return domain.Sale{}, false, err
	}
	if existingHash != command.IdempotencyPayloadHash {
		return domain.Sale{}, false, domain.ErrSaleIdempotencyConflict
	}
	existing, err := loadSale(ctx, tx, existingID, gateway.SaleScope{}, true)
	if err != nil {
		return domain.Sale{}, false, err
	}
	if existing.ReservaID == nil || *existing.ReservaID != command.ReservationID {
		return domain.Sale{}, false, domain.ErrSaleIdempotencyConflict
	}
	return existing, true, nil
}

// reconcileConversion resolves a write whose outcome is unknown (a commit
// that may have landed, or a duplicate key raced in by a concurrent retry).
// It re-runs the conversion's own authorization before handing the venta
// back, so a key alone never discloses it.
func (repository *SaleRepository) reconcileConversion(ctx context.Context, command gateway.ConvertReservationCommand, original error) (domain.Sale, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.Sale{}, original
	}
	defer rollbackTransaction(tx)

	var role string
	var inactive *time.Time
	err = tx.QueryRow(ctx, `SELECT rol, fecha_baja FROM usuarios WHERE id = $1::uuid`, command.ActorID).Scan(&role, &inactive)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Sale{}, domain.ErrActorNoAprovisionado
	}
	if err != nil {
		return domain.Sale{}, original
	}
	if inactive != nil {
		return domain.Sale{}, domain.ErrCuentaInactiva
	}
	if !domain.IsSaleRole(domain.Rol(role)) {
		return domain.Sale{}, domain.ErrNoAutorizado
	}
	inScope, err := reservationInScope(ctx, tx, command.ReservationID, conversionScope(domain.Rol(role), command.ActorAuthProviderID))
	if err != nil {
		return domain.Sale{}, original
	}
	if !inScope {
		return domain.Sale{}, domain.ErrReservationNotFound
	}

	var id, hash string
	err = tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(idempotency_payload_hash, '')
		FROM ventas
		WHERE usuario_alta = $1::uuid AND idempotency_key = $2
	`, command.ActorID, command.IdempotencyKey).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Sale{}, original
	}
	if err != nil {
		return domain.Sale{}, original
	}
	if hash != command.IdempotencyPayloadHash {
		return domain.Sale{}, domain.ErrSaleIdempotencyConflict
	}
	sale, err := loadSale(ctx, tx, id, gateway.SaleScope{}, true)
	if err != nil {
		return domain.Sale{}, original
	}
	if sale.ReservaID == nil || *sale.ReservaID != command.ReservationID ||
		!domain.CanConvertReservation(domain.Rol(role), command.ActorID, sale.Vendedor.ID) {
		return domain.Sale{}, domain.ErrSaleIdempotencyConflict
	}
	return sale, nil
}
