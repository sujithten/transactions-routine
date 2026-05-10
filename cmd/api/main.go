// @title           Transactions Routine API
// @version         1.0
// @description     REST service for managing cardholder accounts and financial transactions.
// @host            localhost:8080
// @BasePath        /

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/ts_project/transactions_routine/db"
	_ "github.com/ts_project/transactions_routine/docs"
	"github.com/ts_project/transactions_routine/internal/handler"
	"github.com/ts_project/transactions_routine/internal/repository"
	"github.com/ts_project/transactions_routine/internal/service"
)

func main() {
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/transactions_db?sslmode=disable")
	addr := getEnv("SERVER_ADDR", ":8080")

	if err := runMigrations(dbURL); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	// repositories
	acctRepo := repository.NewAccountRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	opRepo := repository.NewOperationTypeRepository(pool)
	installmentRepo := repository.NewInstallmentRepository(pool)

	// services
	acctSvc := service.NewAccountService(acctRepo)
	txSvc := service.NewTransactionService(repository.NewPoolBeginner(pool), txRepo, opRepo, acctRepo, installmentRepo)

	// handlers
	acctHandler := handler.NewAccountHandler(acctSvc)
	txHandler := handler.NewTransactionHandler(txSvc)

	// router
	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts", acctHandler.CreateAccount)
	mux.HandleFunc("GET /accounts/{accountId}", acctHandler.GetAccount)
	mux.HandleFunc("POST /transactions", txHandler.CreateTransaction)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func runMigrations(dbURL string) error {
	src, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return err
	}
	// golang-migrate's pgx/v5 driver is registered under the "pgx5" scheme
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
