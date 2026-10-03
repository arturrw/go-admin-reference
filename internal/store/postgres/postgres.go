// Package postgres is the PostgreSQL implementation of httpapi.Store.
//
// Queries live in queries/*.sql and are compiled to type-safe Go by sqlc
// (package db; regenerate with `make sqlc`). Schema changes are goose
// migrations embedded in the binary and applied on startup.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	d "github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/store/postgres/db"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	pool *pgxpool.Pool
	q    *db.Queries
	now  func() time.Time
}

// Open connects, verifies the connection and returns a pool.
func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return pool, nil
}

// Migrate applies pending goose migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()
	return migrateDB(ctx, sqlDB)
}

func migrateDB(ctx context.Context, sqlDB *sql.DB) ([]string, error) {
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, mustSub(migrations, "migrations"))
	if err != nil {
		return nil, err
	}
	results, err := provider.Up(ctx)
	applied := make([]string, 0, len(results))
	for _, r := range results {
		applied = append(applied, r.Source.Path)
	}
	return applied, err
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: db.New(pool), now: time.Now}
}

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// tx runs fn in a transaction, committing on success.
func (s *Store) tx(ctx context.Context, fn func(q *db.Queries) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error { return fn(s.q.WithTx(tx)) })
}

// mapErr turns driver errors into domain errors the HTTP layer understands.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return d.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			switch {
			case strings.Contains(pgErr.ConstraintName, "sku"):
				return d.NewValidationError("sku", "is already used by another product")
			case strings.Contains(pgErr.ConstraintName, "email"):
				return d.NewValidationError("email", "is already a member")
			case strings.Contains(pgErr.ConstraintName, "owner"):
				return d.ErrForbidden
			}
		case "23503": // foreign_key_violation: the referenced row doesn't exist
			return d.ErrNotFound
		case "23514": // check_violation
			return d.NewValidationError(checkField(pgErr.ConstraintName), "is out of range")
		}
	}
	return err
}

// checkField maps a constraint like "products_price_cents_check" to "priceCents".
func checkField(constraint string) string {
	name := strings.TrimSuffix(constraint, "_check")
	if i := strings.Index(name, "_"); i >= 0 {
		name = name[i+1:]
	}
	parts := strings.Split(name, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// escapeLike makes user input literal inside ILIKE '%…%'.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
