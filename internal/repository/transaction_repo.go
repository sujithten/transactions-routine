package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ts_project/transactions_routine/internal/domain"
)

type pgxTransactionRepo struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) TransactionRepository {
	return &pgxTransactionRepo{pool: pool}
}

func (r *pgxTransactionRepo) CreateTransaction(ctx context.Context, dbtx DBTX, t *domain.Transaction) (*domain.Transaction, error) {
	rows, err := dbtx.Query(ctx,
		`INSERT INTO transactions (account_id, operation_type_id, amount, event_date)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, account_id, operation_type_id, amount, event_date, created_on`,
		t.AccountID, t.OperationTypeID, t.Amount, t.EventDate,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[domain.Transaction])
}
