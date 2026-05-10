package service

import (
	"context"
	"testing"

	"github.com/ts_project/transactions_routine/internal/domain"
	"github.com/ts_project/transactions_routine/internal/repository"
)

func newAccountSvc(repo repository.AccountRepository) *AccountService {
	return NewAccountService(repo)
}

// --- CreateAccount ---

func TestCreateAccount_Success(t *testing.T) {
	repo := &stubAccountRepo{account: &domain.Account{ID: "uuid-1", DocumentNumber: "12345678900"}}
	got, err := newAccountSvc(repo).CreateAccount(context.Background(), "12345678900")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DocumentNumber != "12345678900" {
		t.Errorf("expected document_number 12345678900, got %s", got.DocumentNumber)
	}
}

func TestCreateAccount_EmptyDocumentNumber(t *testing.T) {
	_, err := newAccountSvc(&stubAccountRepo{}).CreateAccount(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty document_number, got nil")
	}
}

func TestCreateAccount_DuplicateDocument(t *testing.T) {
	repo := &stubAccountRepo{err: repository.ErrDuplicateDocument}
	_, err := newAccountSvc(repo).CreateAccount(context.Background(), "12345678900")
	if err == nil {
		t.Fatal("expected error for duplicate document, got nil")
	}
}

// --- GetAccount ---

func TestGetAccount_Success(t *testing.T) {
	repo := &stubAccountRepo{account: &domain.Account{ID: "uuid-1", DocumentNumber: "12345678900"}}
	got, err := newAccountSvc(repo).GetAccount(context.Background(), "uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "uuid-1" {
		t.Errorf("expected id uuid-1, got %s", got.ID)
	}
}

func TestGetAccount_NotFound(t *testing.T) {
	repo := &stubAccountRepo{err: repository.ErrNotFound}
	_, err := newAccountSvc(repo).GetAccount(context.Background(), "non-existent")
	if err == nil {
		t.Fatal("expected error for non-existent account, got nil")
	}
}
