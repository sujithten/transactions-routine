package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ts_project/transactions_routine/internal/domain"
)

type pgxInstallmentRepo struct {
	pool *pgxpool.Pool
}

func NewInstallmentRepository(pool *pgxpool.Pool) InstallmentRepository {
	return &pgxInstallmentRepo{pool: pool}
}

func (r *pgxInstallmentRepo) CreatePlanWithSchedules(ctx context.Context, dbtx DBTX, plan *domain.InstallmentPlan) (*domain.InstallmentPlan, error) {
	planRows, err := dbtx.Query(ctx,
		`INSERT INTO installment_plans (transaction_id, total_installments)
		 VALUES ($1, $2)
		 RETURNING id, transaction_id, total_installments, created_on`,
		plan.TransactionID, plan.TotalInstallments,
	)
	if err != nil {
		return nil, fmt.Errorf("insert installment_plan: %w", err)
	}
	saved, err := pgx.CollectExactlyOneRow(planRows, pgx.RowToAddrOfStructByName[domain.InstallmentPlan])
	if err != nil {
		return nil, fmt.Errorf("scan installment_plan: %w", err)
	}
	saved.Schedules = plan.Schedules

	if err := r.insertSchedules(ctx, dbtx, saved); err != nil {
		return nil, err
	}
	return saved, nil
}

func (r *pgxInstallmentRepo) insertSchedules(ctx context.Context, dbtx DBTX, plan *domain.InstallmentPlan) error {
	for i := range plan.Schedules {
		s := &plan.Schedules[i]
		s.InstallmentPlanID = plan.ID
		rows, err := dbtx.Query(ctx,
			`INSERT INTO installment_schedules (installment_plan_id, installment_no, amount, due_date)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id, installment_plan_id, installment_no, amount, due_date, created_on`,
			s.InstallmentPlanID, s.InstallmentNo, s.Amount, s.DueDate,
		)
		if err != nil {
			return fmt.Errorf("insert schedule %d: %w", s.InstallmentNo, err)
		}
		scanned, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[domain.InstallmentSchedule])
		if err != nil {
			return fmt.Errorf("scan schedule %d: %w", s.InstallmentNo, err)
		}
		plan.Schedules[i] = *scanned
	}
	return nil
}
