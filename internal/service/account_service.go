package service

import (
	"context"
	"errors"

	"github.com/ts_project/transactions_routine/internal/domain"
	"github.com/ts_project/transactions_routine/internal/repository"
)

type AccountService struct {
	repo repository.AccountRepository
}

func NewAccountService(repo repository.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

func (s *AccountService) CreateAccount(ctx context.Context, documentNumber string) (*domain.Account, error) {
	if documentNumber == "" {
		return nil, errors.New("document_number is required")
	}
	return s.repo.CreateAccount(ctx, documentNumber)
}

func (s *AccountService) GetAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	return s.repo.GetAccountByID(ctx, accountID)
}
