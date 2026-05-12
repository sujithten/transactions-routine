package service

import (
	"context"
	"fmt"

	"github.com/ts_project/transactions_routine/internal/domain"
	"github.com/ts_project/transactions_routine/internal/repository"
)

type TransactionService struct {
	db         repository.Beginner
	acctRepo   repository.AccountRepository
	strategies map[int]OperationStrategy
}

func NewTransactionService(
	db repository.Beginner,
	acctRepo repository.AccountRepository,
	strategies map[int]OperationStrategy,
) *TransactionService {
	return &TransactionService{
		db:         db,
		acctRepo:   acctRepo,
		strategies: strategies,
	}
}

func (s *TransactionService) CreateTransaction(ctx context.Context, accountID string, operationTypeID int, amount float64, installments int) (*domain.Transaction, error) {
	if _, err := s.acctRepo.GetAccountByID(ctx, accountID); err != nil {
		return nil, err
	}

	strategy, ok := s.strategies[operationTypeID]
	if !ok {
		return nil, repository.ErrNotFound
	}

	pgxTx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer pgxTx.Rollback(ctx)

	created, err := strategy.Execute(ctx, accountID, amount, installments, pgxTx)
	if err != nil {
		return nil, err
	}

	if err := pgxTx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return created, nil
}
