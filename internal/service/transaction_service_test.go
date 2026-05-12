package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ts_project/transactions_routine/internal/domain"
	"github.com/ts_project/transactions_routine/internal/repository"
)

// --- stubs ---

type stubBeginner struct{}

func (s *stubBeginner) Begin(_ context.Context) (repository.Tx, error) {
	return &stubTx{}, nil
}

type stubTx struct{}

func (s *stubTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) { return nil, nil }
func (s *stubTx) Commit(_ context.Context) error                                { return nil }
func (s *stubTx) Rollback(_ context.Context) error                              { return nil }

type stubAccountRepo struct {
	account *domain.Account
	err     error
}

func (s *stubAccountRepo) CreateAccount(_ context.Context, documentNumber string) (*domain.Account, error) {
	return s.account, s.err
}

func (s *stubAccountRepo) GetAccountByID(_ context.Context, _ string) (*domain.Account, error) {
	return s.account, s.err
}

type stubTransactionRepo struct {
	eventDate time.Time
}

func (s *stubTransactionRepo) CreateTransaction(_ context.Context, _ repository.DBTX, tx *domain.Transaction) (*domain.Transaction, error) {
	tx.ID = "00000000-0000-0000-0000-000000000001"
	if !s.eventDate.IsZero() {
		tx.EventDate = s.eventDate
	}
	return tx, nil
}

type stubInstallmentRepo struct{}

func (s *stubInstallmentRepo) CreatePlanWithSchedules(_ context.Context, _ repository.DBTX, plan *domain.InstallmentPlan) (*domain.InstallmentPlan, error) {
	plan.ID = "00000000-0000-0000-0000-000000000002"
	for i := range plan.Schedules {
		plan.Schedules[i].ID = "00000000-0000-0000-0000-00000000000" + string(rune('3'+i))
	}
	return plan, nil
}

func newSvc(txRepo repository.TransactionRepository) *TransactionService {
	return NewTransactionService(
		&stubBeginner{},
		&stubAccountRepo{account: &domain.Account{ID: "00000000-0000-0000-0000-000000000001"}},
		map[int]OperationStrategy{
			1: NewDebitStrategy(txRepo, 1),
			2: NewInstallmentStrategy(txRepo, &stubInstallmentRepo{}),
			3: NewDebitStrategy(txRepo, 3),
			4: NewCreditStrategy(txRepo, 4),
		},
	)
}

// --- CreateTransaction: sign enforcement ---

func TestCreateTransaction_SignEnforced(t *testing.T) {
	tx, err := newSvc(&stubTransactionRepo{}).CreateTransaction(context.Background(), "00000000-0000-0000-0000-000000000001", 1, 50.0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Amount >= 0 {
		t.Errorf("expected negative amount for debit op, got %f", tx.Amount)
	}
}

func TestCreateTransaction_InvalidAccount(t *testing.T) {
	svc := NewTransactionService(
		&stubBeginner{},
		&stubAccountRepo{err: repository.ErrNotFound},
		map[int]OperationStrategy{
			1: NewDebitStrategy(&stubTransactionRepo{}, 1),
		},
	)
	_, err := svc.CreateTransaction(context.Background(), "bad-uuid", 1, 50.0, 0)
	if err == nil {
		t.Fatal("expected error for invalid account, got nil")
	}
}

func TestCreateTransaction_InvalidOperationType(t *testing.T) {
	_, err := newSvc(&stubTransactionRepo{}).CreateTransaction(context.Background(), "00000000-0000-0000-0000-000000000001", 99, 50.0, 0)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown op type, got %v", err)
	}
}

// --- CreateTransaction: installment plan ---

func TestCreateTransaction_InstallmentPlanCreated(t *testing.T) {
	tx, err := newSvc(&stubTransactionRepo{}).CreateTransaction(context.Background(), "00000000-0000-0000-0000-000000000001", 2, 3000.0, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.InstallmentPlan == nil {
		t.Fatal("expected installment_plan to be set, got nil")
	}
	if len(tx.InstallmentPlan.Schedules) != 3 {
		t.Errorf("expected 3 schedules, got %d", len(tx.InstallmentPlan.Schedules))
	}
}

func TestCreateTransaction_InstallmentAmountsAreNegative(t *testing.T) {
	tx, err := newSvc(&stubTransactionRepo{}).CreateTransaction(context.Background(), "00000000-0000-0000-0000-000000000001", 2, 1200.0, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range tx.InstallmentPlan.Schedules {
		if s.Amount >= 0 {
			t.Errorf("installment %d: expected negative amount, got %f", s.InstallmentNo, s.Amount)
		}
	}
}

func TestCreateTransaction_DueDatesAreMonthlyFromEventDate(t *testing.T) {
	base := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	tx, err := newSvc(&stubTransactionRepo{eventDate: base}).CreateTransaction(context.Background(), "00000000-0000-0000-0000-000000000001", 2, 600.0, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, s := range tx.InstallmentPlan.Schedules {
		want := base.AddDate(0, i+1, 0)
		if !s.DueDate.Equal(want) {
			t.Errorf("schedule %d: want due_date %v, got %v", i+1, want, s.DueDate)
		}
	}
}

func TestCreateTransaction_NonInstallmentHasNoPlan(t *testing.T) {
	tx, err := newSvc(&stubTransactionRepo{}).CreateTransaction(context.Background(), "00000000-0000-0000-0000-000000000001", 1, 100.0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.InstallmentPlan != nil {
		t.Errorf("expected no installment_plan for op_type=1, got %+v", tx.InstallmentPlan)
	}
}

func TestCreateTransaction_PerInstallmentAmountIsCorrect(t *testing.T) {
	tx, err := newSvc(&stubTransactionRepo{}).CreateTransaction(context.Background(), "00000000-0000-0000-0000-000000000001", 2, 3000.0, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := -1000.0
	for _, s := range tx.InstallmentPlan.Schedules {
		if s.Amount != want {
			t.Errorf("expected per-installment amount %f, got %f", want, s.Amount)
		}
	}
}

func TestCreateTransaction_CreditVoucherAmountIsPositive(t *testing.T) {
	tx, err := newSvc(&stubTransactionRepo{}).CreateTransaction(context.Background(), "00000000-0000-0000-0000-000000000001", 4, 500.0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Amount <= 0 {
		t.Errorf("expected positive amount for credit voucher, got %f", tx.Amount)
	}
}

func TestCreateTransaction_MissingInstallments(t *testing.T) {
	_, err := newSvc(&stubTransactionRepo{}).CreateTransaction(context.Background(), "00000000-0000-0000-0000-000000000001", 2, 1000.0, 0)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for zero installments, got %v", err)
	}
}
