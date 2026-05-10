package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type poolBeginner struct{ pool *pgxpool.Pool }

func NewPoolBeginner(pool *pgxpool.Pool) Beginner {
	return &poolBeginner{pool: pool}
}

func (p *poolBeginner) Begin(ctx context.Context) (Tx, error) {
	return p.pool.Begin(ctx)
}
