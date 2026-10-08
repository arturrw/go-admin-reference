package postgres

import (
	"context"
	"time"

	d "github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/arturrw/go-admin-reference/internal/seed"
	"github.com/arturrw/go-admin-reference/internal/store/postgres/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func toKey(k db.ApiKey) d.APIKey {
	return d.APIKey{
		ID: k.ID, Name: k.Name, Scope: k.Scope, Last4: k.Last4, CreatedBy: k.CreatedBy, CreatedAt: k.CreatedAt,
		LastUsedAt: k.LastUsedAt, RevokedAt: k.RevokedAt, Hash: k.TokenHash,
	}
}

func (s *Store) ListAPIKeys(ctx context.Context) ([]d.APIKey, error) {
	rows, err := s.q.ListAPIKeys(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]d.APIKey, len(rows))
	for i, r := range rows {
		out[i] = toKey(r)
	}
	return out, nil
}

func (s *Store) CreateAPIKey(ctx context.Context, k d.APIKey) (d.APIKey, error) {
	row, err := s.q.CreateAPIKey(ctx, db.CreateAPIKeyParams{Name: k.Name, Scope: k.Scope, Last4: k.Last4, TokenHash: k.Hash, CreatedBy: k.CreatedBy})
	return toKey(row), mapErr(err)
}

func (s *Store) APIKeyByHash(ctx context.Context, hash []byte) (d.APIKey, error) {
	row, err := s.q.APIKeyByHash(ctx, hash)
	return toKey(row), mapErr(err)
}

func (s *Store) TouchAPIKey(ctx context.Context, id int64) { _ = s.q.TouchAPIKey(ctx, id) }

func (s *Store) RevokeAPIKey(ctx context.Context, id int64) (d.APIKey, error) {
	row, err := s.q.RevokeAPIKey(ctx, id)
	return toKey(row), mapErr(err) // no active row → not found
}

// BackfillAPIKeys gives databases created before API keys existed the demo
// keys a fresh seed has. A table that has ever held a key (even a revoked
// one) is left alone. It reports how many keys were added.
func BackfillAPIKeys(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int, error) {
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM api_keys`).Scan(&n); err != nil || n > 0 {
		return 0, err
	}
	keys := seed.APIKeys(now)
	for _, k := range keys {
		if _, err := pool.Exec(ctx,
			`INSERT INTO api_keys (name, scope, last4, token_hash, created_by, created_at, last_used_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			k.Name, k.Scope, k.Last4, k.Hash, k.CreatedBy, k.CreatedAt, k.LastUsedAt); err != nil {
			return 0, err
		}
	}
	return len(keys), nil
}
