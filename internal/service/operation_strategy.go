package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ts_project/transactions_routine/internal/domain"
	"github.com/ts_project/transactions_routine/internal/repository"
)

var ErrInvalidInput = errors.New("invalid input")

type OperationStrategy interface {
	Execute(ctx context.Context, accountID string, amount float64, installments int, dbtx repository.DBTX) (*domain.Transaction, error)
}

func NewDebitStrategy(txRepo repository.TransactionRepository, opTypeID int) OperationStrategy {
	return &debitStrategy{txRepo: txRepo, opTypeID: opTypeID}
}

func NewCreditStrategy(txRepo repository.TransactionRepository, opTypeID int) OperationStrategy {
	return &creditStrategy{txRepo: txRepo, opTypeID: opTypeID}
}

func NewInstallmentStrategy(txRepo repository.TransactionRepository, installmentRepo repository.InstallmentRepository) OperationStrategy {
	return &installmentStrategy{txRepo: txRepo, installmentRepo: installmentRepo}
}

type debitStrategy struct {
	txRepo   repository.TransactionRepository
	opTypeID int
}

func (d *debitStrategy) Execute(ctx context.Context, accountID string, amount float64, _ int, dbtx repository.DBTX) (*domain.Transaction, error) {
	return d.txRepo.CreateTransaction(ctx, dbtx, &domain.Transaction{
		AccountID:       accountID,
		OperationTypeID: d.opTypeID,
		Amount:          -math.Abs(amount),
		EventDate:       time.Now().UTC(),
	})
}

type creditStrategy struct {
	txRepo   repository.TransactionRepository
	opTypeID int
}

func (c *creditStrategy) Execute(ctx context.Context, accountID string, amount float64, _ int, dbtx repository.DBTX) (*domain.Transaction, error) {
	return c.txRepo.CreateTransaction(ctx, dbtx, &domain.Transaction{
		AccountID:       accountID,
		OperationTypeID: c.opTypeID,
		Amount:          math.Abs(amount),
		EventDate:       time.Now().UTC(),
	})
}

type installmentStrategy struct {
	txRepo          repository.TransactionRepository
	installmentRepo repository.InstallmentRepository
}

func (s *installmentStrategy) Execute(ctx context.Context, accountID string, amount float64, installments int, dbtx repository.DBTX) (*domain.Transaction, error) {
	if installments <= 0 {
		return nil, fmt.Errorf("%w: installments must be greater than 0", ErrInvalidInput)
	}

	now := time.Now().UTC()
	amt := -math.Abs(amount)

	created, err := s.txRepo.CreateTransaction(ctx, dbtx, &domain.Transaction{
		AccountID:       accountID,
		OperationTypeID: 2,
		Amount:          amt,
		EventDate:       now,
	})
	if err != nil {
		return nil, err
	}

	perInstallment := amt / float64(installments)
	schedules := make([]domain.InstallmentSchedule, installments)
	for i := range schedules {
		schedules[i] = domain.InstallmentSchedule{
			InstallmentNo: i + 1,
			Amount:        perInstallment,
			DueDate:       now.AddDate(0, i+1, 0),
		}
	}

	plan, err := s.installmentRepo.CreatePlanWithSchedules(ctx, dbtx, &domain.InstallmentPlan{
		TransactionID:     created.ID,
		TotalInstallments: installments,
		Schedules:         schedules,
	})
	if err != nil {
		return nil, err
	}

	created.InstallmentPlan = plan
	return created, nil
}
