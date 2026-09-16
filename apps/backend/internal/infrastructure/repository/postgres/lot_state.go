package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

const (
	lotStateTransitionConstraint = "lote_estados_transition_chk"
	lotReservationConstraint     = "lote_estados_reserva_lote_fk"
	lotSaleConstraint            = "lote_estados_venta_lote_fk"
)

type LotStateRepository struct {
	pool *pgxpool.Pool
}

func NewLotStateRepository(pool *pgxpool.Pool) *LotStateRepository {
	return &LotStateRepository{pool: pool}
}

var lotExistsInScopeSQL = `
	SELECT EXISTS (
		SELECT 1
		FROM lotes lo
		JOIN loteos l ON l.id = lo.loteo_id
		WHERE lo.id = $2::uuid
			AND lo.loteo_id = $1::uuid
			AND lo.fecha_baja IS NULL
			AND l.fecha_baja IS NULL
			AND ` + fmt.Sprintf(loteoScopedPredicate, 3, 4, 5) + `
	)
`

func (repository *LotStateRepository) LotExists(
	ctx context.Context,
	developmentID, lotID string,
	scope gateway.LoteoScope,
) (bool, error) {
	var exists bool
	err := repository.pool.QueryRow(ctx, lotExistsInScopeSQL,
		developmentID, lotID,
		scope.AssigneeAuthProviderID, scope.ByUserAssignment, scope.ByAgencyAssignment,
	).Scan(&exists)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return false, nil
		}
		return false, err
	}

	return exists, nil
}

func (repository *LotStateRepository) Transition(
	ctx context.Context,
	command domain.LotStateTransition,
) (domain.LotStateEvent, error) {
	if err := command.Validate(); err != nil {
		return domain.LotStateEvent{}, err
	}

	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.LotStateEvent{}, err
	}
	defer tx.Rollback(ctx)

	event, err := transitionLotState(ctx, tx, command)
	if err != nil {
		return domain.LotStateEvent{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.LotStateEvent{}, err
	}

	return event, nil
}

func transitionLotState(
	ctx context.Context,
	tx pgx.Tx,
	command domain.LotStateTransition,
) (domain.LotStateEvent, error) {
	if err := command.Validate(); err != nil {
		return domain.LotStateEvent{}, err
	}

	var current domain.LotState
	err := tx.QueryRow(ctx, `
		SELECT estado_actual
		FROM lotes
		WHERE id = $2::uuid AND loteo_id = $1::uuid AND fecha_baja IS NULL
		FOR UPDATE
	`, command.DevelopmentID, command.LotID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LotStateEvent{}, domain.ErrLoteNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return domain.LotStateEvent{}, domain.ErrLoteNotFound
		}
		return domain.LotStateEvent{}, err
	}
	if current != command.ExpectedState {
		return domain.LotStateEvent{}, domain.ErrLotStateConflict
	}
	return insertLotStateEvent(ctx, tx, command, current, true)
}

func transitionLotStateWithLockedLot(
	ctx context.Context,
	tx pgx.Tx,
	command domain.LotStateTransition,
	current domain.LotState,
) (domain.LotStateEvent, error) {
	if err := command.Validate(); err != nil {
		return domain.LotStateEvent{}, err
	}
	return insertLotStateEvent(ctx, tx, command, current, false)
}

func insertLotStateEvent(
	ctx context.Context,
	tx pgx.Tx,
	command domain.LotStateTransition,
	current domain.LotState,
	verifyApplied bool,
) (domain.LotStateEvent, error) {
	if current != command.ExpectedState {
		return domain.LotStateEvent{}, domain.ErrLotStateConflict
	}

	event := domain.LotStateEvent{
		LotID:         command.LotID,
		PreviousState: current,
		State:         command.NextState,
		Origin:        command.Origin,
		Reason:        command.Reason,
		ActorID:       command.ActorID,
		ReservationID: command.ReservationID,
		SaleID:        command.SaleID,
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO lote_estados (
			lote_id, estado, origen, razon, usuario_modificacion,
			reserva_id, venta_id
		)
		VALUES (
			$1::uuid, $2, $3, NULLIF($4, ''), NULLIF($5, '')::uuid,
			NULLIF($6, '')::uuid, NULLIF($7, '')::uuid
		)
		RETURNING id::text, fecha_creacion
	`, command.LotID, command.NextState, command.Origin, command.Reason,
		command.ActorID, referenceValue(command.ReservationID), referenceValue(command.SaleID),
	).Scan(&event.ID, &event.OccurredAt)
	if err != nil {
		return domain.LotStateEvent{}, mapLotStateWriteError(err)
	}

	if !verifyApplied {
		return event, nil
	}

	var applied domain.LotState
	if err := tx.QueryRow(ctx, `
		SELECT estado_actual FROM lotes WHERE id = $1::uuid
	`, command.LotID).Scan(&applied); err != nil {
		return domain.LotStateEvent{}, err
	}
	if applied != command.NextState {
		return domain.LotStateEvent{}, fmt.Errorf("lote state trigger applied %q, want %q", applied, command.NextState)
	}

	return event, nil
}

func referenceValue(reference *string) string {
	if reference == nil {
		return ""
	}
	return *reference
}

func mapLotStateWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case lotStateTransitionConstraint:
			return domain.ErrInvalidLotStateTransition
		case lotReservationConstraint, lotSaleConstraint:
			return domain.ErrInvalidLotStateReference
		}
	}
	return err
}
