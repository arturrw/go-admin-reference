package postgres

import (
	"context"
	"encoding/json"
	"errors"

	d "github.com/arturrw/go-admin-reference/internal/domain"
	"github.com/jackc/pgx/v5"
)

// GetSettings returns the stored settings; fields missing from an older row
// keep their defaults.
func (s *Store) GetSettings(ctx context.Context) (d.Settings, error) {
	out := d.DefaultSettings()
	raw, err := s.q.GetSettings(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return d.Settings{}, err
	}
	return out, json.Unmarshal(raw, &out)
}

func (s *Store) SaveSettings(ctx context.Context, v d.Settings) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.q.SaveSettings(ctx, raw)
}
