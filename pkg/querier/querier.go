package querier

import (
	"context"

	"github.com/avito-tech/go-transaction-manager/pgxv5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Querier struct {
	pool   *pgxpool.Pool
	getter *pgxv5.CtxGetter
}

func New(pool *pgxpool.Pool, getter *pgxv5.CtxGetter) *Querier {
	return &Querier{
		pool:   pool,
		getter: getter,
	}
}

func (q *Querier) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	executor := q.get(ctx)
	return executor.Exec(ctx, sql, args...)
}

func (q *Querier) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	executor := q.get(ctx)
	return executor.Query(ctx, sql, args...)
}

func (q *Querier) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	executor := q.get(ctx)
	return executor.QueryRow(ctx, sql, args...)
}

func (q *Querier) get(ctx context.Context) pgxv5.Tr {
	return q.getter.DefaultTrOrDB(ctx, q.pool)
}
