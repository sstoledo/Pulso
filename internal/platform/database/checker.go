package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Checker struct {
	Pool *pgxpool.Pool
}

func (c Checker) Check(ctx context.Context) error {
	return c.Pool.Ping(ctx)
}
