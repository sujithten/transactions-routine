package handler

import (
	"context"

	"github.com/ts_project/transactions_routine/internal/domain"
)

type accountService interface {
	CreateAccount(ctx context.Context, documentNumber string) (*domain.Account, error)
	GetAccount(ctx context.Context, accountID string) (*domain.Account, error)
}

type transactionService interface {
	CreateTransaction(ctx context.Context, accountID string, operationTypeID int, amount float64, installments int) (*domain.Transaction, error)
}
