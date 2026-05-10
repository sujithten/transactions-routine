package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ts_project/transactions_routine/internal/domain"
)

type pgxAccountRepo struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) AccountRepository {
	return &pgxAccountRepo{pool: pool}
}

func (r *pgxAccountRepo) CreateAccount(ctx context.Context, documentNumber string) (*domain.Account, error) {
	rows, err := r.pool.Query(ctx,
		`INSERT INTO accounts (document_number) VALUES ($1) RETURNING id, document_number, created_on`,
		documentNumber,
	)
	if err != nil {
		return nil, err
	}
	a, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[domain.Account])
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicateDocument
		}
		return nil, err
	}
	return a, nil
}

func (r *pgxAccountRepo) GetAccountByID(ctx context.Context, accountID string) (*domain.Account, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, document_number, created_on FROM accounts WHERE id = $1`,
		accountID,
	)
	if err != nil {
		return nil, err
	}
	a, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[domain.Account])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}
