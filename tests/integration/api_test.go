//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ts_project/transactions_routine/db"
	"github.com/ts_project/transactions_routine/internal/domain"
	"github.com/ts_project/transactions_routine/internal/handler"
	"github.com/ts_project/transactions_routine/internal/repository"
	"github.com/ts_project/transactions_routine/internal/service"
)

var (
	srv  *httptest.Server
	pool *pgxpool.Pool
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/transactions_db?sslmode=disable"
	}

	if err := runMigrations(dbURL); err != nil {
		panic("migrations failed: " + err.Error())
	}

	var err error
	pool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		panic("db connect: " + err.Error())
	}

	acctRepo := repository.NewAccountRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	installmentRepo := repository.NewInstallmentRepository(pool)

	acctSvc := service.NewAccountService(acctRepo)
	txSvc := service.NewTransactionService(
		repository.NewPoolBeginner(pool),
		acctRepo,
		map[int]service.OperationStrategy{
			1: service.NewDebitStrategy(txRepo, 1),
			2: service.NewInstallmentStrategy(txRepo, installmentRepo),
			3: service.NewDebitStrategy(txRepo, 3),
			4: service.NewCreditStrategy(txRepo, 4),
		},
	)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts", handler.NewAccountHandler(acctSvc).CreateAccount)
	mux.HandleFunc("GET /accounts/{accountId}", handler.NewAccountHandler(acctSvc).GetAccount)
	mux.HandleFunc("POST /transactions", handler.NewTransactionHandler(txSvc).CreateTransaction)

	srv = httptest.NewServer(mux)

	truncateAll(pool)
	code := m.Run()
	truncateAll(pool)

	srv.Close()
	pool.Close()
	os.Exit(code)
}

func runMigrations(dbURL string) error {
	src, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, strings.Replace(dbURL, "postgres://", "pgx5://", 1))
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func truncateAll(p *pgxpool.Pool) {
	_, err := p.Exec(context.Background(),
		"TRUNCATE installment_schedules, installment_plans, transactions, accounts CASCADE")
	if err != nil {
		panic("truncateAll: " + err.Error())
	}
}

// truncate clears all data between tests. CASCADE handles FK order automatically.
func truncate(t *testing.T) {
	t.Helper()
	truncateAll(pool)
}

func postJSON(t *testing.T, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func getJSON(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// createAccount is a test helper that creates an account and returns it.
func createAccount(t *testing.T, documentNumber string) domain.Account {
	t.Helper()
	resp := postJSON(t, "/accounts", map[string]string{"document_number": documentNumber})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("createAccount: expected 201, got %d", resp.StatusCode)
	}
	var a domain.Account
	decodeBody(t, resp, &a)
	return a
}

// ── Account tests ────────────────────────────────────────────────────────────

func TestCreateAccount_Success(t *testing.T) {
	truncate(t)

	resp := postJSON(t, "/accounts", map[string]string{"document_number": "12345678900"})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var account domain.Account
	decodeBody(t, resp, &account)

	if account.ID == "" {
		t.Error("expected non-empty account id")
	}
	if account.DocumentNumber != "12345678900" {
		t.Errorf("expected document_number 12345678900, got %s", account.DocumentNumber)
	}
}

func TestCreateAccount_DuplicateDocument(t *testing.T) {
	truncate(t)
	createAccount(t, "12345678900")

	resp := postJSON(t, "/accounts", map[string]string{"document_number": "12345678900"})

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestCreateAccount_MissingDocumentNumber(t *testing.T) {
	resp := postJSON(t, "/accounts", map[string]string{})

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetAccount_Success(t *testing.T) {
	truncate(t)
	created := createAccount(t, "12345678900")

	resp := getJSON(t, "/accounts/"+created.ID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var fetched domain.Account
	decodeBody(t, resp, &fetched)

	if fetched.ID != created.ID {
		t.Errorf("expected id %s, got %s", created.ID, fetched.ID)
	}
	if fetched.DocumentNumber != created.DocumentNumber {
		t.Errorf("expected document_number %s, got %s", created.DocumentNumber, fetched.DocumentNumber)
	}
}

func TestGetAccount_NotFound(t *testing.T) {
	resp := getJSON(t, "/accounts/00000000-0000-0000-0000-000000000000")

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ── Transaction tests ────────────────────────────────────────────────────────

func TestCreateTransaction_NormalPurchase(t *testing.T) {
	truncate(t)
	account := createAccount(t, "12345678900")

	resp := postJSON(t, "/transactions", map[string]any{
		"account_id":        account.ID,
		"operation_type_id": 1,
		"amount":            50.0,
	})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var tx domain.Transaction
	decodeBody(t, resp, &tx)

	if tx.Amount >= 0 {
		t.Errorf("expected negative amount for normal purchase, got %f", tx.Amount)
	}
	if tx.InstallmentPlan != nil {
		t.Error("expected no installment plan for normal purchase")
	}
}

func TestCreateTransaction_Withdrawal(t *testing.T) {
	truncate(t)
	account := createAccount(t, "12345678900")

	resp := postJSON(t, "/transactions", map[string]any{
		"account_id":        account.ID,
		"operation_type_id": 3,
		"amount":            200.0,
	})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var tx domain.Transaction
	decodeBody(t, resp, &tx)

	if tx.Amount >= 0 {
		t.Errorf("expected negative amount for withdrawal, got %f", tx.Amount)
	}
}

func TestCreateTransaction_CreditVoucher(t *testing.T) {
	truncate(t)
	account := createAccount(t, "12345678900")

	resp := postJSON(t, "/transactions", map[string]any{
		"account_id":        account.ID,
		"operation_type_id": 4,
		"amount":            100.0,
	})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var tx domain.Transaction
	decodeBody(t, resp, &tx)

	if tx.Amount <= 0 {
		t.Errorf("expected positive amount for credit voucher, got %f", tx.Amount)
	}
}

func TestCreateTransaction_WithInstallments(t *testing.T) {
	truncate(t)
	account := createAccount(t, "12345678900")

	resp := postJSON(t, "/transactions", map[string]any{
		"account_id":        account.ID,
		"operation_type_id": 2,
		"amount":            1200.0,
		"installments":      12,
	})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var tx domain.Transaction
	decodeBody(t, resp, &tx)

	if tx.Amount >= 0 {
		t.Errorf("expected negative total amount, got %f", tx.Amount)
	}
	if tx.InstallmentPlan == nil {
		t.Fatal("expected installment plan, got nil")
	}
	if len(tx.InstallmentPlan.Schedules) != 12 {
		t.Errorf("expected 12 schedules, got %d", len(tx.InstallmentPlan.Schedules))
	}
	for _, s := range tx.InstallmentPlan.Schedules {
		if s.Amount >= 0 {
			t.Errorf("schedule %d: expected negative amount, got %f", s.InstallmentNo, s.Amount)
		}
	}
}

func TestCreateTransaction_InvalidAccount(t *testing.T) {
	resp := postJSON(t, "/transactions", map[string]any{
		"account_id":        "00000000-0000-0000-0000-000000000000",
		"operation_type_id": 1,
		"amount":            50.0,
	})

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestCreateTransaction_MissingInstallments(t *testing.T) {
	truncate(t)
	account := createAccount(t, "12345678900")

	resp := postJSON(t, "/transactions", map[string]any{
		"account_id":        account.ID,
		"operation_type_id": 2,
		"amount":            1200.0,
		// installments intentionally omitted
	})

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
