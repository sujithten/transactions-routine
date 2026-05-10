package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ts_project/transactions_routine/internal/domain"
	"github.com/ts_project/transactions_routine/internal/repository"
)

type TransactionService struct {
	db              repository.Beginner
	txRepo          repository.TransactionRepository
	opRepo          repository.OperationTypeRepository
	acctRepo        repository.AccountRepository
	installmentRepo repository.InstallmentRepository
}

func NewTransactionService(
	db repository.Beginner,
	txRepo repository.TransactionRepository,
	opRepo repository.OperationTypeRepository,
	acctRepo repository.AccountRepository,
	installmentRepo repository.InstallmentRepository,
) *TransactionService {
	return &TransactionService{
		db:              db,
		txRepo:          txRepo,
		opRepo:          opRepo,
		acctRepo:        acctRepo,
		installmentRepo: installmentRepo,
	}
}

func (s *TransactionService) CreateTransaction(ctx context.Context, accountID string, operationTypeID int, amount float64, installments int) (*domain.Transaction, error) {
	if _, err := s.acctRepo.GetAccountByID(ctx, accountID); err != nil {
		return nil, err
	}
	if _, err := s.opRepo.GetOperationTypeByID(ctx, operationTypeID); err != nil {
		return nil, err
	}

	t := &domain.Transaction{
		AccountID:       accountID,
		OperationTypeID: operationTypeID,
		Amount:          enforceSign(operationTypeID, amount),
		EventDate:       time.Now().UTC(),
	}

	pgxTx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer pgxTx.Rollback(ctx)

	created, err := s.txRepo.CreateTransaction(ctx, pgxTx, t)
	if err != nil {
		return nil, err
	}

	if operationTypeID == domain.OperationTypeInstallment {
		plan, err := s.buildInstallmentPlan(created, installments)
		if err != nil {
			return nil, err
		}
		saved, err := s.installmentRepo.CreatePlanWithSchedules(ctx, pgxTx, plan)
		if err != nil {
			return nil, err
		}
		created.InstallmentPlan = saved
	}

	if err := pgxTx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return created, nil
}

func (s *TransactionService) buildInstallmentPlan(tx *domain.Transaction, installments int) (*domain.InstallmentPlan, error) {
	perInstallment := -(math.Abs(tx.Amount) / float64(installments))

	schedules := make([]domain.InstallmentSchedule, installments)
	for i := range schedules {
		schedules[i] = domain.InstallmentSchedule{
			InstallmentNo: i + 1,
			Amount:        perInstallment,
			DueDate:       tx.EventDate.AddDate(0, i+1, 0),
		}
	}

	return &domain.InstallmentPlan{
		TransactionID:     tx.ID,
		TotalInstallments: installments,
		Schedules:         schedules,
	}, nil
}

func enforceSign(operationTypeID int, amount float64) float64 {
	abs := math.Abs(amount)
	if _, isDebit := domain.DebitOperationTypes[operationTypeID]; isDebit {
		return -abs
	}
	return abs
}
