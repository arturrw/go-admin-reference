package postgres

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arturrw/go-admin-reference/internal/auth"
	"github.com/arturrw/go-admin-reference/internal/store/postgres/db"
)

// Sessions persists login sessions so they survive restarts and work across
// several API instances. Only token hashes are stored.
type Sessions struct {
	q   *db.Queries
	ttl atomic.Int64 // time.Duration; changed at runtime from Settings
}

func NewSessions(pool *pgxpool.Pool, ttl time.Duration) *Sessions {
	s := &Sessions{q: db.New(pool)}
	s.ttl.Store(int64(ttl))
	return s
}

func (s *Sessions) TTL() time.Duration { return time.Duration(s.ttl.Load()) }

// SetTTL changes how long sessions last from their next use on.
func (s *Sessions) SetTTL(ttl time.Duration) { s.ttl.Store(int64(ttl)) }

func (s *Sessions) Create(ctx context.Context, memberID int64) (string, error) {
	token, err := auth.NewToken()
	if err != nil {
		return "", err
	}
	err = s.q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: auth.HashToken(token), MemberID: memberID, ExpiresAt: time.Now().Add(s.TTL()),
	})
	return token, err
}

// Lookup validates the token and slides its expiry forward.
func (s *Sessions) Lookup(ctx context.Context, token string) (int64, error) {
	id, err := s.q.TouchSession(ctx, db.TouchSessionParams{TokenHash: auth.HashToken(token), ExpiresAt: time.Now().Add(s.TTL())})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, auth.ErrNoSession
	}
	return id, err
}

func (s *Sessions) Delete(ctx context.Context, token string) {
	_ = s.q.DeleteSession(ctx, auth.HashToken(token))
}

func (s *Sessions) DeleteMember(ctx context.Context, memberID int64) {
	_ = s.q.DeleteMemberSessions(ctx, memberID)
}

// PurgeExpired deletes stale rows; run it periodically.
func (s *Sessions) PurgeExpired(ctx context.Context) (int64, error) {
	return s.q.DeleteExpiredSessions(ctx)
}
