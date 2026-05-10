package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/ts_project/transactions_routine/internal/domain"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrDuplicateDocument = errors.New("document number already registered")
)

// DBTX is satisfied by both *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Tx extends DBTX with commit/rollback so the service can manage the boundary.
type Tx interface {
	DBTX
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Beginner starts a Tx; satisfied by a thin wrapper around *pgxpool.Pool.
type Beginner interface {
	Begin(ctx context.Context) (Tx, error)
}

type AccountRepository interface {
	CreateAccount(ctx context.Context, documentNumber string) (*domain.Account, error)
	GetAccountByID(ctx context.Context, accountID string) (*domain.Account, error)
}

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, dbtx DBTX, t *domain.Transaction) (*domain.Transaction, error)
}

type OperationTypeRepository interface {
	GetOperationTypeByID(ctx context.Context, id int) (*domain.OperationType, error)
}

type InstallmentRepository interface {
	CreatePlanWithSchedules(ctx context.Context, dbtx DBTX, plan *domain.InstallmentPlan) (*domain.InstallmentPlan, error)
}
