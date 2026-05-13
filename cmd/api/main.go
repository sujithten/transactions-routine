// @title           Transactions Routine API
// @version         1.0
// @description     REST service for managing cardholder accounts and financial transactions.
// @host            localhost:8080
// @BasePath        /

package main

import (
	"context"
	"log"
	"net/http"
	"os"

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

	if err := db.RunMigrations(dbURL); err != nil {
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
	installmentRepo := repository.NewInstallmentRepository(pool)

	// services
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


func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
