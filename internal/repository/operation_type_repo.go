package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ts_project/transactions_routine/internal/domain"
)

type pgxOperationTypeRepo struct {
	pool *pgxpool.Pool
}

func NewOperationTypeRepository(pool *pgxpool.Pool) OperationTypeRepository {
	return &pgxOperationTypeRepo{pool: pool}
}

func (r *pgxOperationTypeRepo) GetOperationTypeByID(ctx context.Context, id int) (*domain.OperationType, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, description, created_on FROM operation_types WHERE id = $1`,
		id,
	)
	if err != nil {
		return nil, err
	}
	ot, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[domain.OperationType])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return ot, err
}
